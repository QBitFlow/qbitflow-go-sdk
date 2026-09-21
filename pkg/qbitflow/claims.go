package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// ClaimService manages account-claim operations.
//
// Background: organizations can create users before those users have claimed their accounts.
// While unclaimed, payments accumulate in a ledger. When the organization creates a claim
// request, the user receives a link to set up their password and wallet, after which the
// owed funds are transferred to their wallet.
type ClaimService struct {
	client *Client
}

func NewClaimService(client *Client) *ClaimService {
	return &ClaimService{client: client}
}

// OnBehalfOf returns a ClaimService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *ClaimService) OnBehalfOf(userID uint64) *ClaimService {
	return NewClaimService(s.client.withOnBehalfOf(userID))
}

// GetClaimRequestByUserID returns the existing claim request for a user (admin only).
// Returns the same link as CreateClaimRequest if one already exists.
func (s *ClaimService) GetClaimRequestByUserID(userID uint64) (*qbmodels.CreateClaimRequestResponse, error) {
	var result qbmodels.CreateClaimRequestResponse
	endpoint := fmt.Sprintf("/user/claim/request/%d", userID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateClaimRequest creates a claim request for a user within the organization (admin only).
// Returns the claim link the user follows to set up their account.
func (s *ClaimService) CreateClaimRequest(userID uint64) (*qbmodels.CreateClaimRequestResponse, error) {
	body := map[string]uint64{"userId": userID}

	var result qbmodels.CreateClaimRequestResponse
	err := s.client.makeRequest("POST", "/user/claim/request", body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetClaimFunds returns all active claim fund entries for the organization —
// funds that have been computed and are awaiting the organization's approval/transfer.
func (s *ClaimService) GetClaimFunds() ([]qbmodels.ClaimFund, error) {
	var result []qbmodels.ClaimFund
	err := s.client.makeRequest("GET", "/user/claim/funds", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TriggerTestClaimFunds manually computes and creates a claim fund entry for a user
// from their ledger entries (test mode only). In live mode this runs automatically on a schedule.
func (s *ClaimService) TriggerTestClaimFunds(userID uint64) (*qbmodels.SuccessResponse, error) {
	var result qbmodels.SuccessResponse
	endpoint := fmt.Sprintf("/user/claim/funds/test-trigger/%d", userID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
