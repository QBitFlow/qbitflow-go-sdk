package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// TransactionStatusService reads the status of a transaction. The status route is public
// (no authentication), which is why the checkout page can poll it.
//
// To react to completed transactions, prefer webhooks (see WebhookService): QBitFlow
// notifies you, and nothing has to poll. GetTransactionStatus is the on-demand check, for
// example on your success-redirect page.
type TransactionStatusService struct {
	client *Client
}

// NewTransactionStatusService creates the transaction-status service bound to client.
func NewTransactionStatusService(client *Client) *TransactionStatusService {
	return &TransactionStatusService{client: client}
}

// OnBehalfOf returns a TransactionStatusService that acts on behalf of the given
// user within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *TransactionStatusService) OnBehalfOf(userID uint64) *TransactionStatusService {
	return NewTransactionStatusService(s.client.withOnBehalfOf(userID))
}

// GetTransactionStatus polls the current status of a transaction by UUID and type.
//
// A session that exists but has not been processed yet (checkout created, nothing paid)
// is reported by the API as a *qberrors.NotFoundError, as is an unknown UUID.
func (s *TransactionStatusService) GetTransactionStatus(transactionUUID string, transactionType qbmodels.TransactionType) (*qbmodels.TransactionStatus, error) {
	if err := requireNonEmpty(transactionUUID, "Transaction UUID"); err != nil {
		return nil, err
	}
	if err := requireNonEmpty(string(transactionType), "Transaction type"); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/transaction/status?txUUID=%s&txType=%s", escapeQuery(transactionUUID), escapeQuery(string(transactionType)))

	var result qbmodels.TransactionStatus
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
