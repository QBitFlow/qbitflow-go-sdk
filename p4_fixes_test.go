package qbitflow

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// The signature covers t exactly as the header carries it, leading zeros included.
func TestVerifyWebhookSignsRawTimestampText(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"webhook.test","version":"v2","test":true,"createdAt":"2026-10-01T12:00:00Z","data":{}}`)
	const secret = "whsec_raw_t"
	rawT := "0001790856000"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawT + "."))
	mac.Write(body)
	overRaw := hex.EncodeToString(mac.Sum(nil))
	clock := WithClock(func() time.Time { return time.Unix(1790856000, 0) })

	if err := VerifyWebhook(body, "t="+rawT+",v1="+overRaw, secret, clock); err != nil {
		t.Fatalf("signature over the raw t text must verify: %v", err)
	}
	overInt := sign(secret, 1790856000, body) // the integer form "1790856000"
	err := VerifyWebhook(body, "t="+rawT+",v1="+overInt, secret, clock)
	var se *WebhookSignatureError
	if !errors.As(err, &se) || se.Reason != WebhookSignatureNoMatchingSignature {
		t.Fatalf("signature over the re-formatted t must not verify, got %v", err)
	}
	// A t that does not fit an int64 is malformed, not "outside tolerance".
	err = VerifyWebhook(body, "t=99999999999999999999,v1="+overRaw, secret, clock)
	if !errors.As(err, &se) || se.Reason != WebhookSignatureMalformedHeader {
		t.Fatalf("oversized t must be malformedHeader, got %v", err)
	}
	// A whitespace-only header is a missing one.
	err = VerifyWebhook(body, "  \t ", secret, clock)
	if !errors.As(err, &se) || se.Reason != WebhookSignatureMissingHeader {
		t.Fatalf("blank header must be missingHeader, got %v", err)
	}
}

// A JSON null is an empty list for a list result but a ServerError where an object is expected.
func TestNullResponseBody(t *testing.T) {
	ts := newTestServer(t, staticReply(200, "null"))
	c, _ := newTestClient(t, ts)
	if _, err := c.Products.Get(context.Background(), "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f"); err == nil {
		t.Fatal("null object body must fail")
	} else if se := (*ServerError)(nil); !errors.As(err, &se) {
		t.Fatalf("want *ServerError, got %T: %v", err, err)
	}
	list, err := c.Products.List(context.Background(), nil)
	if err != nil || len(list) != 0 {
		t.Fatalf("null list body must decode to an empty list, got %v, %v", list, err)
	}
}
