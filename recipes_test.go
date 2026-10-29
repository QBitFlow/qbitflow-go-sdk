package qbitflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests of the integration helpers (the webhook router and test signatures are in webhook_router_test.go).

const waitUUID = "pay@01a0f755-f200-7000-8000-000000000001"

// newWaitClient is a client whose sleep advances a fake clock: WaitForCompletion runs instantly.
func newWaitClient(t *testing.T, ts *testServer) (*Client, *[]time.Duration) {
	t.Helper()
	c, err := NewClient(testAPIKey, WithBaseURL(ts.URL), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	clock := time.Unix(vectorT, 0)
	sleeps := &[]time.Duration{}
	c.t.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return clock
	}
	c.t.sleep = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		*sleeps = append(*sleeps, d)
		clock = clock.Add(d)
		return ctx.Err()
	}
	return c, sleeps
}

// statusSequence answers the n-th poll with statuses[n] (the last one after that).
func statusSequence(statuses ...string) func(http.ResponseWriter, *http.Request, int) {
	return func(w http.ResponseWriter, _ *http.Request, n int) {
		s := statuses[min(n, len(statuses)-1)]
		jsonReply(w, 200, fmt.Sprintf(`{"uuid":%q,"status":%q}`, waitUUID, s))
	}
}

func TestWaitForCompletion(t *testing.T) {
	t.Run("completes after N polls", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created", "created", "waitingConfirmation", "completed"))
		c, sleeps := newWaitClient(t, ts)
		st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, nil)
		if err != nil || st.Status != CheckoutSessionStatusCompleted {
			t.Fatalf("%+v %v", st, err)
		}
		if n := len(ts.recorded()); n != 4 {
			t.Errorf("%d polls", n)
		}
		if !slices.Equal(*sleeps, []time.Duration{3 * time.Second, 3 * time.Second, 3 * time.Second}) {
			t.Errorf("sleeps %v", *sleeps)
		}
		if r := ts.recorded()[0]; r.method != http.MethodGet || r.path != "/checkout-sessions/pay@01a0f755-f200-7000-8000-000000000001" {
			t.Errorf("poll %s %s", r.method, r.path)
		}
	})
	t.Run("expires", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created", "expired"))
		c, _ := newWaitClient(t, ts)
		st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, &WaitOptions{Interval: 5 * time.Second})
		if err != nil || st.Status != CheckoutSessionStatusExpired || len(ts.recorded()) != 2 {
			t.Fatalf("%+v %v (%d polls)", st, err, len(ts.recorded()))
		}
	})
	t.Run("already final: one poll, no sleep", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("completed"))
		c, sleeps := newWaitClient(t, ts)
		if st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, nil); err != nil || st.Status != CheckoutSessionStatusCompleted || len(*sleeps) != 0 {
			t.Fatalf("%+v %v %v", st, err, *sleeps)
		}
	})
	t.Run("timeout returns the last status", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created", "waitingConfirmation"))
		c, sleeps := newWaitClient(t, ts)
		st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, &WaitOptions{Timeout: 10 * time.Second, Interval: 3 * time.Second})
		if err != nil || st.Status != CheckoutSessionStatusWaitingConfirmation {
			t.Fatalf("%+v %v", st, err)
		}
		// Polls at 0, 3, 6, 9 and 10 s: the last wait is cut to the time left.
		if !slices.Equal(*sleeps, []time.Duration{3 * time.Second, 3 * time.Second, 3 * time.Second, time.Second}) || len(ts.recorded()) != 5 {
			t.Errorf("sleeps %v, %d polls", *sleeps, len(ts.recorded()))
		}
	})
	t.Run("default timeout is 10 minutes", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created"))
		c, sleeps := newWaitClient(t, ts)
		st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, &WaitOptions{})
		if err != nil || st.Status != CheckoutSessionStatusCreated {
			t.Fatalf("%+v %v", st, err)
		}
		var total time.Duration
		for _, d := range *sleeps {
			total += d
		}
		if total != 10*time.Minute || len(ts.recorded()) != 201 {
			t.Errorf("waited %v over %d polls", total, len(ts.recorded()))
		}
	})
	t.Run("interval floor", func(t *testing.T) {
		for _, interval := range []time.Duration{time.Millisecond, 999 * time.Millisecond, -time.Second} {
			ts := newTestServer(t, statusSequence("created", "completed"))
			c, sleeps := newWaitClient(t, ts)
			if _, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, &WaitOptions{Interval: interval}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(*sleeps, []time.Duration{time.Second}) {
				t.Errorf("interval %v: sleeps %v", interval, *sleeps)
			}
		}
	})
	t.Run("errors propagate", func(t *testing.T) {
		ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
			if n == 0 {
				jsonReply(w, 200, `{"uuid":"`+waitUUID+`","status":"created"}`)
				return
			}
			jsonReply(w, 404, `{"error":{"code":"not_found","message":"session not found"}}`)
		})
		c, _ := newWaitClient(t, ts)
		st, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, nil)
		if st != nil || !errors.As(err, new(*NotFoundError)) {
			t.Fatalf("%+v %v", st, err)
		}
		if _, err := c.CheckoutSessions.WaitForCompletion(context.Background(), "not-a-tx", nil); !errors.As(err, new(*ValidationError)) {
			t.Fatalf("bad uuid: %v", err)
		}
	})
	t.Run("context cancellation", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created"))
		c, _ := newWaitClient(t, ts)
		ctx, cancel := context.WithCancel(context.Background())
		c.t.sleep = func(context.Context, time.Duration) error {
			cancel()
			return ctx.Err()
		}
		st, err := c.CheckoutSessions.WaitForCompletion(ctx, waitUUID, nil)
		if st != nil || !errors.Is(err, context.Canceled) || len(ts.recorded()) != 1 {
			t.Fatalf("%+v %v (%d polls)", st, err, len(ts.recorded()))
		}
		// Already cancelled: nothing is sent.
		if _, err := c.CheckoutSessions.WaitForCompletion(ctx, waitUUID, nil); !errors.Is(err, context.Canceled) || len(ts.recorded()) != 1 {
			t.Fatalf("%v (%d polls)", err, len(ts.recorded()))
		}
	})
	t.Run("request options reach every poll", func(t *testing.T) {
		ts := newTestServer(t, statusSequence("created", "completed"))
		c, _ := newWaitClient(t, ts)
		member := "019cadfd-8900-7a00-8000-0000000000a1"
		if _, err := c.CheckoutSessions.WaitForCompletion(context.Background(), waitUUID, nil, WithOnBehalfOf(member)); err != nil {
			t.Fatal(err)
		}
		for _, r := range ts.recorded() {
			if r.header.Get("On-Behalf-Of") != member {
				t.Errorf("On-Behalf-Of %q", r.header.Get("On-Behalf-Of"))
			}
		}
	})
}

