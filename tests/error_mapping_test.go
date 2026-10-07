package tests

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// The payloads below are the API's real error envelopes, captured from the server:
//
//	400 -> {"errors":[{"field":"ProductName","message":"ProductName is too short"},
//	                  {"field":"Price","message":"Price is too short"}]}
//	401 -> {"error":"invalid or missing authentication token"}
//	404 -> {"error":"resource not found"}
//
// Note there is no top-level "message" key on an error response; `message` is the
// success envelope (JSONMessage). Tests assert against these shapes verbatim so a
// future change to the parsing cannot quietly stop matching what the API sends.

// clientAgainst returns an SDK client pointed at a stub server that always replies with
// the given status and body.
func clientAgainst(t *testing.T, status int, body string) *qbitflow.QBitFlow {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	// Retries are off so a stubbed 5xx maps straight to its error type instead of being
	// re-sent three times; the retry policy itself is covered in pkg/qbitflow/retry_test.go.
	return qbitflow.NewWithConfig(qbf.Config{
		APIKey:         "sk_dummy_offline_tests",
		BaseURL:        server.URL,
		DisableRetries: true,
	})
}

// ── Validation errors (400) ──────────────────────────────────────────────────

func TestValidationErrorReportsEveryField(t *testing.T) {
	const body = `{"errors":[{"field":"ProductName","message":"ProductName is too short"},` +
		`{"field":"Price","message":"Price is too short"}]}`

	_, err := clientAgainst(t, http.StatusBadRequest, body).Products.Get(1)
	if err == nil {
		t.Fatal("expected an error")
	}

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}

	// Reporting only the first failure hides the rest of what the caller must fix.
	if len(validationErr.Fields) != 2 {
		t.Fatalf("expected 2 field errors, got %d: %+v", len(validationErr.Fields), validationErr.Fields)
	}

	for _, want := range []string{"ProductName", "ProductName is too short", "Price", "Price is too short"} {
		if !strings.Contains(validationErr.Message, want) {
			t.Errorf("message %q does not mention %q", validationErr.Message, want)
		}
	}

	// The field names have to survive, not just the prose.
	if validationErr.Fields[1].Field != "Price" {
		t.Errorf("second field name = %q, want %q", validationErr.Fields[1].Field, "Price")
	}
}

func TestValidationErrorIsNotDoublePrefixed(t *testing.T) {
	const body = `{"errors":[{"field":"Price","message":"Price is too short"}]}`

	_, err := clientAgainst(t, http.StatusBadRequest, body).Products.Get(1)
	if err == nil {
		t.Fatal("expected an error")
	}

	// Error() adds the "Validation Error: " prefix; the stored message must not already
	// carry one, or it renders twice.
	if got := strings.Count(err.Error(), "Validation Error:"); got != 1 {
		t.Errorf("prefix appears %d times in %q, want 1", got, err.Error())
	}
	if strings.Contains(err.Error(), "; ;") || strings.HasSuffix(err.Error(), "; ") {
		t.Errorf("message has empty joined segments: %q", err.Error())
	}
}

func TestValidationStatusWithoutFieldListStillValidationError(t *testing.T) {
	// A 400 that uses the single-message envelope rather than the field list.
	_, err := clientAgainst(t, http.StatusBadRequest, `{"error":"malformed request"}`).Products.Get(1)

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}
	if validationErr.Message != "malformed request" {
		t.Errorf("message = %q, want %q", validationErr.Message, "malformed request")
	}
}

// ── Other statuses ───────────────────────────────────────────────────────────

func TestNotFoundUsesTheErrorKey(t *testing.T) {
	_, err := clientAgainst(t, http.StatusNotFound, `{"error":"resource not found"}`).Products.Get(1)

	var notFoundErr *qberrors.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected *qberrors.NotFoundError, got %T: %v", err, err)
	}
	if notFoundErr.Message != "resource not found" {
		t.Errorf("message = %q, want %q", notFoundErr.Message, "resource not found")
	}
}

func TestUnauthorizedSurfacesTheAPIMessage(t *testing.T) {
	const body = `{"error":"invalid or missing authentication token"}`

	_, err := clientAgainst(t, http.StatusUnauthorized, body).Products.Get(1)

	var authErr *qberrors.AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *qberrors.AuthenticationError, got %T: %v", err, err)
	}
	if authErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", authErr.StatusCode, http.StatusUnauthorized)
	}
	if authErr.Message != "invalid or missing authentication token" {
		t.Errorf("message = %q", authErr.Message)
	}

	// Existing code that matched the base type keeps working: the typed error unwraps to it.
	var apiErr *qberrors.QBitFlowError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Error("AuthenticationError must unwrap to *QBitFlowError with the same status")
	}
}

