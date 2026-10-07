package qbf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

const (
	defaultBaseURL    = "https://api.qbitflow.app/v1"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	// retryBaseDelay is the first back-off interval; each further retry doubles it
	// (1s, 2s, 4s for the default three retries).
	retryBaseDelay = time.Second
)

// Client is the low-level HTTP client shared by every service. Use qbitflow.New or
// qbitflow.NewWithConfig rather than constructing it directly.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	// maxRetries is how many times a retriable request is re-sent after its first
	// attempt fails with a network error or a 5xx. 0 disables retries (Config.MaxRetries
	// has already been resolved here: its zero value means the default).
	maxRetries int
	// sleep is the back-off function; replaced in tests so retries do not really wait.
	sleep func(time.Duration)

	// onBehalfOfUser, when non-zero, causes requests to be executed on behalf
	// of the given user via the On-Behalf-Of header. Requires an admin-level
	// (organization) API key. Zero means "act at the organization level".
	onBehalfOfUser uint64
}

// Config represents configuration for the QBitFlow client.
//
// Configure the client before using it: the setters on Client (SetBaseURL, SetTimeout) are
// not synchronised with in-flight requests.
type Config struct {
	// APIKey authenticates every request (sent as X-API-Key). A blank or whitespace-only key
	// makes every request fail with a *qberrors.ValidationError before anything is sent.
	APIKey string
	// BaseURL is the API root. Empty means https://api.qbitflow.app/v1. A trailing slash
	// is stripped.
	BaseURL string
	// Timeout bounds each HTTP attempt. Zero means 30 seconds.
	Timeout time.Duration
	// MaxRetries is the number of additional attempts for a retriable request (an
	// idempotent GET) after a network error or a 5xx response.
	//
	// Following the Go convention that the zero value is the default, 0 means the default
	// of 3 retries. A negative value disables retries, as does DisableRetries.
	MaxRetries int
	// DisableRetries turns the retry policy off: every request is attempted exactly once.
	DisableRetries bool
}

// NewClient creates a new QBitFlow client with the given API key and default configuration.
func NewClient(apiKey string) *Client {
	return NewClientWithConfig(Config{APIKey: apiKey})
}

// NewClientWithConfig creates a new QBitFlow client with custom configuration.
func NewClientWithConfig(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}

	maxRetries := config.MaxRetries
	switch {
	case config.DisableRetries || maxRetries < 0:
		maxRetries = 0
	case maxRetries == 0:
		maxRetries = defaultMaxRetries
	}

	return &Client{
		apiKey:  config.APIKey,
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
			// Redirects are never followed. net/http would replay the request against the
			// redirect target with the X-API-Key header (it strips only Authorization and
			// cookies), turn a POST/PUT/DELETE into a GET, and hand back the target's body
			// as if it were the answer. A 3xx is surfaced as a ServerError instead.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxRetries: maxRetries,
		sleep:      time.Sleep,
	}
}

// withOnBehalfOf returns a shallow copy of the client configured to act on
// behalf of the given user. The underlying http.Client is shared so that
// connection pooling is preserved; only the on-behalf-of target differs.
//
// The copy takes a snapshot of the base URL: a later SetBaseURL on the original does not
// reach copies made before it.
//
// A userID of 0 yields a copy that acts at the organization level (no header).
func (c *Client) withOnBehalfOf(userID uint64) *Client {
	clone := *c
	clone.onBehalfOfUser = userID
	return &clone
}

// OnBehalfOf returns a copy of the client that sends On-Behalf-Of: userID on every request
// (0 = organization level, no header). The copy shares the http.Client and its
// connection pool. It backs qbitflow.QBitFlow.OnBehalfOf and the per-service OnBehalfOf.
func (c *Client) OnBehalfOf(userID uint64) *Client {
	return c.withOnBehalfOf(userID)
}

// response is one HTTP exchange's outcome, decoupled from *http.Response so the body
// can be read once and reused by the retry loop and the error mapper.
type response struct {
	status int
	header http.Header
	body   []byte
}

// makeRequest sends a JSON request and decodes the JSON response into result.
//
// GET requests are retried per the retry policy (see send); POST, PUT and DELETE are
// never retried. Use makeActionRequest for the two GET routes that perform an action.
func (c *Client) makeRequest(method, endpoint string, body any, result any) error {
	return c.makeRequestWith(method, endpoint, body, result, method == http.MethodGet)
}

