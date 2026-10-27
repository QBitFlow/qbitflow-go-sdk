package qbitflow

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestPathEscaping(t *testing.T) {
	cases := map[string]string{
		pathf("/webhooks/events/%s", "evt_a/b c"):          "/webhooks/events/evt_a%2Fb%20c",
		pathf("/payments/%s", "pay@019c-1"):                "/payments/pay@019c-1",
		pathf("/webhooks/events/%s", "a+b@example.com"):    "/webhooks/events/a+b@example.com",
		pathf("/webhooks/events/%s", ".."):                 "/webhooks/events/%2E%2E",
		pathf("/webhooks/events/%s", "."):                  "/webhooks/events/%2E",
		pathf("/webhooks/events/%s", "v1..2"):              "/webhooks/events/v1..2",
		pathf("/x/%s/y/%s", "?q=1#f", "%41"):               "/x/%3Fq=1%23f/y/%2541",
		pathf("/subscriptions/%s/%s", "sub@019c", "bills"): "/subscriptions/sub@019c/bills",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("pathf = %q, want %q", got, want)
		}
	}

	// The escaped path reaches the server as is (a / in an id is never a separator).
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	if _, err := c.send(context.Background(), get(pathf("/webhooks/events/%s", "a/b")), nil); err != nil {
		t.Fatal(err)
	}
	if got := ts.recorded()[0].path; got != "/webhooks/events/a%2Fb" {
		t.Errorf("server saw %q", got)
	}
}

func TestQueryEncoding(t *testing.T) {
	after := time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("CEST", 2*3600))
	before := time.Date(2026, 10, 5, 0, 0, 0, 500, time.UTC)
	p := &PaymentListParams{
		Limit: 25, Cursor: "djF8Y3Vyc29y", CustomerUUID: memberUUID, CreatedAfter: &after,
		CreatedBefore: &before, IncludeMembers: true, Refunded: ptr(false), Reference: "order-1042",
	}
	q := p.values()
	if got := q.Get("createdAfter"); got != "2026-10-04T12:00:00+02:00" {
		t.Errorf("createdAfter = %q", got)
	}
	if got := q.Get("createdBefore"); got != "2026-10-05T00:00:00.0000005Z" {
		t.Errorf("createdBefore = %q", got)
	}
	encoded := q.Encode()
	want := "createdAfter=2026-10-04T12%3A00%3A00%2B02%3A00&createdBefore=2026-10-05T00%3A00%3A00.0000005Z&cursor=djF8Y3Vyc29y&customerUuid=" +
		memberUUID + "&includeMembers=true&limit=25&reference=order-1042&refunded=false"
	if encoded != want {
		t.Errorf("query =\n%s\nwant\n%s", encoded, want)
	}

	// Unset filters are omitted.
	if got := (&PaymentListParams{}).values().Encode(); got != "" {
		t.Errorf("empty params encode to %q", got)
	}
	if (*PaymentListParams)(nil).values() != nil {
		t.Error("nil params encode to nil")
	}

	// Sent on the wire with the + escaped.
	ts := newTestServer(t, staticReply(200, `{"items":[],"hasMore":false}`))
	c, _ := newTestClient(t, ts)
	if _, err := c.call[Page[Payment]](context.Background(), endpoint{method: http.MethodGet, path: "/payments", query: q}, nil); err != nil {
		t.Fatal(err)
	}
	raw := ts.recorded()[0].query
	parsed, _ := url.ParseQuery(raw)
	if parsed.Get("createdAfter") != "2026-10-04T12:00:00+02:00" {
		t.Errorf("server parsed createdAfter %q from %q", parsed.Get("createdAfter"), raw)
	}
}

