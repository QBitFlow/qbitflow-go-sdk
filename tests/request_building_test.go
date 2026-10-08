package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// recorder captures the raw request line the SDK produced, so the URL can be asserted
// rather than inferred.
type recorder struct {
	rawPath  string
	rawQuery string
	requests int
}

// clientRecording returns a client pointed at a stub server that records each request
// and answers with the given body.
func clientRecording(t *testing.T, body string) (*qbitflow.QBitFlow, *recorder) {
	t.Helper()

	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.EscapedPath() preserves the encoding as it arrived on the wire, which is
		// the whole point of these assertions; r.URL.Path would show it already decoded.
		rec.rawPath = r.URL.EscapedPath()
		rec.rawQuery = r.URL.RawQuery
		rec.requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := qbitflow.New("sk_dummy_offline_tests")
	client.SetBaseURL(server.URL)

	return client, rec
}

// ── Path escaping ────────────────────────────────────────────────────────────

func TestReferenceWithSlashesStaysOneSegment(t *testing.T) {
	// A reference like this comes straight from the caller's own order system. Raw, the
	// slashes make it address "/product/reference/ORD/2026/17" — a different route.
	client, rec := clientRecording(t, `{"id":1}`)

	_, _ = client.Products.GetByReference("ORD/2026/17")

	if want := "/product/reference/ORD%2F2026%2F17"; rec.rawPath != want {
		t.Errorf("path = %q, want %q", rec.rawPath, want)
	}
}

func TestEscapingCoversEveryUserSuppliedSegment(t *testing.T) {
	const hostile = "a/b"

	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow)
		want string
	}{
		{"customer by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.Get(hostile) },
			"/customer/uuid/a%2Fb"},
		{"customer by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.GetByReference(hostile) },
			"/customer/reference/a%2Fb"},
		{"customer by email", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.GetByEmail("a/b@x.com") },
			"/customer/email/a%2Fb@x.com"},
		{"payment by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetPayment(hostile) },
			"/transaction/payment/a%2Fb"},
		{"payment by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetPaymentByReference(hostile) },
			"/transaction/payment/reference/a%2Fb"},
		{"subscription by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Subscriptions.GetSubscription(hostile) },
			"/transaction/subscription/a%2Fb"},
		{"refund by transaction", func(c *qbitflow.QBitFlow) { _, _ = c.Refunds.GetByTransactionUUID(hostile) },
			"/transaction/refunds/by-transaction/a%2Fb"},
		{"user by email", func(c *qbitflow.QBitFlow) { _, _ = c.Users.GetByEmail("a/b@x.com") },
			"/user/email/a%2Fb@x.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			tc.call(client)

			if rec.rawPath != tc.want {
				t.Errorf("path = %q, want %q", rec.rawPath, tc.want)
			}
		})
	}
}

func TestPrefixedIdentifiersArePassedThroughUnchanged(t *testing.T) {
	// The API's own identifiers carry a "pay@"/"sub@" prefix. '@' is legal in a path
	// segment, so escaping must leave these byte-identical — otherwise every existing
	// lookup would change shape.
	client, rec := clientRecording(t, `{}`)

	const uuid = "pay@550e8400-e29b-41d4-a716-446655440000"
	_, _ = client.Payments.GetPayment(uuid)

	if want := "/transaction/payment/" + uuid; rec.rawPath != want {
		t.Errorf("path = %q, want %q", rec.rawPath, want)
	}
}

func TestQueryValuesAreEscaped(t *testing.T) {
	client, rec := clientRecording(t, `{}`)

	_, _ = client.TransactionStatus.GetTransactionStatus("a&b=c", qbmodels.TransactionTypeOneTimePayment)

	// Raw, the '&' would split into a third query parameter and txUUID would be truncated.
	if !strings.Contains(rec.rawQuery, "txUUID=a%26b%3Dc") {
		t.Errorf("query = %q, want an escaped txUUID", rec.rawQuery)
	}
}

// ── WebSocket URL ────────────────────────────────────────────────────────────

