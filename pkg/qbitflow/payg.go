package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

// PayAsYouGoService handles pay-as-you-go subscription operations
type PayAsYouGoService struct {
	client *Client
}

func NewPayAsYouGoService(client *Client) *PayAsYouGoService {
	return &PayAsYouGoService{client: client}
}

// CreatePAYGSessionOptions represents options for creating a PAYG subscription session
type CreatePAYGSessionOptions struct {
	ProductID    uint64            `json:"productId" binding:"required"` // Product ID for the PAYG subscription
	Frequency    qbmodels.Duration `json:"frequency" binding:"required"` // Billing frequency for the subscription
	FreeCredits  *float64          `json:"freeCredits,omitempty"`        // Optional free credits to start with
	SuccessURL   *string           `json:"successUrl,omitempty"`         // Optional success URL (your customer is redirected here after payment)
	CancelURL    *string           `json:"cancelUrl,omitempty"`          // Optional cancel URL (your customer is redirected here if the payment failed)
	WebhookURL   *string           `json:"webhookUrl,omitempty"`         // Optional webhook URL for payment status updates
	CustomerUUID *string           `json:"customerUUID,omitempty"`       // Optional customer UUID to associate with the subscription
}

// CreateSession creates a new pay-as-you-go subscription session
func (s *PayAsYouGoService) CreateSession(opts *CreatePAYGSessionOptions) (*qbmodels.LinkResponse, error) {
	options := &qbmodels.CreateSubscriptionOptions{
		SubscriptionType: qbmodels.SubscriptionTypePAYG,
		Frequency:        opts.Frequency,
		FreeCredits:      opts.FreeCredits,
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

// GetSession retrieves a PAYG subscription session by its UUID
func (s *PayAsYouGoService) GetSession(sessionUUID string) (*qbmodels.Session, error) {
	var result qbmodels.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a pay-as-you-go subscription by its UUID
func (s *PayAsYouGoService) GetSubscription(subscriptionUUID string) (*qbmodels.PayAsYouGoSubscription, error) {
	var result qbmodels.PayAsYouGoSubscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the payment history for a PAYG subscription
func (s *PayAsYouGoService) GetPaymentHistory(subscriptionUUID string) ([]qbmodels.SubscriptionHistory, error) {
	var result []qbmodels.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel force cancels a PAYG subscription immediately (use with caution)
func (s *PayAsYouGoService) ForceCancel(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle executes a test billing cycle (only works in test mode)
func (s *PayAsYouGoService) ExecuteTestBillingCycle(subscriptionUUID string) (*qbmodels.StatusLinkResponse, error) {
	var result qbmodels.StatusLinkResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IncreaseUnitsCurrentPeriod increases the units for the current billing period
func (s *PayAsYouGoService) IncreaseUnitsCurrentPeriod(subscriptionUUID string, increaseAmount float64) (*qbmodels.PayAsYouGoSubscription, error) {
	req := &qbmodels.IncreaseUnitsRequest{
		SubscriptionUUID: subscriptionUUID,
		IncreaseByAmount: increaseAmount,
	}

	var result qbmodels.PayAsYouGoSubscription
	err := s.client.makeRequest("POST", "/transaction/subscription/payg/increase-units-current-period", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
