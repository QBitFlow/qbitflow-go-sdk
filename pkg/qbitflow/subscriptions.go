package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// SubscriptionService handles recurring subscription operations
type SubscriptionService struct {
	client *Client
}

// NewSubscriptionService creates the subscription service bound to client.
func NewSubscriptionService(client *Client) *SubscriptionService {
	return &SubscriptionService{client: client}
}

// OnBehalfOf returns a SubscriptionService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *SubscriptionService) OnBehalfOf(userID uint64) *SubscriptionService {
	return NewSubscriptionService(s.client.withOnBehalfOf(userID))
}

// CreateSubscriptionSessionOptions are the options for creating a subscription session.
// Provide either ProductID, ProductReference, or the (ProductName + Description + Price) triplet.
//
// Every string option treats "" like nil: the field is left out of the request.
type CreateSubscriptionSessionOptions struct {
	ProductID   *uint64  // Use an existing product by ID (must be > 0)
	ProductName *string  // Or define a product inline (requires Description and Price); producttext, 2-100 chars
	Description *string  // producttext, 2-500 chars
	Price       *float64 // USD, a finite number greater than 0
	// Frequency is the required billing interval (e.g. {Value: 1, Unit: "months"}); Value
	// must be within 1..4294967295.
	Frequency qbmodels.Duration
	// TrialPeriod is an optional free trial before the first charge; Value within
	// 0..4294967295.
	TrialPeriod *qbmodels.Duration
	// MinPeriods is the optional minimum number of billing periods before cancellation is
	// allowed; 0 means none (it is left out of the request).
	MinPeriods *uint32
	SuccessURL *string // absolute http(s) URL
	CancelURL  *string // absolute http(s) URL

	// CustomerUUID pre-associates a customer by their QBitFlow UUID (a bare UUID, no prefix).
	CustomerUUID *string
	// Reference is your own reference for the subscription (e.g. an order/invoice ID). It is
	// echoed back on the resulting Subscription and in webhooks, and usable with
	// GetSubscriptionByReference.
	Reference *string
	// ProductReference selects the product by your own reference (alternative to ProductID).
	ProductReference *string
	// CustomerReference selects an existing customer by your own reference (alternative to
	// CustomerUUID). A new customer is created during checkout if none matches.
	CustomerReference *string
}

// createSubscriptionSessionRequest is the JSON body for POST /transaction/session-checkout/new/subscription
type createSubscriptionSessionRequest struct {
	ProductID         *uint64            `json:"productId,omitempty"`
	ProductName       *string            `json:"productName,omitempty"`
	Description       *string            `json:"description,omitempty"`
	Price             *float64           `json:"price,omitempty"`
	SuccessURL        *string            `json:"successUrl,omitempty"`
	CancelURL         *string            `json:"cancelUrl,omitempty"`
	CustomerUUID      *string            `json:"customerUUID,omitempty"`
	Reference         *string            `json:"reference,omitempty"`
	ProductReference  *string            `json:"productReference,omitempty"`
	CustomerReference *string            `json:"customerReference,omitempty"`
	Frequency         qbmodels.Duration  `json:"frequency"`
	TrialPeriod       *qbmodels.Duration `json:"trialPeriod,omitempty"`
	MinPeriods        *uint32            `json:"minPeriods,omitempty"`
}