// ── Non-2xx that is not 4xx/5xx ──────────────────────────────────────────────

func TestRedirectIsNotTreatedAsSuccess(t *testing.T) {
	// A 304 carries no usable body. Parsing it as a result would hand the caller a
	// zero-valued Product and a nil error.
	product, err := clientAgainst(t, http.StatusNotModified, "").Products.Get(1)
	if err == nil {
		t.Fatalf("a 304 was reported as success, product = %+v", product)
	}
}

func TestRedirectWithBodyIsNotTreatedAsSuccess(t *testing.T) {
	body := `{"id":999,"name":"not the product you asked for"}`

	product, err := clientAgainst(t, http.StatusMovedPermanently, body).Products.Get(1)
	if err == nil {
		t.Fatalf("a 301 was reported as success, product = %+v", product)
	}
}

// ── Malformed bodies ─────────────────────────────────────────────────────────

func TestNonJSONErrorBodyIsPreserved(t *testing.T) {
	// Gateways and proxies return HTML; it is the only clue the caller has.
	const body = `<html><body>502 Bad Gateway</body></html>`

	_, err := clientAgainst(t, http.StatusBadGateway, body).Products.Get(1)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Errorf("raw body was discarded: %q", err.Error())
	}
}

func TestEmptyErrorBodyFallsBackToStatusText(t *testing.T) {
	_, err := clientAgainst(t, http.StatusInternalServerError, "").Products.Get(1)
	if err == nil {
		t.Fatal("expected an error")
	}
	// Previously rendered as "... (Status 500): " with nothing after the colon.
	if !strings.Contains(err.Error(), http.StatusText(http.StatusInternalServerError)) {
		t.Errorf("no fallback message for an empty body: %q", err.Error())
	}
}

// ── Full status → type mapping ───────────────────────────────────────────────

func TestStatusToTypeMapping(t *testing.T) {
	const msg = `{"error":"nope"}`

	cases := []struct {
		status int
		body   string
		check  func(t *testing.T, err error)
	}{
		{422, `{"errors":[{"field":"Email","message":"Email is invalid"}]}`, func(t *testing.T, err error) {
			var e *qberrors.ValidationError
			if !errors.As(err, &e) {
				t.Fatalf("got %T", err)
			}
			if e.StatusCode != 422 || len(e.Fields) != 1 {
				t.Errorf("status=%d fields=%v", e.StatusCode, e.Fields)
			}
		}},
		{403, msg, func(t *testing.T, err error) {
			var e *qberrors.ForbiddenError
			if !errors.As(err, &e) || e.StatusCode != 403 || e.Message != "nope" {
				t.Fatalf("got %T: %v", err, err)
			}
		}},
		{409, `{"error":"Subscription is not due for billing yet"}`, func(t *testing.T, err error) {
			var e *qberrors.ConflictError
			if !errors.As(err, &e) || e.StatusCode != 409 {
				t.Fatalf("got %T: %v", err, err)
			}
			if !strings.Contains(e.Error(), "not due for billing") {
				t.Errorf("message lost: %q", e.Error())
			}
		}},
		{418, msg, func(t *testing.T, err error) {
			// No dedicated type: the base type with the status, and NOT a ValidationError.
			var v *qberrors.ValidationError
			if errors.As(err, &v) {
				t.Fatal("an unmapped 4xx must not be reported as a validation failure")
			}
			var e *qberrors.QBitFlowError
			if !errors.As(err, &e) || e.StatusCode != 418 {
				t.Fatalf("got %T: %v", err, err)
			}
			if _, isServer := err.(*qberrors.ServerError); isServer {
				t.Fatal("an unmapped 4xx must not be a ServerError")
			}
		}},
		{500, msg, func(t *testing.T, err error) {
			var e *qberrors.ServerError
			if !errors.As(err, &e) || e.StatusCode != 500 {
				t.Fatalf("got %T: %v", err, err)
			}
		}},
		{503, `<html>maintenance</html>`, func(t *testing.T, err error) {
			var e *qberrors.ServerError
			if !errors.As(err, &e) || !strings.Contains(e.Message, "maintenance") {
				t.Fatalf("got %T: %v", err, err)
			}
		}},
		{307, ``, func(t *testing.T, err error) {
			var e *qberrors.ServerError
			if !errors.As(err, &e) || e.StatusCode != 307 {
				t.Fatalf("a redirect must be a ServerError, got %T: %v", err, err)
			}
		}},
	}

	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			_, err := clientAgainst(t, tc.status, tc.body).Products.Get(1)
			if err == nil {
				t.Fatal("expected an error")
			}
			tc.check(t, err)
		})
	}
}

