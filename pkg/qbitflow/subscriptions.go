package qbitflow

import (
	"fmt"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

// SubscriptionService handles subscription-related operations
type SubscriptionService struct {
	client *Client
}

// CreateSubscriptionSessionOptions represents options for creating a subscription session
type CreateSubscriptionSessionOptions struct {
	ProductID    uint64           `binding:"required"` // Product ID for the subscription
	Frequency    models.Duration  `binding:"required"` // Billing frequency for the subscription
	TrialPeriod  *models.Duration // Optional trial period for the subscription
	MinPeriods   *uint32          // Optional minimum number of billing periods before cancellation
	SuccessURL   *string          // Optional success URL (your customer is redirected here after payment)
	CancelURL    *string          // Optional cancel URL (your customer is redirected here if the payment failed)
	WebhookURL   *string          // Optional webhook URL for payment status updates
	CustomerUUID *string          // Optional customer UUID to associate with the subscription
}

// CreateSession creates a new subscription session
func (s *SubscriptionService) CreateSession(opts *CreateSubscriptionSessionOptions) (*models.LinkResponse, error) {
	options := &models.CreateSubscriptionOptions{
		SubscriptionType: models.SubscriptionTypeSubscription,
		Frequency:        opts.Frequency,
		TrialPeriod:      opts.TrialPeriod,
		MinPeriods:       opts.MinPeriods,
	}

	req := &models.CreateSessionRequest{
		ProductID:    &opts.ProductID,
		SuccessURL:   opts.SuccessURL,
		CancelURL:    opts.CancelURL,
		WebhookURL:   opts.WebhookURL,
		CustomerUUID: opts.CustomerUUID,
		Options:      options,
	}

	var result models.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSession retrieves a subscription session by its UUID
func (s *SubscriptionService) GetSession(sessionUUID string) (*models.Session, error) {
	var result models.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a subscription by its UUID
func (s *SubscriptionService) GetSubscription(subscriptionUUID string) (*models.Subscription, error) {
	var result models.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the payment history for a subscription
func (s *SubscriptionService) GetPaymentHistory(subscriptionUUID string) ([]models.SubscriptionHistory, error) {
	var result []models.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel force cancels a subscription immediately (use with caution)
func (s *SubscriptionService) ForceCancel(subscriptionUUID string) (*models.SuccessResponse, error) {
	var result models.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle executes a test billing cycle (only works in test mode)
func (s *SubscriptionService) ExecuteTestBillingCycle(subscriptionUUID string) (*models.StatusResponse, error) {
	var result models.StatusResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