// CreateSession creates a new subscription checkout session and returns a checkout link.
//
// The options are validated against the API's binding rules before any request is sent
// (product selection, inline product text and price, redirect URLs, customer UUID, a
// Frequency of 1..4294967295 with a known unit, TrialPeriod when set); a violation —
// including nil options — returns a *qberrors.ValidationError.
func (s *SubscriptionService) CreateSession(opts *CreateSubscriptionSessionOptions) (*qbmodels.LinkResponse, error) {
	if err := requireArgument(opts == nil, "Subscription session options"); err != nil {
		return nil, err
	}
	if err := validateSessionProduct(opts.ProductID, opts.ProductReference, opts.ProductName,
		opts.Description, opts.Price, opts.SuccessURL, opts.CancelURL, opts.CustomerUUID); err != nil {
		return nil, err
	}
	if err := validateDuration("Frequency", opts.Frequency, 1); err != nil {
		return nil, err
	}
	if opts.TrialPeriod != nil {
		if err := validateDuration("TrialPeriod", *opts.TrialPeriod, 0); err != nil {
			return nil, err
		}
	}

	minPeriods := opts.MinPeriods
	if minPeriods != nil && *minPeriods == 0 {
		minPeriods = nil // 0 means "no minimum", which the API expresses by omission
	}

	req := &createSubscriptionSessionRequest{
		ProductID:         opts.ProductID,
		ProductName:       omitEmpty(opts.ProductName),
		Description:       omitEmpty(opts.Description),
		Price:             opts.Price,
		SuccessURL:        omitEmpty(opts.SuccessURL),
		CancelURL:         omitEmpty(opts.CancelURL),
		CustomerUUID:      omitEmpty(opts.CustomerUUID),
		Reference:         omitEmpty(opts.Reference),
		ProductReference:  omitEmpty(opts.ProductReference),
		CustomerReference: omitEmpty(opts.CustomerReference),
		Frequency:         opts.Frequency,
		TrialPeriod:       opts.TrialPeriod,
		MinPeriods:        minPeriods,
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/new/subscription", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSession retrieves a checkout session by its UUID. The result is the unified
// SessionCheckout; use IsPayment / IsSubscription to tell the kinds apart.
//
// closeToExpireError controls whether the API rejects a session that is close to expiry.
// Pass nil to use the API's default (true).
func (s *SubscriptionService) GetSession(sessionUUID string, closeToExpireError *bool) (*qbmodels.SessionCheckout, error) {
	if err := requireNonEmpty(sessionUUID, "Session UUID"); err != nil {
		return nil, err
	}

	return getSession(s.client, sessionUUID, closeToExpireError)
}

// GetSubscription retrieves a subscription by its UUID — the subscription's own sub@ UUID
// (from a subscription webhook's SubscriptionUUID, or GetSubscriptionByReference), which
// differs from the checkout session's UUID.
func (s *SubscriptionService) GetSubscription(subscriptionUUID string) (*qbmodels.Subscription, error) {
	if err := requireNonEmpty(subscriptionUUID, "Subscription UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", escapePath(subscriptionUUID))
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscriptionByReference retrieves a subscription by the reference you assigned when creating
// the session. Lets you resolve a subscription from your own order/invoice ID without storing
// QBitFlow's UUID.
func (s *SubscriptionService) GetSubscriptionByReference(reference string) (*qbmodels.Subscription, error) {
	if err := requireNonEmpty(reference, "Subscription reference"); err != nil {
		return nil, err
	}

	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/reference/%s/%s", qbmodels.SubscriptionTypeSubscription, escapePath(reference))
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the billing history for a subscription (a public route: an
// unknown subscription returns an empty list rather than an error).
func (s *SubscriptionService) GetPaymentHistory(subscriptionUUID string) ([]qbmodels.SubscriptionHistory, error) {
	if err := requireNonEmpty(subscriptionUUID, "Subscription UUID"); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", escapePath(subscriptionUUID))
	return getList[qbmodels.SubscriptionHistory](s.client, endpoint)
}

// ForceCancel cancels a subscription without the customer signing on-chain. It queues an
// on-chain cancellation and returns immediately; the subscription's status reflects the
// cancellation once the transaction settles.
//
// This request is never retried: a timeout may already have queued the cancellation.
func (s *SubscriptionService) ForceCancel(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	if err := requireNonEmpty(subscriptionUUID, "Subscription UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", escapePath(subscriptionUUID))
	err := s.client.makeActionRequest(endpoint, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle manually triggers a billing cycle for a test subscription (live
// subscriptions are billed automatically). The API enforces the schedule: a subscription
// that is not yet due is refused with a *qberrors.ConflictError (HTTP 409).
//
// This request is never retried: a timeout may already have run the cycle.
func (s *SubscriptionService) ExecuteTestBillingCycle(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	if err := requireNonEmpty(subscriptionUUID, "Subscription UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", escapePath(subscriptionUUID))
	err := s.client.makeActionRequest(endpoint, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
