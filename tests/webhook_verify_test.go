package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
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

// ── Cross-SDK golden vectors ─────────────────────────────────────────────────

// goldenVectors are generated with the Go reference implementation (the same
// encoding/json the server uses) and shared verbatim with the JS, Python and PHP SDKs, so
// all four are pinned to byte-identical canonical forms and signatures. They cover key
// ordering at every level, HTML/Unicode escaping (\u003c \u003e \u0026 \u2028 \u2029),
// float64 number rendering, empty containers and the real webhook shapes.
var goldenVectors = []struct {
	name      string
	payload   string
	secret    string
	timestamp string
	canonical string
	signature string
}{
	{
		name:      "empty_object",
		payload:   "{}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{}",
		signature: "sha256=122d17b8a94e9170db2d91edb1a4060a1847a8de8020e3a1a20ea84293f28d22",
	},
	{
		name:      "key_order_and_types",
		payload:   "{\"zeta\":1,\"alpha\":\"a\",\"Beta\":true,\"nested\":{\"y\":null,\"x\":[1,2,{\"b\":\"B\",\"a\":\"A\"}]},\"\u00e9moji\":\"\u2615\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"Beta\":true,\"alpha\":\"a\",\"nested\":{\"x\":[1,2,{\"a\":\"A\",\"b\":\"B\"}],\"y\":null},\"zeta\":1,\"\u00e9moji\":\"\u2615\"}",
		signature: "sha256=078c8862e2a4b67245ff3c04405d313c1815d19e3f246446583919260df7332d",
	},
	{
		name:      "html_and_unicode_escaping",
		payload:   "{\"msg\":\"<b>ok</b> & done / fine \\\"q\\\" \\\\ back\",\"line\":\"a\u2028b\u2029c\\ttab\\nnl\",\"dq\":\"<script>\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"dq\":\"\\u003cscript\\u003e\",\"line\":\"a\\u2028b\\u2029c\\ttab\\nnl\",\"msg\":\"\\u003cb\\u003eok\\u003c/b\\u003e \\u0026 done / fine \\\"q\\\" \\\\ back\"}",
		signature: "sha256=220a10812e56a3cd202440cdbe7fa255515979760c4d15f821b3a6fea34564a4",
	},
	{
		name:      "numbers_float64_semantics",
		payload:   "{\"n\":[1e-7,1e-6,0.00001234,0.0001234,0.001,1e15,1e16,1e20,1e21,1.5e22,123456789.123,0.1,100.0,10.5,0,5e-324,1.7976931348623157e308,9007199254740993,2.5e-5,1234567890123456789,19.99,-5]}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"n\":[1e-7,0.000001,0.00001234,0.0001234,0.001,1000000000000000,10000000000000000,100000000000000000000,1e+21,1.5e+22,123456789.123,0.1,100,10.5,0,5e-324,1.7976931348623157e+308,9007199254740992,0.000025,1234567890123456800,19.99,-5]}",
		signature: "sha256=2969a1f1c606f2cefbc8dadef939ccf2337d0388827819c888137454deec56b8",
	},
	{
		name:      "empty_containers",
		payload:   "{\"emptyObj\":{},\"emptyArr\":[],\"list\":[{},[],[[]],[{}]],\"s\":\"\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"emptyArr\":[],\"emptyObj\":{},\"list\":[{},[],[[]],[{}]],\"s\":\"\"}",
		signature: "sha256=de1584470053d814ee8903db86fbe07517e13a0282c1e682587ed77659f9231a",
	},
	{
		name:      "transaction_webhook",
		payload:   "{\"uuid\":\"pay@01997c89-d0e9-7c9a-9886-fe7709919695\",\"txType\":\"payment\",\"status\":{\"status\":\"completed\",\"txHash\":\"0x9f2c\",\"settlementDetails\":{\"feeBps\":100,\"txMetadata\":{\"networkFees\":{\"amount\":\"21000\",\"unitsConsumed\":21000},\"blockData\":{\"number\":\"1234567\",\"timestamp\":1758539990},\"mainCurrencyPriceUSD\":0.00001234},\"txAmounts\":{\"usd\":{\"platform\":0.2,\"merchant\":19.79},\"minUnits\":{\"platform\":\"200000\",\"organization\":\"0\",\"referral\":\"0\",\"merchant\":\"19790000\"}}}},\"session\":{\"uuid\":\"pay@01997c89-d0e9-7c9a-9886-fe7709919695\",\"reference\":\"ORD/2026/17\",\"productId\":42,\"productName\":\"Caf\u00e9 Premium\",\"description\":\"Best <coffee> & tea\",\"price\":19.99,\"successUrl\":\"https://shop.example.com/ok?o=17&x=1\",\"organizationId\":7,\"organizationName\":\"Acme\",\"feeBps\":100,\"userId\":0,\"test\":false,\"customerUUID\":\"3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f\",\"txType\":\"payment\",\"availableCurrencies\":[1,2,3]},\"managementPageLink\":\"https://app.qbitflow.app/m/abc\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"managementPageLink\":\"https://app.qbitflow.app/m/abc\",\"session\":{\"availableCurrencies\":[1,2,3],\"customerUUID\":\"3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f\",\"description\":\"Best \\u003ccoffee\\u003e \\u0026 tea\",\"feeBps\":100,\"organizationId\":7,\"organizationName\":\"Acme\",\"price\":19.99,\"productId\":42,\"productName\":\"Caf\u00e9 Premium\",\"reference\":\"ORD/2026/17\",\"successUrl\":\"https://shop.example.com/ok?o=17\\u0026x=1\",\"test\":false,\"txType\":\"payment\",\"userId\":0,\"uuid\":\"pay@01997c89-d0e9-7c9a-9886-fe7709919695\"},\"status\":{\"settlementDetails\":{\"feeBps\":100,\"txAmounts\":{\"minUnits\":{\"merchant\":\"19790000\",\"organization\":\"0\",\"platform\":\"200000\",\"referral\":\"0\"},\"usd\":{\"merchant\":19.79,\"platform\":0.2}},\"txMetadata\":{\"blockData\":{\"number\":\"1234567\",\"timestamp\":1758539990},\"mainCurrencyPriceUSD\":0.00001234,\"networkFees\":{\"amount\":\"21000\",\"unitsConsumed\":21000}}},\"status\":\"completed\",\"txHash\":\"0x9f2c\"},\"txType\":\"payment\",\"uuid\":\"pay@01997c89-d0e9-7c9a-9886-fe7709919695\"}",
		signature: "sha256=c6e845486a0c5d2b9f3d1fca4a9c4190649ed46b06db74d71bd2703358a411ec",
	},
	{
		name:      "subscription_billing_webhook",
		payload:   "{\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"subscriptionReference\":\"plan-pro\",\"type\":\"billing\",\"data\":{\"uuid\":\"sub-hist@01a0c5be-0000-7000-8000-000000000002\",\"createdAt\":\"2026-09-21T22:02:09.986381+02:00\",\"from\":\"0xfrom\",\"to\":\"0xto\",\"amount\":9.99,\"amountMinUnits\":\"9990000\",\"currencyId\":2,\"test\":true,\"transactionHash\":\"0xhash\",\"organizationId\":7,\"userId\":3,\"name\":\"Pro\",\"description\":\"Monthly\",\"productId\":9,\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"metadata\":{\"feeBps\":100,\"txMetadata\":{\"networkFees\":{\"amount\":\"0\",\"unitsConsumed\":0},\"blockData\":{\"number\":\"\",\"timestamp\":0}},\"txAmounts\":{\"usd\":{\"platform\":0.0999,\"merchant\":9.8901},\"minUnits\":{\"platform\":\"99900\",\"organization\":\"0\",\"referral\":\"0\",\"merchant\":\"9890100\"}}}}}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"data\":{\"amount\":9.99,\"amountMinUnits\":\"9990000\",\"createdAt\":\"2026-09-21T22:02:09.986381+02:00\",\"currencyId\":2,\"description\":\"Monthly\",\"from\":\"0xfrom\",\"metadata\":{\"feeBps\":100,\"txAmounts\":{\"minUnits\":{\"merchant\":\"9890100\",\"organization\":\"0\",\"platform\":\"99900\",\"referral\":\"0\"},\"usd\":{\"merchant\":9.8901,\"platform\":0.0999}},\"txMetadata\":{\"blockData\":{\"number\":\"\",\"timestamp\":0},\"networkFees\":{\"amount\":\"0\",\"unitsConsumed\":0}}},\"name\":\"Pro\",\"organizationId\":7,\"productId\":9,\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"test\":true,\"to\":\"0xto\",\"transactionHash\":\"0xhash\",\"userId\":3,\"uuid\":\"sub-hist@01a0c5be-0000-7000-8000-000000000002\"},\"subscriptionReference\":\"plan-pro\",\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"type\":\"billing\"}",
		signature: "sha256=c69da12a34355745027c9db89877cdd8187df497171c2ee591bf7a80abd36187",
	},
	{
		name:      "status_transition_webhook",
		payload:   "{\"type\":\"status_transition\",\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"data\":{\"previousStatus\":\"active\",\"currentStatus\":\"low_on_funds\",\"updatedAt\":\"2026-09-22T10:00:00Z\"}}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"data\":{\"currentStatus\":\"low_on_funds\",\"previousStatus\":\"active\",\"updatedAt\":\"2026-09-22T10:00:00Z\"},\"subscriptionUUID\":\"sub@01a0c5be-0000-7000-8000-000000000001\",\"type\":\"status_transition\"}",
		signature: "sha256=88708bfd796485695fd051a03a32ee3374ecf7b02ead27b1bbcf67504380f46a",
	},
	// The five vectors below (shared verbatim with the other SDKs, golden/vectors-new.json)
	// cover decimal strings of 19+ digits staying strings, C0 controls, lone surrogates
	// becoming U+FFFD, -0 and exponent forms, and keys ordered by UTF-8 bytes (not UTF-16
	// code units).
	{
		name:      "decimal strings of 19+ digits stay strings",
		payload:   "{\"amountMinUnits\":\"1000000000000000000\",\"txAmounts\":{\"minUnits\":{\"merchant\":\"99000000000000000000\",\"platform\":\"1000000000000000000000\",\"referral\":\"0\"}},\"big\":12345678901234567890,\"id\":\"0001\",\"neg\":\"-12345678901234567890\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"amountMinUnits\":\"1000000000000000000\",\"big\":12345678901234567000,\"id\":\"0001\",\"neg\":\"-12345678901234567890\",\"txAmounts\":{\"minUnits\":{\"merchant\":\"99000000000000000000\",\"platform\":\"1000000000000000000000\",\"referral\":\"0\"}}}",
		signature: "sha256=88423199655b9f5e5db5e4b4fbdaea52d61f64f08837cad0a65bab1ea87b6a19",
	},
	{
		name:      "control characters",
		payload:   "{\"s\":\"a\\bb\\fc\\u0000d\\u001fe\\u007ff\\u0080g\",\"k\\u0001\":\"x\",\"tab\":\"\\t\",\"cr\":\"\\r\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"cr\":\"\\r\",\"k\\u0001\":\"x\",\"s\":\"a\\bb\\fc\\u0000d\\u001fe\x7ff\u0080g\",\"tab\":\"\\t\"}",
		signature: "sha256=4118c0a289dd2e4bb4ab62ad2b13a7e1ff60981717c75d6c1ce1c4848e82d5f0",
	},
	{
		name:      "lone surrogates become U+FFFD",
		payload:   "{\"hi\":\"\\ud800x\",\"lo\":\"\\udfff\",\"pair\":\"\U0001f600\"}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"hi\":\"\ufffdx\",\"lo\":\"\ufffd\",\"pair\":\"\U0001f600\"}",
		signature: "sha256=b31ca0014545f32bf62e937955fe50e11c5a6cf68f57052008edd0cfb2af29d3",
	},
	{
		name:      "negative zero and exponent forms",
		payload:   "{\"a\":-0,\"b\":-0.0,\"c\":1E2,\"d\":1e-7,\"e\":-1.5e-10,\"f\":0.0,\"g\":-0.000001}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"a\":-0,\"b\":-0,\"c\":100,\"d\":1e-7,\"e\":-1.5e-10,\"f\":0,\"g\":-0.000001}",
		signature: "sha256=45e3033a130332ab91bce58115d670ac83a9d4a41b59f51323c4e20625bcd2bc",
	},
	{
		name:      "keys are ordered by UTF-8 bytes",
		payload:   "{\"\U0001f600\":1,\"\uffff\":2,\"a\":3,\"\u00e9\":4,\"Z\":5,\"\":6,\"aa\":7,\"a\\u0000\":8}",
		secret:    "whsec_golden_secret_0123456789",
		timestamp: "1758540000",
		canonical: "{\"\":6,\"Z\":5,\"a\":3,\"a\\u0000\":8,\"aa\":7,\"\u00e9\":4,\"\uffff\":2,\"\U0001f600\":1}",
		signature: "sha256=3b3885769c26eac4d862579cb4782135f5021d1ba5fa6c801d50f9228c164fb9",
	},
}