func TestSubscriptionHasAccess(t *testing.T) {
	end := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	s := &Subscription{CurrentPeriodEnd: &end, Status: SubscriptionStatusCancelled}
	cases := []struct {
		at   time.Time
		want bool
	}{
		{end.Add(-time.Nanosecond), true},
		{end, false}, // equal: no access
		{end.Add(time.Nanosecond), false},
		{end.AddDate(-1, 0, 0), true},
	}
	for _, tc := range cases {
		if got := s.HasAccess(tc.at); got != tc.want {
			t.Errorf("HasAccess(%v) = %v", tc.at, got)
		}
	}
	if (&Subscription{Status: SubscriptionStatusActive}).HasAccess(time.Now()) {
		t.Error("nil CurrentPeriodEnd has access")
	}
	var nilSub *Subscription
	if nilSub.HasAccess(time.Now()) {
		t.Error("nil subscription has access")
	}
	// The zero time is now.
	future, past := time.Now().Add(time.Hour), time.Now().Add(-time.Hour)
	if !(&Subscription{CurrentPeriodEnd: &future}).HasAccess(time.Time{}) || (&Subscription{CurrentPeriodEnd: &past}).HasAccess(time.Time{}) {
		t.Error("zero time is not now")
	}
	// Promoted to the events' data.
	changed := &SubscriptionStatusChanged{Subscription: *s}
	if !changed.HasAccess(end.Add(-time.Hour)) {
		t.Error("SubscriptionStatusChanged.HasAccess")
	}
}

