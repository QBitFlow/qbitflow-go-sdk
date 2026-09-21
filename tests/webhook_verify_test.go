package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

const testSecret = "whsec_test_0123456789abcdef"

func nowTS() string { return strconv.FormatInt(time.Now().Unix(), 10) }

// ── Canonical JSON ───────────────────────────────────────────────────────────

func TestCanonicalJSONSortsKeys(t *testing.T) {
	t.Run("key order in the input does not change the output", func(t *testing.T) {
		a, err := qbf.CanonicalJSON([]byte(`{"b":2,"a":1,"c":3}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, err := qbf.CanonicalJSON([]byte(`{"c":3,"a":1,"b":2}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("canonical forms differ: %s vs %s", a, b)
		}
		if string(a) != `{"a":1,"b":2,"c":3}` {
			t.Errorf("unexpected canonical form: %s", a)
		}
	})

	t.Run("nested objects are sorted too", func(t *testing.T) {
		got, err := qbf.CanonicalJSON([]byte(`{"z":{"y":1,"x":2},"a":[{"d":4,"c":3}]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"a":[{"c":3,"d":4}],"z":{"x":2,"y":1}}`
		if string(got) != want {
			t.Errorf("nested sort wrong:\n got %s\nwant %s", got, want)
		}
	})

	t.Run("array order is preserved", func(t *testing.T) {
		// Arrays are ordered by definition: sorting them would change meaning.
		got, _ := qbf.CanonicalJSON([]byte(`{"items":[3,1,2]}`))
		if string(got) != `{"items":[3,1,2]}` {
			t.Errorf("array order must be preserved, got %s", got)
		}
	})

	t.Run("whitespace in the input is dropped", func(t *testing.T) {
		got, _ := qbf.CanonicalJSON([]byte("{\n  \"a\" : 1,\n  \"b\":  2\n}"))
		if string(got) != `{"a":1,"b":2}` {
			t.Errorf("whitespace not normalised, got %s", got)
		}
	})

	t.Run("HTML characters are escaped the way the Go server escapes them", func(t *testing.T) {
		// This pins the cross-SDK contract: encoding/json escapes <, > and & by default,
		// and the JS, Python and PHP SDKs reproduce it so all four agree byte for byte.
		//
		// The expected sequences are built from bytes rather than written as literals, so
		// no editor or tooling can silently un-escape them and make this test vacuous.
		got, err := qbf.CanonicalJSON([]byte(`{"url":"https://x.test/a?b=1&c=2","tag":"<b>"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(got)

		bs := string([]byte{92}) // a single backslash
		for raw, escaped := range map[string]string{
			"<": bs + "u003c",
			">": bs + "u003e",
			"&": bs + "u0026",
		} {
			if strings.Contains(s, raw) {
				t.Errorf("%q must be escaped in the canonical form, got %s", raw, s)
			}
			if !strings.Contains(s, escaped) {
				t.Errorf("expected %q to appear as %q, got %s", raw, escaped, s)
			}
		}
	})

	t.Run("invalid JSON is rejected", func(t *testing.T) {
		if _, err := qbf.CanonicalJSON([]byte(`{not json`)); err == nil {
			t.Error("expected invalid JSON to error")
		}
	})
}

// ── Signature computation ────────────────────────────────────────────────────

func TestComputeWebhookSignature(t *testing.T) {
	t.Run("is stable and correctly formatted", func(t *testing.T) {
		sig, err := qbf.ComputeWebhookSignature(testSecret, "1700000000", []byte(`{"a":1}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(sig, "sha256=") {
			t.Errorf("signature must carry the sha256= prefix, got %q", sig)
		}
		if len(sig) != len("sha256=")+64 {
			t.Errorf("expected a 64-char hex digest, got %q", sig)
		}
		again, _ := qbf.ComputeWebhookSignature(testSecret, "1700000000", []byte(`{"a":1}`))
		if sig != again {
			t.Error("signature is not deterministic")
		}
	})

	t.Run("key order does not change the signature", func(t *testing.T) {
		a, _ := qbf.ComputeWebhookSignature(testSecret, "1700000000", []byte(`{"a":1,"b":2}`))
		b, _ := qbf.ComputeWebhookSignature(testSecret, "1700000000", []byte(`{"b":2,"a":1}`))
		if a != b {
			t.Error("reordering keys must not change the signature")
		}
	})

	t.Run("the timestamp is part of the signed message", func(t *testing.T) {
		a, _ := qbf.ComputeWebhookSignature(testSecret, "1700000000", []byte(`{"a":1}`))
		b, _ := qbf.ComputeWebhookSignature(testSecret, "1700000001", []byte(`{"a":1}`))
		if a == b {
			t.Error("a different timestamp must produce a different signature")
		}
	})

	t.Run("the secret is part of the signed message", func(t *testing.T) {
		a, _ := qbf.ComputeWebhookSignature("secret-one", "1700000000", []byte(`{"a":1}`))
		b, _ := qbf.ComputeWebhookSignature("secret-two", "1700000000", []byte(`{"a":1}`))
		if a == b {
			t.Error("a different secret must produce a different signature")
		}
	})

	t.Run("a missing secret or timestamp is rejected", func(t *testing.T) {
		if _, err := qbf.ComputeWebhookSignature("", "1700000000", []byte(`{}`)); err == nil {
			t.Error("expected an empty secret to be rejected")
		}
		if _, err := qbf.ComputeWebhookSignature(testSecret, "", []byte(`{}`)); err == nil {
			t.Error("expected an empty timestamp to be rejected")
		}
	})
}

// ── Verification ─────────────────────────────────────────────────────────────

func TestVerifyWebhookSignature(t *testing.T) {
	payload := []byte(`{"uuid":"pay@abc","txType":"payment","status":{"status":"completed"}}`)

	t.Run("accepts a genuine webhook", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, nil); err != nil {
			t.Errorf("a genuine webhook must verify, got: %v", err)
		}
	})

	t.Run("accepts a payload whose keys were reordered in transit", func(t *testing.T) {
		// The whole point of canonicalisation: a proxy or framework may reorder keys.
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, []byte(`{"a":1,"b":2}`))
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, []byte(`{"b":2,"a":1}`), nil); err != nil {
			t.Errorf("reordered keys must still verify, got: %v", err)
		}
	})

	t.Run("rejects a tampered payload", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		tampered := []byte(`{"uuid":"pay@EVIL","txType":"payment","status":{"status":"completed"}}`)
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, tampered, nil); err == nil {
			t.Error("a tampered payload must be rejected")
		}
	})

	t.Run("rejects a tampered timestamp", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		other := strconv.FormatInt(time.Now().Unix()-30, 10)
		if err := qbf.VerifyWebhookSignature(testSecret, other, sig, payload, nil); err == nil {
			t.Error("a tampered timestamp must be rejected")
		}
	})

	t.Run("rejects the wrong secret", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if err := qbf.VerifyWebhookSignature("whsec_wrong", ts, sig, payload, nil); err == nil {
			t.Error("the wrong secret must be rejected")
		}
	})

	t.Run("rejects a replayed webhook outside the window", func(t *testing.T) {
		old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		sig, _ := qbf.ComputeWebhookSignature(testSecret, old, payload)
		err := qbf.VerifyWebhookSignature(testSecret, old, sig, payload, nil)
		if err == nil {
			t.Fatal("a 10-minute-old webhook must be rejected by the 5-minute default window")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("expected an expiry error, got: %v", err)
		}
	})

	t.Run("rejects a timestamp too far in the future", func(t *testing.T) {
		// Guards against a forged future timestamp extending the replay window.
		future := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
		sig, _ := qbf.ComputeWebhookSignature(testSecret, future, payload)
		if err := qbf.VerifyWebhookSignature(testSecret, future, sig, payload, nil); err == nil {
			t.Error("a far-future timestamp must be rejected")
		}
	})

	t.Run("honours a custom replay window", func(t *testing.T) {
		old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		sig, _ := qbf.ComputeWebhookSignature(testSecret, old, payload)
		opts := &qbf.VerifyOptions{MaxTimestampAge: 30 * time.Minute}
		if err := qbf.VerifyWebhookSignature(testSecret, old, sig, payload, opts); err != nil {
			t.Errorf("a 30-minute window must accept a 10-minute-old webhook, got: %v", err)
		}
	})

	t.Run("rejects a non-numeric timestamp", func(t *testing.T) {
		sig, _ := qbf.ComputeWebhookSignature(testSecret, "not-a-number", payload)
		if err := qbf.VerifyWebhookSignature(testSecret, "not-a-number", sig, payload, nil); err == nil {
			t.Error("a non-numeric timestamp must be rejected")
		}
	})

	t.Run("rejects an empty or malformed signature", func(t *testing.T) {
		ts := nowTS()
		for _, bad := range []string{"", "sha256=", "deadbeef", "sha512=abc"} {
			if err := qbf.VerifyWebhookSignature(testSecret, ts, bad, payload, nil); err == nil {
				t.Errorf("signature %q must be rejected", bad)
			}
		}
	})
}

