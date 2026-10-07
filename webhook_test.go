package qbitflow

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The shared vector (behaviour §7, docs webhooks.md).
const (
	vectorBody = `{"createdAt":"2026-10-01T12:00:00Z","data":{},"id":"evt_3f1c2d4e-5a6b-5c7d-8e9f-0a1b2c3d4e5f","test":false,"type":"webhook.test","version":"v2"}`
	vectorT    = int64(1790856000)
	vectorNew  = "4a158046f55556e922bdec376a917c3ac338ba575b495f825427541a60bd2f4d" // whsec_new_secret
	vectorOld  = "1cc78a6297ad96dd282d9d0399def50ed42a4e651885ac79f8528194227bc4f5" // whsec_old_secret
)

var vectorHeader = "t=1790856000,v1=" + vectorNew + ",v1=" + vectorOld

func at(seconds int64) VerifyOption {
	return WithClock(func() time.Time { return time.Unix(vectorT+seconds, 0) })
}

func sign(secret string, t int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(t, 10) + "."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func reasonOf(err error) WebhookSignatureReason {
	var se *WebhookSignatureError
	if errors.As(err, &se) {
		return se.Reason
	}
	return ""
}

func TestWebhookVector(t *testing.T) {
	body := []byte(vectorBody)
	if got := sign("whsec_new_secret", vectorT, body); got != vectorNew {
		t.Fatalf("HMAC of the vector = %s, want %s", got, vectorNew)
	}
	if got := sign("whsec_old_secret", vectorT, body); got != vectorOld {
		t.Fatalf("HMAC of the vector (old) = %s", got)
	}

	// At t+60 both secrets of the rotation verify; any other secret does not.
	for _, secret := range []string{"whsec_new_secret", "whsec_old_secret"} {
		if err := VerifyWebhook(body, vectorHeader, secret, at(60)); err != nil {
			t.Errorf("%s at t+60: %v", secret, err)
		}
	}
	for _, secret := range []string{"whsec_other", "new_secret", "whsec_new_secret "} {
		if r := reasonOf(VerifyWebhook(body, vectorHeader, secret, at(60))); r != WebhookSignatureNoMatchingSignature {
			t.Errorf("%q at t+60: reason %q", secret, r)
		}
	}
	// At t+360 (6 min) every secret is refused: too old.
	for _, secret := range []string{"whsec_new_secret", "whsec_old_secret", "whsec_other"} {
		if r := reasonOf(VerifyWebhook(body, vectorHeader, secret, at(360))); r != WebhookSignatureTimestampOutsideTolerance {
			t.Errorf("%s at t+360: reason %q", secret, r)
		}
	}
}

func TestWebhookTolerance(t *testing.T) {
	body := []byte(vectorBody)
	header := "t=1790856000,v1=" + vectorNew
	cases := []struct {
		offset int64
		opts   []VerifyOption
		ok     bool
	}{
		{0, nil, true},
		{300, nil, true},
		{301, nil, false},
		{-300, nil, true}, // t in the future: either direction
		{-301, nil, false},
		{60, []VerifyOption{WithTolerance(10 * time.Second)}, false},
		{10, []VerifyOption{WithTolerance(10 * time.Second)}, true},
		{299, []VerifyOption{WithTolerance(0)}, true}, // 0 or less keeps 300 s
		{301, []VerifyOption{WithTolerance(-time.Second)}, false},
		{3000, []VerifyOption{WithTolerance(time.Hour)}, true},
	}
	for _, tc := range cases {
		err := VerifyWebhook(body, header, "whsec_new_secret", append([]VerifyOption{at(tc.offset)}, tc.opts...)...)
		if (err == nil) != tc.ok {
			t.Errorf("offset %+d %v: err %v", tc.offset, tc.opts, err)
		}
		if err != nil && reasonOf(err) != WebhookSignatureTimestampOutsideTolerance {
			t.Errorf("offset %+d: reason %q", tc.offset, reasonOf(err))
		}
	}
	// A timestamp in the far future (time arithmetic saturates) is refused too.
	far := "t=9223372036854775807,v1=" + vectorNew
	if r := reasonOf(VerifyWebhook(body, far, "whsec_new_secret", at(0))); r != WebhookSignatureTimestampOutsideTolerance {
		t.Errorf("far future: reason %q", r)
	}
}

func TestWebhookHeaderParsing(t *testing.T) {
	body := []byte(vectorBody)
	const ts = "1790856000"
	cases := []struct {
		name   string
		header string
		reason WebhookSignatureReason // "" = valid
	}{
		{"rotation", vectorHeader, ""},
		{"new only", "t=" + ts + ",v1=" + vectorNew, ""},
		{"v1 first", "v1=" + vectorNew + ",t=" + ts, ""},
		{"spaces around parts", "  t = " + ts + " ,  v1 = " + vectorNew + "  ", ""},
		{"bad v1 then good", "t=" + ts + ",v1=deadbeef,v1=" + vectorNew, ""},
		{"good v1 then bad", "t=" + ts + ",v1=" + vectorNew + ",v1=deadbeef", ""},
		{"unknown scheme ignored", "t=" + ts + ",v0=" + vectorNew + ",v1=" + vectorNew + ",v2=x", ""},
		{"parts without =", "t=" + ts + ",junk,v1=" + vectorNew + ",", ""},
		{"empty", "", WebhookSignatureMissingHeader},
		{"blank", "   ", WebhookSignatureMissingHeader},
		{"garbage", "garbage", WebhookSignatureMalformedHeader},
		{"no t", "v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"no v1", "t=" + ts, WebhookSignatureMalformedHeader},
		{"only unknown scheme", "t=" + ts + ",v0=" + vectorNew, WebhookSignatureMalformedHeader},
		{"sha256 legacy scheme", "sha256=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t not a number", "t=abc,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t negative", "t=-1790856000,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t with sign", "t=+1790856000,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t decimal", "t=1790856000.0,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t empty", "t=,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t non-ASCII digits", "t=１７９０８５６０００,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"t overflow", "t=99999999999999999999,v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"duplicate t", "t=" + ts + ",t=" + ts + ",v1=" + vectorNew, WebhookSignatureMalformedHeader},
		{"uppercase hex", "t=" + ts + ",v1=" + strings.ToUpper(vectorNew), WebhookSignatureNoMatchingSignature},
		{"truncated hex", "t=" + ts + ",v1=" + vectorNew[:63], WebhookSignatureNoMatchingSignature},
		{"value with =", "t=" + ts + ",v1=" + vectorNew + "=", WebhookSignatureNoMatchingSignature},
		{"empty v1", "t=" + ts + ",v1=", WebhookSignatureNoMatchingSignature},
		{"other t", "t=1790856001,v1=" + vectorNew, WebhookSignatureNoMatchingSignature},
	}
	for _, tc := range cases {
		err := VerifyWebhook(body, tc.header, "whsec_new_secret", at(30))
		if got := reasonOf(err); got != tc.reason {
			t.Errorf("%s: reason %q, want %q (%v)", tc.name, got, tc.reason, err)
		}
		if err != nil {
			var se *WebhookSignatureError
			var base *APIError
			if !errors.As(err, &se) || !errors.As(err, &base) || base.StatusCode != 0 {
				t.Errorf("%s: %T must be a *WebhookSignatureError with no status", tc.name, err)
			}
		}
	}
}

func TestWebhookRawBody(t *testing.T) {
	header := "t=1790856000,v1=" + vectorNew
	// Any change to the bytes breaks the signature: never re-serialize.
	for _, body := range []string{
		strings.Replace(vectorBody, `"test":false`, `"test":true`, 1),
		strings.Replace(vectorBody, `,"id"`, `, "id"`, 1),
		vectorBody + "\n",
	} {
		if r := reasonOf(VerifyWebhook([]byte(body), header, "whsec_new_secret", at(0))); r != WebhookSignatureNoMatchingSignature {
			t.Errorf("tampered body accepted: %q", body)
		}
	}
	// Non-ASCII and non-UTF-8 bodies are signed as bytes.
	for _, body := range [][]byte{[]byte(`{"name":"Zoë – 李小龙 ☃"}`), {0xff, 0xfe, '{', '}'}, {}} {
		h := "t=1790856000,v1=" + sign("whsec_new_secret", vectorT, body)
		if err := VerifyWebhook(body, h, "whsec_new_secret", at(0)); err != nil {
			t.Errorf("body %q: %v", body, err)
		}
	}
	// The key is the whole secret, whsec_ prefix included.
	h := "t=1790856000,v1=" + sign("new_secret", vectorT, []byte(vectorBody))
	if r := reasonOf(VerifyWebhook([]byte(vectorBody), h, "whsec_new_secret", at(0))); r != WebhookSignatureNoMatchingSignature {
		t.Error("the whsec_ prefix must be part of the key")
	}
	// An empty secret is a configuration error.
	var ve *ValidationError
	if err := VerifyWebhook([]byte(vectorBody), header, "", at(0)); !errors.As(err, &ve) {
		t.Errorf("empty secret: %v", err)
	}
}

func TestVerifyWebhookRequest(t *testing.T) {
	newReq := func(body []byte, header string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(body))
		if header != "" {
			r.Header.Set(SignatureHeader, header)
		}
		return r
	}

	r := newReq([]byte(vectorBody), vectorHeader)
	body, err := VerifyWebhookRequest(r, "whsec_old_secret", at(60))
	if err != nil || string(body) != vectorBody {
		t.Fatalf("got %q, %v", body, err)
	}
	again, _ := io.ReadAll(r.Body)
	if string(again) != vectorBody {
		t.Error("the body must be readable again")
	}

	if r := reasonOf(func() error {
		_, err := VerifyWebhookRequest(newReq([]byte(vectorBody), ""), "whsec_new_secret", at(0))
		return err
	}()); r != WebhookSignatureMissingHeader {
		t.Errorf("missing header: %q", r)
	}

	big := bytes.Repeat([]byte("x"), maxWebhookBody+1)
	_, err = VerifyWebhookRequest(newReq(big, "t=1790856000,v1="+sign("whsec_new_secret", vectorT, big)), "whsec_new_secret", at(0))
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("over 1 MiB: %v", err)
	}
	exact := bytes.Repeat([]byte("x"), maxWebhookBody)
	if _, err := VerifyWebhookRequest(newReq(exact, "t=1790856000,v1="+sign("whsec_new_secret", vectorT, exact)), "whsec_new_secret", at(0)); err != nil {
		t.Errorf("exactly 1 MiB: %v", err)
	}
	if _, err := VerifyWebhookRequest(nil, "whsec_x"); !errors.As(err, &ve) {
		t.Errorf("nil request: %v", err)
	}
}