// TestSubscriptionHasAccessUntil: AccessUntil decides when set; CurrentPeriodEnd is the fallback
// for a subscription stored before the API sent it.
func TestSubscriptionHasAccessUntil(t *testing.T) {
	end := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	until := end.Add(time.Hour)
	// An active subscription between CurrentPeriodEnd and AccessUntil: its renewal is on its way.
	active := &Subscription{Status: SubscriptionStatusActive, CurrentPeriodEnd: &end, AccessUntil: &until}
	cases := []struct {
		at   time.Time
		want bool
	}{
		{end.Add(-time.Nanosecond), true},
		{end, true},
		{end.Add(30 * time.Minute), true},
		{until.Add(-time.Nanosecond), true},
		{until, false}, // equal: no access
		{until.Add(time.Nanosecond), false},
	}
	for _, tc := range cases {
		if got := active.HasAccess(tc.at); got != tc.want {
			t.Errorf("active HasAccess(%v) = %v", tc.at, got)
		}
	}
	// AccessUntil is read even when it comes before CurrentPeriodEnd: it decides, not the later one.
	early := end.Add(-time.Hour)
	if (&Subscription{CurrentPeriodEnd: &end, AccessUntil: &early}).HasAccess(end.Add(-30 * time.Minute)) {
		t.Error("AccessUntil not read first")
	}
	// Only AccessUntil set.
	if !(&Subscription{AccessUntil: &until}).HasAccess(end) {
		t.Error("AccessUntil alone")
	}
	// Absent AccessUntil (stored before it existed): CurrentPeriodEnd decides.
	stored := &Subscription{Status: SubscriptionStatusActive, CurrentPeriodEnd: &end}
	if !stored.HasAccess(end.Add(-time.Nanosecond)) || stored.HasAccess(end) || stored.HasAccess(end.Add(30*time.Minute)) {
		t.Error("CurrentPeriodEnd fallback")
	}
	// Decoded: accessUntil absent or present.
	old, err := decodeJSON[Subscription]([]byte(`{"status":"active","currentPeriodEnd":"2026-11-01T12:00:00Z"}`))
	if err != nil || old.AccessUntil != nil || !old.HasAccess(end.Add(-time.Minute)) || old.HasAccess(end.Add(time.Minute)) {
		t.Errorf("decoded without accessUntil %+v %v", old, err)
	}
	cur, err := decodeJSON[Subscription]([]byte(`{"status":"active","currentPeriodEnd":"2026-11-01T12:00:00Z","accessUntil":"2026-11-01T13:00:00Z"}`))
	if err != nil || cur.AccessUntil == nil || !cur.AccessUntil.Equal(until) || !cur.HasAccess(end.Add(time.Minute)) {
		t.Errorf("decoded with accessUntil %+v %v", cur, err)
	}
}

func TestFormatAmount(t *testing.T) {
	ok := []struct {
		minUnits string
		decimals int
		want     string
	}{
		{"1500000", 6, "1.5"},
		{"1000000", 6, "1"},
		{"1", 6, "0.000001"},
		{"0", 6, "0"},
		{"-10004200", 6, "-10.0042"},
		{"123", 0, "123"},
		{"000120", 2, "1.2"},
		{"-0", 6, "0"},
		{"-000", 0, "0"},
		{"100", 2, "1"},
		{"999999999999999999999999999999", 18, "999999999999.999999999999999999"},
		{"1", 77, "0." + strings.Repeat("0", 76) + "1"},
	}
	for _, tc := range ok {
		got, err := FormatAmount(tc.minUnits, tc.decimals)
		if err != nil || got != tc.want {
			t.Errorf("FormatAmount(%q, %d) = %q, %v; want %q", tc.minUnits, tc.decimals, got, err, tc.want)
		}
	}
	bad := []struct {
		minUnits string
		decimals int
	}{
		{"", 6}, {"-", 6}, {"1.5", 6}, {"1e6", 6}, {"+1", 6}, {" 1", 6}, {"1,000", 6}, {"--1", 6}, {"١", 6},
		{"1", -1}, {"1", 78},
	}
	for _, tc := range bad {
		if got, err := FormatAmount(tc.minUnits, tc.decimals); !errors.As(err, new(*ValidationError)) {
			t.Errorf("FormatAmount(%q, %d) = %q, %v; want a ValidationError", tc.minUnits, tc.decimals, got, err)
		}
	}

	usdc := &Currency{Symbol: "USDC", Decimals: 6}
	if got, err := usdc.FormatAmount("1500000"); err != nil || got != "1.5" {
		t.Errorf("Currency.FormatAmount = %q, %v", got, err)
	}
	var none *Currency
	if _, err := none.FormatAmount("1"); !errors.As(err, new(*ValidationError)) {
		t.Errorf("nil currency: %v", err)
	}
}

