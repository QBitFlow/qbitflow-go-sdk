package qbitflow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		code   string
		want   any // a nil pointer of the expected concrete type
	}{
		{400, "validation_failed", (*ValidationError)(nil)},
		{400, "bad_request", (*BadRequestError)(nil)},
		{400, "foreign_key_violation", (*BadRequestError)(nil)},
		{400, "invalid_signature", (*BadRequestError)(nil)},
		{400, "", (*BadRequestError)(nil)},
		{401, "unauthorized", (*AuthenticationError)(nil)},
		{403, "forbidden", (*PermissionDeniedError)(nil)},
		{403, "policy_disabled", (*PermissionDeniedError)(nil)},
		{403, "plan_required", (*PermissionDeniedError)(nil)},
		{404, "not_found", (*NotFoundError)(nil)},
		{409, "unique_violation", (*ConflictError)(nil)},
		{409, "tx_already_sent", (*ConflictError)(nil)},
		{409, "merchant_not_ready", (*ConflictError)(nil)},
		{409, "held_funds_pending", (*ConflictError)(nil)},
		{409, "idempotency_key_in_use", (*ConflictError)(nil)},
		{410, "merchant_closed", (*GoneError)(nil)},
		{422, "idempotency_key_reused", (*IdempotencyError)(nil)},
		{422, "something_else", (*APIError)(nil)},
		{413, "request_too_large", (*APIError)(nil)},
		{405, "method_not_allowed", (*APIError)(nil)},
		{429, "rate_limit_exceeded", (*RateLimitError)(nil)},
		{500, "internal", (*ServerError)(nil)},
		{503, "network_unavailable", (*ServerError)(nil)},
		{504, "timeout", (*ServerError)(nil)},
		{301, "", (*ServerError)(nil)},
		{304, "", (*ServerError)(nil)},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d %s", tc.status, tc.code), func(t *testing.T) {
			body := fmt.Sprintf(`{"error":"the message","code":%q,"requestId":"req-body"}`, tc.code)
			err := errorFromResponse(&response{status: tc.status, header: http.Header{}, body: []byte(body)}, time.Now())
			if reflect.TypeOf(err) != reflect.TypeOf(tc.want) {
				t.Fatalf("type %T, want %T", err, tc.want)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("every error must unwrap to *APIError")
			}
			var base QBitFlowError
			if !errors.As(err, &base) {
				t.Fatal("every error must be a QBitFlowError")
			}
			if apiErr.StatusCode != tc.status || apiErr.Code != tc.code || apiErr.Message != "the message" || apiErr.RequestID != "req-body" {
				t.Errorf("payload %+v", apiErr)
			}
			if apiErr.Details == nil {
				t.Error("Details must never be nil")
			}
		})
	}
}

func TestErrorDetailsAndFields(t *testing.T) {
	body := `{
		"error": "name must be at least 2 characters",
		"code": "validation_failed",
		"details": {"errors": [
			{"field": "name", "message": "name must be at least 2 characters"},
			{"field": "frequency.unit", "message": "frequency.unit is required", "reason": "reserved"},
			"garbage",
			{"other": 1}
		], "max": 5},
		"errors": [{"field": "TOPLEVEL", "message": "must be ignored"}],
		"requestId": "abc-123",
		"debug": "stack trace"
	}`
	err := errorFromResponse(&response{status: 400, header: http.Header{"X-Request-Id": {"from-header"}}, body: []byte(body)}, time.Now())
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %T", err)
	}
	want := []FieldError{{"name", "name must be at least 2 characters"}, {"frequency.unit", "frequency.unit is required"}}
	if !reflect.DeepEqual(ve.FieldErrors, want) {
		t.Errorf("FieldErrors = %+v", ve.FieldErrors)
	}
	if ve.RequestID != "abc-123" {
		t.Errorf("RequestID = %q (the body wins over the header)", ve.RequestID)
	}
	if ve.Details["max"] != float64(5) {
		t.Errorf("Details = %v", ve.Details)
	}
	wantMsg := "name must be at least 2 characters (status 400, code validation_failed, request abc-123); " +
		"name: name must be at least 2 characters; frequency.unit: frequency.unit is required"
	if ve.Error() != wantMsg {
		t.Errorf("Error() =\n%q\nwant\n%q", ve.Error(), wantMsg)
	}
	if string(ve.RawBody) != body {
		t.Error("RawBody must keep the body (debug included)")
	}
}

