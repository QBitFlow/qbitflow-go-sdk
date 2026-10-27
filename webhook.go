package qbitflow

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Webhook delivery headers.
const (
	// SignatureHeader carries "t=<unix seconds>,v1=<hex>" (two v1 during a secret rotation).
	SignatureHeader = "QBitFlow-Signature"
	// EventIDHeader carries the event's id (evt_…): deduplicate on it.
	EventIDHeader = "QBitFlow-Event-Id"
	// EventTypeHeader carries the event's type.
	EventTypeHeader = "QBitFlow-Event-Type"
)

const (
	// DefaultWebhookTolerance is how far a signature's timestamp may be from now.
	DefaultWebhookTolerance = 300 * time.Second
	// maxWebhookBody bounds the body VerifyWebhookRequest reads (the API's own limit).
	maxWebhookBody = 1 << 20
)

// verifyConfig is what the VerifyOptions set.
type verifyConfig struct {
	tolerance time.Duration
	now       func() time.Time
}

// VerifyOption configures VerifyWebhook, VerifyWebhookRequest and ConstructEvent.
type VerifyOption func(*verifyConfig)

// WithTolerance sets how far the signature's timestamp may be from now, in either direction
// (default 300 s; 0 or less keeps the default).
func WithTolerance(tolerance time.Duration) VerifyOption {
	return func(c *verifyConfig) {
		if tolerance > 0 {
			c.tolerance = tolerance
		}
	}
}

// WithClock sets the clock the timestamp is checked against (tests, replays).
func WithClock(now func() time.Time) VerifyOption {
	return func(c *verifyConfig) {
		if now != nil {
			c.now = now
		}
	}
}

// VerifyWebhook checks a webhook delivery's signature: signatureHeader is the
// QBitFlow-Signature header ("t=<unix seconds>,v1=<hex>[,v1=<hex>]"), secret the endpoint's
// whsec_… secret, rawBody the body exactly as received (never re-serialized).
//
// It accepts the delivery when t is within the tolerance (300 s) of now and any v1 equals
// hex(HMAC-SHA256(secret, t + "." + rawBody)), compared in constant time; during a secret
// rotation either secret's signature matches. It returns nil, or a *WebhookSignatureError
// whose Reason says why (an empty secret is a *ValidationError).
func VerifyWebhook(rawBody []byte, signatureHeader, secret string, opts ...VerifyOption) error {
	cfg := verifyConfig{tolerance: DefaultWebhookTolerance, now: time.Now}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if secret == "" {
		return newFieldError("secret", "is required (the endpoint's whsec_… secret)")
	}
	if strings.TrimSpace(signatureHeader) == "" {
		return newSignatureError(WebhookSignatureMissingHeader, "missing QBitFlow-Signature header")
	}

	timestamp, rawT, signatures, ok := parseSignatureHeader(signatureHeader)
	if !ok {
		return newSignatureError(WebhookSignatureMalformedHeader, "malformed QBitFlow-Signature header: it needs one t=<unix seconds> and at least one v1=<signature>")
	}

	age := cfg.now().Sub(time.Unix(timestamp, 0))
	if age > cfg.tolerance || age < -cfg.tolerance {
		return newSignatureError(WebhookSignatureTimestampOutsideTolerance, "webhook timestamp is outside the tolerance ("+cfg.tolerance.String()+")")
	}

	// The signed text is t exactly as the header carries it (not its re-formatted integer).
	expected := []byte(webhookSignature(secret, rawT, rawBody))

	matched := false
	for _, sig := range signatures {
		// Every v1 is compared, in constant time: never stop at the first.
		if hmac.Equal(expected, []byte(sig)) {
			matched = true
		}
	}
	if !matched {
		return newSignatureError(WebhookSignatureNoMatchingSignature, "no webhook signature matches")
	}
	return nil
}

// webhookSignature is hex(HMAC-SHA256(secret, t + "." + rawBody)), a v1 signature.
func webhookSignature(secret, t string, rawBody []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(t))
	mac.Write([]byte{'.'})
	mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