func TestQueryEncodersPerParams(t *testing.T) {
	cases := []struct {
		got  url.Values
		want string
	}{
		{(&ProductListParams{IncludeHidden: true, Subscription: ptr(true)}).values(), "includeHidden=true&subscription=true"},
		{(&ProductListParams{Limit: 50, Cursor: "c", Reference: "pro"}).values(), "cursor=c&limit=50&reference=pro"},
		{(&ProductListParams{}).values(), ""},
		{(&CustomerListParams{Limit: 5, Email: "a+b@x.io", Verified: ptr(false)}).values(), "email=a%2Bb%40x.io&limit=5&verified=false"},
		{(&CustomerListParams{Reference: "crm:1"}).values(), "reference=crm%3A1"},
		{(&TransactionListParams{Type: TransactionKindBill, SubscriptionUUID: "sub@x", Refunded: ptr(true)}).values(), "refunded=true&subscriptionUuid=sub%40x&type=bill"},
		{(&FailureListParams{Kind: FailureKindBill, Category: FailureCategoryOther, UserUUID: memberUUID}).values(), "category=other&kind=bill&userUuid=" + memberUUID},
		{(&SubscriptionListParams{Status: SubscriptionStatusPastDue, Reference: "r-1", Cursor: "c"}).values(), "cursor=c&reference=r-1&status=pastDue"},
		{(&BillListParams{Limit: 100}).values(), "limit=100"},
		{(&RefundListParams{IncludeMembers: ptr(false), Held: ptr(true)}).values(), "held=true&includeMembers=false"},
		{(&RefundListParams{Limit: 3, Cursor: "c", UserUUID: memberUUID}).values(), "cursor=c&limit=3&userUuid=" + memberUUID},
		{(&RefundListParams{Status: []RefundStatus{RefundStatusApproved, RefundStatusRejected}, TxUUID: "pay@x"}).values(), "status=approved&status=rejected&txUuid=pay%40x"},
		{(&MemberListParams{Limit: -1}).values(), "limit=-1"},
		{(&InvitationListParams{Status: InvitationStatusPending}).values(), "status=pending"},
		{(&EventListParams{Cursor: "djF8ZXZ0", Type: EventTypePaymentCompleted, IncludeMembers: true}).values(), "cursor=djF8ZXZ0&includeMembers=true&type=payment.completed"},
		{(&EndpointListParams{Limit: 10, Cursor: "c"}).values(), "cursor=c&limit=10"},
		{(&ReadParams{IncludeMembers: true}).values(), "includeMembers=true"},
		{(&CancelSubscriptionParams{Immediate: ptr(false)}).values(), "immediate=false"},
		{(&CancelSubscriptionParams{}).values(), ""},
		{(&WalletListParams{WithBalances: true}).values(), "withBalances=true"},
		{(&AcceptedCurrenciesParams{UserUUID: memberUUID}).values(), "userUuid=" + memberUUID},
		{(&CurrencyListParams{Test: true}).values(), "test=true"},
		{(&CurrencyListParams{Main: true}).values(), "main=true"},
	}
	for i, tc := range cases {
		if got := tc.got.Encode(); got != tc.want {
			t.Errorf("case %d: %q, want %q", i, got, tc.want)
		}
	}

	q, err := accountingQuery("2026-06-01", "2026-09-01", "csv")
	if err != nil || q.Encode() != "format=csv&from=2026-06-01&to=2026-09-01" {
		t.Errorf("accountingQuery = %v, %v", q, err)
	}
	for _, bad := range [][2]string{{"2026-09-02", "2026-09-01"}, {"2026-02-30", "2026-03-01"}, {"", "2026-01-01"}, {"2026/01/01", "2026-01-02"}} {
		if _, err := accountingQuery(bad[0], bad[1], "json"); err == nil {
			t.Errorf("accountingQuery(%q, %q) accepted", bad[0], bad[1])
		}
	}
}