func TestGoldenVectorsMatchTheReferenceImplementation(t *testing.T) {
	for _, tc := range goldenVectors {
		t.Run(tc.name, func(t *testing.T) {
			canonical, err := qbf.CanonicalJSON([]byte(tc.payload))
			if err != nil {
				t.Fatalf("CanonicalJSON: %v", err)
			}
			if string(canonical) != tc.canonical {
				t.Errorf("canonical form differs:\n got %s\nwant %s", canonical, tc.canonical)
			}

			sig, err := qbf.ComputeWebhookSignature(tc.secret, tc.timestamp, []byte(tc.payload))
			if err != nil {
				t.Fatalf("ComputeWebhookSignature: %v", err)
			}
			if sig != tc.signature {
				t.Errorf("signature = %s, want %s", sig, tc.signature)
			}

			// The vectors carry a fixed timestamp, so verify with the replay check off.
			opts := &qbf.VerifyOptions{SkipTimestampCheck: true}
			if err := qbf.VerifyWebhookSignature(tc.secret, tc.timestamp, tc.signature, []byte(tc.payload), opts); err != nil {
				t.Errorf("golden signature must verify, got: %v", err)
			}
		})
	}
}

// ── Replay window edge cases ─────────────────────────────────────────────────

func TestReplayWindowIsOverflowSafe(t *testing.T) {
	// A difference of centuries overflows a time.Duration (nanoseconds) and would wrap
	// around into the accepted window. The timestamps are signed, so these are genuine
	// signatures: only the window check stands between them and acceptance.
	payload := []byte(`{"a":1}`)
	now := time.Now().Unix()
	for _, offset := range []int64{10_000_000_000, 9_300_000_000, -10_000_000_000, 1 << 62} {
		ts := strconv.FormatInt(now+offset, 10)
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, nil); err == nil {
			t.Errorf("a timestamp %d s away must be rejected", offset)
		}
	}
}

