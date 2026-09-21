package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/utils"
)

// RefundService handles refund-related operations
type RefundService struct {
	client *Client
}

func NewRefundService(client *Client) *RefundService {
	return &RefundService{client: client}
}

// OnBehalfOf returns a RefundService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *RefundService) OnBehalfOf(userID uint64) *RefundService {
	return NewRefundService(s.client.withOnBehalfOf(userID))
}

// GetByTransactionUUID retrieves the refund associated with a transaction UUID (public endpoint).
func (s *RefundService) GetByTransactionUUID(transactionUUID string) (*qbmodels.RefundEntry, error) {
	var result qbmodels.RefundEntry
	endpoint := fmt.Sprintf("/transaction/refunds/by-transaction/%s", transactionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves all active refunds for the authenticated organization.
func (s *RefundService) GetAll() ([]qbmodels.RefundEntry, error) {
	var result []qbmodels.RefundEntry
	err := s.client.makeRequest("GET", "/transaction/refunds/all", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetAllInactive retrieves inactive (processed/rejected) refunds with cursor-based pagination.
func (s *RefundService) GetAllInactive(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.RefundEntry], error) {
	endpoint := utils.CursorQueryBuilder("/transaction/refunds/all/inactive", limit, cursor)

	var result qbmodels.CursorData[qbmodels.RefundEntry]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
