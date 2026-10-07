package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// AccountingService handles accounting and financial export operations
type AccountingService struct {
	client *Client
}

// NewAccountingService creates the accounting service bound to client.
func NewAccountingService(client *Client) *AccountingService {
	return &AccountingService{client: client}
}

// OnBehalfOf returns an AccountingService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *AccountingService) OnBehalfOf(userID uint64) *AccountingService {
	return NewAccountingService(s.client.withOnBehalfOf(userID))
}

// AccountingExportParams specifies the date range for an accounting export.
//
// From and To are inclusive, formatted YYYY-MM-DD, real calendar dates, and From must not
// be after To — the API's own rules, checked before the request. The maximum span is
// enforced by the API alone.
type AccountingExportParams struct {
	From string // Start date, inclusive (YYYY-MM-DD)
	To   string // End date, inclusive (YYYY-MM-DD)
}

// Validate checks the date range against the API's rules.
func (p AccountingExportParams) Validate() error {
	return validateAccountingRange(p.From, p.To)
}

// Export retrieves accounting events for the given date range as structured JSON
// (`format=json`). Use ExportCSV for the CSV rendering.
func (s *AccountingService) Export(params AccountingExportParams) ([]qbmodels.AccountingEvent, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/accounting/export?from=%s&to=%s&format=json", escapeQuery(params.From), escapeQuery(params.To))

	return getList[qbmodels.AccountingEvent](s.client, endpoint)
}

// ExportCSV retrieves accounting events for the given date range as CSV text
// (`format=csv`). The body always starts with the header row, followed by one row per
// event; write it to a file as-is. An API error (a JSON error body) is returned as the
// usual typed error.
func (s *AccountingService) ExportCSV(params AccountingExportParams) (string, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("/accounting/export?from=%s&to=%s&format=csv", escapeQuery(params.From), escapeQuery(params.To))

	return s.client.makeTextRequest(endpoint)
}
