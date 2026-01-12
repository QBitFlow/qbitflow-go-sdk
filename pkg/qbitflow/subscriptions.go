package qbitflow

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

// SubscriptionService handles subscription-related operations
type SubscriptionService struct {
	client *Client
}

// CreateSubscriptionSessionOptions represents options for creating a subscription session
type CreateSubscriptionSessionOptions struct {
	ProductID    uint64             `binding:"required"` // Product ID for the subscription
	Frequency    qbmodels.Duration  `binding:"required"` // Billing frequency for the subscription
	TrialPeriod  *qbmodels.Duration // Optional trial period for the subscription
	MinPeriods   *uint32            // Optional minimum number of billing periods before cancellation
	SuccessURL   *string            // Optional success URL (your customer is redirected here after payment)
	CancelURL    *string            // Optional cancel URL (your customer is redirected here if the payment failed)
	WebhookURL   *string            // Optional webhook URL for payment status updates
	CustomerUUID *string            // Optional customer UUID to associate with the subscription
}

// CreateSession creates a new subscription session
func (s *SubscriptionService) CreateSession(opts *CreateSubscriptionSessionOptions) (*qbmodels.LinkResponse, error) {
	options := &qbmodels.CreateSubscriptionOptions{
		SubscriptionType: qbmodels.SubscriptionTypeSubscription,
		Frequency:        opts.Frequency,
		TrialPeriod:      opts.TrialPeriod,
		MinPeriods:       opts.MinPeriods,
	}

	req := &qbmodels.CreateSessionRequest{
		ProductID:    &opts.ProductID,
		SuccessURL:   opts.SuccessURL,
		CancelURL:    opts.CancelURL,
		WebhookURL:   opts.WebhookURL,
		CustomerUUID: opts.CustomerUUID,
		Options:      options,
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSession retrieves a subscription session by its UUID
func (s *SubscriptionService) GetSession(sessionUUID string) (*qbmodels.Session, error) {
	var result qbmodels.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a subscription by its UUID
func (s *SubscriptionService) GetSubscription(subscriptionUUID string) (*qbmodels.Subscription, error) {
	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the payment history for a subscription
func (s *SubscriptionService) GetPaymentHistory(subscriptionUUID string) ([]qbmodels.SubscriptionHistory, error) {
	var result []qbmodels.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel force cancels a subscription immediately (use with caution)
func (s *SubscriptionService) ForceCancel(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle executes a test billing cycle (only works in test mode)
func (s *SubscriptionService) ExecuteTestBillingCycle(subscriptionUUID string) (*qbmodels.StatusLinkResponse, error) {
	var result qbmodels.StatusLinkResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
