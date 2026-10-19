package qbitflow

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// sequence answers the scripted (status, body) pairs in order, then repeats the last one.
func sequence(replies ...[2]any) func(http.ResponseWriter, *http.Request, int) {
	return func(w http.ResponseWriter, _ *http.Request, n int) {
		reply := replies[min(n, len(replies)-1)]
		jsonReply(w, reply[0].(int), reply[1].(string))
	}
}

func reply(status int, body string) [2]any { return [2]any{status, body} }

var (
	ok200       = reply(200, `{"ok":true}`)
	err500      = reply(500, `{"error":"boom","code":"internal"}`)
	err503      = reply(503, `{"error":"no network","code":"network_unavailable"}`)
	err504      = reply(504, `{"error":"too slow","code":"timeout"}`)
	keyInUse409 = reply(409, `{"error":"in use","code":"idempotency_key_in_use"}`)
)

var idempotentCreate = endpoint{method: http.MethodPost, path: "/products", body: map[string]any{"name": "Pro"}, idempotent: true}

func TestRetryMatrix(t *testing.T) {
	cases := []struct {
		name       string
		e          endpoint
		replies    [][2]any
		attempts   int
		sleeps     []time.Duration
		wantErr    any // pointer to the expected error type, nil for success
		maxRetries *int
	}{
		{"GET 500 exhausts retries", get("/x"), [][2]any{err500}, 4, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}, new(*ServerError), nil},
		{"GET 503 then 200", get("/x"), [][2]any{err503, ok200}, 2, []time.Duration{time.Second}, nil, nil},
		{"GET 504 then 200", get("/x"), [][2]any{err504, ok200}, 2, []time.Duration{time.Second}, nil, nil},
		{"GET 502 non-JSON then 200", get("/x"), [][2]any{reply(502, `<html>bad gateway</html>`), ok200}, 2, []time.Duration{time.Second}, nil, nil},
		{"GET 400 not retried", get("/x"), [][2]any{reply(400, `{"error":"bad","code":"bad_request"}`)}, 1, nil, new(*BadRequestError), nil},
		{"GET 401 not retried", get("/x"), [][2]any{reply(401, `{"error":"no","code":"unauthorized"}`)}, 1, nil, new(*AuthenticationError), nil},
		{"GET 403 not retried", get("/x"), [][2]any{reply(403, `{"error":"no","code":"forbidden"}`)}, 1, nil, new(*PermissionDeniedError), nil},
		{"GET 404 not retried", get("/x"), [][2]any{reply(404, `{"error":"no","code":"not_found"}`)}, 1, nil, new(*NotFoundError), nil},
		{"GET 409 in_use not retried (not a create)", get("/x"), [][2]any{keyInUse409}, 1, nil, new(*ConflictError), nil},
		{"POST action 500 not retried", endpoint{method: http.MethodPost, path: "/expire"}, [][2]any{err500}, 1, nil, new(*ServerError), nil},
		{"POST action 503 not retried", endpoint{method: http.MethodPost, path: "/cancel"}, [][2]any{err503}, 1, nil, new(*ServerError), nil},
		{"PATCH 500 not retried", endpoint{method: http.MethodPatch, path: "/products/x", body: map[string]any{}}, [][2]any{err500}, 1, nil, new(*ServerError), nil},
		{"DELETE 503 not retried", endpoint{method: http.MethodDelete, path: "/products/x"}, [][2]any{err503}, 1, nil, new(*ServerError), nil},
		{"POST action 429 not retried", endpoint{method: http.MethodPost, path: "/trust"}, [][2]any{reply(429, `{"error":"slow","code":"rate_limit_exceeded"}`)}, 1, nil, new(*RateLimitError), nil},
		{"create 500, 500, 201", idempotentCreate, [][2]any{err500, err500, reply(201, `{}`)}, 3, []time.Duration{time.Second, 2 * time.Second}, nil, nil},
		{"create 409 in_use then 201", idempotentCreate, [][2]any{keyInUse409, reply(201, `{}`)}, 2, []time.Duration{time.Second}, nil, nil},
		{"create 409 unique_violation not retried", idempotentCreate, [][2]any{reply(409, `{"error":"dup","code":"unique_violation","details":{"field":"reference"}}`)}, 1, nil, new(*ConflictError), nil},
		{"create 422 key reused not retried", idempotentCreate, [][2]any{reply(422, `{"error":"reused","code":"idempotency_key_reused"}`)}, 1, nil, new(*IdempotencyError), nil},
		{"create 400 validation not retried", idempotentCreate, [][2]any{reply(400, `{"error":"bad","code":"validation_failed"}`)}, 1, nil, new(*ValidationError), nil},
		{"3xx not retried", get("/x"), [][2]any{reply(302, ``)}, 1, nil, new(*ServerError), nil},
		{"maxRetries 0 disables", get("/x"), [][2]any{err500}, 1, nil, new(*ServerError), ptr(0)},
		{"maxRetries 1", get("/x"), [][2]any{err500}, 2, []time.Duration{time.Second}, new(*ServerError), ptr(1)},
		{"maxRetries 5", get("/x"), [][2]any{err503}, 6, []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}, new(*ServerError), ptr(5)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t, sequence(tc.replies...))
			var opts []ClientOption
			if tc.maxRetries != nil {
				opts = append(opts, WithMaxRetries(*tc.maxRetries))
			}
			c, sleeps := newTestClient(t, ts, opts...)
			_, err := c.send(context.Background(), tc.e, nil)

			if got := len(ts.recorded()); got != tc.attempts {
				t.Errorf("attempts = %d, want %d", got, tc.attempts)
			}
			if !slices.Equal(*sleeps, tc.sleeps) {
				t.Errorf("sleeps = %v, want %v", *sleeps, tc.sleeps)
			}
			switch target := tc.wantErr.(type) {
			case nil:
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			default:
				if err == nil || !errors.As(err, target) {
					t.Errorf("error = %T %v, want %T", err, err, target)
				}
			}
		})
	}
}

