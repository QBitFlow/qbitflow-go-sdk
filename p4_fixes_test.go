package qbitflow

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
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

// Every answer is a JSON object (lists are pages): a null body is a ServerError, for a list too.
func TestNullResponseBody(t *testing.T) {
	ts := newTestServer(t, staticReply(200, "null"))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	if _, err := c.Products.Get(ctx, "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f"); !errors.As(err, new(*ServerError)) {
		t.Fatalf("null object body: want *ServerError, got %v", err)
	}
	if _, err := c.Products.List(ctx, nil); !errors.As(err, new(*ServerError)) {
		t.Fatalf("null page body: want *ServerError, got %v", err)
	}
	if _, err := c.Currencies.List(ctx, nil); !errors.As(err, new(*ServerError)) {
		t.Fatalf("null bounded list body: want *ServerError, got %v", err)
	}
}

// A "." or ".." path segment would be resolved as a relative path: refused before any request.
// A reference is never "." or ".." either (the API's reference rule), so GetByReference refuses
// them too; "..." is an ordinary reference, sent as the filter.
func TestDotPathSegmentsRefused(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		ref, _ := json.Marshal(r.URL.Query().Get("reference")) // the item carries the reference asked for
		jsonReply(w, 200, `{"items":[{"uuid":"sub@1","reference":`+string(ref)+`}],"hasMore":false}`)
	})
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	for _, id := range []string{".", ".."} {
		_, err := c.Webhooks.Events.Get(ctx, id)
		var ve *ValidationError
		if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "id" {
			t.Errorf("event id %q: want a ValidationError on id, got %v", id, err)
		}
		_, err = c.Subscriptions.GetByReference(ctx, id)
		if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "reference" {
			t.Errorf("reference %q: want a ValidationError on reference, got %v", id, err)
		}
	}
	if n := len(ts.recorded()); n != 0 {
		t.Fatalf("%d requests sent", n)
	}
	if _, err := c.Subscriptions.GetByReference(ctx, "..."); err != nil {
		t.Fatalf(`reference "..." must be sent as a filter: %v`, err)
	}
	if got := ts.recorded()[0].query; got != "reference=..." {
		t.Errorf("query %q", got)
	}
}

// Data that does not match its known type is a 400 even when no typed handler is registered.
func TestRouterRefusesMalformedDataWithoutTypedHandler(t *testing.T) {
	const secret = "whsec_router"
	body := []byte(`{"id":"evt_x","type":"payment.completed","version":"v2","test":true,"createdAt":"2026-10-01T12:00:00Z","data":{"amount":10}}`)
	now := time.Now()
	anyRan := false
	r := NewWebhookRouter(secret).OnAny(func(context.Context, *Event) error { anyRan = true; return nil })
	res := r.Handle(context.Background(), body, SignWebhook(body, secret, now))
	if res.Status != 400 || anyRan {
		t.Fatalf("want 400 and no handler run, got %d (onAny ran: %v, err %v)", res.Status, anyRan, res.Err)
	}
}
