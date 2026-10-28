package qbitflow

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

const testAPIKey = "sk_test_key_123"

// recorded is one request the test server received.
type recorded struct {
	method string
	path   string // escaped path, as sent
	query  string
	header http.Header
	body   string
}

// testServer is an httptest server answering with a scripted handler and recording requests.
type testServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []recorded
}

func (s *testServer) recorded() []recorded {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recorded(nil), s.requests...)
}

func newTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, n int)) *testServer {
	t.Helper()
	ts := &testServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ts.mu.Lock()
		n := len(ts.requests)
		ts.requests = append(ts.requests, recorded{
			method: r.Method,
			path:   r.URL.EscapedPath(),
			query:  r.URL.RawQuery,
			header: r.Header.Clone(),
			body:   string(body),
		})
		ts.mu.Unlock()
		handler(w, r, n)
	}))
	t.Cleanup(ts.Close)
	return ts
}

// newTestClient builds a client on ts with a recording sleep (no real waits).
func newTestClient(t *testing.T, ts *testServer, opts ...ClientOption) (*Client, *[]time.Duration) {
	t.Helper()
	c, err := NewClient(testAPIKey, append([]ClientOption{WithBaseURL(ts.URL)}, opts...)...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var mu sync.Mutex
	sleeps := &[]time.Duration{}
	c.t.sleep = func(_ context.Context, d time.Duration) error {
		mu.Lock()
		defer mu.Unlock()
		*sleeps = append(*sleeps, d)
		return nil
	}
	return c, sleeps
}

// jsonReply writes a JSON answer.
func jsonReply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// staticReply answers every request with the same status and body.
func staticReply(status int, body string) func(http.ResponseWriter, *http.Request, int) {
	return func(w http.ResponseWriter, _ *http.Request, _ int) { jsonReply(w, status, body) }
}

func get(path string) endpoint { return endpoint{method: http.MethodGet, path: path} }

func ptr[T any](v T) *T { return &v }
