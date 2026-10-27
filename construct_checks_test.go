package qbitflow

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewClientKeyCharset (behaviour §2): after trimming, the key starts with sk_ and every
// character is printable ASCII without spaces (0x21–0x7E); else a *ValidationError on apiKey
// whose message never quotes the key. Beyond that the key is opaque.
func TestNewClientKeyCharset(t *testing.T) {
	const secret = "SECRETPART"
	for name, key := range map[string]string{
		"inner space": "sk_live_" + secret + " x",
		"tab":         "sk_live_" + secret + "\tx",
		"CR LF":       "sk_live_" + secret + "\r\nX-Evil: 1",
		"NUL":         "sk_live_" + secret + "\x00",
		"DEL":         "sk_live_" + secret + "\x7f",
		"non-ASCII":   "sk_live_" + secret + "é",
		"NBSP inside": "sk_live_" + secret + " x",
	} {
		_, err := NewClient(key)
		var ve *ValidationError
		if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "apiKey" {
			t.Errorf("%s: want a *ValidationError on apiKey, got %v", name, err)
			continue
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("%s: the message quotes the key: %v", name, err)
		}
	}
	for _, key := range []string{
		"sk_test_019eca82-5680-7b00-8000-0000000000b1_s3cr3t", // API v2's shape
		"sk_019eca82-5680-7b00-8000-0000000000b1_live_s3cr3t", // an earlier shape: the API's 401
		"sk_123_live_abc", "sk_!~#%&*+=?^`{|}", "\t sk_padded \r\n",
	} {
		if _, err := NewClient(key); err != nil {
			t.Errorf("NewClient(%q): %v", key, err)
		}
	}
}

// TestWithBaseURLRefusesCredentialsQueryFragment (behaviour §1): an http(s) URL with a host and
// no user:pass@, query or fragment; the message never quotes the URL.
func TestWithBaseURLRefusesCredentialsQueryFragment(t *testing.T) {
	for name, base := range map[string]string{
		"credentials":  "https://user:SECRETPART@api.example.com/v2",
		"user only":    "https://SECRETPART@api.example.com/v2",
		"query":        "https://api.example.com/v2?token=SECRETPART",
		"empty query":  "https://api.example.com/v2?",
		"fragment":     "https://api.example.com/v2#SECRETPART",
		"empty frag":   "https://api.example.com/v2#",
		"no host":      "https:///v2",
		"scheme only":  "https://",
		"other scheme": "ws://api.example.com/SECRETPART",
	} {
		_, err := NewClient("sk_x", WithBaseURL(base))
		var ve *ValidationError
		if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "baseUrl" {
			t.Errorf("%s: want a *ValidationError on baseUrl, got %v", name, err)
			continue
		}
		if strings.Contains(err.Error(), "SECRETPART") || strings.Contains(err.Error(), "example.com") {
			t.Errorf("%s: the message quotes the URL: %v", name, err)
		}
	}
	for _, base := range []string{"https://api.example.com/v2", "HTTP://localhost:8080/v2/", "https://api.example.com:8443"} {
		if _, err := NewClient("sk_x", WithBaseURL(base)); err != nil {
			t.Errorf("WithBaseURL(%q): %v", base, err)
		}
	}
}

// TestTimeoutCoversTheBody (behaviour §1): the per-attempt timeout bounds the attempt through
// the last byte of the body; a server trickling it is a timed-out *NetworkError, retried like one.
func TestTimeoutCoversTheBody(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"credential":`))
		w.(http.Flusher).Flush()
		select { // never finishes the body within the client's timeout
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	c, sleeps := newTestClient(t, ts, WithTimeout(150*time.Millisecond), WithMaxRetries(1))
	start := time.Now()
	_, err := c.Me(context.Background())
	elapsed := time.Since(start)
	var ne *NetworkError
	if !errors.As(err, &ne) || !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("want a timed-out *NetworkError, got %T %v", err, err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("the call took %v: the body was not bounded", elapsed)
	}
	if len(ts.recorded()) != 2 || len(*sleeps) != 1 {
		t.Errorf("a timed-out attempt is retried: %d attempts, sleeps %v", len(ts.recorded()), *sleeps)
	}
}