func TestRateLimitCarriesRetryAfter(t *testing.T) {
	t.Run("with header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "12")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"too many requests"}`))
		}))
		t.Cleanup(server.Close)
		client := qbitflow.NewWithConfig(qbf.Config{APIKey: "k", BaseURL: server.URL, DisableRetries: true})

		_, err := client.Products.Get(1)

		var e *qberrors.RateLimitError
		if !errors.As(err, &e) {
			t.Fatalf("expected *qberrors.RateLimitError, got %T: %v", err, err)
		}
		if e.RetryAfter == nil || *e.RetryAfter != 12 {
			t.Errorf("RetryAfter = %v, want 12", e.RetryAfter)
		}
		if !strings.Contains(e.Error(), "retry after 12s") {
			t.Errorf("Error() = %q", e.Error())
		}
	})

	t.Run("without header", func(t *testing.T) {
		_, err := clientAgainst(t, http.StatusTooManyRequests, `{"error":"slow down"}`).Products.Get(1)
		var e *qberrors.RateLimitError
		if !errors.As(err, &e) {
			t.Fatalf("expected *qberrors.RateLimitError, got %T: %v", err, err)
		}
		if e.RetryAfter != nil {
			t.Errorf("RetryAfter = %v, want nil", *e.RetryAfter)
		}
	})
}

func TestFieldErrorsAreParsedForAnyStatus(t *testing.T) {
	// The errors[] list used to be consulted only on 400/422; the API may attach it to any
	// status, and the fields are the actionable part.
	const body = `{"error":"forbidden field","errors":[{"field":"OrganizationFeeBps","message":"requires admin authority"}]}`

	_, err := clientAgainst(t, http.StatusForbidden, body).Products.Get(1)

	var e *qberrors.ForbiddenError
	if !errors.As(err, &e) {
		t.Fatalf("expected *qberrors.ForbiddenError, got %T: %v", err, err)
	}
	if len(e.Fields) != 1 || e.Fields[0].Field != "OrganizationFeeBps" {
		t.Errorf("fields = %+v", e.Fields)
	}
	// `error` wins the message; the field list stays in Fields.
	if e.Message != "forbidden field" {
		t.Errorf("message = %q", e.Message)
	}
	if got := qberrors.FieldsOf(err); len(got) != 1 {
		t.Errorf("FieldsOf = %+v", got)
	}
	if got := qberrors.StatusCodeOf(err); got != 403 {
		t.Errorf("StatusCodeOf = %d", got)
	}
}

func TestMessagePrecedence(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"error beats errors[] and message", `{"error":"E","errors":[{"field":"F","message":"M"}],"message":"X"}`, "E"},
		{"errors[] beats message", `{"errors":[{"field":"F","message":"M"}],"message":"X"}`, "F: M"},
		{"message alone", `{"message":"only message"}`, "only message"},
		{"non-JSON body", `plain text failure`, "plain text failure"},
		{"empty body falls back to status text", ``, http.StatusText(http.StatusForbidden)},
		{"errors[] of the wrong shape is ignored", `{"error":"E","errors":"not a list"}`, "E"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clientAgainst(t, http.StatusForbidden, tc.body).Products.Get(1)
			var e *qberrors.ForbiddenError
			if !errors.As(err, &e) {
				t.Fatalf("got %T: %v", err, err)
			}
			if e.Message != tc.want {
				t.Errorf("message = %q, want %q", e.Message, tc.want)
			}
		})
	}
}

