package qbf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

// BASE_URL is the API path prefix of the webhook routes (not the client base URL).
const BASE_URL = "/webhooks"

// WebhookService verifies incoming webhooks. Verify asks the API to check the signature
// (no secret needed); VerifyLocal and the package-level VerifyWebhookSignature check it
// locally with your webhook secret and no network call.
type WebhookService struct {
	client *Client
}

// Header names carried by every webhook QBitFlow sends.
const (
	// HeaderSignature carries the HMAC signature, formatted "sha256=<hex>".
	HeaderSignature = "X-Webhook-Signature-256"
	// HeaderTimestamp carries the send time, in unix seconds.
	HeaderTimestamp = "X-Webhook-Timestamp"
	// HeaderWebhookID carries the transaction id (e.g. "pay@<uuid>"), or TEST_WEBHOOK_ID
	// for the dashboard's connectivity probe.
	HeaderWebhookID = "X-Webhook-Id"

	// TEST_WEBHOOK_ID is the X-Webhook-Id value of the dashboard's "Test webhook" probe,
	// sent to confirm your endpoint is reachable. Verify its signature like any other
	// delivery, then acknowledge it with a 200 without processing.
	TEST_WEBHOOK_ID = "test-webhook-id"
)

// NewWebhookService creates the webhook service bound to client.
func NewWebhookService(client *Client) *WebhookService {
	return &WebhookService{client: client}
}

// OnBehalfOf returns a WebhookService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *WebhookService) OnBehalfOf(userID uint64) *WebhookService {
	return NewWebhookService(s.client.withOnBehalfOf(userID))
}

// GetSignatureHeader returns HeaderSignature ("X-Webhook-Signature-256").
func (s *WebhookService) GetSignatureHeader() string {
	return HeaderSignature
}

// GetTimestampHeader returns HeaderTimestamp ("X-Webhook-Timestamp").
func (s *WebhookService) GetTimestampHeader() string {
	return HeaderTimestamp
}

// GetWebhookIDHeader returns HeaderWebhookID ("X-Webhook-Id").
func (s *WebhookService) GetWebhookIDHeader() string {
	return HeaderWebhookID
}

// Verify verifies the authenticity of a webhook request using the provided payload,
// signature, and timestamp, by asking the QBitFlow API (no secret needed).
//
// payload is the webhook body: raw JSON ([]byte, json.RawMessage or string — pass the
// body exactly as received) or an already-decoded generic value such as the
// map[string]any produced by json.Unmarshal. Raw JSON is forwarded verbatim, so numbers
// and the {} / [] distinction survive the round-trip. Do not pass an SDK struct such as
// SessionWebhookResponse: re-encoding it drops fields the struct does not model, and the
// signature then no longer matches. A missing payload, invalid JSON, or a value that
// cannot be encoded (NaN, ±Inf) is a *qberrors.ValidationError, raised before any request.
//
// The API answers 200 when the signature is valid and 400 when it is not, so only an
// HTTP 400 yields (false, nil). Every other failure — an unreachable API, a 5xx, an
// expired API key (401), an insufficient role (403), even a 422 — comes back as its own
// typed error with ok=false.
//
// That distinction matters: reporting a failure as "not verified" would make an outage
// indistinguishable from a forged signature, and a handler that drops unverified events
// would silently discard real payments for as long as the outage lasted. On a non-nil
// error, answer non-200 so QBitFlow retries the delivery.
//
//	ok, err := client.Webhooks.Verify(body, sig, ts)
//	if err != nil {
//	    w.WriteHeader(http.StatusServiceUnavailable) // transient: let QBitFlow retry
//	    return
//	}
//	if !ok {
//	    w.WriteHeader(http.StatusBadRequest) // genuinely rejected
//	    return
//	}
func (s *WebhookService) Verify(payload any, signature, timestamp string) (bool, error) {
	if signature == "" || timestamp == "" {
		return false, qberrors.NewValidationError("missing required webhook headers for verification")
	}

	raw, err := webhookPayloadJSON(payload)
	if err != nil {
		return false, err
	}

	var result any

	if err := s.client.makeRequest("POST", BASE_URL+"/verify", map[string]any{
		"payload":           raw,
		"receivedSignature": signature,
		"receivedTimestamp": timestamp,
	}, &result); err != nil {
		// A 400 is the API's "signature rejected" answer — a verification result, not an
		// error the caller has to handle.
		var validationErr *qberrors.ValidationError
		if errors.As(err, &validationErr) && validationErr.StatusCode == http.StatusBadRequest {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

// webhookPayloadJSON turns a webhook payload into the JSON sent to the verify endpoint.
// Raw JSON is validated and forwarded verbatim; any other value is encoded.
func webhookPayloadJSON(payload any) (json.RawMessage, error) {
	var raw []byte
	switch v := payload.(type) {
	case nil:
		return nil, qberrors.NewValidationError("webhook payload is required")
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	case string:
		raw = []byte(v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload cannot be encoded as JSON: %v", err))
		}
		return encoded, nil
	}

	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, qberrors.NewValidationError("webhook payload is required")
	}
	if !json.Valid(raw) {
		return nil, qberrors.NewValidationError("webhook payload is not valid JSON")
	}
	return json.RawMessage(raw), nil
}