// makeActionRequest sends a GET that performs a non-idempotent action (force-cancel,
// execute-billing, the claim-funds test trigger). It is never retried: a timeout on the
// first attempt may already have performed the action, and a second attempt would perform
// it twice.
func (c *Client) makeActionRequest(endpoint string, result any) error {
	return c.makeRequestWith(http.MethodGet, endpoint, nil, result, false)
}

func (c *Client) makeRequestWith(method, endpoint string, body any, result any, retriable bool) error {
	resp, err := c.send(method, endpoint, body, retriable)
	if err != nil {
		return err
	}

	// Nothing to decode: the caller does not want a result, or the API answered 204.
	if result == nil || resp.status == http.StatusNoContent {
		return nil
	}

	// A 2xx without a body where a result is expected would otherwise hand the caller a
	// zero-valued struct and a nil error, indistinguishable from a real record.
	if len(bytes.TrimSpace(resp.body)) == 0 {
		return qberrors.NewServerError("empty response body", resp.status, nil)
	}

	if err := json.Unmarshal(resp.body, result); err != nil {
		// A 2xx that is not the JSON we expect is a response-shape failure — a proxy's HTML
		// page served with a 200, or a field of the wrong JSON type — and is reported as a
		// server error rather than handed back as a partially decoded result.
		return qberrors.NewServerError("failed to parse response", resp.status, err)
	}

	return nil
}

// makeTextRequest sends a GET and returns the raw response body as text (used for the
// CSV accounting export, whose body is not JSON).
func (c *Client) makeTextRequest(endpoint string) (string, error) {
	resp, err := c.send(http.MethodGet, endpoint, nil, true)
	if err != nil {
		return "", err
	}
	return string(resp.body), nil
}

// send performs the request with the retry policy and maps any failure to a typed error.
//
// Retry policy (identical across the QBitFlow SDKs):
//   - only retriable requests are re-sent — idempotent GETs, excluding the three GET action
//     routes (force-cancel, execute-billing, claim-funds test-trigger); POST/PUT/DELETE are
//     attempted once;
//   - a retry happens only after a network/timeout error or a 5xx response; 4xx, 429 and
//     3xx are returned immediately (redirects are never followed);
//   - back-off is exponential, 1s * 2^n (1s, 2s, 4s), for Config.MaxRetries retries.
func (c *Client) send(method, endpoint string, body any, retriable bool) (*response, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, qberrors.NewValidationError("API key is required")
	}

	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	url := c.baseURL + endpoint

	var payload []byte
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			// The body cannot be represented as JSON (NaN or ±Inf in a float field, …):
			// the caller's input is invalid, and nothing is sent.
			return nil, qberrors.NewValidationError(fmt.Sprintf("request body cannot be encoded as JSON: %v", err))
		}
		payload = jsonData
	}

	// Validate the URL once, up front. A malformed base URL is a configuration error,
	// not a transient failure, so it must not be retried.
	if _, err := http.NewRequest(method, url, nil); err != nil {
		return nil, qberrors.NewQBitFlowError("failed to create request", 0, err)
	}

	attempts := 1
	if retriable {
		attempts += c.maxRetries
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			c.sleep(retryBaseDelay << (attempt - 1))
		}

		resp, err := c.attempt(method, url, payload)
		if err != nil {
			lastErr = err // network / timeout: retriable
			continue
		}

		if resp.status >= 500 {
			lastErr = c.handleErrorResponse(resp) // 5xx: retriable
			continue
		}

		// Anything else outside 2xx is final. A 3xx reaching this point means the request
		// was not fulfilled (http.Client already followed the redirects it is willing to
		// follow), so parsing its body as a result would hand the caller a zero-valued
		// struct instead of surfacing the problem.
		if resp.status < 200 || resp.status > 299 {
			return nil, c.handleErrorResponse(resp)
		}

		return resp, nil
	}

	return nil, lastErr
}