func TestNetworkFailureIsANetworkError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	client := qbitflow.NewWithConfig(qbf.Config{APIKey: "k", BaseURL: url, DisableRetries: true})
	_, err := client.Products.Get(1)

	var e *qberrors.NetworkError
	if !errors.As(err, &e) {
		t.Fatalf("expected *qberrors.NetworkError, got %T: %v", err, err)
	}
	if e.StatusCode != 0 || e.Err == nil {
		t.Errorf("network error must have status 0 and a cause: %+v", e.QBitFlowError)
	}
	if !strings.HasPrefix(e.Error(), "Network Error: ") {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestClientSideValidationErrorHasStatusZero(t *testing.T) {
	_, err := clientAgainst(t, http.StatusOK, `{}`).Products.Get(0)

	var e *qberrors.ValidationError
	if !errors.As(err, &e) {
		t.Fatalf("got %T: %v", err, err)
	}
	if e.StatusCode != 0 || len(e.Fields) != 0 {
		t.Errorf("client-side validation must carry status 0 and no fields: %+v", e)
	}
}

// ── One base type for every error ────────────────────────────────────────────

func TestEveryErrorMatchesTheBaseType(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"validation", http.StatusBadRequest, `{"errors":[{"field":"Price","message":"too low"}]}`},
		{"not found", http.StatusNotFound, `{"error":"resource not found"}`},
		{"unauthorized", http.StatusUnauthorized, `{"error":"no"}`},
		{"conflict", http.StatusConflict, `{"error":"not due"}`},
		{"server", http.StatusInternalServerError, `{"error":"boom"}`},
		{"other 4xx", http.StatusGone, `{"error":"gone"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clientAgainst(t, tc.status, tc.body).Products.Get(1)

			var base *qberrors.QBitFlowError
			if !errors.As(err, &base) {
				t.Fatalf("%T does not match *qberrors.QBitFlowError", err)
			}
			if base.StatusCode != tc.status || base.Message == "" {
				t.Errorf("base view = %+v", base)
			}

			// Wrapped by the caller, the helpers still find the status and fields.
			wrapped := fmt.Errorf("loading product: %w", err)
			if qberrors.StatusCodeOf(wrapped) != tc.status {
				t.Errorf("StatusCodeOf(wrapped) = %d, want %d", qberrors.StatusCodeOf(wrapped), tc.status)
			}
			if tc.status == http.StatusBadRequest && len(qberrors.FieldsOf(wrapped)) != 1 {
				t.Errorf("FieldsOf(wrapped) = %+v", qberrors.FieldsOf(wrapped))
			}
		})
	}

	t.Run("client-side validation", func(t *testing.T) {
		_, err := clientAgainst(t, http.StatusOK, `{}`).Products.Get(0)
		var base *qberrors.QBitFlowError
		if !errors.As(err, &base) || base.StatusCode != 0 {
			t.Errorf("a client-side ValidationError must match the base type with status 0, got %T: %v", err, err)
		}
	})

	t.Run("non-SDK errors", func(t *testing.T) {
		if qberrors.StatusCodeOf(errors.New("plain")) != 0 || qberrors.FieldsOf(nil) != nil {
			t.Error("a non-SDK error carries no status or fields")
		}
	})
}

// ── Redirects and empty bodies through the public client ─────────────────────

func TestRealRedirectIsAServerErrorAndIsNotFollowed(t *testing.T) {
	var contacted int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacted++
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(target.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/product/", http.StatusFound)
	}))
	t.Cleanup(api.Close)

	client := qbitflow.NewWithConfig(qbf.Config{APIKey: "sk_secret", BaseURL: api.URL, DisableRetries: true})

	// Followed, a redirected DELETE would become a GET answered by the other host — and be
	// reported as a successful delete.
	err := client.Products.Delete(1)
	var serverErr *qberrors.ServerError
	if !errors.As(err, &serverErr) || serverErr.StatusCode != http.StatusFound {
		t.Fatalf("expected a ServerError with status 302, got %T: %v", err, err)
	}

	_, err = client.Products.Create(&qbf.CreateProduct{Name: "Widget", Description: "A widget", Price: 1})
	if !errors.As(err, &serverErr) {
		t.Fatalf("expected a ServerError, got %T: %v", err, err)
	}
	if contacted != 0 {
		t.Errorf("the redirect target was contacted %d time(s) with the API key", contacted)
	}
}

func TestEmpty2xxWhereAResultIsExpectedIsAServerError(t *testing.T) {
	_, err := clientAgainst(t, http.StatusOK, "").Payments.GetPayment("pay@1")

	var serverErr *qberrors.ServerError
	if !errors.As(err, &serverErr) || serverErr.StatusCode != http.StatusOK {
		t.Fatalf("expected a ServerError with status 200, got %T: %v", err, err)
	}

	// A DELETE has nothing to decode: an empty 2xx is its success.
	if err := clientAgainst(t, http.StatusOK, "").Customers.Delete("c1"); err != nil {
		t.Errorf("an empty 2xx DELETE must succeed, got %v", err)
	}
}

// ── CSV export errors ────────────────────────────────────────────────────────

func TestCSVExportParsesJSONErrorBodies(t *testing.T) {
	params := qbf.AccountingExportParams{From: "2026-01-01", To: "2026-04-10"}

	_, err := clientAgainst(t, http.StatusBadRequest, `{"error":"'to' date cannot be more than 3 months after 'from' date"}`).
		Accounting.ExportCSV(params)
	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) || !strings.Contains(validationErr.Message, "3 months") {
		t.Fatalf("expected the API's ValidationError, got %T: %v", err, err)
	}

	_, err = clientAgainst(t, http.StatusForbidden, `{"error":"forbidden"}`).Accounting.ExportCSV(params)
	var forbidden *qberrors.ForbiddenError
	if !errors.As(err, &forbidden) || forbidden.Message != "forbidden" {
		t.Fatalf("expected a ForbiddenError, got %T: %v", err, err)
	}
}
