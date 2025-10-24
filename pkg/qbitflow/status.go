package qbitflow

import (
	"fmt"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

// TransactionStatusService handles transaction status operations
type TransactionStatusService struct {
	client *Client
}

// GetTransactionStatus retrieves the status of a transaction by its UUID
func (s *TransactionStatusService) GetTransactionStatus(transactionUUID string, transactionType models.TransactionType) (*models.TransactionStatus, error) {
	endpoint := fmt.Sprintf("/transaction/status?transactionUUID=%s&transactionStatusType=%s", transactionUUID, transactionType)

	var result models.TransactionStatus
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