func TestParseAmount(t *testing.T) {
	ok := []struct {
		amount   string
		decimals int
		want     string
	}{
		{"1.5", 6, "1500000"},
		{"0.000001", 6, "1"},
		{"10", 2, "1000"},
		{"-0.5", 2, "-50"},
		{"-0", 2, "0"},
		{"-0.00", 2, "0"},
		{"0", 0, "0"},
		{"007.50", 2, "750"},
		{"123", 0, "123"},
		{"1.000000", 6, "1000000"},
	}
	for _, tc := range ok {
		got, err := ParseAmount(tc.amount, tc.decimals)
		if err != nil || got != tc.want {
			t.Errorf("ParseAmount(%q, %d) = %q, %v; want %q", tc.amount, tc.decimals, got, err, tc.want)
		}
	}
	bad := []struct {
		amount   string
		decimals int
	}{
		{"1.0000001", 6}, {"1.5", 0}, {"", 2}, {"-", 2}, {".5", 2}, {"1.", 2}, {"1e3", 2}, {"1,000", 2},
		{"+1", 2}, {"1.2.3", 2}, {" 1", 2}, {"1", -1}, {"1", 78},
	}
	for _, tc := range bad {
		if got, err := ParseAmount(tc.amount, tc.decimals); !errors.As(err, new(*ValidationError)) {
			t.Errorf("ParseAmount(%q, %d) = %q, %v; want a ValidationError", tc.amount, tc.decimals, got, err)
		}
	}
	// Round trip.
	for _, s := range []string{"1", "0.000001", "-10.0042", "123456789.123456"} {
		units, err := ParseAmount(s, 6)
		if err != nil {
			t.Fatal(err)
		}
		if back, err := FormatAmount(units, 6); err != nil || back != s {
			t.Errorf("round trip %q → %q → %q %v", s, units, back, err)
		}
	}
}

func TestExportWindows(t *testing.T) {
	cases := []struct {
		from, to string
		want     [][2]string
	}{
		// A yearly export: windows of 92 days, both ends included.
		{"2026-01-01", "2026-12-31", [][2]string{
			{"2026-01-01", "2026-04-02"}, {"2026-04-03", "2026-07-03"}, {"2026-07-04", "2026-10-03"}, {"2026-10-04", "2026-12-31"},
		}},
		{"2026-01-01", "2026-04-02", [][2]string{{"2026-01-01", "2026-04-02"}}},                               // 92 days, counted inclusively
		{"2026-01-01", "2026-04-03", [][2]string{{"2026-01-01", "2026-04-02"}, {"2026-04-03", "2026-04-03"}}}, // 93 days
		{"2026-03-03", "2026-03-03", [][2]string{{"2026-03-03", "2026-03-03"}}},
		{"2024-02-01", "2024-06-30", [][2]string{{"2024-02-01", "2024-05-02"}, {"2024-05-03", "2024-06-30"}}}, // leap year
	}
	for _, tc := range cases {
		got, err := exportWindows(tc.from, tc.to)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s → %s: %v %v", tc.from, tc.to, got, err)
		}
		// Every window covers at most 92 days, both ends included, with no gap or overlap.
		for i, w := range got {
			from, _ := time.Parse(time.DateOnly, w[0])
			to, _ := time.Parse(time.DateOnly, w[1])
			if days := int(to.Sub(from).Hours()/24) + 1; days > 92 || days < 1 {
				t.Errorf("%v: %d days", w, days)
			}
			if i > 0 {
				prev, _ := time.Parse(time.DateOnly, got[i-1][1])
				if !prev.AddDate(0, 0, 1).Equal(from) {
					t.Errorf("gap or overlap before %v", w)
				}
			}
		}
	}
}

