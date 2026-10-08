package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
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

	client := qbitflow.New("sk_dummy_offline_tests")
	client.SetBaseURL(server.URL)

	return client
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

	var apiErr *qberrors.QBitFlowError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *qberrors.QBitFlowError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
	if apiErr.Message != "invalid or missing authentication token" {
		t.Errorf("message = %q", apiErr.Message)
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