func TestWebSocketURLUsesAWebSocketScheme(t *testing.T) {
	cases := []struct{ base, wantPrefix string }{
		{"https://api.qbitflow.app/v1", "wss://api.qbitflow.app/v1"},
		{"http://localhost:3001", "ws://localhost:3001"},
	}

	for _, tc := range cases {
		t.Run(tc.base, func(t *testing.T) {
			client := qbitflow.NewWithConfig(qbf.Config{APIKey: "sk_dummy", BaseURL: tc.base})

			url := client.TransactionStatus.GetTransactionStatusWSURL(
				"pay@1", qbmodels.TransactionTypeOneTimePayment)

			// An http(s) URL is rejected outright by a WebSocket dialer
			// ("malformed ws or wss URL"), so the scheme has to be rewritten.
			if !strings.HasPrefix(url, tc.wantPrefix) {
				t.Errorf("url = %q, want prefix %q", url, tc.wantPrefix)
			}
			if strings.HasPrefix(url, "http") {
				t.Errorf("url still carries an http scheme: %q", url)
			}
		})
	}
}

func TestWebSocketURLEscapesQueryValues(t *testing.T) {
	client := qbitflow.NewWithConfig(qbf.Config{
		APIKey:  "sk_dummy",
		BaseURL: "https://api.qbitflow.app/v1",
	})

	url := client.TransactionStatus.GetTransactionStatusWSURL(
		"a&b", qbmodels.TransactionTypeOneTimePayment)

	if !strings.Contains(url, "txUUID=a%26b") {
		t.Errorf("url = %q, want an escaped txUUID", url)
	}
}

// ── Argument validation ──────────────────────────────────────────────────────

func TestEmptyIdentifierIsRejectedBeforeAnyRequest(t *testing.T) {
	// Get("") previously built "/customer/uuid/", which is the *collection* route — a
	// different, valid endpoint. The call appeared to succeed against the wrong resource
	// instead of failing fast.
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"customers.Get", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Get(""); return err }},
		{"customers.GetByEmail", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetByEmail(""); return err }},
		{"customers.Delete", func(c *qbitflow.QBitFlow) error { return c.Customers.Delete("") }},
		{"payments.GetPayment", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPayment(""); return err }},
		{"subscriptions.GetSubscription", func(c *qbitflow.QBitFlow) error {
			_, err := c.Subscriptions.GetSubscription("")
			return err
		}},
		{"refunds.GetByTransactionUUID", func(c *qbitflow.QBitFlow) error {
			_, err := c.Refunds.GetByTransactionUUID("")
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)

			err := tc.call(client)
			if err == nil {
				t.Fatal("expected a validation error")
			}

			var validationErr *qberrors.ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("expected *qberrors.ValidationError, got %T: %v", err, err)
			}
			if rec.requests != 0 {
				t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
			}
		})
	}
}

func TestZeroIdentifierIsRejectedBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"users.GetByID", func(c *qbitflow.QBitFlow) error { _, err := c.Users.GetByID(0); return err }},
		{"users.Delete", func(c *qbitflow.QBitFlow) error { return c.Users.Delete(0) }},
		{"products.Get", func(c *qbitflow.QBitFlow) error { _, err := c.Products.Get(0); return err }},
		{"products.Delete", func(c *qbitflow.QBitFlow) error { return c.Products.Delete(0) }},
		{"apiKeys.GetForUser", func(c *qbitflow.QBitFlow) error { _, err := c.ApiKeys.GetForUser(0); return err }},
		{"claims.CreateClaimRequest", func(c *qbitflow.QBitFlow) error {
			_, err := c.Claims.CreateClaimRequest(0)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)

			err := tc.call(client)
			if err == nil {
				t.Fatal("expected a validation error")
			}

			var validationErr *qberrors.ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("expected *qberrors.ValidationError, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), "must be positive") {
				t.Errorf("message = %q, want it to mention %q", err.Error(), "must be positive")
			}
			if rec.requests != 0 {
				t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
			}
		})
	}
}

func TestClientSideValidationUsesTheSDKErrorType(t *testing.T) {
	// These guards previously returned raw fmt.Errorf values, which matched no SDK error
	// type, so `errors.As(err, &validationErr)` missed them entirely.
	client, _ := clientRecording(t, `{}`)

	_, err := client.Customers.GetByEmail("not-an-email")

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}

	// The message must not echo the caller's value back — it used to interpolate the
	// address, putting user data into error strings and logs.
	if strings.Contains(err.Error(), "not-an-email") {
		t.Errorf("error message leaks the input value: %q", err.Error())
	}
}

