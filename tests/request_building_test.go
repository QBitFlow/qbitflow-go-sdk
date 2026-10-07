package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// recorder captures what the SDK put on the wire for the last request, so the method,
// URL, headers and body can be asserted rather than inferred.
type recorder struct {
	mu       sync.Mutex
	method   string
	rawPath  string
	rawQuery string
	header   http.Header
	body     string
	requests int
}

func (r *recorder) record(req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.method = req.Method
	// EscapedPath preserves the encoding as it arrived on the wire, which is the whole
	// point of the escaping assertions; URL.Path would show it already decoded.
	r.rawPath = req.URL.EscapedPath()
	r.rawQuery = req.URL.RawQuery
	r.header = req.Header.Clone()
	r.body = string(body)
	r.requests++
}

// clientRecording returns a client pointed at a stub server that records each request
// and answers 200 with the given body.
func clientRecording(t *testing.T, body string) (*qbitflow.QBitFlow, *recorder) {
	t.Helper()
	return clientRecordingStatus(t, http.StatusOK, body)
}

// clientRecordingStatus is clientRecording with an explicit status code.
func clientRecordingStatus(t *testing.T, status int, body string) (*qbitflow.QBitFlow, *recorder) {
	t.Helper()

	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := qbitflow.NewWithConfig(qbf.Config{
		APIKey:         "sk_dummy_offline_tests",
		BaseURL:        server.URL,
		DisableRetries: true,
	})

	return client, rec
}

// decodedBody parses the recorded JSON request body.
func decodedBody(t *testing.T, rec *recorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(rec.body), &body); err != nil {
		t.Fatalf("request body %q is not a JSON object: %v", rec.body, err)
	}
	return body
}

// ── Routes ───────────────────────────────────────────────────────────────────