func TestExportRange(t *testing.T) {
	queries := func(ts *testServer) []string {
		var out []string
		for _, r := range ts.recorded() {
			q, _ := url.ParseQuery(r.query)
			out = append(out, q.Get("from")+".."+q.Get("to")+":"+q.Get("format"))
		}
		return out
	}

	t.Run("JSON yearly", func(t *testing.T) {
		ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, n int) {
			jsonReply(w, 200, fmt.Sprintf(`{"items":[{"type":"payment","txUuid":"pay@%d","grossAmountUsd":"10"}],"hasMore":false}`, n))
		})
		c, _ := newTestClient(t, ts)
		events, err := c.Accounting.ExportJSONRange(context.Background(), "2026-01-01", "2026-12-31")
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			"2026-01-01..2026-04-02:json", "2026-04-03..2026-07-03:json", "2026-07-04..2026-10-03:json", "2026-10-04..2026-12-31:json",
		}
		if got := queries(ts); !slices.Equal(got, want) {
			t.Errorf("requests %v", got)
		}
		if len(events) != 4 || events[0].TxUUID != "pay@0" || events[3].TxUUID != "pay@3" || events[0].GrossAmountUSD != "10" {
			t.Errorf("events %+v", events)
		}
	})
	t.Run("92 days is one request, 93 two", func(t *testing.T) {
		ts := newTestServer(t, staticReply(200, `{"items":[],"hasMore":false}`))
		c, _ := newTestClient(t, ts)
		events, err := c.Accounting.ExportJSONRange(context.Background(), "2026-01-01", "2026-04-02")
		if err != nil || events == nil || len(events) != 0 || len(ts.recorded()) != 1 {
			t.Fatalf("%v %v (%d requests)", events, err, len(ts.recorded()))
		}
		if _, err := c.Accounting.ExportJSONRange(context.Background(), "2026-01-01", "2026-04-03"); err != nil || len(ts.recorded()) != 3 {
			t.Fatalf("%v (%d requests)", err, len(ts.recorded()))
		}
	})
	t.Run("validation before sending", func(t *testing.T) {
		ts := newTestServer(t, staticReply(200, `{"items":[],"hasMore":false}`))
		c, _ := newTestClient(t, ts)
		for _, r := range [][2]string{{"2026-02-01", "2026-01-01"}, {"2026-13-01", "2026-12-01"}, {"", "2026-01-01"}} {
			if _, err := c.Accounting.ExportJSONRange(context.Background(), r[0], r[1]); !errors.As(err, new(*ValidationError)) {
				t.Errorf("%v: %v", r, err)
			}
			if _, err := c.Accounting.ExportCSVRange(context.Background(), r[0], r[1]); !errors.As(err, new(*ValidationError)) {
				t.Errorf("CSV %v: %v", r, err)
			}
		}
		if len(ts.recorded()) != 0 {
			t.Errorf("%d requests sent", len(ts.recorded()))
		}
	})
	t.Run("an error stops it", func(t *testing.T) {
		ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
			if n == 1 {
				jsonReply(w, 400, `{"error":"validation failed","code":"validation_failed","details":{"max":92,"errors":[{"field":"to","message":"too long"}]}}`)
				return
			}
			jsonReply(w, 200, `{"items":[],"hasMore":false}`)
		})
		c, _ := newTestClient(t, ts)
		if _, err := c.Accounting.ExportJSONRange(context.Background(), "2026-01-01", "2026-12-31"); !errors.As(err, new(*ValidationError)) || len(ts.recorded()) != 2 {
			t.Fatalf("%v (%d requests)", err, len(ts.recorded()))
		}
	})
	t.Run("CSV header once", func(t *testing.T) {
		bodies := []string{
			"type,amount\r\npayment,1\r\n",
			"type,amount\r\n", // only a header: adds nothing
			"type,amount\r\nrefund,2\r\npayment,3\r\n",
			"type,amount\r\nfee,4", // no final line ending
		}
		ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write([]byte(bodies[n]))
		})
		c, _ := newTestClient(t, ts)
		csv, err := c.Accounting.ExportCSVRange(context.Background(), "2026-01-01", "2026-12-31")
		if err != nil {
			t.Fatal(err)
		}
		if want := "type,amount\r\npayment,1\r\nrefund,2\r\npayment,3\r\nfee,4"; csv != want {
			t.Errorf("csv %q, want %q", csv, want)
		}
		if got := queries(ts); len(got) != 4 || !strings.HasSuffix(got[0], ":csv") {
			t.Errorf("requests %v", got)
		}
	})
	t.Run("CSV edge cases", func(t *testing.T) {
		var b strings.Builder
		for _, part := range []string{"", "h\n", "h\nr1", "h\nr2\n", "h"} {
			appendCSVWindow(&b, part)
		}
		if got := b.String(); got != "h\nr1\nr2\n" {
			t.Errorf("%q", got)
		}
		b.Reset()
		appendCSVWindow(&b, "h\r\nr1")
		appendCSVWindow(&b, "h\r\nr2\r\n")
		if got := b.String(); got != "h\r\nr1\r\nr2\r\n" {
			t.Errorf("%q", got)
		}
	})
}

