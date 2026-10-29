package qbitflow

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

const memberUUID = "019eca82-5680-7b00-8000-0000000000b1"

func TestNewClientAPIKey(t *testing.T) {
	for _, key := range []string{"", "   ", "pk_live_123", "SK_live", "key"} {
		_, err := NewClient(key)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("NewClient(%q): want *ValidationError, got %v", key, err)
			continue
		}
		if ve.StatusCode != 0 || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "apiKey" {
			t.Errorf("NewClient(%q): unexpected error %+v", key, ve.APIError)
		}
	}
	for _, key := range []string{"sk_", "sk_123_live_abc", "sk_019eca82-5680-7b00-8000-0000000000b1_test_x", "  sk_padded  "} {
		if _, err := NewClient(key); err != nil {
			t.Errorf("NewClient(%q): %v", key, err)
		}
	}
}

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient("sk_x")
	if err != nil {
		t.Fatal(err)
	}
	if DefaultBaseURL != "https://api.qbitflow.app/v2" || c.t.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q", c.t.baseURL)
	}
	if c.t.timeout != 30*time.Second || c.t.maxRetries != 3 {
		t.Errorf("timeout %v, maxRetries %d", c.t.timeout, c.t.maxRetries)
	}
	if c.onBehalfOf != "" || c.onBehalfOfErr != nil {
		t.Errorf("onBehalfOf = %q / %v", c.onBehalfOf, c.onBehalfOfErr)
	}
	if c.t.httpClient.CheckRedirect == nil {
		t.Error("redirects must be refused")
	}
	for name, svc := range map[string]any{
		"Products": c.Products, "Customers": c.Customers, "CheckoutSessions": c.CheckoutSessions,
		"Payments": c.Payments, "Failures": c.Failures, "Subscriptions": c.Subscriptions,
		"Refunds": c.Refunds, "Members": c.Members, "Invitations": c.Invitations, "Wallets": c.Wallets,
		"Accounting": c.Accounting, "Webhooks": c.Webhooks, "Webhooks.Endpoints": c.Webhooks.Endpoints,
		"Webhooks.Events": c.Webhooks.Events, "Currencies": c.Currencies,
	} {
		if svc == nil {
			t.Errorf("service %s not wired", name)
		}
	}
}

func TestNewClientOptions(t *testing.T) {
	bad := map[string]ClientOption{
		"baseUrl ftp":        WithBaseURL("ftp://api.example.com"),
		"baseUrl relative":   WithBaseURL("/v2"),
		"baseUrl empty":      WithBaseURL(""),
		"timeout zero":       WithTimeout(0),
		"timeout negative":   WithTimeout(-time.Second),
		"maxRetries":         WithMaxRetries(-1),
		"onBehalfOf garbage": WithOnBehalfOf("42"),
		"onBehalfOf nil":     WithOnBehalfOf("00000000-0000-0000-0000-000000000000"),
		"onBehalfOf spaces":  WithOnBehalfOf(" " + memberUUID),
	}
	for name, opt := range bad {
		_, err := NewClient("sk_x", opt)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want *ValidationError, got %v", name, err)
		}
	}

	custom := &http.Client{Timeout: 5 * time.Second}
	c, err := NewClient("sk_x", WithBaseURL("https://sandbox.example.com/v2///"), WithTimeout(2*time.Second),
		WithMaxRetries(0), WithHTTPClient(custom), WithOnBehalfOf(memberUUID), nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.t.baseURL != "https://sandbox.example.com/v2" {
		t.Errorf("trailing slashes not stripped: %q", c.t.baseURL)
	}
	if c.t.timeout != 2*time.Second || c.t.maxRetries != 0 || c.onBehalfOf != memberUUID {
		t.Errorf("options not applied: %+v %q", c.t, c.onBehalfOf)
	}
	if c.t.httpClient == custom || custom.CheckRedirect != nil {
		t.Error("the caller's http.Client must be copied, not modified")
	}
	if c.t.httpClient.Timeout != 5*time.Second || c.t.httpClient.CheckRedirect == nil {
		t.Error("the copy keeps the caller's settings and refuses redirects")
	}
}