// SignWebhook returns the QBitFlow-Signature header QBitFlow would send with rawBody, signed
// with secret at t ("t=<unix seconds>,v1=<hex>"; the zero time means now). Use it to test a
// webhook handler: VerifyWebhook(body, SignWebhook(body, secret, time.Now()), secret) passes.
func SignWebhook(rawBody []byte, secret string, t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	ts := strconv.FormatInt(t.Unix(), 10)
	return "t=" + ts + ",v1=" + webhookSignature(secret, ts, rawBody)
}

// parseSignatureHeader splits "t=…,v1=…[,v1=…]": parts on ",", each on its first "=", spaces
// trimmed, other keys ignored. It needs exactly one t made of ASCII digits that fits an int64,
// and a v1. rawT is t's text as received: the signature covers that text.
func parseSignatureHeader(header string) (timestamp int64, rawT string, signatures []string, ok bool) {
	seenT := false
	for part := range strings.SplitSeq(header, ",") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "t":
			if seenT || !isASCIIDigits(value) {
				return 0, "", nil, false
			}
			t, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, "", nil, false
			}
			timestamp, rawT, seenT = t, value, true
		case "v1":
			signatures = append(signatures, value)
		}
	}
	return timestamp, rawT, signatures, seenT && len(signatures) > 0
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func newSignatureError(reason WebhookSignatureReason, message string) *WebhookSignatureError {
	return &WebhookSignatureError{APIError: &APIError{Message: message, Details: map[string]any{}}, Reason: reason}
}

// VerifyWebhookRequest reads a webhook delivery's body (at most 1 MiB) and verifies it with
// its QBitFlow-Signature header. It returns the raw body for ParseEvent, and puts it back on
// r.Body so later handlers can read it again.
func VerifyWebhookRequest(r *http.Request, secret string, opts ...VerifyOption) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, newFieldError("body", "is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	_ = r.Body.Close()
	if err != nil {
		v := newValidationError("cannot read the webhook body")
		v.Err = err
		return nil, v
	}
	if len(body) > maxWebhookBody {
		return nil, newFieldError("body", "must be at most 1 MiB")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := VerifyWebhook(body, r.Header.Get(SignatureHeader), secret, opts...); err != nil {
		return nil, err
	}
	return body, nil
}

// ParseEvent parses a webhook body (or an event of the log) without verifying it: use it on a
// body already verified (VerifyWebhook, Webhooks.VerifyRemote). A body that is not a JSON
// object, or whose version is not v2, is a *ValidationError; an unknown type is not an error
// (its Data stays raw).
func ParseEvent(rawBody []byte) (*Event, error) {
	trimmed := bytes.TrimSpace(rawBody)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, newFieldError("body", "must be a JSON object (a webhook event)")
	}
	// The envelope's version is always v2 (v1 payloads are retired): checked, not exposed.
	var e struct {
		Event
		Version string `json:"version"`
	}
	if err := json.Unmarshal(trimmed, &e); err != nil {
		v := newFieldError("body", "is not a valid webhook event")
		v.Err = err
		return nil, v
	}
	if e.Version != "v2" {
		return nil, newFieldError("version", "must be v2 (unsupported payload version)")
	}
	return &e.Event, nil
}

// ConstructEvent verifies a webhook delivery (VerifyWebhook) and parses it (ParseEvent).
func ConstructEvent(rawBody []byte, signatureHeader, secret string, opts ...VerifyOption) (*Event, error) {
	if err := VerifyWebhook(rawBody, signatureHeader, secret, opts...); err != nil {
		return nil, err
	}
	return ParseEvent(rawBody)
}

// Verify checks a webhook delivery's signature locally (see VerifyWebhook).
func (s *WebhooksService) Verify(rawBody []byte, signatureHeader, secret string, opts ...VerifyOption) error {
	return VerifyWebhook(rawBody, signatureHeader, secret, opts...)
}

// ConstructEvent verifies and parses a webhook delivery (see ConstructEvent).
func (s *WebhooksService) ConstructEvent(rawBody []byte, signatureHeader, secret string, opts ...VerifyOption) (*Event, error) {
	return ConstructEvent(rawBody, signatureHeader, secret, opts...)
}

// ParseEvent parses a webhook body without verifying it (see ParseEvent).
func (s *WebhooksService) ParseEvent(rawBody []byte) (*Event, error) {
	return ParseEvent(rawBody)
}
