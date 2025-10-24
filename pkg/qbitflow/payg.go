package qbitflow

import (
	"fmt"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

// PayAsYouGoService handles pay-as-you-go subscription operations
type PayAsYouGoService struct {
	client *Client
}

// CreatePAYGSessionOptions represents options for creating a PAYG subscription session
type CreatePAYGSessionOptions struct {
	ProductID    uint64          `json:"productId" binding:"required"` // Product ID for the PAYG subscription
	Frequency    models.Duration `json:"frequency" binding:"required"` // Billing frequency for the subscription
	FreeCredits  *float64        `json:"freeCredits,omitempty"`        // Optional free credits to start with
	SuccessURL   *string         `json:"successUrl,omitempty"`         // Optional success URL (your customer is redirected here after payment)
	CancelURL    *string         `json:"cancelUrl,omitempty"`          // Optional cancel URL (your customer is redirected here if the payment failed)
	WebhookURL   *string         `json:"webhookUrl,omitempty"`         // Optional webhook URL for payment status updates
	CustomerUUID *string         `json:"customerUUID,omitempty"`       // Optional customer UUID to associate with the subscription
}

// CreateSession creates a new pay-as-you-go subscription session
func (s *PayAsYouGoService) CreateSession(opts *CreatePAYGSessionOptions) (*models.LinkResponse, error) {
	options := &models.CreateSubscriptionOptions{
		SubscriptionType: models.SubscriptionTypePAYG,
		Frequency:        opts.Frequency,
		FreeCredits:      opts.FreeCredits,
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

// GetSession retrieves a PAYG subscription session by its UUID
func (s *PayAsYouGoService) GetSession(sessionUUID string) (*models.Session, error) {
	var result models.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a pay-as-you-go subscription by its UUID
func (s *PayAsYouGoService) GetSubscription(subscriptionUUID string) (*models.PayAsYouGoSubscription, error) {
	var result models.PayAsYouGoSubscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the payment history for a PAYG subscription
func (s *PayAsYouGoService) GetPaymentHistory(subscriptionUUID string) ([]models.SubscriptionHistory, error) {
	var result []models.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel force cancels a PAYG subscription immediately (use with caution)
func (s *PayAsYouGoService) ForceCancel(subscriptionUUID string) (*models.SuccessResponse, error) {
	var result models.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle executes a test billing cycle (only works in test mode)
func (s *PayAsYouGoService) ExecuteTestBillingCycle(subscriptionUUID string) (*models.StatusResponse, error) {
	var result models.StatusResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IncreaseUnitsCurrentPeriod increases the units for the current billing period
func (s *PayAsYouGoService) IncreaseUnitsCurrentPeriod(subscriptionUUID string, increaseAmount float64) (*models.PayAsYouGoSubscription, error) {
	req := &models.IncreaseUnitsRequest{
		SubscriptionUUID: subscriptionUUID,
		IncreaseByAmount: increaseAmount,
	}

	var result models.PayAsYouGoSubscription
	err := s.client.makeRequest("POST", "/transaction/subscription/payg/increase-units-current-period", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