func TestRetryIdempotencyKey(t *testing.T) {
	ts := newTestServer(t, sequence(err503, keyInUse409, reply(201, `{}`)))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	if _, err := c.send(ctx, idempotentCreate, nil); err != nil {
		t.Fatal(err)
	}
	reqs := ts.recorded()
	if len(reqs) != 3 {
		t.Fatalf("attempts = %d", len(reqs))
	}
	key := reqs[0].header.Get("Idempotency-Key")
	if !isUUID(key) || key[14] != '4' || !strings.ContainsRune("89ab", rune(key[19])) {
		t.Errorf("key %q is not a UUID v4", key)
	}
	for i, r := range reqs {
		if got := r.header.Get("Idempotency-Key"); got != key {
			t.Errorf("attempt %d key %q, want the first attempt's %q", i, got, key)
		}
		if r.body != `{"name":"Pro"}` {
			t.Errorf("attempt %d body %q", i, r.body)
		}
	}

	// A new call gets a new key.
	if _, err := c.send(ctx, idempotentCreate, nil); err != nil {
		t.Fatal(err)
	}
	if next := ts.recorded()[3].header.Get("Idempotency-Key"); next == key || !isUUID(next) {
		t.Errorf("second call key %q (first %q)", next, key)
	}
}

func TestIdempotencyKeyOption(t *testing.T) {
	ts := newTestServer(t, sequence(err500, reply(201, `{}`)))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	if _, err := c.send(ctx, idempotentCreate, []RequestOption{WithIdempotencyKey("order-1042:create~v1")}); err != nil {
		t.Fatal(err)
	}
	for i, r := range ts.recorded() {
		if got := r.header.Get("Idempotency-Key"); got != "order-1042:create~v1" {
			t.Errorf("attempt %d key %q", i, got)
		}
	}

	for _, key := range []string{"has space", "tab\tkey", "é", strings.Repeat("k", 256)} {
		_, err := c.send(ctx, idempotentCreate, []RequestOption{WithIdempotencyKey(key)})
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.FieldErrors[0].Field != "Idempotency-Key" {
			t.Errorf("WithIdempotencyKey(%q): %v", key, err)
		}
	}
	if _, err := c.send(ctx, idempotentCreate, []RequestOption{WithIdempotencyKey(strings.Repeat("k", 255))}); err != nil {
		t.Errorf("255 chars: %v", err)
	}

	// Ignored (and never sent) on other methods.
	n := len(ts.recorded())
	if _, err := c.send(ctx, get("/x"), []RequestOption{WithIdempotencyKey("has space")}); err != nil {
		t.Errorf("GET with a key: %v", err)
	}
	if got := ts.recorded()[n].header.Get("Idempotency-Key"); got != "" {
		t.Errorf("GET sent Idempotency-Key %q", got)
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		header   string
		body     string
		attempts int
		sleeps   []time.Duration
		after    time.Duration // RetryAfter on the final error (when it fails)
	}{
		{"delta seconds", "5", `{"error":"slow","code":"rate_limit_exceeded"}`, 2, []time.Duration{5 * time.Second}, 0},
		{"below the backoff", "0", `{"error":"slow"}`, 2, []time.Duration{time.Second}, 0},
		{"HTTP-date", now.Add(3 * time.Second).Format(http.TimeFormat), `{"error":"slow"}`, 2, []time.Duration{3 * time.Second}, 0},
		{"details fallback", "", `{"error":"slow","details":{"retryAfterSeconds":7,"limit":60,"periodSeconds":60}}`, 2, []time.Duration{7 * time.Second}, 0},
		{"over 60 s: no retry", "120", `{"error":"slow","details":{"limit":50,"periodSeconds":3600}}`, 1, nil, 120 * time.Second},
		{"HTTP-date over 60 s: no retry", now.Add(90 * time.Second).Format(http.TimeFormat), `{"error":"slow"}`, 1, nil, 90 * time.Second},
		{"exactly 60 s: retried", "60", `{"error":"slow"}`, 2, []time.Duration{60 * time.Second}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
				if n == 0 {
					if tc.header != "" {
						w.Header().Set("Retry-After", tc.header)
					}
					jsonReply(w, 429, tc.body)
					return
				}
				jsonReply(w, 200, `{}`)
			})
			c, sleeps := newTestClient(t, ts)
			c.t.now = func() time.Time { return now }
			_, err := c.send(context.Background(), get("/x"), nil)

			if got := len(ts.recorded()); got != tc.attempts {
				t.Errorf("attempts = %d, want %d", got, tc.attempts)
			}
			if !slices.Equal(*sleeps, tc.sleeps) {
				t.Errorf("sleeps = %v, want %v", *sleeps, tc.sleeps)
			}
			if tc.attempts == 1 {
				var rl *RateLimitError
				if !errors.As(err, &rl) {
					t.Fatalf("want *RateLimitError, got %v", err)
				}
				if rl.RetryAfter != tc.after {
					t.Errorf("RetryAfter = %v, want %v", rl.RetryAfter, tc.after)
				}
			} else if err != nil {
				t.Errorf("unexpected error %v", err)
			}
		})
	}
}

