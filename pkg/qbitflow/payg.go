package qbf

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// PayAsYouGoService handles pay-as-you-go subscription operations.
//
// NOTE: PAYG session creation is currently disabled on the API.
// Creating new PAYG sessions is temporarily unavailable; existing PAYG subscriptions
// can still be queried and managed via GetSubscription, GetPaymentHistory, ForceCancel,
// and ExecuteTestBillingCycle.
type PayAsYouGoService struct {
	client *Client
}

func NewPayAsYouGoService(client *Client) *PayAsYouGoService {
	return &PayAsYouGoService{client: client}
}

// OnBehalfOf returns a PayAsYouGoService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *PayAsYouGoService) OnBehalfOf(userID uint64) *PayAsYouGoService {
	return NewPayAsYouGoService(s.client.withOnBehalfOf(userID))
}

// CreateSession is currently disabled — PAYG session creation is not yet available.
//
// func (s *PayAsYouGoService) CreateSession(opts *CreatePAYGSessionOptions) (*qbmodels.LinkResponse, error) {
// 	// PAYG session creation endpoint has been temporarily removed from the API.
// 	// Use SubscriptionService.CreateSession for regular subscriptions.
// 	return nil, qberrors.NewValidationError("PAYG session creation is currently disabled")
// }

// GetSession retrieves a PAYG subscription session by its UUID.
func (s *PayAsYouGoService) GetSession(sessionUUID string) (*qbmodels.SessionCheckout, error) {
	var result qbmodels.SessionCheckout
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscription retrieves a pay-as-you-go subscription by its UUID.
func (s *PayAsYouGoService) GetSubscription(subscriptionUUID string) (*qbmodels.Subscription, error) {
	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSubscriptionByReference retrieves a pay-as-you-go subscription by the reference you assigned
// when creating the session.
func (s *PayAsYouGoService) GetSubscriptionByReference(reference string) (*qbmodels.Subscription, error) {
	if reference == "" {
		return nil, qberrors.NewValidationError("subscription reference cannot be empty")
	}
	var result qbmodels.Subscription
	endpoint := fmt.Sprintf("/transaction/subscription/reference/%s/%s", qbmodels.SubscriptionTypePAYG, reference)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentHistory retrieves the billing history for a PAYG subscription.
func (s *PayAsYouGoService) GetPaymentHistory(subscriptionUUID string) ([]qbmodels.SubscriptionHistory, error) {
	var result []qbmodels.SubscriptionHistory
	endpoint := fmt.Sprintf("/transaction/subscription/history/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ForceCancel immediately cancels a PAYG subscription (use with caution).
func (s *PayAsYouGoService) ForceCancel(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/force-cancel/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecuteTestBillingCycle manually triggers a billing cycle for a PAYG subscription (test mode only).
func (s *PayAsYouGoService) ExecuteTestBillingCycle(subscriptionUUID string) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/transaction/subscription/processing/execute-billing/%s", subscriptionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
