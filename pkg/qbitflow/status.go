package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// TransactionStatusService handles transaction status operations
type TransactionStatusService struct {
	client *Client
}

func NewTransactionStatusService(client *Client) *TransactionStatusService {
	return &TransactionStatusService{client: client}
}

// OnBehalfOf returns a TransactionStatusService that acts on behalf of the given
// user within the same organization. Requires an admin-level (organization) API key.
func (s *TransactionStatusService) OnBehalfOf(userID uint64) *TransactionStatusService {
	return NewTransactionStatusService(s.client.withOnBehalfOf(userID))
}

// GetTransactionStatus polls the current status of a transaction by UUID and type.
func (s *TransactionStatusService) GetTransactionStatus(transactionUUID string, transactionType qbmodels.TransactionType) (*qbmodels.TransactionStatus, error) {
	endpoint := fmt.Sprintf("/transaction/status?txUUID=%s&txType=%s", transactionUUID, transactionType)

	var result qbmodels.TransactionStatus
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTransactionStatusWSURL returns the WebSocket URL to subscribe to real-time status updates
// for a transaction. Connect to this URL with a WebSocket client.
func (s *TransactionStatusService) GetTransactionStatusWSURL(transactionUUID string, transactionType qbmodels.TransactionType) string {
	return fmt.Sprintf("%s/transaction/status/ws?txUUID=%s&txType=%s",
		s.client.baseURL, transactionUUID, transactionType)
}