func TestErrorFallbacks(t *testing.T) {
	cases := []struct {
		name, body, header string
		status             int
		message, requestID string
		code               string
	}{
		{"no body", "", "hdr-1", 404, "not found", "hdr-1", ""},
		{"html body", "<html>gateway</html>", "", 502, "bad gateway", "", ""},
		{"json array", `[1,2]`, "hdr-2", 500, "internal server error", "hdr-2", ""},
		{"wrong types", `{"error": 5, "code": ["x"], "details": "str", "requestId": 7}`, "hdr-3", 403, "forbidden", "hdr-3", ""},
		{"legacy code", `{"error":"boom","code":"error"}`, "", 409, "boom", "", "error"},
		{"header only id", `{"error":"x"}`, "hdr-4", 401, "x", "hdr-4", ""},
		{"unknown status", `{}`, "", 499, "http status 499", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("X-Request-Id", tc.header)
			}
			err := errorFromResponse(&response{status: tc.status, header: h, body: []byte(tc.body)}, time.Now())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("got %T", err)
			}
			if apiErr.Message != tc.message || apiErr.RequestID != tc.requestID || apiErr.Code != tc.code {
				t.Errorf("got message %q request %q code %q", apiErr.Message, apiErr.RequestID, apiErr.Code)
			}
			if apiErr.Details == nil || len(apiErr.FieldErrors) != 0 {
				t.Errorf("details %v fields %v", apiErr.Details, apiErr.FieldErrors)
			}
		})
	}
}

func TestErrorMessageFormat(t *testing.T) {
	cases := []struct {
		err  *APIError
		want string
	}{
		{&APIError{Message: "not found", StatusCode: 404, Code: "not_found", RequestID: "r1"}, "not found (status 404, code not_found, request r1)"},
		{&APIError{Message: "boom", StatusCode: 500}, "boom (status 500)"},
		{&APIError{Message: "validation failed", FieldErrors: []FieldError{{"apiKey", "apiKey is required"}}}, "validation failed; apiKey: apiKey is required"},
		{&APIError{Message: "request failed", Err: errors.New("dial tcp: refused")}, "request failed: dial tcp: refused"},
		{&APIError{}, "qbitflow error"},
	}
	for _, tc := range cases {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}

func TestErrorsThroughTheClient(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		w.Header().Set("X-Request-Id", "hdr-id")
		jsonReply(w, 403, `{"error":"members.products is off","code":"policy_disabled","details":{"policy":"members.products"}}`)
	})
	c, _ := newTestClient(t, ts)
	_, err := c.Me(context.Background())
	var pd *PermissionDeniedError
	if !errors.As(err, &pd) {
		t.Fatalf("got %T", err)
	}
	if pd.Details["policy"] != "members.products" || pd.RequestID != "hdr-id" || pd.Code != "policy_disabled" {
		t.Errorf("%+v", pd.APIError)
	}
	// errors.As reaches the payload through the concrete type.
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr != pd.APIError {
		t.Error("errors.As(*APIError) must return the embedded payload")
	}
}

func TestClientSideErrors(t *testing.T) {
	ve := newFieldError("price", "must be a number above 0")
	if ve.StatusCode != 0 || ve.Code != "" || ve.Details == nil {
		t.Errorf("%+v", ve.APIError)
	}
	if ve.Error() != "validation failed; price: price must be a number above 0" {
		t.Errorf("Error() = %q", ve.Error())
	}
	var base QBitFlowError
	if !errors.As(error(ve), &base) {
		t.Error("a client-side error is a QBitFlowError")
	}
}

func TestAsWebhookSignatureError(t *testing.T) {
	mk := func(status int, code string) error {
		return errorFromResponse(&response{status: status, header: http.Header{}, body: []byte(`{"error":"x","code":"` + code + `"}`)}, time.Now())
	}
	err := asWebhookSignatureError(mk(400, "invalid_signature"))
	var se *WebhookSignatureError
	if !errors.As(err, &se) || se.Reason != WebhookSignatureInvalidSignature || se.StatusCode != 400 {
		t.Errorf("invalid_signature: %#v", err)
	}
	for _, other := range []error{mk(400, "validation_failed"), mk(400, "bad_request"), mk(404, "not_found"), nil} {
		if got := asWebhookSignatureError(other); got != other {
			t.Errorf("%v must be unchanged, got %v", other, got)
		}
	}
}