func TestEveryMethodHitsTheDocumentedRoute(t *testing.T) {
	limit := uint16(5)
	cursor := "c1"
	params := qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-31"}

	cases := []struct {
		name   string
		reply  string
		call   func(c *qbitflow.QBitFlow) error
		method string
		path   string
		query  string
	}{
		{"ApiKeys.GetAll", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.ApiKeys.GetAll(); return err },
			"GET", "/api-key/", ""},
		{"ApiKeys.GetForUser", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.ApiKeys.GetForUser(4); return err },
			"GET", "/api-key/user/4", ""},
		{"Customers.Create", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Customers.Create(&qbf.CreateCustomer{Name: "Jane", LastName: "Doe", Email: "j@x.io"})
			return err
		}, "POST", "/customer/", ""},
		{"Customers.Update", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Customers.Update("c1", &qbf.UpdateCustomer{Name: new("Jane")})
			return err
		}, "PUT", "/customer/c1", ""},
		{"Customers.Get", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Get("c1"); return err },
			"GET", "/customer/uuid/c1", ""},
		{"Customers.GetByEmail", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetByEmail("j@x.io"); return err },
			"GET", "/customer/email/j@x.io", ""},
		{"Customers.GetByReference", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetByReference("r1"); return err },
			"GET", "/customer/reference/r1", ""},
		{"Customers.GetAll", `{"items":[]}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetAll(&limit, &cursor); return err },
			"GET", "/customer/all", "cursor=c1&limit=5"},
		{"Customers.Delete", ``, func(c *qbitflow.QBitFlow) error { return c.Customers.Delete("c1") },
			"DELETE", "/customer/uuid/c1", ""},
		{"Products.Create", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Products.Create(&qbf.CreateProduct{Name: "Widget", Description: "A widget", Price: 1})
			return err
		}, "POST", "/product/", ""},
		{"Products.Get", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.Get(3); return err },
			"GET", "/product/id/3", ""},
		{"Products.GetAll", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.GetAll(); return err },
			"GET", "/product/", ""},
		{"Products.GetByReference", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.GetByReference("p1"); return err },
			"GET", "/product/reference/p1", ""},
		{"Products.Update", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Products.Update(3, &qbf.UpdateProduct{Price: new(2.0)})
			return err
		}, "PUT", "/product/3", ""},
		{"Products.Delete", ``, func(c *qbitflow.QBitFlow) error { return c.Products.Delete(3) },
			"DELETE", "/product/3", ""},
		{"Payments.GetSession", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetSession("pay@1", nil); return err },
			"GET", "/transaction/session-checkout/pay@1", ""},
		{"Subscriptions.GetSession", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetSession("sub@1", nil); return err },
			"GET", "/transaction/session-checkout/sub@1", ""},
		{"Payments.CreateSession", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(1))})
			return err
		}, "POST", "/transaction/session-checkout/new/payment", ""},
		{"Subscriptions.CreateSession", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
				ProductID: new(uint64(1)), Frequency: qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths}})
			return err
		}, "POST", "/transaction/session-checkout/new/subscription", ""},
		{"TransactionStatus.GetTransactionStatus", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.TransactionStatus.GetTransactionStatus("pay@1", qbmodels.TransactionTypeOneTimePayment)
			return err
		}, "GET", "/transaction/status", "txUUID=pay%401&txType=payment"},
		{"Refunds.GetByTransactionUUID", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetByTransactionUUID("pay@1"); return err },
			"GET", "/transaction/refunds/by-transaction/pay@1", ""},
		{"Refunds.GetAll", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetAll(); return err },
			"GET", "/transaction/refunds/all", ""},
		{"Refunds.GetAllInactive", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetAllInactive(&limit, nil); return err },
			"GET", "/transaction/refunds/all/inactive", "limit=5"},
		{"Payments.GetAllPayments", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetAllPayments(nil, &cursor); return err },
			"GET", "/transaction/payments", "cursor=c1"},
		{"Payments.GetAllCombinedPayments", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Payments.GetAllCombinedPayments(nil, nil)
			return err
		}, "GET", "/transaction/payments/combined", ""},
		{"Payments.GetPayment", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPayment("pay@1"); return err },
			"GET", "/transaction/payment/pay@1", ""},
		{"Payments.GetPaymentByReference", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPaymentByReference("o1"); return err },
			"GET", "/transaction/payment/reference/o1", ""},
		{"Payments.GetCustomerForTransaction", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Payments.GetCustomerForTransaction("pay@1")
			return err
		}, "GET", "/transaction/customer/pay@1", ""},
		{"Subscriptions.GetPaymentHistory", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetPaymentHistory("sub@1"); return err },
			"GET", "/transaction/subscription/history/sub@1", ""},
		{"Subscriptions.GetSubscription", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetSubscription("sub@1"); return err },
			"GET", "/transaction/subscription/sub@1", ""},
		{"Subscriptions.GetSubscriptionByReference", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Subscriptions.GetSubscriptionByReference("plan-1")
			return err
		}, "GET", "/transaction/subscription/reference/subscription/plan-1", ""},
		{"Subscriptions.ForceCancel", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.ForceCancel("sub@1"); return err },
			"GET", "/transaction/subscription/processing/force-cancel/sub@1", ""},
		{"Subscriptions.ExecuteTestBillingCycle", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Subscriptions.ExecuteTestBillingCycle("sub@1")
			return err
		}, "GET", "/transaction/subscription/processing/execute-billing/sub@1", ""},
		{"Users.Create", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Users.Create(&qbf.CreateUser{Name: "Al", LastName: "Smith", Email: "a@x.io", Role: qbmodels.RoleUser})
			return err
		}, "POST", "/user/", ""},
		{"Users.GetAll", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Users.GetAll(); return err },
			"GET", "/user/all", ""},
		{"Users.GetByID", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Users.GetByID(9); return err },
			"GET", "/user/id/9", ""},
		{"Users.Get", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Users.Get(); return err },
			"GET", "/user/", ""},
		{"Users.GetByEmail", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Users.GetByEmail("a@x.io"); return err },
			"GET", "/user/email/a@x.io", ""},
		{"Users.Update", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Users.Update(9, &qbf.UpdateUser{Name: new("Al")})
			return err
		}, "PUT", "/user/9", ""},
		{"Users.Delete", ``, func(c *qbitflow.QBitFlow) error { return c.Users.Delete(9) },
			"DELETE", "/user/9", ""},
		{"Claims.CreateClaimRequest", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Claims.CreateClaimRequest(9); return err },
			"POST", "/user/claim/request", ""},
		{"Claims.GetClaimRequestByUserID", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Claims.GetClaimRequestByUserID(9); return err },
			"GET", "/user/claim/request/9", ""},
		{"Claims.GetClaimFunds", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Claims.GetClaimFunds(); return err },
			"GET", "/user/claim/funds", ""},
		{"Claims.TriggerTestClaimFunds", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Claims.TriggerTestClaimFunds(9); return err },
			"GET", "/user/claim/funds/test-trigger/9", ""},
		{"Accounting.Export", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Accounting.Export(params); return err },
			"GET", "/accounting/export", "from=2026-01-01&to=2026-01-31&format=json"},
		{"Accounting.ExportCSV", `paymentId`, func(c *qbitflow.QBitFlow) error { _, err := c.Accounting.ExportCSV(params); return err },
			"GET", "/accounting/export", "from=2026-01-01&to=2026-01-31&format=csv"},
		{"Webhooks.Verify", `{}`, func(c *qbitflow.QBitFlow) error {
			_, err := c.Webhooks.Verify([]byte(`{}`), "sha256=x", "1")
			return err
		},
			"POST", "/webhooks/verify", ""},
		{"Currencies.GetAllAvailable", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Currencies.GetAllAvailable(true); return err },
			"GET", "/utils/all-available-currencies", "test=true"},
		{"Currencies.GetAllMain", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Currencies.GetAllMain(false); return err },
			"GET", "/utils/all-main-currencies", "test=false"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, tc.reply)
			if err := tc.call(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.method != tc.method || rec.rawPath != tc.path || rec.rawQuery != tc.query {
				t.Errorf("got %s %s?%s, want %s %s?%s", rec.method, rec.rawPath, rec.rawQuery, tc.method, tc.path, tc.query)
			}
		})
	}
}

// ── Path escaping ────────────────────────────────────────────────────────────

func TestReferenceWithSlashesStaysOneSegment(t *testing.T) {
	// The '/' is encoded as %2F so the value stays one path segment. (The API currently
	// cannot route such a reference and answers 404; the SDK still encodes it correctly.)
	client, rec := clientRecording(t, `{"id":1}`)

	_, _ = client.Products.GetByReference("ORD/2026/17")

	if want := "/product/reference/ORD%2F2026%2F17"; rec.rawPath != want {
		t.Errorf("path = %q, want %q", rec.rawPath, want)
	}
}

func TestEscapingCoversEveryUserSuppliedSegment(t *testing.T) {
	const hostile = "a/b?c#d"

	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow)
		want string
	}{
		{"customer by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.Get(hostile) },
			"/customer/uuid/a%2Fb%3Fc%23d"},
		{"customer by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.GetByReference(hostile) },
			"/customer/reference/a%2Fb%3Fc%23d"},
		{"customer by email", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.GetByEmail("a/b@x.com") },
			"/customer/email/a%2Fb@x.com"},
		{"customer update", func(c *qbitflow.QBitFlow) { _, _ = c.Customers.Update(hostile, &qbf.UpdateCustomer{}) },
			"/customer/a%2Fb%3Fc%23d"},
		{"customer delete", func(c *qbitflow.QBitFlow) { _ = c.Customers.Delete(hostile) },
			"/customer/uuid/a%2Fb%3Fc%23d"},
		{"product by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Products.GetByReference(hostile) },
			"/product/reference/a%2Fb%3Fc%23d"},
		{"payment by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetPayment(hostile) },
			"/transaction/payment/a%2Fb%3Fc%23d"},
		{"payment by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetPaymentByReference(hostile) },
			"/transaction/payment/reference/a%2Fb%3Fc%23d"},
		{"customer for transaction", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetCustomerForTransaction(hostile) },
			"/transaction/customer/a%2Fb%3Fc%23d"},
		{"session", func(c *qbitflow.QBitFlow) { _, _ = c.Payments.GetSession(hostile, nil) },
			"/transaction/session-checkout/a%2Fb%3Fc%23d"},
		{"subscription by uuid", func(c *qbitflow.QBitFlow) { _, _ = c.Subscriptions.GetSubscription(hostile) },
			"/transaction/subscription/a%2Fb%3Fc%23d"},
		{"subscription by reference", func(c *qbitflow.QBitFlow) { _, _ = c.Subscriptions.GetSubscriptionByReference(hostile) },
			"/transaction/subscription/reference/subscription/a%2Fb%3Fc%23d"},
		{"subscription history", func(c *qbitflow.QBitFlow) { _, _ = c.Subscriptions.GetPaymentHistory(hostile) },
			"/transaction/subscription/history/a%2Fb%3Fc%23d"},
		{"refund by transaction", func(c *qbitflow.QBitFlow) { _, _ = c.Refunds.GetByTransactionUUID(hostile) },
			"/transaction/refunds/by-transaction/a%2Fb%3Fc%23d"},
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
	// segment, so escaping must leave these byte-identical.
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

	cursor := "a+b&c"
	client, rec = clientRecording(t, `{"items":[]}`)
	_, _ = client.Payments.GetAllPayments(nil, &cursor)
	if rec.rawQuery != "cursor=a%2Bb%26c" {
		t.Errorf("cursor query = %q, want an escaped cursor", rec.rawQuery)
	}
}

// ── Argument validation ──────────────────────────────────────────────────────

func expectNoRequestValidationError(t *testing.T, err error, rec *recorder) {
	t.Helper()
	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}
	if rec.requests != 0 {
		t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
	}
}

func TestEmptyIdentifierIsRejectedBeforeAnyRequest(t *testing.T) {
	// Get("") would build "/customer/uuid/", which is a different route — the call would
	// appear to succeed against the wrong resource instead of failing fast.
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"customers.Get", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Get(""); return err }},
		{"customers.GetByEmail", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetByEmail(""); return err }},
		{"customers.GetByReference", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetByReference(" "); return err }},
		{"customers.Delete", func(c *qbitflow.QBitFlow) error { return c.Customers.Delete("") }},
		{"products.GetByReference", func(c *qbitflow.QBitFlow) error { _, err := c.Products.GetByReference(""); return err }},
		{"payments.GetPayment", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPayment(""); return err }},
		{"payments.GetPaymentByReference", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPaymentByReference(""); return err }},
		{"payments.GetCustomerForTransaction", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetCustomerForTransaction(""); return err }},
		{"subscriptions.GetSubscription", func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetSubscription(""); return err }},
		{"subscriptions.GetPaymentHistory", func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetPaymentHistory(""); return err }},
		{"subscriptions.ForceCancel", func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.ForceCancel(""); return err }},
		{"refunds.GetByTransactionUUID", func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetByTransactionUUID(""); return err }},
		{"users.GetByEmail", func(c *qbitflow.QBitFlow) error { _, err := c.Users.GetByEmail(""); return err }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			expectNoRequestValidationError(t, tc.call(client), rec)
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
		{"claims.CreateClaimRequest", func(c *qbitflow.QBitFlow) error { _, err := c.Claims.CreateClaimRequest(0); return err }},
		{"claims.GetClaimRequestByUserID", func(c *qbitflow.QBitFlow) error { _, err := c.Claims.GetClaimRequestByUserID(0); return err }},
		{"claims.TriggerTestClaimFunds", func(c *qbitflow.QBitFlow) error { _, err := c.Claims.TriggerTestClaimFunds(0); return err }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			err := tc.call(client)
			expectNoRequestValidationError(t, err, rec)
			if !strings.Contains(err.Error(), "must be positive") {
				t.Errorf("message = %q, want it to mention %q", err.Error(), "must be positive")
			}
		})
	}
}

func TestNilArgumentsAreAValidationErrorNotAPanic(t *testing.T) {
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"Payments.CreateSession", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.CreateSession(nil); return err }},
		{"Subscriptions.CreateSession", func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.CreateSession(nil); return err }},
		{"Customers.Create", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Create(nil); return err }},
		{"Customers.Update", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Update("c1", nil); return err }},
		{"Products.Create", func(c *qbitflow.QBitFlow) error { _, err := c.Products.Create(nil); return err }},
		{"Products.Update", func(c *qbitflow.QBitFlow) error { _, err := c.Products.Update(1, nil); return err }},
		{"Users.Create", func(c *qbitflow.QBitFlow) error { _, err := c.Users.Create(nil); return err }},
		{"Users.Update", func(c *qbitflow.QBitFlow) error { _, err := c.Users.Update(1, nil); return err }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			expectNoRequestValidationError(t, tc.call(client), rec)
		})
	}
}

func TestPageLimitMustBePositive(t *testing.T) {
	zero := uint16(0)
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"Customers.GetAll", func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetAll(&zero, nil); return err }},
		{"Payments.GetAllPayments", func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetAllPayments(&zero, nil); return err }},
		{"Payments.GetAllCombinedPayments", func(c *qbitflow.QBitFlow) error {
			_, err := c.Payments.GetAllCombinedPayments(&zero, nil)
			return err
		}},
		{"Refunds.GetAllInactive", func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetAllInactive(&zero, nil); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			expectNoRequestValidationError(t, tc.call(client), rec)
		})
	}

	t.Run("an empty cursor requests the first page", func(t *testing.T) {
		client, rec := clientRecording(t, `{"items":[]}`)
		empty := ""
		if _, err := client.Customers.GetAll(nil, &empty); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.rawQuery != "" {
			t.Errorf("query = %q, want none", rec.rawQuery)
		}
	})
}

func TestClientSideValidationUsesTheSDKErrorType(t *testing.T) {
	client, rec := clientRecording(t, `{}`)

	_, err := client.Customers.GetByEmail("not-an-email")
	expectNoRequestValidationError(t, err, rec)

	// The message must not echo the caller's value back: user data would end up in error
	// strings and logs.
	if strings.Contains(err.Error(), "not-an-email") {
		t.Errorf("error message leaks the input value: %q", err.Error())
	}
}

// ── On-Behalf-Of ─────────────────────────────────────────────────────────────

// onBehalfOfCalls exercises one request of every service through a given client.
var onBehalfOfCalls = []struct {
	name  string
	reply string
	call  func(c *qbitflow.QBitFlow) error
}{
	{"Customers", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.Get("c1"); return err }},
	{"Products", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.Get(1); return err }},
	{"Users", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Users.Get(); return err }},
	{"ApiKeys", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.ApiKeys.GetForUser(1); return err }},
	{"Webhooks", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Webhooks.Verify("{}", "sha256=x", "1"); return err }},
	{"Currencies", `[]`, func(c *qbitflow.QBitFlow) error { _, err := c.Currencies.GetAllMain(false); return err }},
	{"Payments", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPayment("pay@1"); return err }},
	{"Subscriptions", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetSubscription("sub@1"); return err }},
	{"TransactionStatus", `{}`, func(c *qbitflow.QBitFlow) error {
		_, err := c.TransactionStatus.GetTransactionStatus("pay@1", "payment")
		return err
	}},
	{"Refunds", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Refunds.GetByTransactionUUID("pay@1"); return err }},
	{"Accounting", `id`, func(c *qbitflow.QBitFlow) error {
		_, err := c.Accounting.ExportCSV(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-02"})
		return err
	}},
	{"Claims", `{}`, func(c *qbitflow.QBitFlow) error { _, err := c.Claims.GetClaimRequestByUserID(1); return err }},
}

func TestPerServiceOnBehalfOf(t *testing.T) {
	scoped := map[string]func(c *qbitflow.QBitFlow) error{
		"Customers": func(c *qbitflow.QBitFlow) error { _, err := c.Customers.OnBehalfOf(42).Get("c1"); return err },
		"Products":  func(c *qbitflow.QBitFlow) error { _, err := c.Products.OnBehalfOf(42).Get(1); return err },
		"Users":     func(c *qbitflow.QBitFlow) error { _, err := c.Users.OnBehalfOf(42).Get(); return err },
		"ApiKeys":   func(c *qbitflow.QBitFlow) error { _, err := c.ApiKeys.OnBehalfOf(42).GetForUser(1); return err },
		"Webhooks": func(c *qbitflow.QBitFlow) error {
			_, err := c.Webhooks.OnBehalfOf(42).Verify("{}", "sha256=x", "1")
			return err
		},
		"Payments": func(c *qbitflow.QBitFlow) error { _, err := c.Payments.OnBehalfOf(42).GetPayment("pay@1"); return err },
		"Subscriptions": func(c *qbitflow.QBitFlow) error {
			_, err := c.Subscriptions.OnBehalfOf(42).GetSubscription("sub@1")
			return err
		},
		"TransactionStatus": func(c *qbitflow.QBitFlow) error {
			_, err := c.TransactionStatus.OnBehalfOf(42).GetTransactionStatus("pay@1", "payment")
			return err
		},
		"Refunds": func(c *qbitflow.QBitFlow) error {
			_, err := c.Refunds.OnBehalfOf(42).GetByTransactionUUID("pay@1")
			return err
		},
		"Accounting": func(c *qbitflow.QBitFlow) error {
			_, err := c.Accounting.OnBehalfOf(42).ExportCSV(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-02"})
			return err
		},
		"Claims": func(c *qbitflow.QBitFlow) error {
			_, err := c.Claims.OnBehalfOf(42).GetClaimRequestByUserID(1)
			return err
		},
	}

	for _, tc := range onBehalfOfCalls {
		call, ok := scoped[tc.name]
		if !ok {
			continue // Currencies: public routes, no per-service OnBehalfOf
		}
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, tc.reply)
			if err := call(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rec.header.Get("On-Behalf-Of"); got != "42" {
				t.Errorf("On-Behalf-Of = %q, want 42", got)
			}

			// The service it was derived from is unaffected.
			if err := tc.call(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rec.header.Get("On-Behalf-Of"); got != "" {
				t.Errorf("the original service sent On-Behalf-Of %q", got)
			}
		})
	}
}

func TestClientLevelOnBehalfOf(t *testing.T) {
	for _, tc := range onBehalfOfCalls {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, tc.reply)

			// Every service of the scoped client carries the header — including
			// Currencies, which has no per-service OnBehalfOf.
			if err := tc.call(client.OnBehalfOf(42)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rec.header.Get("On-Behalf-Of"); got != "42" {
				t.Errorf("On-Behalf-Of = %q, want 42", got)
			}

			// The original client is untouched.
			if err := tc.call(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rec.header.Get("On-Behalf-Of"); got != "" {
				t.Errorf("the original client sent On-Behalf-Of %q", got)
			}

			// 0 means organization level: no header.
			if err := tc.call(client.OnBehalfOf(0)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := rec.header.Get("On-Behalf-Of"); got != "" {
				t.Errorf("OnBehalfOf(0) sent On-Behalf-Of %q", got)
			}
		})
	}

	t.Run("a per-service OnBehalfOf on the scoped client replaces the target", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		if _, err := client.OnBehalfOf(42).Products.OnBehalfOf(9).Get(1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := rec.header.Get("On-Behalf-Of"); got != "9" {
			t.Errorf("On-Behalf-Of = %q, want 9", got)
		}
	})
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
		{"validation 422", http.StatusUnprocessableEntity, `{"error":"bad body"}`},
		{"api failure", http.StatusInternalServerError, `{"error":"boom"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := clientRecordingStatus(t, tc.status, tc.body)

			ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=good", "123")

			if err == nil || ok {
				t.Fatalf("expected an error and ok=false for status %d, got ok=%v err=%v", tc.status, ok, err)
			}
			if qberrors.StatusCodeOf(err) != tc.status {
				t.Errorf("status = %d, want %d", qberrors.StatusCodeOf(err), tc.status)
			}
		})
	}
}