func TestRateLimitBackoffGrows(t *testing.T) {
	// Retry-After 3 s: attempts wait max(3 s, 1 s·2^n) = 3 s, 3 s, 4 s.
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		w.Header().Set("Retry-After", "3")
		jsonReply(w, 429, `{"error":"slow","code":"rate_limit_exceeded","details":{"limit":60,"periodSeconds":60,"retryAfterSeconds":3}}`)
	})
	c, sleeps := newTestClient(t, ts)
	_, err := c.send(context.Background(), get("/x"), nil)
	if want := []time.Duration{3 * time.Second, 3 * time.Second, 4 * time.Second}; !slices.Equal(*sleeps, want) {
		t.Errorf("sleeps = %v, want %v", *sleeps, want)
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.Limit != 60 || rl.PeriodSeconds != 60 || rl.RetryAfter != 3*time.Second {
		t.Errorf("err = %#v", err)
	}
}

func TestNetworkErrorRetried(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
		if n < 2 {
			conn, _, _ := w.(http.Hijacker).Hijack()
			_ = conn.Close() // no response
			return
		}
		jsonReply(w, 200, `{"ok":true}`)
	})
	c, sleeps := newTestClient(t, ts)
	if _, err := c.send(context.Background(), get("/x"), nil); err != nil {
		t.Fatalf("want success after 2 network errors, got %v", err)
	}
	if len(ts.recorded()) != 3 || len(*sleeps) != 2 {
		t.Errorf("attempts %d sleeps %v", len(ts.recorded()), *sleeps)
	}

	// Exhausted: a *NetworkError.
	down := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		_ = conn.Close()
	})
	c2, _ := newTestClient(t, down)
	_, err := c2.send(context.Background(), get("/x"), nil)
	var ne *NetworkError
	if !errors.As(err, &ne) || ne.StatusCode != 0 || ne.Err == nil {
		t.Errorf("want *NetworkError, got %#v", err)
	}
	if len(down.recorded()) != 4 || !IsRetryable(err) {
		t.Errorf("attempts %d", len(down.recorded()))
	}

	// A non-retried write is attempted once.
	c3, _ := newTestClient(t, down)
	before := len(down.recorded())
	_, err = c3.send(context.Background(), endpoint{method: http.MethodPost, path: "/expire"}, nil)
	if !errors.As(err, &ne) || len(down.recorded())-before != 1 {
		t.Errorf("POST: %v, attempts %d", err, len(down.recorded())-before)
	}
}