// ── Webhook verification contract ────────────────────────────────────────────

func TestWebhookVerifyReportsRejectionWithoutAnError(t *testing.T) {
	client, _ := clientRecordingStatus(t, http.StatusBadRequest, `{"error":"invalid signature"}`)

	ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=bad", "123")

	if err != nil {
		t.Fatalf("a rejected signature must not be an error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for a rejected signature")
	}
}

func TestWebhookVerifyPropagatesNonRejectionFailures(t *testing.T) {
	// Reporting these as "not verified" would make an outage indistinguishable from a
	// forged signature, and a handler that drops unverified events would silently
	// discard real payments.
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"expired api key", http.StatusUnauthorized, `{"error":"invalid token"}`},
		{"insufficient role", http.StatusForbidden, `{"error":"forbidden"}`},
		{"api failure", http.StatusInternalServerError, `{"error":"boom"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := clientRecordingStatus(t, tc.status, tc.body)

			ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=good", "123")

			if err == nil {
				t.Fatalf("expected an error for status %d, got ok=%v", tc.status, ok)
			}
			if ok {
				t.Error("expected ok=false alongside the error")
			}
		})
	}
}

func TestWebhookVerifyAcceptsAValidSignature(t *testing.T) {
	client, _ := clientRecordingStatus(t, http.StatusOK, `{"message":"valid"}`)

	ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=good", "123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true")
	}
}

func TestWebhookVerifyRejectsMissingHeadersWithAValidationError(t *testing.T) {
	client, rec := clientRecording(t, `{}`)

	_, err := client.Webhooks.Verify([]byte(`{"a":1}`), "", "123")

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}
	if rec.requests != 0 {
		t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
	}
}

// clientRecordingStatus is clientRecording with an explicit status code.
func clientRecordingStatus(t *testing.T, status int, body string) (*qbitflow.QBitFlow, *recorder) {
	t.Helper()

	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.rawPath = r.URL.EscapedPath()
		rec.requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := qbitflow.New("sk_dummy_offline_tests")
	client.SetBaseURL(server.URL)

	return client, rec
}

// ── closeToExpireError ───────────────────────────────────────────────────────

func TestGetSessionOmitsCloseToExpireErrorByDefault(t *testing.T) {
	// The API defaults this to true. Hard-coding false suppressed the "session close to
	// expiry" error that JS and PHP surface, so the same call behaved differently per SDK.
	client, rec := clientRecording(t, `{}`)

	_, _ = client.Payments.GetSession("pay@1", nil)

	if rec.rawQuery != "" {
		t.Errorf("query = %q, want it omitted so the API default applies", rec.rawQuery)
	}
}

func TestGetSessionSendsCloseToExpireErrorWhenSet(t *testing.T) {
	for _, want := range []bool{true, false} {
		t.Run(map[bool]string{true: "true", false: "false"}[want], func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)

			_, _ = client.Payments.GetSession("pay@1", &want)

			expected := "closeToExpireError=" + map[bool]string{true: "true", false: "false"}[want]
			if rec.rawQuery != expected {
				t.Errorf("query = %q, want %q", rec.rawQuery, expected)
			}
		})
	}
}

func TestGetSessionValidatesTheUUID(t *testing.T) {
	client, rec := clientRecording(t, `{}`)

	_, err := client.Payments.GetSession("", nil)

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}
	if rec.requests != 0 {
		t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
	}
}

// ── Subscription webhook envelope ────────────────────────────────────────────

// The API wraps subscription webhooks in an envelope carrying the subscription identity
// and a `type` discriminator, with the event-specific payload in `data`. The SDK used to
// model the inner payload flat at the top level, so decoding a real delivery produced an
// empty transition and no way to tell a billing event from a status change.
const statusTransitionDelivery = `{
	"subscriptionUUID": "sub@01a0c5be",
	"subscriptionReference": "ORD-1",
	"type": "status_transition",
	"data": {
		"previousStatus": "trial",
		"currentStatus": "active",
		"updatedAt": "2026-03-01T12:00:00Z"
	}
}`

const billingDelivery = `{
	"subscriptionUUID": "sub@01a0c5be",
	"type": "billing",
	"data": {
		"uuid": "sub-hist@1",
		"subscriptionUUID": "sub@01a0c5be",
		"amount": 29.99,
		"amountMinUnits": "29990000",
		"createdAt": "2026-03-01T12:00:00Z"
	}
}`

func TestSubscriptionWebhookDecodesAStatusTransition(t *testing.T) {
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(statusTransitionDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.SubscriptionUUID != "sub@01a0c5be" {
		t.Errorf("subscriptionUUID = %q, want %q", event.SubscriptionUUID, "sub@01a0c5be")
	}
	if event.Type != qbmodels.SubscriptionWebhookTypeStatusTransition {
		t.Fatalf("type = %q, want %q", event.Type, qbmodels.SubscriptionWebhookTypeStatusTransition)
	}
	if event.SubscriptionReference == nil || *event.SubscriptionReference != "ORD-1" {
		t.Errorf("subscriptionReference = %v, want %q", event.SubscriptionReference, "ORD-1")
	}

	transition, err := event.StatusTransition()
	if err != nil {
		t.Fatalf("StatusTransition: %v", err)
	}
	if transition.PreviousStatus != qbmodels.SubscriptionStatusTrial {
		t.Errorf("previousStatus = %q, want %q", transition.PreviousStatus, qbmodels.SubscriptionStatusTrial)
	}
	if transition.CurrentStatus != qbmodels.SubscriptionStatusActive {
		t.Errorf("currentStatus = %q, want %q", transition.CurrentStatus, qbmodels.SubscriptionStatusActive)
	}
	if transition.UpdatedAt.IsZero() {
		t.Error("updatedAt did not decode")
	}
}

func TestSubscriptionWebhookDecodesABilling(t *testing.T) {
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(billingDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.Type != qbmodels.SubscriptionWebhookTypeBilling {
		t.Fatalf("type = %q, want %q", event.Type, qbmodels.SubscriptionWebhookTypeBilling)
	}

	billing, err := event.Billing()
	if err != nil {
		t.Fatalf("Billing: %v", err)
	}
	if billing.Amount != 29.99 {
		t.Errorf("amount = %v, want 29.99", billing.Amount)
	}
	// The reference is omitempty and absent here.
	if event.SubscriptionReference != nil {
		t.Errorf("subscriptionReference = %v, want nil", event.SubscriptionReference)
	}
}

func TestSubscriptionWebhookRefusesToUnwrapTheWrongPayload(t *testing.T) {
	// A mis-routed payload must fail loudly rather than yield a zero-valued struct.
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(billingDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, err := event.StatusTransition(); err == nil {
		t.Error("expected an error unwrapping a billing delivery as a status transition")
	}
}

// ── Role hierarchy ───────────────────────────────────────────────────────────

func TestUserRoleCoversTheWholeHierarchy(t *testing.T) {
	// The hierarchy is handle < user < admin < owner. `handle` and `owner` were missing,
	// so reading an owner or a handle user yielded a role matching no constant and every
	// `user.Role == qbmodels.RoleAdmin`-style check silently fell through.
	want := map[qbmodels.UserRole]string{
		qbmodels.RoleHandle: "handle",
		qbmodels.RoleUser:   "user",
		qbmodels.RoleAdmin:  "admin",
		qbmodels.RoleOwner:  "owner",
	}

	for role, expected := range want {
		t.Run(expected, func(t *testing.T) {
			if string(role) != expected {
				t.Errorf("wire value = %q, want %q", string(role), expected)
			}
		})
	}
}

func TestUserRoleDecodesEveryRoleTheAPIReturns(t *testing.T) {
	for _, role := range []string{"handle", "user", "admin", "owner"} {
		t.Run(role, func(t *testing.T) {
			body := `{"id":1,"name":"N","lastName":"L","email":"e@x.com","role":"` + role + `"}`
			client, _ := clientRecording(t, body)

			user, err := client.Users.GetByID(1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(user.Role) != role {
				t.Errorf("role = %q, want %q", user.Role, role)
			}
		})
	}
}

// ── Product text validation ──────────────────────────────────────────────────

func TestProductTextLengthIsCountedInRunes(t *testing.T) {
	// The server counts runes; the SDK used to count bytes, so a 60-character accented or
	// CJK name (120+ bytes) was rejected locally even though the API accepts it.
	accented := strings.Repeat("é", 60) // 60 runes, 120 bytes
	cjk := strings.Repeat("商", 60)      // 60 runes, 180 bytes

	for _, name := range []string{accented, cjk} {
		t.Run(fmt.Sprintf("%d runes / %d bytes", len([]rune(name)), len(name)), func(t *testing.T) {
			update := qbf.UpdateProduct{Name: &name}
			if err := update.Validate(); err != nil {
				t.Errorf("valid multi-byte name rejected: %v", err)
			}

			create := qbf.CreateProduct{Name: name, Description: "A fine description", Price: 1}
			if err := create.Validate(); err != nil {
				t.Errorf("valid multi-byte name rejected on create: %v", err)
			}
		})
	}
}

func TestProductTextStillEnforcesTheRuneBounds(t *testing.T) {
	tooLong := strings.Repeat("é", 101) // 101 runes — over the limit in runes, not just bytes
	update := qbf.UpdateProduct{Name: &tooLong}

	err := update.Validate()
	if err == nil {
		t.Fatal("expected a 101-rune name to be rejected")
	}

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("expected *qberrors.ValidationError, got %T", err)
	}
}

func TestProductValidationRejectsMarkup(t *testing.T) {
	// The API binds these fields `producttext`; neither validator checked the character set
	// before, so markup reached the server and came back as a 400.
	markup := "<script>alert(1)</script>"

	update := qbf.UpdateProduct{Name: &markup}
	if err := update.Validate(); err == nil {
		t.Error("expected markup to be rejected on update")
	}

	create := qbf.CreateProduct{Name: markup, Description: "A fine description", Price: 1}
	if err := create.Validate(); err == nil {
		t.Error("expected markup to be rejected on create")
	}
}

func TestProductCreateRequiresADescription(t *testing.T) {
	// The API binds description `required,producttext,min=2,max=500`; create used to check
	// only the name.
	create := qbf.CreateProduct{Name: "Widget", Price: 1}

	if err := create.Validate(); err == nil {
		t.Error("expected a missing description to be rejected")
	}
}

// ── Combined feed ────────────────────────────────────────────────────────────

func TestCombinedPaymentDecodesTheCurrency(t *testing.T) {
	// Captured from the live combined feed: the API sends an expanded `currency` here even
	// though the reference documents it as never serialized. Go was the only SDK missing it.
	const body = `{"items":[{
		"source":"payment","uuid":"pay@1","amount":9.99,"amountMinUnits":"",
		"currencyId":7,
		"currency":{"id":7,"symbol":"USDC","name":"USD Coin","decimals":6,
		            "address":"GZJyATVw4RQTZxwRqzWwDKomh5q1jJfwhGVhVHRNZLFo","test":false},
		"customerUUID":"c1","test":false,"transactionHash":"h",
		"createdAt":"2026-03-01T12:00:00Z","name":"n","description":"d"
	}],"nextCursor":null}`

	client, _ := clientRecording(t, body)

	page, err := client.Payments.GetAllCombinedPayments(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	item := page.Items[0]
	if item.Currency == nil {
		t.Fatal("currency was not decoded")
	}
	if item.Currency.Symbol != "USDC" {
		t.Errorf("symbol = %q, want %q", item.Currency.Symbol, "USDC")
	}
	if item.Currency.Address != "GZJyATVw4RQTZxwRqzWwDKomh5q1jJfwhGVhVHRNZLFo" {
		t.Errorf("address = %q", item.Currency.Address)
	}
	// The combined feed sends an empty min-units string; it must not break decoding.
	if item.AmountMinUnits != "" {
		t.Errorf("amountMinUnits = %q, want empty", item.AmountMinUnits)
	}
}

func TestCombinedPaymentToleratesAnAbsentCurrency(t *testing.T) {
	// If the server ever honours its own resource:"ignore" tag, the field disappears — the
	// pointer must come back nil rather than a zero-valued currency that looks real.
	const body = `{"items":[{"source":"payment","uuid":"pay@1","currencyId":7,
		"createdAt":"2026-03-01T12:00:00Z"}],"nextCursor":null}`

	client, _ := clientRecording(t, body)

	page, err := client.Payments.GetAllCombinedPayments(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Items[0].Currency != nil {
		t.Error("expected a nil currency when the API omits it")
	}
	if page.Items[0].CurrencyID != 7 {
		t.Errorf("currencyId = %d, want 7", page.Items[0].CurrencyID)
	}
}