func TestConstructEvent(t *testing.T) {
	e, err := ConstructEvent([]byte(vectorBody), vectorHeader, "whsec_new_secret", at(60))
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "evt_3f1c2d4e-5a6b-5c7d-8e9f-0a1b2c3d4e5f" || e.Type != EventTypeWebhookTest || e.Version != "v2" || e.Test ||
		!e.CreatedAt.Equal(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)) || e.UserUUID != "" {
		t.Errorf("event %+v", e)
	}
	if _, err := ConstructEvent([]byte(vectorBody), vectorHeader, "whsec_other", at(60)); reasonOf(err) != WebhookSignatureNoMatchingSignature {
		t.Errorf("bad secret: %v", err)
	}

	// The client's Webhooks service delegates to the same functions.
	c, err := NewClient("sk_x")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Webhooks.Verify([]byte(vectorBody), vectorHeader, "whsec_old_secret", at(60)); err != nil {
		t.Error(err)
	}
	if e, err := c.Webhooks.ConstructEvent([]byte(vectorBody), vectorHeader, "whsec_old_secret", at(60)); err != nil || e.Type != EventTypeWebhookTest {
		t.Error(e, err)
	}
	if e, err := c.Webhooks.ParseEvent([]byte(vectorBody)); err != nil || e.Type != EventTypeWebhookTest {
		t.Error(e, err)
	}

	// A v1 body signed correctly is still refused: the endpoint must move to v2.
	v1 := []byte(`{"uuid":"pay@x","txType":"payment","status":{"status":"completed"}}`)
	_, err = ConstructEvent(v1, "t=1790856000,v1="+sign("whsec_new_secret", vectorT, v1), "whsec_new_secret", at(0))
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.FieldErrors[0].Field != "version" {
		t.Errorf("v1 body: %v", err)
	}
}
