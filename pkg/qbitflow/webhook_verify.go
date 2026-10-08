package qbf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

// DefaultMaxTimestampAge is how far a webhook's timestamp may be from the current clock
// before the webhook is rejected as a replay.
//
// It must match the server's MaxTimestampAge. Five minutes is the SDK default; override it
// with VerifyOptions.MaxTimestampAge if your deployment uses a different window.
const DefaultMaxTimestampAge = 5 * time.Minute

// signaturePrefix is the algorithm marker QBitFlow prefixes to the hex digest.
const signaturePrefix = "sha256="

// VerifyOptions tunes local webhook verification.
type VerifyOptions struct {
	// MaxTimestampAge is the replay window. Zero means DefaultMaxTimestampAge.
	MaxTimestampAge time.Duration

	// Now overrides the clock. Only useful in tests; nil means time.Now.
	Now func() time.Time

	// SkipTimestampCheck disables the replay check entirely.
	//
	// Leave this false in production: without it a captured webhook can be replayed
	// forever. It exists for replaying stored webhooks in a test harness.
	SkipTimestampCheck bool
}

func (o *VerifyOptions) maxAge() time.Duration {
	if o == nil || o.MaxTimestampAge <= 0 {
		return DefaultMaxTimestampAge
	}
	return o.MaxTimestampAge
}

func (o *VerifyOptions) now() time.Time {
	if o == nil || o.Now == nil {
		return time.Now()
	}
	return o.Now()
}

// CanonicalJSON renders a webhook payload the way QBitFlow signs it.
//
// The signature is computed over a *canonical* rendering rather than the bytes as they
// arrived, because JSON object key order is not significant and intermediaries (proxies,
// frameworks, logging layers) routinely re-serialize a body and reorder keys. Signing the
// raw bytes would make verification fail for a payload that is in fact untouched.
//
// Canonical here means: object keys sorted lexicographically at every level, no
// insignificant whitespace. This function reaches that form by decoding the payload into
// Go's generic representation and re-encoding it — encoding/json sorts map keys, so the
// result is stable regardless of the input's original ordering.
//
// payload may be raw JSON ([]byte, json.RawMessage or string) or an already-decoded value.
//
// NOTE ON ESCAPING: encoding/json escapes <, > and & as <, > and & by
// default, and the server signs with the same default. The other QBitFlow SDKs reproduce
// this escaping deliberately so that all four compute an identical signature.
func CanonicalJSON(payload any) ([]byte, error) {
	var decoded any

	switch v := payload.(type) {
	case nil:
		return nil, qberrors.NewValidationError("webhook payload is required")
	case []byte:
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload is not valid JSON: %v", err))
		}
	case json.RawMessage:
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload is not valid JSON: %v", err))
		}
	case string:
		if err := json.Unmarshal([]byte(v), &decoded); err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload is not valid JSON: %v", err))
		}
	default:
		// Round-trip anything else so struct field order cannot leak into the signature:
		// only the sorted-map form is ever signed.
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload cannot be encoded: %v", err))
		}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, qberrors.NewValidationError(fmt.Sprintf("webhook payload cannot be normalised: %v", err))
		}
	}

	return json.Marshal(decoded)
}

// ComputeWebhookSignature returns the signature QBitFlow would send for this payload.
//
// The signed message is `<timestamp>.<canonical-json>` and the result is the hex-encoded
// HMAC-SHA256 of that message under the webhook secret, prefixed with "sha256=" — exactly
// the value delivered in the X-Webhook-Signature-256 header.
//
// This is exported mainly so you can generate valid webhooks in your own tests; to check
// an incoming webhook use VerifyWebhookSignature, which also enforces the replay window
// and compares in constant time.
func ComputeWebhookSignature(secret, timestamp string, payload any) (string, error) {
	if secret == "" {
		return "", qberrors.NewValidationError("webhook secret is required")
	}
	if timestamp == "" {
		return "", qberrors.NewValidationError("webhook timestamp is required")
	}

	canonical, err := CanonicalJSON(payload)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(canonical)

	return signaturePrefix + hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyWebhookSignature checks a webhook locally, without calling the QBitFlow API.
//
// Use it when you would rather not make a network round-trip per webhook, or want
// verification to keep working when the API is unreachable. It performs the same three
// checks the server does:
//
//  1. The timestamp is within MaxTimestampAge of now, which is what stops a captured
//     webhook from being replayed later.
//  2. The HMAC-SHA256 of `<timestamp>.<canonical-json>` under your secret matches.
//  3. The comparison is constant-time, so a timing side channel cannot be used to guess
//     the signature byte by byte.
//
// The secret comes from the QBitFlow dashboard. Treat it like a password: keep it in your
// environment or secret manager, never in source control, and never send it anywhere.
//
// A nil return means the webhook is authentic. Any error means do not trust the payload.
//
//	err := qbf.VerifyWebhookSignature(secret, timestamp, signature, body, nil)
//	if err != nil {
//		http.Error(w, "invalid webhook", http.StatusBadRequest)
//		return
//	}
func VerifyWebhookSignature(secret, timestamp, signature string, payload any, opts *VerifyOptions) error {
	if signature == "" {
		return qberrors.NewValidationError("webhook signature is required")
	}

	if !opts.skipTimestamp() {
		if err := verifyTimestamp(timestamp, opts); err != nil {
			return err
		}
	}

	expected, err := ComputeWebhookSignature(secret, timestamp, payload)
	if err != nil {
		return err
	}

	// Constant-time: comparing with == would leak how many leading bytes matched.
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return qberrors.NewValidationError("webhook signature mismatch")
	}

	return nil
}