// ── Request helper ───────────────────────────────────────────────────────────

func TestVerifyWebhookRequest(t *testing.T) {
	payload := []byte(`{"uuid":"pay@abc","txType":"payment"}`)

	newReq := func(ts, sig string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(payload))
		if sig != "" {
			r.Header.Set("X-Webhook-Signature-256", sig)
		}
		if ts != "" {
			r.Header.Set("X-Webhook-Timestamp", ts)
		}
		r.Header.Set("X-Webhook-Id", "pay@abc")
		return r
	}

	t.Run("verifies and returns the body", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)

		body, err := qbf.VerifyWebhookRequest(newReq(ts, sig), testSecret, nil)
		if err != nil {
			t.Fatalf("expected the request to verify, got: %v", err)
		}
		if !bytes.Equal(body, payload) {
			t.Errorf("body not returned intact: %s", body)
		}

		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("returned body must be decodable: %v", err)
		}
	})

	t.Run("rejects a request with missing headers", func(t *testing.T) {
		ts := nowTS()
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if _, err := qbf.VerifyWebhookRequest(newReq("", sig), testSecret, nil); err == nil {
			t.Error("a missing timestamp header must be rejected")
		}
		if _, err := qbf.VerifyWebhookRequest(newReq(ts, ""), testSecret, nil); err == nil {
			t.Error("a missing signature header must be rejected")
		}
	})

	t.Run("extracts headers regardless of casing", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(payload))
		r.Header.Set("x-webhook-signature-256", "sha256=abc")
		r.Header.Set("x-webhook-timestamp", "1700000000")
		r.Header.Set("x-webhook-id", "pay@abc")

		h := qbf.ExtractWebhookHeaders(r.Header)
		if h.Signature != "sha256=abc" || h.Timestamp != "1700000000" || h.WebhookID != "pay@abc" {
			t.Errorf("headers not extracted: %+v", h)
		}
	})

	t.Run("identifies the dashboard test webhook", func(t *testing.T) {
		h := qbf.WebhookHeaders{WebhookID: qbf.TEST_WEBHOOK_ID}
		if !h.IsTest() {
			t.Error("the test webhook id must be recognised")
		}
		if (qbf.WebhookHeaders{WebhookID: "pay@abc"}).IsTest() {
			t.Error("a real transaction id must not be flagged as a test")
		}
	})
}
