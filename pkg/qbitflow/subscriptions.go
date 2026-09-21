package qbf

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// SubscriptionService handles recurring subscription operations
type SubscriptionService struct {
	client *Client
}

func NewSubscriptionService(client *Client) *SubscriptionService {
	return &SubscriptionService{client: client}
}

// OnBehalfOf returns a SubscriptionService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *SubscriptionService) OnBehalfOf(userID uint64) *SubscriptionService {
	return NewSubscriptionService(s.client.withOnBehalfOf(userID))
}

// CreateSubscriptionSessionOptions are the options for creating a subscription session.
// Provide either ProductID, ProductReference, or the (ProductName + Description + Price) triplet.
type CreateSubscriptionSessionOptions struct {
	ProductID   *uint64 // Use an existing product by ID
	ProductName *string // Or define a product inline (requires Description and Price)
	Description *string
	Price       *float64
	Frequency   qbmodels.Duration  // Required: billing interval (e.g., {Value:1, Unit:"months"})
	TrialPeriod *qbmodels.Duration // Optional: free trial before first charge
	MinPeriods  *uint32            // Optional: minimum billing periods before cancellation is allowed
	SuccessURL  *string
	CancelURL   *string

	// CustomerUUID pre-associates a customer by their QBitFlow UUID.
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
func (s *SubscriptionService) CreateSession(opts *CreateSubscriptionSessionOptions) (*qbmodels.LinkResponse, error) {
	if opts.ProductID == nil && opts.ProductReference == nil && (opts.ProductName == nil || opts.Description == nil || opts.Price == nil) {
		return nil, qberrors.NewValidationError("either ProductID, ProductReference, or ProductName, Description, and Price must be provided")
	}
	if opts.Frequency.Value <= 0 {
		return nil, qberrors.NewValidationError("Frequency.Value must be greater than zero")
	}
	// Mirror the API's binding rules for an inline product and the redirect URLs, so
	// invalid input is caught before the round-trip.
	if err := validateInlineProduct(opts.ProductName, opts.Description, opts.Price, opts.SuccessURL, opts.CancelURL); err != nil {
		return nil, err
	}

	req := &createSubscriptionSessionRequest{
		ProductID:         opts.ProductID,
		ProductName:       opts.ProductName,
		Description:       opts.Description,
		Price:             opts.Price,
		SuccessURL:        opts.SuccessURL,
		CancelURL:         opts.CancelURL,
		CustomerUUID:      opts.CustomerUUID,
		Reference:         opts.Reference,
		ProductReference:  opts.ProductReference,
		CustomerReference: opts.CustomerReference,
		Frequency:         opts.Frequency,
		TrialPeriod:       opts.TrialPeriod,
		MinPeriods:        opts.MinPeriods,
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/new/subscription", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSession retrieves a checkout session by its UUID.
func (s *SubscriptionService) GetSession(sessionUUID string) (*qbmodels.SessionCheckout, error) {
	var result qbmodels.SessionCheckout
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a subscription by its UUID.
func (s *SubscriptionService) GetSubscription(subscriptionUUID string) (*qbmodels.Subscription, error) {
	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
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
	if reference == "" {
		return nil, qberrors.NewValidationError("subscription reference cannot be empty")
	}
	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/reference/%s/%s", qbmodels.SubscriptionTypeSubscription, reference)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the billing history for a subscription.
func (s *SubscriptionService) GetPaymentHistory(subscriptionUUID string) ([]qbmodels.SubscriptionHistory, error) {
	var result []qbmodels.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel immediately cancels a subscription, bypassing the normal user-signed cancellation flow.
func (s *SubscriptionService) ForceCancel(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle manually triggers a billing cycle for a subscription (test mode only).
func (s *SubscriptionService) ExecuteTestBillingCycle(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
