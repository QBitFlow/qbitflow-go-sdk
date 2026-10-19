package qbitflow

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// maxExportDays is the longest window one export request covers: to - from <= 95 days.
const maxExportDays = 95

// ExportJSON returns the accounting events between two dates, both YYYY-MM-DD and included
// (GET /accounting/export?format=json). The dates are the transactions' (their block's day, in
// UTC), so consecutive exports never miss nor repeat an event. Events are ordered by TxTimeUTC,
// then Type, then PaymentUUID. The API allows at most 95 days (400 beyond); the SDK checks the
// dates and from <= to before sending.
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

// ExportJSONRange is ExportJSON over any range: it checks the dates like ExportJSON, splits
// the range into consecutive windows of at most 95 days ([from, from+95d], the next one
// starting the day after), requests them in order and returns every window's events, in
// order. A range of 95 days or less is one request. The first error stops it.
func (s *AccountingService) ExportJSONRange(ctx context.Context, from, to string, opts ...RequestOption) ([]AccountingEvent, error) {
	windows, err := exportWindows(from, to)
	if err != nil {
		return nil, err
	}
	var all []AccountingEvent
	for _, w := range windows {
		events, err := s.ExportJSON(ctx, w[0], w[1], opts...)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
	}
	if all == nil {
		all = []AccountingEvent{}
	}
	return all, nil
}

// ExportCSVRange is ExportCSV over any range, split like ExportJSONRange: the first window's
// header line once, then every window's rows, with the line endings the API sent.
func (s *AccountingService) ExportCSVRange(ctx context.Context, from, to string, opts ...RequestOption) (string, error) {
	windows, err := exportWindows(from, to)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, w := range windows {
		csv, err := s.ExportCSV(ctx, w[0], w[1], opts...)
		if err != nil {
			return "", err
		}
		appendCSVWindow(&out, csv)
	}
	return out.String(), nil
}

// appendCSVWindow appends one window's CSV: whole when it is the first with content, else
// without its header line (a window with only its header adds nothing).
func appendCSVWindow(out *strings.Builder, csv string) {
	if out.Len() == 0 {
		out.WriteString(csv)
		return
	}
	header, rows, found := strings.Cut(csv, "\n")
	if !found || rows == "" {
		return
	}
	if acc := out.String(); !strings.HasSuffix(acc, "\n") {
		// The previous window's last row had no line ending: give it this window's.
		if strings.HasSuffix(header, "\r") {
			out.WriteString("\r\n")
		} else {
			out.WriteString("\n")
		}
	}
	out.WriteString(rows)
}

// exportWindows checks an export range and splits it into windows of at most 95 days, on
// calendar dates.
func exportWindows(from, to string) ([][2]string, error) {
	if _, err := accountingQuery(from, to, "json"); err != nil {
		return nil, err
	}
	start, _ := time.Parse(time.DateOnly, from)
	end, _ := time.Parse(time.DateOnly, to)
	var windows [][2]string
	for !start.After(end) {
		windowEnd := start.AddDate(0, 0, maxExportDays)
		if windowEnd.After(end) {
			windowEnd = end
		}
		windows = append(windows, [2]string{start.Format(time.DateOnly), windowEnd.Format(time.DateOnly)})
		start = windowEnd.AddDate(0, 0, 1)
	}
	return windows, nil
}