func TestHeaders(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{"credential":"apiKey"}`))
	c, _ := newTestClient(t, ts)

	if _, err := c.Me(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := c.call[map[string]any](context.Background(),
		endpoint{method: http.MethodPost, path: "/products", body: map[string]any{"name": "x"}, idempotent: true},
		[]RequestOption{WithRequestID("req-1.a:b_c")})
	if err != nil {
		t.Fatal(err)
	}

	reqs := ts.recorded()
	getReq, postReq := reqs[0], reqs[1]
	for _, r := range reqs {
		if got := r.header.Get("X-API-Key"); got != testAPIKey {
			t.Errorf("X-API-Key = %q", got)
		}
		if got := r.header.Get("User-Agent"); got != "qbitflow-go/3.1.0" {
			t.Errorf("User-Agent = %q", got)
		}
		if got := r.header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if r.header.Get("On-Behalf-Of") != "" {
			t.Error("On-Behalf-Of sent without being set")
		}
	}
	if getReq.method != "GET" || getReq.path != "/me" {
		t.Errorf("GET /me: %s %s", getReq.method, getReq.path)
	}
	for _, h := range []string{"Content-Type", "Idempotency-Key", "X-Request-Id"} {
		if getReq.header.Get(h) != "" {
			t.Errorf("GET sent %s", h)
		}
	}
	if postReq.header.Get("Content-Type") != "application/json" || postReq.body != `{"name":"x"}` {
		t.Errorf("POST: content-type %q body %q", postReq.header.Get("Content-Type"), postReq.body)
	}
	if postReq.header.Get("X-Request-Id") != "req-1.a:b_c" {
		t.Errorf("X-Request-Id = %q", postReq.header.Get("X-Request-Id"))
	}
	if !isUUID(postReq.header.Get("Idempotency-Key")) {
		t.Errorf("Idempotency-Key = %q", postReq.header.Get("Idempotency-Key"))
	}
}

func TestRequestIDValidation(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	for _, id := range []string{"has space", "slash/no", "é", string(make([]byte, 129))} {
		_, err := c.Me(context.Background(), WithRequestID(id))
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.FieldErrors[0].Field != "X-Request-Id" {
			t.Errorf("WithRequestID(%q): %v", id, err)
		}
	}
	if n := len(ts.recorded()); n != 0 {
		t.Errorf("%d requests sent despite invalid ids", n)
	}
}

func TestOnBehalfOf(t *testing.T) {
	const other = "01a05cd7-2a00-7d00-8000-0000000000d1"
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	member := c.OnBehalfOf(memberUUID)
	orgOnly := member.OnBehalfOf("")

	if member.t != c.t {
		t.Error("OnBehalfOf must share the transport")
	}
	if member.Products.client != member || c.Products.client != c {
		t.Error("services must point at their own client")
	}

	ctx := context.Background()
	calls := []struct {
		client *Client
		opts   []RequestOption
		want   string
	}{
		{c, nil, ""},
		{member, nil, memberUUID},
		{member, []RequestOption{WithOnBehalfOf(other)}, other}, // request wins
		{member, []RequestOption{WithOnBehalfOf("")}, ""},       // "" forces organization level
		{c, []RequestOption{WithOnBehalfOf(other)}, other},
		{orgOnly, nil, ""},
	}
	for _, call := range calls {
		if _, err := call.client.Me(ctx, call.opts...); err != nil {
			t.Fatal(err)
		}
	}
	for i, r := range ts.recorded() {
		if got := r.header.Get("On-Behalf-Of"); got != calls[i].want {
			t.Errorf("call %d: On-Behalf-Of = %q, want %q", i, got, calls[i].want)
		}
		if _, set := r.header["On-Behalf-Of"]; set && calls[i].want == "" {
			t.Errorf("call %d: empty On-Behalf-Of header sent", i)
		}
	}

	// Client-level config is validated by NewClient.
	c2, err := NewClient(testAPIKey, WithBaseURL(ts.URL), WithOnBehalfOf(memberUUID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.Me(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ts.recorded()[len(calls)].header.Get("On-Behalf-Of"); got != memberUUID {
		t.Errorf("client option: On-Behalf-Of = %q", got)
	}
}

func TestOnBehalfOfInvalid(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	bad := c.OnBehalfOf("not-a-uuid") // lazy: chaining works, every call fails
	for range 2 {
		_, err := bad.Me(ctx)
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.FieldErrors[0].Field != "onBehalfOf" {
			t.Fatalf("want onBehalfOf ValidationError, got %v", err)
		}
	}
	if _, err := c.OnBehalfOf("00000000-0000-0000-0000-000000000000").Me(ctx); err == nil {
		t.Error("the nil UUID must be refused")
	}
	if _, err := c.Me(ctx, WithOnBehalfOf("123")); err == nil {
		t.Error("an invalid request-level On-Behalf-Of must be refused")
	}
	if n := len(ts.recorded()); n != 0 {
		t.Errorf("%d requests sent", n)
	}
	// The original client is unaffected.
	if _, err := c.Me(ctx); err != nil {
		t.Errorf("original client: %v", err)
	}
	// Uppercase UUIDs are UUIDs.
	if _, err := c.OnBehalfOf("019ECA82-5680-7B00-8000-0000000000B1").Me(ctx); err != nil {
		t.Errorf("uppercase UUID: %v", err)
	}
}

func TestMe(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{
		"credential": "apiKey",
		"apiKeyUuid": "019cadfd-8900-7b00-8000-0000000000f1",
		"role": "user",
		"onBehalfOfUserUuid": "019eca82-5680-7b00-8000-0000000000b1",
		"space": {
			"uuid": "019eca82-5680-7c00-8000-0000000000b2",
			"organizationUuid": "019cadfd-8900-7a00-8000-0000000000a1",
			"organizationName": "Example Shop",
			"userUuid": "019eca82-5680-7b00-8000-0000000000b1",
			"member": {"userUuid": "019eca82-5680-7b00-8000-0000000000b1", "firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.com"},
			"test": true,
			"role": "admin",
			"organizationFeePercent": "7.5",
			"trustedAt": "2026-10-02T12:00:00Z",
			"createdAt": "2026-10-01T12:00:00Z"
		},
		"spaces": [],
		"user": {"ignored": true}
	}`))
	c, _ := newTestClient(t, ts)
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Role != RoleUser || me.OnBehalfOfUserUUID != memberUUID {
		t.Errorf("me = %+v", me)
	}
	if me.Space == nil || !me.Space.Test || me.Space.OrganizationName != "Example Shop" || me.Space.Member == nil || me.Space.Member.FirstName != "Ada" ||
		me.Space.Role != RoleAdmin || me.Space.OrganizationFeePercent != "7.5" || me.Space.TrustedAt == nil || me.Space.CreatedAt.IsZero() {
		t.Errorf("space = %+v", me.Space)
	}
}
