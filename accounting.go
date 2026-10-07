package qbitflow

import (
	"context"
	"net/http"
)

// ExportJSON returns the accounting events between two dates, both YYYY-MM-DD and included
// (GET /accounting/export?format=json). The API allows at most 95 days (400 beyond); the SDK
// checks the dates and from <= to before sending.
func (s *AccountingService) ExportJSON(ctx context.Context, from, to string, opts ...RequestOption) ([]AccountingEvent, error) {
	query, err := accountingQuery(from, to, "json")
	if err != nil {
		return nil, err
	}
	return s.client.call[[]AccountingEvent](ctx, endpoint{method: http.MethodGet, path: "/accounting/export", query: query}, opts)
}

// ExportCSV returns the same export as ExportJSON as CSV text
// (GET /accounting/export?format=csv). Errors are still JSON, and typed as usual.
func (s *AccountingService) ExportCSV(ctx context.Context, from, to string, opts ...RequestOption) (string, error) {
	query, err := accountingQuery(from, to, "csv")
	if err != nil {
		return "", err
	}
	return s.client.callText(ctx, endpoint{method: http.MethodGet, path: "/accounting/export", query: query}, opts)
}