func TestWebhookVerifyAcceptsAValidSignature(t *testing.T) {
	client, _ := clientRecordingStatus(t, http.StatusOK, `{"message":"webhook verified!"}`)

	ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=good", "123")

	if err != nil || !ok {
		t.Fatalf("expected ok=true, got ok=%v err=%v", ok, err)
	}
}

func TestWebhookVerifyPropagatesNetworkFailures(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	client := qbitflow.NewWithConfig(qbf.Config{APIKey: "k", BaseURL: url, DisableRetries: true})

	ok, err := client.Webhooks.Verify([]byte(`{"a":1}`), "sha256=good", "123")

	var netErr *qberrors.NetworkError
	if !errors.As(err, &netErr) || ok {
		t.Fatalf("an unreachable API must surface as NetworkError, got ok=%v %T: %v", ok, err, err)
	}
}

func TestWebhookVerifyRequestBody(t *testing.T) {
	t.Run("raw bytes are forwarded verbatim", func(t *testing.T) {
		client, rec := clientRecording(t, `{"message":"ok"}`)

		// {} and [] must survive, and so must the number literal.
		_, err := client.Webhooks.Verify([]byte(`{"obj":{},"arr":[],"n":1.50,"html":"<b>"}`), "sha256=x", "1700000000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var body struct {
			Payload           json.RawMessage `json:"payload"`
			ReceivedSignature string          `json:"receivedSignature"`
			ReceivedTimestamp string          `json:"receivedTimestamp"`
		}
		if err := json.Unmarshal([]byte(rec.body), &body); err != nil {
			t.Fatalf("body %q: %v", rec.body, err)
		}
		if body.ReceivedSignature != "sha256=x" || body.ReceivedTimestamp != "1700000000" {
			t.Errorf("signature/timestamp not sent: %s", rec.body)
		}
		// encoding/json HTML-escapes the forwarded bytes (< and > become backslash-u003c/e),
		// which the server decodes back to the same string. The sequences are built from
		// bytes so no tooling can silently un-escape them.
		bs := string([]byte{92})
		want := `{"obj":{},"arr":[],"n":1.50,"html":"` + bs + "u003cb" + bs + `u003e"}`
		if string(body.Payload) != want {
			t.Errorf("payload = %s, want %s", body.Payload, want)
		}
	})

	t.Run("a string or a decoded value is accepted", func(t *testing.T) {
		for _, payload := range []any{`{"a":1}`, json.RawMessage(`{"a":1}`), map[string]any{"a": 1}} {
			client, rec := clientRecording(t, `{"message":"ok"}`)
			if _, err := client.Webhooks.Verify(payload, "sha256=x", "1"); err != nil {
				t.Fatalf("%T: unexpected error: %v", payload, err)
			}
			if !strings.Contains(rec.body, `"payload":{"a":1}`) {
				t.Errorf("%T: body = %s", payload, rec.body)
			}
		}
	})

	t.Run("a missing, empty, invalid or unencodable payload is rejected locally", func(t *testing.T) {
		for _, payload := range []any{nil, []byte(nil), []byte(""), "  ", "null", "{not json", map[string]any{"n": math.NaN()}} {
			client, rec := clientRecording(t, `{}`)
			ok, err := client.Webhooks.Verify(payload, "sha256=x", "1")
			if ok {
				t.Errorf("%#v: ok must be false", payload)
			}
			expectNoRequestValidationError(t, err, rec)
		}
	})

	t.Run("missing headers are rejected locally", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Webhooks.Verify([]byte(`{"a":1}`), "", "123")
		expectNoRequestValidationError(t, err, rec)
	})
}

