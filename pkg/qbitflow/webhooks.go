package qbf

import (
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/pkg/errors"
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
)

func NewWebhookService(client *Client) *WebhookService {
	return &WebhookService{client: client}
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

// Interface to represent any webhook payload data that can be passed to the Verify method for verification.
type WebhookData interface {
	GetWebhookData() any
}

// VerifyWebhook verifies the authenticity of a webhook request using the provided payload, signature, and timestamp
func (s *WebhookService) Verify(payload WebhookData, signature, timestamp string) (bool, error) {
	if signature == "" || timestamp == "" {
		return false, qberrors.NewQBitFlowError("missing required webhook headers for verification", 0, nil)
	}

	var result any

	if err := s.client.makeRequest("POST", BASE_URL+"/verify", map[string]any{
		"payload":           payload,
		"receivedSignature": signature,
		"receivedTimestamp": timestamp,
	}, &result); err != nil {
		return false, err
	}
	return true, nil
}