func TestBodyEncodingFailures(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	cases := []struct {
		body  any
		field string
	}{
		{map[string]any{"priceUsd": math.NaN()}, ""},
		{map[string]any{"price": math.Inf(1)}, ""},
		{CreateCustomerParams{FirstName: "Ad\xffa", Email: "a@b.co"}, "firstName"},
		{CreateProductParams{Name: "Pro", PriceUSD: "1", Metadata: map[string]string{"k": "v\xff"}}, "metadata.k"},
		{UpdateCustomerParams{Address: ptr("bad \xfe")}, "address"},
		{CreateSubscriptionSessionParams{ProductUUID: memberUUID, Frequency: &Duration{1, "mon\xffths"}}, "frequency.unit"},
		{map[string]any{"bad\xffkey": 1}, ""},
		{CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{"payment.completed", "x\xff"}}, "events[1]"},
		// A params type with its own MarshalJSON is still named by its wire names.
		{&CreateInvitationParams{Email: "a\xff@b.co"}, "email"},
		{&CreateInvitationParams{Email: "a@b.co", RedirectURL: "https://x.io/\xff"}, "redirectUrl"},
	}
	for i, tc := range cases {
		_, err := c.send(context.Background(), endpoint{method: http.MethodPost, path: "/x", body: tc.body}, nil)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("case %d: want *ValidationError, got %v", i, err)
			continue
		}
		if tc.field != "" && (len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != tc.field) {
			t.Errorf("case %d: fields %+v, want %s", i, ve.FieldErrors, tc.field)
		}
	}
	if n := len(ts.recorded()); n != 0 {
		t.Errorf("%d requests sent", n)
	}

	// Valid UTF-8, including U+FFFD itself, is sent untouched.
	if _, err := c.send(context.Background(), endpoint{method: http.MethodPost, path: "/x", body: map[string]string{"name": "O’Brien ☃ �"}}, nil); err != nil {
		t.Fatal(err)
	}
	if got := ts.recorded()[0].body; got != `{"name":"O’Brien ☃ `+"�"+`"}` {
		t.Errorf("body %q", got)
	}
}

type decodeTarget struct {
	S    string             `json:"s"`
	N    int                `json:"n"`
	U    uint64             `json:"u"`
	F    float64            `json:"f"`
	B    bool               `json:"b"`
	T    time.Time          `json:"t"`
	P    *time.Time         `json:"p"`
	L    []int              `json:"l"`
	O    struct{ X int }    `json:"o"`
	M    map[string]string  `json:"m"`
	Enum SubscriptionStatus `json:"enum"`
	Dec  string             `json:"dec"`
}

func TestDecodingPolicy(t *testing.T) {
	ok := []struct {
		name, body string
		check      func(decodeTarget) bool
	}{
		{"absent fields are zero", `{}`, func(v decodeTarget) bool {
			return v.S == "" && v.N == 0 && !v.B && v.T.IsZero() && v.P == nil && v.L == nil && v.O.X == 0
		}},
		{"null fields are zero", `{"s":null,"n":null,"f":null,"b":null,"t":null,"p":null,"l":null,"o":null,"m":null}`, func(v decodeTarget) bool {
			return v.S == "" && v.N == 0 && v.T.IsZero() && v.P == nil && v.L == nil
		}},
		{"unknown keys ignored", `{"zzz":{"a":[1]},"s":"x"}`, func(v decodeTarget) bool { return v.S == "x" }},
		{"unknown enum kept", `{"enum":"hibernating"}`, func(v decodeTarget) bool { return v.Enum == "hibernating" }},
		{"int into float", `{"f":3}`, func(v decodeTarget) bool { return v.F == 3 }},
		{"integral float into int", `{"n":2.0,"u":1e3,"f":1.5}`, func(v decodeTarget) bool { return v.N == 2 && v.U == 1000 && v.F == 1.5 }},
		{"decimal strings stay strings", `{"dec":"-10004200.000001"}`, func(v decodeTarget) bool { return v.Dec == "-10004200.000001" }},
		{"any RFC 3339 offset and precision (the API sends Z, ms)", `{"t":"2026-09-13T21:23:26.620071+02:00"}`, func(v decodeTarget) bool {
			return v.T.Equal(time.Date(2026, 9, 13, 19, 23, 26, 620071000, time.UTC))
		}},
	}
	for _, tc := range ok {
		v, err := decodeResponse[decodeTarget](&response{status: 200, header: http.Header{}, body: []byte(tc.body)})
		if err != nil || !tc.check(v) {
			t.Errorf("%s: %+v, %v", tc.name, v, err)
		}
	}

	bad := map[string]string{
		"empty body":          ``,
		"blank body":          "  \n",
		"html":                `<html>ok</html>`,
		"string for int":      `{"n":"5"}`,
		"number for string":   `{"s":5}`,
		"bool for string":     `{"s":true}`,
		"object for list":     `{"l":{}}`,
		"number for time":     `{"t":5}`,
		"bad time":            `{"t":"yesterday"}`,
		"fractional into int": `{"n":2.5}`,
		"negative into uint":  `{"u":-1}`,
		"array for object":    `[1]`,
		"truncated":           `{"s":"x"`,
		"null body":           `null`,
	}
	for name, body := range bad {
		_, err := decodeResponse[decodeTarget](&response{status: 201, header: http.Header{"X-Request-Id": {"rid"}}, body: []byte(body)})
		var se *ServerError
		if !errors.As(err, &se) || se.StatusCode != 201 || se.RequestID != "rid" {
			t.Errorf("%s: want *ServerError with the status, got %v", name, err)
		}
		if IsRetryable(err) && se.StatusCode < 500 {
			t.Errorf("%s: a response-shape failure is not retryable", name)
		}
	}
}