func TestPerAttemptTimeout(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, n int) {
		if n == 0 {
			select { // hang past the timeout
			case <-release:
			case <-r.Context().Done():
			}
			return
		}
		jsonReply(w, 200, `{"ok":true}`)
	})
	c, sleeps := newTestClient(t, ts, WithTimeout(100*time.Millisecond))
	v, err := c.call[map[string]bool](context.Background(), get("/x"), nil)
	if err != nil || !v["ok"] {
		t.Fatalf("want success on the retry, got %v %v", v, err)
	}
	if len(*sleeps) != 1 {
		t.Errorf("sleeps %v", *sleeps)
	}

	c2, _ := newTestClient(t, ts, WithTimeout(100*time.Millisecond), WithMaxRetries(0))
	ts.mu.Lock()
	ts.requests = nil // the next request is n == 0 again: it hangs
	ts.mu.Unlock()
	_, err = c2.send(context.Background(), get("/x"), nil)
	var ne *NetworkError
	if !errors.As(err, &ne) || !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("want a timeout *NetworkError, got %v", err)
	}
}

func TestContextCancelled(t *testing.T) {
	ts := newTestServer(t, sequence(err503))
	c, err := NewClient(testAPIKey, WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}
	// Real sleep: the cancelled context interrupts the 1 s back-off.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for len(ts.recorded()) == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	start := time.Now()
	_, err = c.send(ctx, get("/x"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
	var ne *NetworkError
	if !errors.As(err, &ne) {
		t.Errorf("want *NetworkError, got %T", err)
	}
	if time.Since(start) > 900*time.Millisecond || len(ts.recorded()) != 1 {
		t.Errorf("not interrupted: %v, %d attempts", time.Since(start), len(ts.recorded()))
	}

	// Already cancelled: no retry.
	done, cancelDone := context.WithCancel(context.Background())
	cancelDone()
	_, err = c.send(done, get("/x"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: %v", err)
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		if r.URL.Path == "/elsewhere" {
			jsonReply(w, 200, `{"hijacked":true}`)
			return
		}
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	})
	c, sleeps := newTestClient(t, ts)
	for _, e := range []endpoint{get("/me"), idempotentCreate} {
		_, err := c.call[map[string]any](context.Background(), e, nil)
		var se *ServerError
		if !errors.As(err, &se) || se.StatusCode != 302 {
			t.Errorf("%s: want 302 *ServerError, got %v", e.method, err)
		}
		if IsRetryable(err) {
			t.Error("a 3xx is not retryable")
		}
	}
	for _, r := range ts.recorded() {
		if r.path == "/elsewhere" {
			t.Error("redirect followed")
		}
	}
	if len(ts.recorded()) != 2 || len(*sleeps) != 0 {
		t.Errorf("attempts %d sleeps %v", len(ts.recorded()), *sleeps)
	}
}

func TestIsRetryable(t *testing.T) {
	mk := func(status int, code string) error {
		return errorFromResponse(&response{status: status, header: http.Header{}, body: []byte(`{"error":"x","code":"` + code + `"}`)}, time.Now())
	}
	cases := []struct {
		err  error
		want bool
	}{
		{mk(500, "internal"), true},
		{mk(503, "network_unavailable"), true},
		{mk(504, "timeout"), true},
		{mk(429, "rate_limit_exceeded"), true},
		{mk(409, "idempotency_key_in_use"), true},
		{mk(409, "unique_violation"), false},
		{mk(422, "idempotency_key_reused"), false},
		{mk(400, "validation_failed"), false},
		{mk(404, "not_found"), false},
		{mk(302, ""), false},
		{newNetworkError("down", errors.New("dial")), true},
		{newValidationError("bad"), false},
		{errors.New("other"), false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := IsRetryable(tc.err); got != tc.want {
			t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestNewUUIDv4(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := newUUIDv4()
		if !isUUID(id) || id[14] != '4' || !strings.ContainsRune("89ab", rune(id[19])) || seen[id] {
			t.Fatalf("bad or repeated UUID v4 %q", id)
		}
		seen[id] = true
	}
}