func TestReplayWindowBoundaries(t *testing.T) {
	payload := []byte(`{"a":1}`)
	fixed := time.Unix(1_758_540_000, 0)
	opts := &qbf.VerifyOptions{Now: func() time.Time { return fixed }}

	for _, tc := range []struct {
		offset int64
		ok     bool
	}{{-300, true}, {300, true}, {-301, false}, {301, false}, {0, true}} {
		ts := strconv.FormatInt(fixed.Unix()+tc.offset, 10)
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, opts)
		if (err == nil) != tc.ok {
			t.Errorf("offset %d: err=%v, want ok=%v", tc.offset, err, tc.ok)
		}
	}

	t.Run("a non-positive MaxTimestampAge means the 5-minute default", func(t *testing.T) {
		for _, age := range []time.Duration{0, -time.Minute} {
			o := &qbf.VerifyOptions{Now: opts.Now, MaxTimestampAge: age}
			ts := strconv.FormatInt(fixed.Unix()-299, 10)
			sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
			if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, o); err != nil {
				t.Errorf("MaxTimestampAge %v: a 299 s old webhook must verify, got %v", age, err)
			}
			ts = strconv.FormatInt(fixed.Unix()-301, 10)
			sig, _ = qbf.ComputeWebhookSignature(testSecret, ts, payload)
			if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, o); err == nil {
				t.Errorf("MaxTimestampAge %v: a 301 s old webhook must be rejected", age)
			}
		}
	})
}