func TestStatusTextAndVoid(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		switch r.URL.Path {
		case "/cancel":
			jsonReply(w, 202, `{"uuid":"sub@1","status":"active"}`)
		case "/csv":
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write([]byte("txTimeUtc,type,txUuid\n2026-10-01T12:00:00Z,payment,pay@1\n"))
		case "/delete":
			w.WriteHeader(204)
		case "/nocontent":
			w.WriteHeader(204)
		case "/empty":
			w.WriteHeader(200)
		case "/csv-error":
			jsonReply(w, 400, `{"error":"validation failed","code":"validation_failed","details":{"max":92,"errors":[{"field":"to","message":"to must be at most 92 days after from, both included"}]}}`)
		}
	})
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	sub, status, err := c.callStatus[Subscription](ctx, endpoint{method: http.MethodPost, path: "/cancel"}, nil)
	if err != nil || status != 202 || sub.UUID != "sub@1" {
		t.Errorf("callStatus: %v %d %v", sub, status, err)
	}

	text, err := c.callText(ctx, get("/csv"), nil)
	if err != nil || text != "txTimeUtc,type,txUuid\n2026-10-01T12:00:00Z,payment,pay@1\n" {
		t.Errorf("callText: %q %v", text, err)
	}
	if got := ts.recorded()[1].header.Get("Accept"); got != "text/csv, application/json" {
		t.Errorf("CSV Accept = %q", got)
	}
	var ve *ValidationError
	if _, err := c.callText(ctx, get("/csv-error"), nil); !errors.As(err, &ve) || ve.FieldErrors[0].Field != "to" || ve.Details["max"] != float64(92) {
		t.Errorf("CSV error not parsed as JSON: %v", err)
	}

	for _, path := range []string{"/delete", "/nocontent", "/empty"} {
		if err := c.callVoid(ctx, endpoint{method: http.MethodDelete, path: path}, nil); err != nil {
			t.Errorf("callVoid %s: %v", path, err)
		}
	}
	if _, err := c.callRef[Product](ctx, get("/empty"), nil); !errors.As(err, new(*ServerError)) {
		t.Errorf("empty 200 where JSON is expected: %v", err)
	}
	if _, err := c.callRef[Product](ctx, get("/nocontent"), nil); !errors.As(err, new(*ServerError)) {
		t.Errorf("204 where JSON is expected: %v", err)
	}
}