// ── closeToExpireError ───────────────────────────────────────────────────────

func TestGetSessionOmitsCloseToExpireErrorByDefault(t *testing.T) {
	client, rec := clientRecording(t, `{}`)

	_, _ = client.Payments.GetSession("pay@1", nil)

	if rec.rawQuery != "" {
		t.Errorf("query = %q, want it omitted so the API default applies", rec.rawQuery)
	}
}

func TestGetSessionSendsCloseToExpireErrorWhenSet(t *testing.T) {
	for _, want := range []bool{true, false} {
		t.Run(fmt.Sprint(want), func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)

			_, _ = client.Subscriptions.GetSession("sub@1", &want)

			if expected := fmt.Sprintf("closeToExpireError=%t", want); rec.rawQuery != expected {
				t.Errorf("query = %q, want %q", rec.rawQuery, expected)
			}
		})
	}
}

// ── Subscription webhook envelope ────────────────────────────────────────────

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
		"currencyId": 2,
		"currency": {"id": 2, "name": "USD Coin", "symbol": "USDC", "decimals": 6, "address": "0xa0b8", "mainCurrencyId": 1, "mainCurrency": null, "test": true},
		"customerUUID": "3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f",
		"metadata": {"feeBps": 100, "txMetadata": {"networkFees": {"amount": "0", "unitsConsumed": 0}, "blockData": {"number": "1", "timestamp": 1}}, "txAmounts": {"usd": {"platform": 0.3, "merchant": 29.69}, "minUnits": {"platform": "300000", "organization": "0", "referral": "0", "merchant": "29690000"}}},
		"createdAt": "2026-03-01T12:00:00Z"
	}
}`

func TestSubscriptionWebhookDecodesAStatusTransition(t *testing.T) {
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(statusTransitionDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.SubscriptionUUID != "sub@01a0c5be" || event.SubscriptionReference != "ORD-1" {
		t.Errorf("envelope = %+v", event)
	}
	if event.Type != qbmodels.SubscriptionWebhookTypeStatusTransition {
		t.Fatalf("type = %q", event.Type)
	}

	transition, err := event.StatusTransition()
	if err != nil {
		t.Fatalf("StatusTransition: %v", err)
	}
	if transition.PreviousStatus != qbmodels.SubscriptionStatusTrial ||
		transition.CurrentStatus != qbmodels.SubscriptionStatusActive || transition.UpdatedAt.IsZero() {
		t.Errorf("transition = %+v", transition)
	}
}

func TestSubscriptionWebhookDecodesABilling(t *testing.T) {
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(billingDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	billing, err := event.Billing()
	if err != nil {
		t.Fatalf("Billing: %v", err)
	}
	if billing.Amount != 29.99 || billing.Currency.Symbol != "USDC" || billing.Metadata.FeeBps != 100 {
		t.Errorf("billing = %+v", billing)
	}
	if billing.CustomerUUID == nil || *billing.CustomerUUID != "3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f" {
		t.Errorf("customerUUID = %v", billing.CustomerUUID)
	}
	// subscriptionReference is omitempty and absent: it decodes to "".
	if event.SubscriptionReference != "" {
		t.Errorf("subscriptionReference = %q, want empty", event.SubscriptionReference)
	}
}

func TestSubscriptionWebhookRefusesToUnwrapTheWrongPayload(t *testing.T) {
	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal([]byte(billingDelivery), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	_, err := event.StatusTransition()
	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("expected a ValidationError unwrapping a billing delivery as a transition, got %T: %v", err, err)
	}

	event.Data = json.RawMessage(`{"amount":"not a number"}`)
	if _, err := event.Billing(); !errors.As(err, &validationErr) {
		t.Errorf("expected a ValidationError for undecodable data, got %T: %v", err, err)
	}
}

// ── Role hierarchy ───────────────────────────────────────────────────────────

func TestUserRoleDecodesEveryRoleTheAPIReturns(t *testing.T) {
	for _, role := range []string{"handle", "user", "admin", "owner", "future_role"} {
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

	tooLong := strings.Repeat("é", 101)
	if err := (&qbf.UpdateProduct{Name: &tooLong}).Validate(); err == nil {
		t.Error("expected a 101-rune name to be rejected")
	}
}

func TestProductValidationRejectsMarkup(t *testing.T) {
	markup := "<script>alert(1)</script>"

	if err := (&qbf.UpdateProduct{Name: &markup}).Validate(); err == nil {
		t.Error("expected markup to be rejected on update")
	}
	if err := (&qbf.CreateProduct{Name: markup, Description: "A fine description", Price: 1}).Validate(); err == nil {
		t.Error("expected markup to be rejected on create")
	}
	if err := (&qbf.CreateProduct{Name: "Widget", Price: 1}).Validate(); err == nil {
		t.Error("expected a missing description to be rejected")
	}
}

// ── Response hydration ───────────────────────────────────────────────────────

const currencyJSON = `{"id":7,"name":"USD Coin","symbol":"USDC","decimals":6,"address":"GZJyATVw4RQTZxwRqzWwDKomh5q1jJfwhGVhVHRNZLFo","mainCurrencyId":3,"mainCurrency":{"id":3,"name":"Solana","symbol":"SOL","decimals":9,"address":"","mainCurrency":null,"test":false},"test":false}`

func TestPaymentDecodesEveryField(t *testing.T) {
	body := `{"uuid":"pay@1","createdAt":"2026-03-01T12:00:00Z","from":"0xf","to":"0xt","amount":9.99,
		"amountMinUnits":"9990000","currencyId":7,"currency":` + currencyJSON + `,"test":false,
		"transactionHash":"0xh","organizationId":5,"userId":3,"reference":"order-1","name":"n","description":"d",
		"productId":11,"customerUUID":"3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f",
		"metadata":{"feeBps":100,"organizationFee":{"organizationId":5,"organization":"0xorg","feeBps":50},
		"txMetadata":{"networkFees":{"amount":"21000","unitsConsumed":21000},"blockData":{"number":"12","timestamp":1758539990}},
		"txAmounts":{"usd":{"platform":0.1,"organization":0.05,"merchant":9.84},"minUnits":{"platform":"1","organization":"2","referral":"0","merchant":"3"}}}}`
	client, _ := clientRecording(t, body)

	p, err := client.Payments.GetPayment("pay@1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Currency.Symbol != "USDC" || p.Currency.MainCurrency == nil || p.Currency.MainCurrency.Symbol != "SOL" ||
		p.Currency.MainCurrencyID == nil || *p.Currency.MainCurrencyID != 3 {
		t.Errorf("currency = %+v", p.Currency)
	}
	if p.OrganizationID != 5 || p.UserID != 3 || p.ProductID != 11 || p.Reference == nil || *p.Reference != "order-1" {
		t.Errorf("payment = %+v", p)
	}
	if p.CustomerUUID == nil || *p.CustomerUUID != "3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f" {
		t.Errorf("customerUUID = %v", p.CustomerUUID)
	}
	fee := p.Metadata.OrganizationFee
	if fee == nil || fee.OrganizationID != 5 || fee.Organization != "0xorg" || fee.FeeBps != 50 {
		t.Errorf("organizationFee = %+v", fee)
	}
	if p.Metadata.ReferralFee != nil {
		t.Errorf("referralFee = %+v, want nil", p.Metadata.ReferralFee)
	}
	if p.Metadata.TxMetadata.BlockData.Timestamp != 1758539990 || p.Metadata.TxAmounts.MinUnits.Merchant != "3" {
		t.Errorf("metadata = %+v", p.Metadata)
	}
}

func TestAbsentAndNullFieldsDecodeToZeroValues(t *testing.T) {
	// An org-level payment without product or customer: the API omits the omitempty fields
	// and may send null for others. Non-nullable fields decode to zero values — never an
	// error, never a nil you have to check.
	body := `{"uuid":"pay@1","createdAt":null,"amount":1,"currencyId":7,"currency":null,"name":null,"metadata":null}`
	client, _ := clientRecording(t, body)

	p, err := client.Payments.GetPayment("pay@1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.UserID != 0 || p.ProductID != 0 || p.Name != "" || !p.CreatedAt.IsZero() {
		t.Errorf("payment = %+v", p)
	}
	if p.Currency != (qbmodels.Currency{}) || p.Metadata.FeeBps != 0 || p.Metadata.TxAmounts.MinUnits.Merchant != "" {
		t.Errorf("nested values must be zero-valued: currency=%+v metadata=%+v", p.Currency, p.Metadata)
	}
	if p.CustomerUUID != nil || p.Reference != nil {
		t.Errorf("nullable fields must be nil: %+v", p)
	}
}

func TestCombinedPaymentDecodesTheCurrency(t *testing.T) {
	const body = `{"items":[{
		"source":"payment","uuid":"pay@1","amount":9.99,"amountMinUnits":"",
		"currencyId":7,"currency":` + currencyJSON + `,
		"customerUUID":"00000000-0000-0000-0000-000000000000","test":false,"transactionHash":"h",
		"createdAt":"2026-03-01T12:00:00Z","name":"n","description":"d","metadata":null
	},{"source":"subscription_history","uuid":"sub-hist@1","subscriptionUUID":"sub@1","productId":4,"currencyId":7}],"nextCursor":"next-1"}`

	client, _ := clientRecording(t, body)

	page, err := client.Payments.GetAllCombinedPayments(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 || !page.HasMore() || *page.NextCursor != "next-1" {
		t.Fatalf("page = %+v", page)
	}

	first, second := page.Items[0], page.Items[1]
	if first.Currency.Symbol != "USDC" || first.Currency.Address != "GZJyATVw4RQTZxwRqzWwDKomh5q1jJfwhGVhVHRNZLFo" {
		t.Errorf("currency = %+v", first.Currency)
	}
	if first.Metadata != nil || first.ProductID != nil || first.SubscriptionUUID != nil {
		t.Errorf("nullable fields of a payment row: %+v", first)
	}
	if second.SubscriptionUUID == nil || *second.SubscriptionUUID != "sub@1" || second.ProductID == nil || *second.ProductID != 4 {
		t.Errorf("subscription-history row: %+v", second)
	}
	if second.Currency != (qbmodels.Currency{}) {
		t.Errorf("an absent currency decodes to the zero value, got %+v", second.Currency)
	}
}

func TestSubscriptionDecodesEveryField(t *testing.T) {
	body := `{"uuid":"sub@1","reference":null,"createdAt":"2026-03-01T12:00:00Z","updatedAt":"2026-03-02T12:00:00Z",
		"from":"0xf","to":"0xt","productId":4,"subscriptionHash":"0xs","currencyId":7,"currency":` + currencyJSON + `,
		"frequency":2592000,"allowance":"29.97","subscriptionStatus":"trial","stopped":false,
		"lastBillingDate":"0001-01-01T00:00:00Z","nextBillingDate":"2026-04-01T12:00:00Z",
		"test":true,"organizationId":5,"userId":0}`
	client, _ := clientRecording(t, body)

	sub, err := client.Subscriptions.GetSubscription("sub@1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Currency.Symbol != "USDC" || sub.Frequency != 2592000 || sub.Allowance != "29.97" {
		t.Errorf("subscription = %+v", sub)
	}
	// Never billed: the zero time, not a nil pointer.
	if !sub.LastBillingDate.IsZero() || sub.NextBillingDate.IsZero() {
		t.Errorf("billing dates = %v / %v", sub.LastBillingDate, sub.NextBillingDate)
	}
	if sub.Reference != nil || sub.CustomerUUID != nil || sub.MinimumCancellationDate != nil {
		t.Errorf("absent nullable fields must be nil: %+v", sub)
	}
}

func TestSubscriptionHistoryDecodes(t *testing.T) {
	client, _ := clientRecording(t, `[{"uuid":"sub-hist@1","subscriptionUUID":"sub@1","currencyId":3,
		"currency":`+currencyJSON+`,"organizationId":5,"userId":2,"metadata":{"feeBps":100}}]`)
	history, err := client.Subscriptions.GetPaymentHistory("sub@1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].Currency.Symbol != "USDC" || history[0].Metadata.FeeBps != 100 ||
		history[0].OrganizationID != 5 || history[0].UserID != 2 || history[0].CustomerUUID != nil {
		t.Errorf("history = %+v", history)
	}
}

func TestTransactionStatusFields(t *testing.T) {
	const body = `{"status":"completed","txHash":"0xabc","settlementDetails":{"feeBps":100,` +
		`"organizationFee":{"organizationId":5,"organization":"0xorg","feeBps":50},` +
		`"referralFee":{"referralId":8,"referrer":"0xref","feeBps":10,"deadline":"2026-03-01T12:00:00Z"},` +
		`"txMetadata":{"networkFees":{"amount":"0","unitsConsumed":0},"blockData":{"number":"1","timestamp":1}},` +
		`"txAmounts":{"usd":{"platform":0.1,"merchant":9.89},"minUnits":{"platform":"1","organization":"0","referral":"0","merchant":"9"}}}}`
	client, _ := clientRecording(t, body)

	status, err := client.TransactionStatus.GetTransactionStatus("pay@1", qbmodels.TransactionTypeOneTimePayment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// message is omitempty and absent: "".
	if status.Message != "" || status.TxHash != "0xabc" || status.SettlementDetails == nil {
		t.Fatalf("status = %+v", status)
	}
	org, ref := status.SettlementDetails.OrganizationFee, status.SettlementDetails.ReferralFee
	if org == nil || org.OrganizationID != 5 || org.Organization != "0xorg" {
		t.Errorf("organizationFee = %+v", org)
	}
	if ref == nil || ref.ReferralID != 8 || ref.Referrer != "0xref" || ref.Deadline.IsZero() {
		t.Errorf("referralFee = %+v", ref)
	}
	if status.SettlementDetails.TxAmounts.Usd.Organization != 0 {
		t.Errorf("an absent organization USD amount decodes to 0")
	}
}

func TestRefundEntryFields(t *testing.T) {
	// A refund not handled yet: merchantMessage and txHash are "" (plain strings on the
	// server, never null), respondedAt and metadata are null.
	const pending = `{"uuid":"refund@1","txId":"pay@1","test":false,"reason":"changed my mind","status":"pending",` +
		`"createdAt":"2026-03-01T12:00:00Z","merchantMessage":"","respondedAt":null,"userId":0,"organizationId":5,` +
		`"txHash":"","amountMinUnits":"100","metadata":null}`
	client, _ := clientRecording(t, pending)

	refund, err := client.Refunds.GetByTransactionUUID("pay@1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refund.TxHash != "" || refund.MerchantMessage != "" || refund.RespondedAt != nil || refund.Metadata != nil ||
		refund.OrganizationID != 5 || refund.Status != qbmodels.RefundStatusPending {
		t.Errorf("pending refund = %+v", refund)
	}

	const handled = `[{"uuid":"refund@1","txId":"pay@1","status":"approved","userId":3,"organizationId":7,` +
		`"txHash":"0xdeadbeef","merchantMessage":"done","respondedAt":"2026-03-02T12:00:00Z",` +
		`"metadata":{"networkFees":{"amount":"5","unitsConsumed":1},"blockData":{"number":"9","timestamp":9},"mainCurrencyPriceUSD":150.5}}]`
	client, _ = clientRecording(t, handled)
	refunds, err := client.Refunds.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := refunds[0]
	if r.UserID != 3 || r.OrganizationID != 7 || r.TxHash != "0xdeadbeef" || r.MerchantMessage != "done" ||
		r.RespondedAt == nil || r.Metadata == nil || r.Metadata.MainCurrencyPriceUSD != 150.5 {
		t.Errorf("handled refund = %+v", r)
	}
}

func TestCustomerOptionalFieldsAreStrings(t *testing.T) {
	client, _ := clientRecording(t, `{"uuid":"c1","name":"Jane","lastName":"Doe","email":"j@x.io","organizationId":5,"test":false}`)

	c, err := client.Customers.Get("c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.PhoneNumber != "" || c.Address != "" || c.Reference != "" || c.UserID != 0 || c.OrganizationID != 5 {
		t.Errorf("customer = %+v", c)
	}
}

func TestSessionCheckoutFields(t *testing.T) {
	client, _ := clientRecording(t, `{"uuid":"pay@1","txType":"payment","organizationId":5,"organizationName":"Acme",
		"feeBps":100,"test":false,"customerUUID":"3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f","availableCurrencies":null}`)

	s, err := client.Payments.GetSession("pay@1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.OrganizationID != 5 || s.FeeBps != 100 || s.CustomerUUID == nil || s.UserID != 0 || s.Reference != "" {
		t.Errorf("session = %+v", s)
	}
	if s.AvailableCurrencies == nil || len(s.AvailableCurrencies) != 0 {
		t.Errorf("a null availableCurrencies must be an empty list, got %#v", s.AvailableCurrencies)
	}
}

func TestSubscriptionSessionDecodesUpgradingFromTrial(t *testing.T) {
	client, _ := clientRecording(t, `{"uuid":"sub@1","txType":"createSubscription","frequency":2592000,"upgradingFromTrial":true}`)

	session, err := client.Subscriptions.GetSession("sub@1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !session.UpgradingFromTrial || !session.IsSubscription() || session.IsPayment() {
		t.Errorf("session = %+v", session)
	}
}

func TestSessionKindHelpers(t *testing.T) {
	cases := []struct {
		session               qbmodels.SessionCheckout
		isPayment, isSubscrip bool
	}{
		{qbmodels.SessionCheckout{TxType: qbmodels.TransactionTypeOneTimePayment}, true, false},
		{qbmodels.SessionCheckout{TxType: qbmodels.TransactionTypeOneTimePayment, Frequency: 60}, true, false},
		{qbmodels.SessionCheckout{TxType: qbmodels.TransactionTypeCreateSubscription}, false, true},
		// An unknown or newer type is neither.
		{qbmodels.SessionCheckout{TxType: qbmodels.TransactionTypeCreatePAYGSubscription}, false, false},
		{qbmodels.SessionCheckout{TxType: "somethingNew"}, false, false},
		// Without txType, the billing frequency decides.
		{qbmodels.SessionCheckout{}, true, false},
		{qbmodels.SessionCheckout{Frequency: 60}, false, true},
	}
	for _, tc := range cases {
		if tc.session.IsPayment() != tc.isPayment || tc.session.IsSubscription() != tc.isSubscrip {
			t.Errorf("%+v: IsPayment=%v IsSubscription=%v, want %v/%v", tc.session,
				tc.session.IsPayment(), tc.session.IsSubscription(), tc.isPayment, tc.isSubscrip)
		}
	}
}

func TestTransactionWebhookWithNullStatusDecodes(t *testing.T) {
	const body = `{"uuid":"pay@1","txType":"payment","status":null,"session":{"uuid":"pay@1","txType":"payment"}}`

	var event qbmodels.SessionWebhookResponse
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Status != nil || event.ManagementPageLink != "" || !event.Session.IsPayment() || event.Session.CustomerUUID != nil {
		t.Errorf("event = %+v", event)
	}
}

func TestNullListsAreEmptyLists(t *testing.T) {
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow) (int, bool, error)
	}{
		{"Products.GetAll", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Products.GetAll()
			return len(v), v != nil, err
		}},
		{"Users.GetAll", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Users.GetAll()
			return len(v), v != nil, err
		}},
		{"ApiKeys.GetAll", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.ApiKeys.GetAll()
			return len(v), v != nil, err
		}},
		{"Refunds.GetAll", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Refunds.GetAll()
			return len(v), v != nil, err
		}},
		{"Claims.GetClaimFunds", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Claims.GetClaimFunds()
			return len(v), v != nil, err
		}},
		{"Currencies.GetAllMain", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Currencies.GetAllMain(false)
			return len(v), v != nil, err
		}},
		{"Subscriptions.GetPaymentHistory", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Subscriptions.GetPaymentHistory("sub@1")
			return len(v), v != nil, err
		}},
		{"Accounting.Export", func(c *qbitflow.QBitFlow) (int, bool, error) {
			v, err := c.Accounting.Export(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-02"})
			return len(v), v != nil, err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := clientRecording(t, `null`)
			n, nonNil, err := tc.call(client)
			if err != nil || n != 0 || !nonNil {
				t.Errorf("len=%d nonNil=%v err=%v, want an empty non-nil list", n, nonNil, err)
			}
		})
	}

	t.Run("CursorData items", func(t *testing.T) {
		for _, body := range []string{`{"items":null,"nextCursor":null}`, `{}`} {
			client, _ := clientRecording(t, body)
			page, err := client.Customers.GetAll(nil, nil)
			if err != nil || page.Items == nil || len(page.Items) != 0 || page.HasMore() {
				t.Errorf("body %s: page=%+v err=%v", body, page, err)
			}
		}
	})
}

func TestWrongJSONTypeIsAServerError(t *testing.T) {
	cases := []struct {
		name string
		body string
		call func(c *qbitflow.QBitFlow) error
	}{
		{"string for a number", `{"id":"1"}`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.Get(1); return err }},
		{"object for a string", `{"uuid":{"a":1}}`, func(c *qbitflow.QBitFlow) error { _, err := c.Payments.GetPayment("pay@1"); return err }},
		{"string for an object", `{"currency":"USDC"}`, func(c *qbitflow.QBitFlow) error { _, err := c.Subscriptions.GetSubscription("sub@1"); return err }},
		{"object for a list", `{"items":{}}`, func(c *qbitflow.QBitFlow) error { _, err := c.Customers.GetAll(nil, nil); return err }},
		{"object for a top-level list", `{"a":1}`, func(c *qbitflow.QBitFlow) error { _, err := c.Products.GetAll(); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := clientRecording(t, tc.body)
			err := tc.call(client)

			var serverErr *qberrors.ServerError
			if !errors.As(err, &serverErr) || serverErr.StatusCode != http.StatusOK {
				t.Errorf("expected a ServerError with status 200, got %T: %v", err, err)
			}
		})
	}

	t.Run("an integer into a float field is fine", func(t *testing.T) {
		client, _ := clientRecording(t, `{"id":1,"price":10}`)
		p, err := client.Products.Get(1)
		if err != nil || p.Price != 10 {
			t.Errorf("price=%v err=%v", p, err)
		}
	})
}

// ── execute-billing / force-cancel / test-trigger ────────────────────────────

func TestGETActionsReturnTheMessageEnvelope(t *testing.T) {
	client, _ := clientRecording(t, `{"message":"Billing executed successfully"}`)
	resp, err := client.Subscriptions.ExecuteTestBillingCycle("sub@1")
	if err != nil || resp.Message != "Billing executed successfully" {
		t.Errorf("execute-billing: %+v %v", resp, err)
	}

	client, _ = clientRecording(t, `{"message":"Subscription cancelled successfully"}`)
	resp, err = client.Subscriptions.ForceCancel("sub@1")
	if err != nil || resp.Message != "Subscription cancelled successfully" {
		t.Errorf("force-cancel: %+v %v", resp, err)
	}

	client, _ = clientRecording(t, `{"message":"ok"}`)
	resp, err = client.Claims.TriggerTestClaimFunds(42)
	if err != nil || resp.Message != "ok" {
		t.Errorf("test-trigger: %+v %v", resp, err)
	}
}

func TestExecuteBillingNotDueIsAConflict(t *testing.T) {
	client, _ := clientRecordingStatus(t, http.StatusConflict, `{"error":"Subscription is not due for billing yet"}`)

	_, err := client.Subscriptions.ExecuteTestBillingCycle("sub@1")

	var conflict *qberrors.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected *qberrors.ConflictError, got %T: %v", err, err)
	}
}

// ── Enums ────────────────────────────────────────────────────────────────────

func TestUnknownEnumValuesAreKeptVerbatim(t *testing.T) {
	// A status this SDK does not know yet must survive decoding untouched — never an error,
	// never coerced to a default that could grant or revoke access wrongly.
	client, _ := clientRecording(t, `{"uuid":"sub@1","subscriptionStatus":"future_status"}`)
	sub, err := client.Subscriptions.GetSubscription("sub@1")
	if err != nil || string(sub.SubscriptionStatus) != "future_status" {
		t.Errorf("subscription status = %v, err %v", sub, err)
	}

	client, _ = clientRecording(t, `{"uuid":"refund@1","status":"escalated"}`)
	refund, err := client.Refunds.GetByTransactionUUID("pay@1")
	if err != nil || string(refund.Status) != "escalated" {
		t.Errorf("refund status = %v, err %v", refund, err)
	}

	client, _ = clientRecording(t, `{"status":"somethingNew"}`)
	status, err := client.TransactionStatus.GetTransactionStatus("pay@1", "payment")
	if err != nil || string(status.Status) != "somethingNew" {
		t.Errorf("transaction status = %v, err %v", status, err)
	}

	client, _ = clientRecording(t, `[{"type":"subHistory"},{"type":"subscriptionHistory"}]`)
	events, err := client.Accounting.Export(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-02"})
	if err != nil || events[0].Type != qbmodels.AccountingEventTypeSubHistory ||
		events[1].Type != qbmodels.AccountingEventTypeSubscriptionHistory {
		t.Errorf("accounting types = %+v, err %v", events, err)
	}
}
