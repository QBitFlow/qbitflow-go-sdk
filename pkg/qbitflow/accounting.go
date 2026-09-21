package qbf

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// AccountingService handles accounting and financial export operations
type AccountingService struct {
	client *Client
}

func NewAccountingService(client *Client) *AccountingService {
	return &AccountingService{client: client}
}

// OnBehalfOf returns an AccountingService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *AccountingService) OnBehalfOf(userID uint64) *AccountingService {
	return NewAccountingService(s.client.withOnBehalfOf(userID))
}

// AccountingExportParams specifies the date range for an accounting export.
// From and To must be in YYYY-MM-DD format.
type AccountingExportParams struct {
	From string // Start date, inclusive (YYYY-MM-DD)
	To   string // End date, inclusive (YYYY-MM-DD)
}

// Export retrieves accounting events for the given date range as structured JSON.
// For CSV exports, use the QBitFlow web app.
func (s *AccountingService) Export(params AccountingExportParams) ([]qbmodels.AccountingEvent, error) {
	if params.From == "" || params.To == "" {
		return nil, qberrors.NewValidationError("From and To dates are required")
	}

	endpoint := fmt.Sprintf("/accounting/export?from=%s&to=%s&format=json", params.From, params.To)

	var result []qbmodels.AccountingEvent
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
