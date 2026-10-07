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
		pathf("/product/reference/%s", "a/b c"):                                     "/product/reference/a%2Fb%20c",
		pathf("/transaction/payment/%s", "pay@019c-1"):                              "/transaction/payment/pay@019c-1",
		pathf("/customer/email/%s", "a+b@example.com"):                              "/customer/email/a+b@example.com",
		pathf("/product/reference/%s", ".."):                                        "/product/reference/%2E%2E",
		pathf("/product/reference/%s", "."):                                         "/product/reference/%2E",
		pathf("/product/reference/%s", "v1..2"):                                     "/product/reference/v1..2",
		pathf("/x/%s/y/%s", "?q=1#f", "%41"):                                        "/x/%3Fq=1%23f/y/%2541",
		pathf("/transaction/subscription/reference/%s/%s", "subscription", "ord:1"): "/transaction/subscription/reference/subscription/ord:1",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("pathf = %q, want %q", got, want)
		}
	}

	// The escaped path reaches the server as is (a / in a reference is never a separator).
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	if _, err := c.send(context.Background(), get(pathf("/product/reference/%s", "a/b")), nil); err != nil {
		t.Fatal(err)
	}
	if got := ts.recorded()[0].path; got != "/product/reference/a%2Fb" {
		t.Errorf("server saw %q", got)
	}
}

func TestQueryEncoding(t *testing.T) {
	after := time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("CEST", 2*3600))
	before := time.Date(2026, 10, 5, 0, 0, 0, 500, time.UTC)
	p := &PaymentListParams{
		Limit: 25, Cursor: "019c-cursor", CustomerUUID: memberUUID, CreatedAfter: &after,
		CreatedBefore: &before, IncludeMembers: true, Refunded: ptr(false),
	}
	q := p.values()
	if got := q.Get("createdAfter"); got != "2026-10-04T12:00:00+02:00" {
		t.Errorf("createdAfter = %q", got)
	}
	if got := q.Get("createdBefore"); got != "2026-10-05T00:00:00.0000005Z" {
		t.Errorf("createdBefore = %q", got)
	}
	encoded := q.Encode()
	want := "createdAfter=2026-10-04T12%3A00%3A00%2B02%3A00&createdBefore=2026-10-05T00%3A00%3A00.0000005Z&cursor=019c-cursor&customerUuid=" +
		memberUUID + "&includeMembers=true&limit=25&refunded=false"
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
	ts := newTestServer(t, staticReply(200, `{"items":[],"nextCursor":null}`))
	c, _ := newTestClient(t, ts)
	if _, err := c.call[Page[Payment]](context.Background(), endpoint{method: http.MethodGet, path: "/transaction/payments", query: q}, nil); err != nil {
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
		{(&ProductListParams{}).values(), ""},
		{(&CustomerListParams{Limit: 5, Email: "a+b@x.io", Verified: ptr(false)}).values(), "email=a%2Bb%40x.io&limit=5&verified=false"},
		{(&CombinedPaymentListParams{Source: CombinedPaymentSourceSubscriptionHistory, SubscriptionUUID: "sub@x"}).values(), "source=subscriptionHistory&subscriptionUuid=sub%40x"},
		{(&FailureListParams{Kind: FailureKindBill, Category: FailureCategoryOther, UserUUID: memberUUID}).values(), "category=other&kind=bill&userUuid=" + memberUUID},
		{(&SubscriptionListParams{Status: SubscriptionStatusPastDue, Reference: "r-1", Cursor: "c"}).values(), "cursor=c&reference=r-1&status=pastDue"},
		{(&BillListParams{Limit: 100}).values(), "limit=100"},
		{(&RefundListParams{Limit: 3, Cursor: "c", IncludeMembers: ptr(false), Held: ptr(true)}).values(false), "held=true&includeMembers=false"},
		{(&RefundListParams{Limit: 3, Cursor: "c", UserUUID: memberUUID}).values(true), "cursor=c&limit=3&userUuid=" + memberUUID},
		{(&MemberListParams{Limit: -1}).values(), "limit=-1"},
		{(&InvitationListParams{Status: InvitationStatusPending}).values(), "status=pending"},
		{(&EventListParams{Cursor: "evt_1", Type: EventTypePaymentCompleted, IncludeMembers: true}).values(), "cursor=evt_1&includeMembers=true&type=payment.completed"},
		{(&ReadParams{IncludeMembers: true}).values(), "includeMembers=true"},
		{(&CancelSubscriptionParams{Immediate: ptr(false)}).values(), "immediate=false"},
		{(&CancelSubscriptionParams{}).values(), ""},
		{(&WalletListParams{WithBalances: true}).values(), "withBalances=true"},
		{(&SupportedCurrenciesParams{UserUUID: memberUUID}).values(), "userUuid=" + memberUUID},
		{(&CurrencyListParams{Test: true}).values(), "test=true"},
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
		{CreateProductParams{Name: "Pro", Price: math.NaN()}, ""},
		{map[string]any{"price": math.Inf(1)}, ""},
		{CreateCustomerParams{Name: "Ad\xffa", Email: "a@b.co"}, "name"},
		{UpdateCustomerParams{Address: ptr("bad \xfe")}, "address"},
		{CreateSubscriptionSessionParams{ProductUUID: memberUUID, Frequency: &Duration{1, "mon\xffths"}}, "frequency.unit"},
		{map[string]any{"bad\xffkey": 1}, ""},
		{CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{"payment.completed", "x\xff"}}, "events[1]"},
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
		{"local offsets and microseconds", `{"t":"2026-09-13T21:23:26.620071+02:00"}`, func(v decodeTarget) bool {
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
			_, _ = w.Write([]byte("paymentUuid,type\npay@1,payment\n"))
		case "/delete":
			jsonReply(w, 200, `{"message":"deleted"}`)
		case "/nocontent":
			w.WriteHeader(204)
		case "/empty":
			w.WriteHeader(200)
		case "/csv-error":
			jsonReply(w, 400, `{"error":"'to' date cannot be more than 3 months after 'from' date","code":"bad_request"}`)
		}
	})
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	sub, status, err := c.callStatus[Subscription](ctx, endpoint{method: http.MethodPost, path: "/cancel"}, nil)
	if err != nil || status != 202 || sub.UUID != "sub@1" {
		t.Errorf("callStatus: %v %d %v", sub, status, err)
	}

	text, err := c.callText(ctx, get("/csv"), nil)
	if err != nil || text != "paymentUuid,type\npay@1,payment\n" {
		t.Errorf("callText: %q %v", text, err)
	}
	if got := ts.recorded()[1].header.Get("Accept"); got != "text/csv, application/json" {
		t.Errorf("CSV Accept = %q", got)
	}
	if _, err := c.callText(ctx, get("/csv-error"), nil); !errors.As(err, new(*BadRequestError)) {
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