// attempt performs a single HTTP exchange.
func (c *Client) attempt(method, url string, payload []byte) (*response, error) {
	var reqBody io.Reader
	if payload != nil {
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, qberrors.NewQBitFlowError("failed to create request", 0, err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/csv")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.onBehalfOfUser != 0 {
		req.Header.Set("On-Behalf-Of", strconv.FormatUint(c.onBehalfOfUser, 10))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, qberrors.NewNetworkError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, qberrors.NewNetworkError("failed to read response body", err)
	}

	return &response{status: resp.StatusCode, header: resp.Header, body: respBody}, nil
}

// errorEnvelope is what the SDK could extract from an error body.
type errorEnvelope struct {
	errorMsg string
	message  string
	fields   []qberrors.FieldError
	rawText  string // the body when it was not JSON
}

// parseErrorEnvelope reads both envelopes the API uses — a single message,
//
//	{"error":"resource not found"}
//
// and a field-level list for validation failures,
//
//	{"errors":[{"field":"Price","message":"Price is too short"}]}
//
// — tolerating either, both, or neither. Each key is decoded independently so an
// unexpected shape in one never hides the others.
func parseErrorEnvelope(body []byte) errorEnvelope {
	var env errorEnvelope

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		// Not a JSON object — keep the text, which is usually a gateway or proxy page and
		// is the only clue the caller has.
		env.rawText = strings.TrimSpace(string(body))
		return env
	}

	if v, ok := raw["error"]; ok {
		_ = json.Unmarshal(v, &env.errorMsg)
	}
	if v, ok := raw["message"]; ok {
		_ = json.Unmarshal(v, &env.message)
	}
	if v, ok := raw["errors"]; ok {
		_ = json.Unmarshal(v, &env.fields)
	}

	return env
}

// handleErrorResponse converts a non-2xx response into a typed SDK error.
//
// Message precedence: `error` → joined `errors[]` → `message` → non-JSON body text →
// HTTP status text. The `errors[]` field list is kept on the error for any status.
func (c *Client) handleErrorResponse(resp *response) error {
	env := parseErrorEnvelope(resp.body)

	message := env.errorMsg
	if message == "" {
		message = qberrors.JoinFieldErrors(env.fields)
	}
	if message == "" {
		message = env.message
	}
	if message == "" {
		message = env.rawText
	}
	if message == "" {
		message = http.StatusText(resp.status)
	}

	status := resp.status
	switch {
	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		return qberrors.NewAPIValidationError(message, status, env.fields)
	case status == http.StatusUnauthorized:
		return qberrors.NewAuthenticationError(message, status, env.fields)
	case status == http.StatusForbidden:
		return qberrors.NewForbiddenError(message, status, env.fields)
	case status == http.StatusNotFound:
		return qberrors.NewNotFoundErrorWithFields(message, env.fields)
	case status == http.StatusConflict:
		return qberrors.NewConflictError(message, status, env.fields)
	case status == http.StatusTooManyRequests:
		return qberrors.NewRateLimitError(message, status, env.fields, parseRetryAfter(resp.header))
	case status >= 500, status >= 300 && status < 400, status < 200:
		return qberrors.NewServerErrorWithFields(message, status, env.fields)
	}

	return qberrors.NewQBitFlowErrorWithFields(message, status, env.fields)
}

// parseRetryAfter reads a Retry-After header in either of its two forms — delta-seconds
// ("120") or an HTTP-date ("Wed, 23 Sep 2026 10:00:00 GMT", converted to the seconds
// remaining from now, never below 0). It returns nil when the header is absent or
// malformed.
func parseRetryAfter(header http.Header) *int {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return nil
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return nil
		}
		return &seconds
	}

	when, err := http.ParseTime(value)
	if err != nil {
		return nil
	}
	seconds := int(time.Until(when) / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	return &seconds
}

// SetBaseURL sets a custom base URL for the client (a trailing slash is stripped).
//
// Call it before using the client: it is not synchronised with in-flight requests, and
// OnBehalfOf copies made earlier keep the base URL they were created with.
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = strings.TrimSuffix(baseURL, "/")
}

// SetTimeout sets a custom timeout for HTTP requests. The http.Client is shared with every
// OnBehalfOf copy, so the new timeout applies to them too.
//
// Call it before using the client: changing the timeout while requests are in flight is a
// data race.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}
