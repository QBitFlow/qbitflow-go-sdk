package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// RefundService handles refund-related operations
type RefundService struct {
	client *Client
}

// NewRefundService creates the refund service bound to client.
func NewRefundService(client *Client) *RefundService {
	return &RefundService{client: client}
}

// OnBehalfOf returns a RefundService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *RefundService) OnBehalfOf(userID uint64) *RefundService {
	return NewRefundService(s.client.withOnBehalfOf(userID))
}

// GetByTransactionUUID retrieves the refund associated with a transaction UUID (e.g. the
// pay@ id of the refunded payment).
func (s *RefundService) GetByTransactionUUID(transactionUUID string) (*qbmodels.RefundEntry, error) {
	if err := requireNonEmpty(transactionUUID, "Transaction UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.RefundEntry
	endpoint := fmt.Sprintf("/transaction/refunds/by-transaction/%s", escapePath(transactionUUID))
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves the refunds awaiting your response (RespondedAt nil), as a plain list.
func (s *RefundService) GetAll() ([]qbmodels.RefundEntry, error) {
	return getList[qbmodels.RefundEntry](s.client, "/transaction/refunds/all")
}

// GetAllInactive retrieves one page of handled refunds (approved, refused or failed —
// RespondedAt set). limit (optional) must be positive; pass the previous page's NextCursor
// as cursor to fetch the next page.
func (s *RefundService) GetAllInactive(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.RefundEntry], error) {
	return getPage[qbmodels.RefundEntry](s.client, "/transaction/refunds/all/inactive", limit, cursor)
}