func TestTimestampParsingMatchesGoParseInt(t *testing.T) {
	// The timestamp is parsed like the server's strconv.ParseInt: an optional sign and ASCII
	// digits, nothing else.
	payload := []byte(`{"a":1}`)
	fixed := time.Unix(1_758_540_000, 0)
	opts := &qbf.VerifyOptions{Now: func() time.Time { return fixed }}

	for _, ts := range []string{" 1758540000", "1758540000 ", "1758540000.0", "0x68d1b0e0", "1_758_540_000", "١٧٥٨٥٤٠٠٠٠"} {
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, opts); err == nil {
			t.Errorf("timestamp %q must be rejected", ts)
		}
	}
	for _, ts := range []string{"+1758540000", "1758540000"} {
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)
		if err := qbf.VerifyWebhookSignature(testSecret, ts, sig, payload, opts); err != nil {
			t.Errorf("timestamp %q must be accepted, got %v", ts, err)
		}
	}
}

// ── Request helper bounds ────────────────────────────────────────────────────

func TestVerifyWebhookRequestBoundsTheBody(t *testing.T) {
	ts := nowTS()

	t.Run("a body over the limit is refused", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), qbf.MaxWebhookBodySize+1)
		r := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(big))
		r.Header.Set("X-Webhook-Signature-256", "sha256=00")
		r.Header.Set("X-Webhook-Timestamp", ts)

		_, err := qbf.VerifyWebhookRequest(r, testSecret, nil)
		var validationErr *qberrors.ValidationError
		if !errors.As(err, &validationErr) || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("expected a size ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("a body exactly at the limit is read", func(t *testing.T) {
		// A JSON string padded to exactly the limit.
		body := append([]byte(`"`), bytes.Repeat([]byte("a"), qbf.MaxWebhookBodySize-2)...)
		body = append(body, '"')
		sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, body)
		r := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
		r.Header.Set("X-Webhook-Signature-256", sig)
		r.Header.Set("X-Webhook-Timestamp", ts)

		got, err := qbf.VerifyWebhookRequest(r, testSecret, nil)
		if err != nil || len(got) != qbf.MaxWebhookBodySize {
			t.Fatalf("expected the %d-byte body to verify, got %d bytes, err %v", qbf.MaxWebhookBodySize, len(got), err)
		}
	})
}