func (o *VerifyOptions) skipTimestamp() bool {
	return o != nil && o.SkipTimestampCheck
}

// verifyTimestamp enforces the replay window, comparing in absolute terms so a webhook
// from a clock slightly ahead of ours is treated the same as one slightly behind.
func verifyTimestamp(timestamp string, opts *VerifyOptions) error {
	if timestamp == "" {
		return qberrors.NewValidationError("webhook timestamp is required")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return qberrors.NewValidationError("webhook timestamp is not a unix-seconds integer")
	}

	age := time.Duration(math.Abs(float64(opts.now().Unix()-ts))) * time.Second
	if age > opts.maxAge() {
		return qberrors.NewValidationError(
			fmt.Sprintf("webhook timestamp expired: age %s exceeds the maximum of %s", age, opts.maxAge()))
	}

	return nil
}

// WebhookHeaders holds the QBitFlow headers carried by an incoming webhook.
type WebhookHeaders struct {
	// Signature is the X-Webhook-Signature-256 value, formatted "sha256=<hex>".
	Signature string
	// Timestamp is the X-Webhook-Timestamp value, in unix seconds.
	Timestamp string
	// WebhookID is the X-Webhook-Id value: the transaction id, e.g. "pay@<uuid>".
	WebhookID string
}

// IsTest reports whether this is the dashboard's connectivity test webhook rather than a
// real transaction. Those carry a fixed id and no transaction behind them.
func (h WebhookHeaders) IsTest() bool {
	return h.WebhookID == TEST_WEBHOOK_ID
}

// ExtractWebhookHeaders pulls the QBitFlow headers out of an http.Header.
//
// http.Header canonicalises keys, so this works regardless of the casing the sender used.
func ExtractWebhookHeaders(h http.Header) WebhookHeaders {
	return WebhookHeaders{
		Signature: h.Get(HeaderSignature),
		Timestamp: h.Get(HeaderTimestamp),
		WebhookID: h.Get(HeaderWebhookID),
	}
}

// VerifyWebhookRequest reads and verifies an incoming *http.Request in one step, and
// returns the raw body so you can decode it once verification has passed.
//
// It consumes r.Body. The body is returned rather than re-decoded for you, so you stay in
// control of how the payload is parsed.
//
//	body, err := qbf.VerifyWebhookRequest(r, secret, nil)
//	if err != nil {
//		http.Error(w, "invalid webhook", http.StatusBadRequest)
//		return
//	}
//	var event qbmodels.SessionWebhookResponse
//	_ = json.Unmarshal(body, &event)
//
// Never act on the body before this returns nil: an unverified webhook may be forged.
func VerifyWebhookRequest(r *http.Request, secret string, opts *VerifyOptions) ([]byte, error) {
	if r == nil {
		return nil, qberrors.NewValidationError("request is required")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, qberrors.NewValidationError(fmt.Sprintf("could not read webhook body: %v", err))
	}

	headers := ExtractWebhookHeaders(r.Header)
	if headers.Signature == "" || headers.Timestamp == "" {
		return nil, qberrors.NewValidationError(
			fmt.Sprintf("missing webhook headers: %s and %s are both required", HeaderSignature, HeaderTimestamp))
	}

	if err := VerifyWebhookSignature(secret, headers.Timestamp, headers.Signature, body, opts); err != nil {
		return nil, err
	}

	return body, nil
}

// VerifyLocal verifies a webhook without contacting the API, as a method on the service so
// it sits beside Verify (which asks the API to check the signature for you).
//
// Local verification needs your webhook secret but no network call; Verify needs no secret
// but one round-trip per webhook. Prefer this one in a hot path.
func (s *WebhookService) VerifyLocal(secret, timestamp, signature string, payload any, opts *VerifyOptions) error {
	return VerifyWebhookSignature(secret, timestamp, signature, payload, opts)
}
