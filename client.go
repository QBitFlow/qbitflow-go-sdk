package qbitflow

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// The environment variables NewClientFromEnv reads.
const (
	envAPIKey     = "QBITFLOW_API_KEY"      // the API key (required)
	envBaseURL    = "QBITFLOW_BASE_URL"     // the API root (optional)
	envOnBehalfOf = "QBITFLOW_ON_BEHALF_OF" // the member the client acts for (optional)
)

// Client is the QBitFlow API client. Create it with NewClient and share it: it is safe for
// concurrent use. Each service field groups the methods of one API area.
type Client struct {
	// Products manages the products (GET/POST/PUT/DELETE /product…).
	Products *ProductsService
	// Customers manages the customers (/customer…).
	Customers *CustomersService
	// CheckoutSessions creates checkout sessions and reads or expires them (/transaction/session-checkout…).
	CheckoutSessions *CheckoutSessionsService
	// Payments reads the one-time payments and the combined payment feed (/transaction/payment…).
	Payments *PaymentsService
	// Failures reads the failed payment attempts (/transaction/failures).
	Failures *FailuresService
	// Subscriptions reads, bills and cancels subscriptions (/transaction/subscription…).
	Subscriptions *SubscriptionsService
	// Refunds lists and initiates refunds (/transaction/refunds…).
	Refunds *RefundsService
	// Members manages the organization's members and their held funds (organization key).
	Members *MembersService
	// Invitations invites members (organization key).
	Invitations *InvitationsService
	// Wallets reads the wallets and the currencies they accept (/wallet…).
	Wallets *WalletsService
	// Accounting exports the accounting events (/accounting/export).
	Accounting *AccountingService
	// Webhooks verifies webhooks and manages the endpoints (Endpoints) and the event log (Events).
	Webhooks *WebhooksService
	// Currencies reads the currency catalog (/utils…, public).
	Currencies *CurrenciesService

	t *transport

	// onBehalfOf is the client's default On-Behalf-Of ("" = organization level).
	onBehalfOf string
	// onBehalfOfErr is set by OnBehalfOf with an invalid UUID: every request then fails with it.
	onBehalfOfErr error
}

// NewClient creates a client for apiKey (sent as X-API-Key). The key must be non-blank and
// start with "sk_"; the rest is opaque. Options: WithBaseURL, WithTimeout, WithMaxRetries,
// WithHTTPClient, WithOnBehalfOf. A bad key or option returns a *ValidationError.
//
// NewClient sends nothing: call Client.Me to check the key online.
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, newFieldError("apiKey", "is required")
	}
	if !strings.HasPrefix(apiKey, "sk_") {
		return nil, newFieldError("apiKey", "must be a QBitFlow API key (sk_…)")
	}

	cfg := clientConfig{baseURL: DefaultBaseURL, timeout: DefaultTimeout, maxRetries: DefaultMaxRetries}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.applyClient(&cfg); err != nil {
			return nil, err
		}
	}

	// The SDK works on its own copy of the http.Client: it never follows redirects. net/http
	// would replay the request (X-API-Key included) at the redirect's target, turn a POST into
	// a GET and hand back the target's body as the answer; a 3xx is a ServerError instead.
	httpClient := &http.Client{}
	if cfg.httpClient != nil {
		copied := *cfg.httpClient
		httpClient = &copied
	}
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	t := &transport{
		apiKey:     apiKey,
		baseURL:    cfg.baseURL,
		httpClient: httpClient,
		timeout:    cfg.timeout,
		maxRetries: cfg.maxRetries,
		sleep:      sleepContext,
		now:        time.Now,
		newKey:     newUUIDv4,
	}
	return newClient(t, cfg.onBehalfOf, nil), nil
}

// NewClientFromEnv creates a client from the environment: the API key from QBITFLOW_API_KEY
// (required: a *ValidationError naming it otherwise), the base URL from QBITFLOW_BASE_URL and
// the default On-Behalf-Of from QBITFLOW_ON_BEHALF_OF when they are set (blank is unset). opts
// come after them, so an explicit WithBaseURL or WithOnBehalfOf overrides the environment.
func NewClientFromEnv(opts ...ClientOption) (*Client, error) {
	apiKey := strings.TrimSpace(os.Getenv(envAPIKey))
	if apiKey == "" {
		return nil, newFieldError(envAPIKey, "is not set (the API key, sk_…)")
	}
	var fromEnv []ClientOption
	if base := strings.TrimSpace(os.Getenv(envBaseURL)); base != "" {
		fromEnv = append(fromEnv, WithBaseURL(base))
	}
	if member := strings.TrimSpace(os.Getenv(envOnBehalfOf)); member != "" {
		fromEnv = append(fromEnv, WithOnBehalfOf(member))
	}
	return NewClient(apiKey, append(fromEnv, opts...)...)
}

// newClient wires a Client and its services around a shared transport.
func newClient(t *transport, onBehalfOf string, onBehalfOfErr error) *Client {
	c := &Client{t: t, onBehalfOf: onBehalfOf, onBehalfOfErr: onBehalfOfErr}
	c.Products = &ProductsService{client: c}
	c.Customers = &CustomersService{client: c}
	c.CheckoutSessions = &CheckoutSessionsService{client: c}
	c.Payments = &PaymentsService{client: c}
	c.Failures = &FailuresService{client: c}
	c.Subscriptions = &SubscriptionsService{client: c}
	c.Refunds = &RefundsService{client: c}
	c.Members = &MembersService{client: c}
	c.Invitations = &InvitationsService{client: c}
	c.Wallets = &WalletsService{client: c}
	c.Accounting = &AccountingService{client: c}
	c.Webhooks = &WebhooksService{
		client:    c,
		Endpoints: &WebhookEndpointsService{client: c},
		Events:    &WebhookEventsService{client: c},
	}
	c.Currencies = &CurrenciesService{client: c}
	return c
}

// OnBehalfOf returns a client that acts in a member's space: every request sends
// On-Behalf-Of: userUUID (a member's userUuid; organization key only). It shares this
// client's connections and configuration; "" returns one at the organization level.
//
// An invalid UUID (or the nil UUID) is reported lazily, so calls can chain: every request of
// the returned client fails with a *ValidationError.
func (c *Client) OnBehalfOf(userUUID string) *Client {
	return newClient(c.t, userUUID, checkOnBehalfOf(userUUID))
}

// Me returns what the API key is (GET /me): its role, its space (organization, member, test
// or live mode) and the member it acts for. The recommended start-up check, e.g. assert
// me.Space.Test in a test environment.
func (c *Client) Me(ctx context.Context, opts ...RequestOption) (*Me, error) {
	return c.callRef[Me](ctx, endpoint{method: http.MethodGet, path: "/me"}, opts)
}
