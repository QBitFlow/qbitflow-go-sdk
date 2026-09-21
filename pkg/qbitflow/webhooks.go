package qbf

import (
	"encoding/json"

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

// VerifyWebhook verifies the authenticity of a webhook request using the provided payload, signature, and timestamp
func (s *WebhookService) Verify(payload []byte, signature, timestamp string) (bool, error) {
	if signature == "" || timestamp == "" {
		return false, qberrors.NewQBitFlowError("missing required webhook headers for verification", 0, nil)
	}

	var result any

	if err := s.client.makeRequest("POST", BASE_URL+"/verify", map[string]any{
		"payload":           json.RawMessage(payload),
		"receivedSignature": signature,
		"receivedTimestamp": timestamp,
	}, &result); err != nil {
		return false, err
	}
	return true, nil
}