func TestNewClientFromEnv(t *testing.T) {
	member := "019cadfd-8900-7a00-8000-0000000000a1"

	t.Run("missing key", func(t *testing.T) {
		t.Setenv("QBITFLOW_API_KEY", "  ")
		_, err := NewClientFromEnv()
		var ve *ValidationError
		if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "QBITFLOW_API_KEY" || !strings.Contains(ve.FieldErrors[0].Message, "QBITFLOW_API_KEY") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("QBITFLOW_API_KEY", testAPIKey)
		t.Setenv("QBITFLOW_BASE_URL", "")
		t.Setenv("QBITFLOW_ON_BEHALF_OF", "")
		c, err := NewClientFromEnv()
		if err != nil || c.t.baseURL != DefaultBaseURL || *c.t.apiKey != testAPIKey || c.onBehalfOf != "" {
			t.Fatalf("%+v %v", c, err)
		}
	})
	t.Run("environment", func(t *testing.T) {
		t.Setenv("QBITFLOW_API_KEY", testAPIKey)
		t.Setenv("QBITFLOW_BASE_URL", "https://api.example.com/v2/")
		t.Setenv("QBITFLOW_ON_BEHALF_OF", member)
		c, err := NewClientFromEnv(WithMaxRetries(0))
		if err != nil || c.t.baseURL != "https://api.example.com/v2" || c.onBehalfOf != member || c.t.maxRetries != 0 {
			t.Fatalf("%+v %v", c, err)
		}
	})
	t.Run("options override", func(t *testing.T) {
		t.Setenv("QBITFLOW_API_KEY", testAPIKey)
		t.Setenv("QBITFLOW_BASE_URL", "https://api.example.com/v2")
		t.Setenv("QBITFLOW_ON_BEHALF_OF", member)
		c, err := NewClientFromEnv(WithBaseURL("https://other.example.com"), WithOnBehalfOf(""))
		if err != nil || c.t.baseURL != "https://other.example.com" || c.onBehalfOf != "" {
			t.Fatalf("%+v %v", c, err)
		}
	})
	t.Run("invalid values", func(t *testing.T) {
		t.Setenv("QBITFLOW_API_KEY", "pk_nope")
		if _, err := NewClientFromEnv(); !errors.As(err, new(*ValidationError)) {
			t.Errorf("bad key: %v", err)
		}
		t.Setenv("QBITFLOW_API_KEY", testAPIKey)
		t.Setenv("QBITFLOW_BASE_URL", "ftp://x")
		if _, err := NewClientFromEnv(); !errors.As(err, new(*ValidationError)) {
			t.Errorf("bad base URL: %v", err)
		}
		t.Setenv("QBITFLOW_BASE_URL", "")
		t.Setenv("QBITFLOW_ON_BEHALF_OF", "not-a-uuid")
		if _, err := NewClientFromEnv(); !errors.As(err, new(*ValidationError)) {
			t.Errorf("bad On-Behalf-Of: %v", err)
		}
	})
}

func TestPlaceholders(t *testing.T) {
	if PlaceholderUUID != "{{UUID}}" || PlaceholderTransactionType != "{{TRANSACTION_TYPE}}" {
		t.Fatal(PlaceholderUUID, PlaceholderTransactionType)
	}
	// Sent literally, never URL-encoded.
	ts := newTestServer(t, staticReply(201, `{"uuid":"pay@x","link":"https://pay.example.com","expiresAt":"2026-10-01T12:00:00Z"}`))
	c, _ := newTestClient(t, ts)
	_, err := c.CheckoutSessions.CreatePayment(context.Background(), &CreatePaymentSessionParams{
		ProductName: "Pro plan", PriceUSD: "1",
		SuccessURL: "https://shop.example.com/thanks?session=" + PlaceholderUUID + "&type=" + PlaceholderTransactionType,
	})
	if err != nil {
		t.Fatal(err)
	}
	var sent struct {
		SuccessURL string `json:"successUrl"`
	}
	if err := json.Unmarshal([]byte(ts.recorded()[0].body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.SuccessURL != "https://shop.example.com/thanks?session={{UUID}}&type={{TRANSACTION_TYPE}}" {
		t.Errorf("successUrl %q", sent.SuccessURL)
	}
}