func TestCanonicalJSONOfAStructDiffersFromTheRawBody(t *testing.T) {
	// Documented pitfall: an SDK struct does not model every field, so its canonical form is
	// not the one QBitFlow signed. Raw bytes and a generic map both round-trip.
	raw := []byte(`{"uuid":"pay@1","txType":"payment","status":{"status":"completed"},"session":{"uuid":"pay@1","extra":"x"}}`)

	fromRaw, _ := qbf.CanonicalJSON(raw)

	var generic map[string]any
	_ = json.Unmarshal(raw, &generic)
	fromMap, _ := qbf.CanonicalJSON(generic)
	if !bytes.Equal(fromRaw, fromMap) {
		t.Errorf("a generic map must canonicalise like the raw body:\n%s\n%s", fromRaw, fromMap)
	}

	var typed qbmodels.SessionWebhookResponse
	_ = json.Unmarshal(raw, &typed)
	fromStruct, _ := qbf.CanonicalJSON(typed)
	if bytes.Equal(fromRaw, fromStruct) {
		t.Error("expected the struct's canonical form to differ (it drops unmodelled fields)")
	}
}

func TestSwappedArgumentsFailSafelyWithAHint(t *testing.T) {
	ts := nowTS()
	payload := []byte(`{"a":1}`)
	sig, _ := qbf.ComputeWebhookSignature(testSecret, ts, payload)

	// signature and timestamp swapped
	err := qbf.VerifyWebhookSignature(testSecret, sig, ts, payload, nil)
	if err == nil || !strings.Contains(err.Error(), "argument order") {
		t.Errorf("expected an argument-order hint, got %v", err)
	}
}
