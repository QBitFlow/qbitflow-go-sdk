package qbf

import (
	"encoding/json"
	"errors"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

const BASE_URL = "/webhooks"

type WebhookService struct {
	client *Client
}

const (
	// HMAC headers
	HeaderSignature = "X-Webhook-Signature-256"
	HeaderTimestamp = "X-Webhook-Timestamp"
	HeaderWebhookID = "X-Webhook-ID"

	// Frontend test webhook ID for testing purposes and ensure that the webhook can be reached
	TEST_WEBHOOK_ID = "test-webhook-id"
)

func NewWebhookService(client *Client) *WebhookService {
	return &WebhookService{client: client}
}

// OnBehalfOf returns a WebhookService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *WebhookService) OnBehalfOf(userID uint64) *WebhookService {
	return NewWebhookService(s.client.withOnBehalfOf(userID))
}

func (s *WebhookService) GetSignatureHeader() string {
	return HeaderSignature
}

func (s *WebhookService) GetTimestampHeader() string {
	return HeaderTimestamp
}

func (s *WebhookService) GetWebhookIDHeader() string {
	return HeaderWebhookID
}

// Verify verifies the authenticity of a webhook request using the provided payload,
// signature, and timestamp.
//
// The API answers 200 when the signature is valid and 400 when it is not, so only a 400
// yields (false, nil). Every other failure — an unreachable API, a 5xx, an expired API
// key (401), an insufficient role (403) — comes back as a non-nil error.
//
// That distinction matters: reporting a failure as "not verified" would make an outage
// indistinguishable from a forged signature, and a handler that drops unverified events
// would silently discard real payments for as long as the outage lasted. On a non-nil
// error, answer non-200 so QBitFlow retries the delivery.
//
//	ok, err := client.Webhooks.Verify(body, sig, ts)
//	if err != nil {
//		w.WriteHeader(http.StatusServiceUnavailable) // transient: let QBitFlow retry
//		return
//	}
//	if !ok {
//		w.WriteHeader(http.StatusBadRequest) // genuinely rejected
//		return
//	}
func (s *WebhookService) Verify(payload []byte, signature, timestamp string) (bool, error) {
	if signature == "" || timestamp == "" {
		return false, qberrors.NewValidationError("missing required webhook headers for verification")
	}

	var result any

	if err := s.client.makeRequest("POST", BASE_URL+"/verify", map[string]any{
		"payload":           json.RawMessage(payload),
		"receivedSignature": signature,
		"receivedTimestamp": timestamp,
	}, &result); err != nil {
		// A 400 is the API's "signature rejected" answer — a verification result, not an
		// error the caller has to handle.
		var validationErr *qberrors.ValidationError
		if errors.As(err, &validationErr) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}
