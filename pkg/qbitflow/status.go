package qbf

import (
	"fmt"
	"strings"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v3/pkg/models"
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
	endpoint := fmt.Sprintf("/transaction/status?txUUID=%s&txType=%s", escapeQuery(transactionUUID), escapeQuery(string(transactionType)))

	var result qbmodels.TransactionStatus
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTransactionStatusWSURL returns the WebSocket URL to subscribe to real-time status updates
// for a transaction. Connect to this URL with a WebSocket client.
//
// The scheme is rewritten from http(s) to ws(s). The client's base URL is an HTTP endpoint,
// and WebSocket dialers reject an http:// URL outright ("malformed ws or wss URL"), so the
// returned value would not be usable without it.
//
// The status endpoint is public, so the URL carries no credential and needs no API key.
//
//	wsURL := client.TransactionStatus.GetTransactionStatusWSURL(uuid, qbmodels.TransactionTypeOneTimePayment)
//	// wss://api.qbitflow.app/v1/transaction/status/ws?txUUID=...&txType=payment
func (s *TransactionStatusService) GetTransactionStatusWSURL(transactionUUID string, transactionType qbmodels.TransactionType) string {
	base := s.client.baseURL
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}

	return fmt.Sprintf("%s/transaction/status/ws?txUUID=%s&txType=%s",
		base, escapeQuery(transactionUUID), escapeQuery(string(transactionType)))
}
