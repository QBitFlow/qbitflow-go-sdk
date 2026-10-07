package qbf

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

// These tests live in package qbf so they can replace the client's sleep function: the
// retry policy is asserted by recording the back-off intervals instead of waiting them.

type recordedRequest struct {
	method string
	path   string
	header http.Header
}

// stubServer answers each request with the next status in the sequence (the last one
// repeats) and records what it received.
type stubServer struct {
	mu       sync.Mutex
	statuses []int
	body     string
	requests []recordedRequest
	server   *httptest.Server
}

func newStubServer(t *testing.T, body string, statuses ...int) *stubServer {
	t.Helper()
	s := &stubServer{statuses: statuses, body: body}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{method: r.Method, path: r.URL.Path, header: r.Header.Clone()})
		idx := len(s.requests) - 1
		if idx >= len(s.statuses) {
			idx = len(s.statuses) - 1
		}
		status := s.statuses[idx]
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "30")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(s.body))
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *stubServer) hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// testClient returns a client pointed at url whose sleeps are recorded rather than slept.
func testClient(t *testing.T, url string, config Config) (*Client, *[]time.Duration) {
	t.Helper()
	config.APIKey = "sk_dummy_offline_tests"
	config.BaseURL = url
	client := NewClientWithConfig(config)

	var sleeps []time.Duration
	client.sleep = func(d time.Duration) { sleeps = append(sleeps, d) }
	return client, &sleeps
}

func TestRetryGETOn5xxThenSucceeds(t *testing.T) {
	stub := newStubServer(t, `{"id":1}`, 500, 503, 200)
	client, sleeps := testClient(t, stub.server.URL, Config{})

	var out struct {
		ID int `json:"id"`
	}
	if err := client.makeRequest(http.MethodGet, "/product/id/1", nil, &out); err != nil {
		t.Fatalf("expected the third attempt to succeed, got %v", err)
	}
	if out.ID != 1 {
		t.Errorf("result not decoded: %+v", out)
	}
	if stub.hits() != 3 {
		t.Errorf("attempts = %d, want 3", stub.hits())
	}
	want := []time.Duration{time.Second, 2 * time.Second}
	if len(*sleeps) != len(want) || (*sleeps)[0] != want[0] || (*sleeps)[1] != want[1] {
		t.Errorf("back-off = %v, want %v", *sleeps, want)
	}
}

func TestRetryDefaultIsThreeRetriesWithExponentialBackoff(t *testing.T) {
	stub := newStubServer(t, `{"error":"boom"}`, 500)
	client, sleeps := testClient(t, stub.server.URL, Config{})

	err := client.makeRequest(http.MethodGet, "/product/", nil, nil)

	var serverErr *qberrors.ServerError
	if !errors.As(err, &serverErr) {
		t.Fatalf("expected *qberrors.ServerError after exhausting retries, got %T: %v", err, err)
	}
	if serverErr.StatusCode != 500 || serverErr.Message != "boom" {
		t.Errorf("unexpected error contents: status=%d message=%q", serverErr.StatusCode, serverErr.Message)
	}
	if stub.hits() != 4 {
		t.Errorf("attempts = %d, want 4 (1 + 3 retries)", stub.hits())
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	if len(*sleeps) != 3 {
		t.Fatalf("back-off = %v, want %v", *sleeps, want)
	}
	for i := range want {
		if (*sleeps)[i] != want[i] {
			t.Errorf("back-off[%d] = %v, want %v", i, (*sleeps)[i], want[i])
		}
	}

	// The new type still satisfies a pre-existing *QBitFlowError target.
	var base *qberrors.QBitFlowError
	if !errors.As(err, &base) || base.StatusCode != 500 {
		t.Error("ServerError must unwrap to *QBitFlowError")
	}
}

func TestRetryNetworkErrorIsRetriedThenReported(t *testing.T) {
	// Start and immediately close a server so every attempt is refused.
	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()

	client, sleeps := testClient(t, url, Config{MaxRetries: 2})

	err := client.makeRequest(http.MethodGet, "/product/", nil, nil)

	var netErr *qberrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *qberrors.NetworkError, got %T: %v", err, err)
	}
	if netErr.StatusCode != 0 || netErr.Err == nil {
		t.Errorf("network error must carry status 0 and the transport cause: %+v", netErr.QBitFlowError)
	}
	if len(*sleeps) != 2 {
		t.Errorf("expected 2 back-offs for MaxRetries=2, got %v", *sleeps)
	}
}

func TestRetryNeverOnPOSTPUTDELETE(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			stub := newStubServer(t, `{"error":"boom"}`, 500)
			client, sleeps := testClient(t, stub.server.URL, Config{})

			err := client.makeRequest(method, "/product/", map[string]string{"name": "x"}, nil)

			var serverErr *qberrors.ServerError
			if !errors.As(err, &serverErr) {
				t.Fatalf("expected *qberrors.ServerError, got %T: %v", err, err)
			}
			if stub.hits() != 1 {
				t.Errorf("%s was attempted %d times; non-idempotent methods must never be retried", method, stub.hits())
			}
			if len(*sleeps) != 0 {
				t.Errorf("unexpected back-off for %s: %v", method, *sleeps)
			}
		})
	}
}

func TestRetryNeverOn4xx429Or3xx(t *testing.T) {
	cases := []struct {
		status int
		body   string
		check  func(t *testing.T, err error)
	}{
		{400, `{"errors":[{"field":"Price","message":"too low"}]}`, func(t *testing.T, err error) {
			var e *qberrors.ValidationError
			if !errors.As(err, &e) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
		}},
		{404, `{"error":"resource not found"}`, func(t *testing.T, err error) {
			var e *qberrors.NotFoundError
			if !errors.As(err, &e) {
				t.Fatalf("expected NotFoundError, got %T", err)
			}
		}},
		{429, `{"error":"slow down"}`, func(t *testing.T, err error) {
			var e *qberrors.RateLimitError
			if !errors.As(err, &e) {
				t.Fatalf("expected RateLimitError, got %T", err)
			}
			if e.RetryAfter == nil || *e.RetryAfter != 30 {
				t.Errorf("RetryAfter = %v, want 30", e.RetryAfter)
			}
		}},
		{302, ``, func(t *testing.T, err error) {
			var e *qberrors.ServerError
			if !errors.As(err, &e) {
				t.Fatalf("expected ServerError for a redirect, got %T: %v", err, err)
			}
		}},
	}

	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			stub := newStubServer(t, tc.body, tc.status)
			client, sleeps := testClient(t, stub.server.URL, Config{})

			err := client.makeRequest(http.MethodGet, "/product/", nil, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			tc.check(t, err)
			if stub.hits() != 1 {
				t.Errorf("status %d was retried (%d attempts)", tc.status, stub.hits())
			}
			if len(*sleeps) != 0 {
				t.Errorf("unexpected back-off: %v", *sleeps)
			}
		})
	}
}

func TestRetryNeverOnGETActions(t *testing.T) {
	for _, action := range []string{"force-cancel", "execute-billing", "claim-funds-test-trigger"} {
		t.Run(action, func(t *testing.T) {
			stub := newStubServer(t, `{"error":"boom"}`, 500)
			client, sleeps := testClient(t, stub.server.URL, Config{})
			subs := NewSubscriptionService(client)

			var err error
			switch action {
			case "force-cancel":
				_, err = subs.ForceCancel("sub@1")
			case "execute-billing":
				_, err = subs.ExecuteTestBillingCycle("sub@1")
			default:
				_, err = NewClaimService(client).TriggerTestClaimFunds(7)
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if stub.hits() != 1 {
				t.Errorf("%s was attempted %d times; a GET action must never be retried", action, stub.hits())
			}
			if len(*sleeps) != 0 {
				t.Errorf("unexpected back-off: %v", *sleeps)
			}
		})
	}
}

func TestRetryCanBeDisabledOrTuned(t *testing.T) {
	t.Run("DisableRetries", func(t *testing.T) {
		stub := newStubServer(t, `{"error":"boom"}`, 500)
		client, sleeps := testClient(t, stub.server.URL, Config{DisableRetries: true})

		_ = client.makeRequest(http.MethodGet, "/product/", nil, nil)
		if stub.hits() != 1 || len(*sleeps) != 0 {
			t.Errorf("DisableRetries must attempt exactly once: attempts=%d sleeps=%v", stub.hits(), *sleeps)
		}
	})

	t.Run("MaxRetries=1", func(t *testing.T) {
		stub := newStubServer(t, `{"error":"boom"}`, 500)
		client, sleeps := testClient(t, stub.server.URL, Config{MaxRetries: 1})

		_ = client.makeRequest(http.MethodGet, "/product/", nil, nil)
		if stub.hits() != 2 {
			t.Errorf("attempts = %d, want 2", stub.hits())
		}
		if len(*sleeps) != 1 || (*sleeps)[0] != time.Second {
			t.Errorf("back-off = %v, want [1s]", *sleeps)
		}
	})

	t.Run("MaxRetries=0 means the default of 3", func(t *testing.T) {
		client := NewClientWithConfig(Config{APIKey: "k", MaxRetries: 0})
		if client.maxRetries != defaultMaxRetries {
			t.Errorf("maxRetries = %d, want %d", client.maxRetries, defaultMaxRetries)
		}
	})

	t.Run("a negative MaxRetries disables retries", func(t *testing.T) {
		stub := newStubServer(t, `{"error":"boom"}`, 503)
		client, sleeps := testClient(t, stub.server.URL, Config{MaxRetries: -1})

		_ = client.makeRequest(http.MethodGet, "/product/", nil, nil)
		if stub.hits() != 1 || len(*sleeps) != 0 {
			t.Errorf("MaxRetries=-1 must attempt exactly once: attempts=%d sleeps=%v", stub.hits(), *sleeps)
		}
	})
}

func TestRedirectsAreNeverFollowed(t *testing.T) {
	// A real redirect carries a Location. net/http would follow it, re-send the request
	// (as a GET for POST/PUT/DELETE) with the X-API-Key header to whatever host it names,
	// and hand back that host's body as the result.
	var elsewhere int
	var leakedKey string
	var mu sync.Mutex
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		elsewhere++
		leakedKey = r.Header.Get("X-API-Key")
		mu.Unlock()
		_, _ = w.Write([]byte(`{"id":999,"name":"not yours"}`))
	}))
	t.Cleanup(other.Close)

	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			t.Run(fmt.Sprintf("%d %s", status, method), func(t *testing.T) {
				api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, other.URL+"/product/", status)
				}))
				t.Cleanup(api.Close)
				client, sleeps := testClient(t, api.URL, Config{})

				var out map[string]any
				err := client.makeRequest(method, "/product/1", map[string]string{"name": "x"}, &out)

				var serverErr *qberrors.ServerError
				if !errors.As(err, &serverErr) || serverErr.StatusCode != status {
					t.Fatalf("expected a ServerError with status %d, got %T: %v", status, err, err)
				}
				if len(*sleeps) != 0 {
					t.Errorf("a redirect must not be retried: %v", *sleeps)
				}
			})
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if elsewhere != 0 || leakedKey != "" {
		t.Errorf("the redirect target was contacted %d time(s), key %q", elsewhere, leakedKey)
	}
}

func TestParseRetryAfter(t *testing.T) {
	header := func(v string) http.Header {
		h := http.Header{}
		if v != "" {
			h.Set("Retry-After", v)
		}
		return h
	}

	if got := parseRetryAfter(header("120")); got == nil || *got != 120 {
		t.Errorf("delta-seconds: got %v, want 120", got)
	}
	for _, bad := range []string{"", "-5", "soon", "1.5"} {
		if got := parseRetryAfter(header(bad)); got != nil {
			t.Errorf("%q: got %d, want nil", bad, *got)
		}
	}

	future := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(header(future)); got == nil || *got < 85 || *got > 90 {
		t.Errorf("HTTP-date in 90 s: got %v, want about 90", got)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(header(past)); got == nil || *got != 0 {
		t.Errorf("HTTP-date in the past: got %v, want 0", got)
	}
}

func TestBlankAPIKeyIsRejectedBeforeAnyRequest(t *testing.T) {
	for _, key := range []string{"", "   ", "\t\n"} {
		stub := newStubServer(t, `{}`, 200)
		client := NewClientWithConfig(Config{APIKey: key, BaseURL: stub.server.URL})

		err := client.makeRequest(http.MethodGet, "/product/", nil, nil)
		var validationErr *qberrors.ValidationError
		if !errors.As(err, &validationErr) {
			t.Errorf("key %q: expected a ValidationError, got %T: %v", key, err, err)
		}
		if stub.hits() != 0 {
			t.Errorf("key %q: %d request(s) sent", key, stub.hits())
		}
	}
}

func TestUnencodableBodyIsAValidationError(t *testing.T) {
	stub := newStubServer(t, `{}`, 200)
	client, _ := testClient(t, stub.server.URL, Config{})

	err := client.makeRequest(http.MethodPost, "/product/", map[string]float64{"price": math.NaN()}, nil)

	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected a ValidationError, got %T: %v", err, err)
	}
	if stub.hits() != 0 {
		t.Errorf("%d request(s) sent", stub.hits())
	}
}

func TestRequestHeaders(t *testing.T) {
	stub := newStubServer(t, `{}`, 200)
	client, _ := testClient(t, stub.server.URL, Config{})

	t.Run("GET carries auth and user-agent but no content type", func(t *testing.T) {
		_ = client.makeRequest(http.MethodGet, "/product/", nil, nil)
		h := stub.requests[len(stub.requests)-1].header
		if got := h.Get("X-API-Key"); got != "sk_dummy_offline_tests" {
			t.Errorf("X-API-Key = %q", got)
		}
		if got := h.Get("User-Agent"); got != "qbitflow-go/"+Version {
			t.Errorf("User-Agent = %q, want %q", got, "qbitflow-go/"+Version)
		}
		if got := h.Get("Content-Type"); got != "" {
			t.Errorf("Content-Type must not be sent without a body, got %q", got)
		}
		if got := h.Get("On-Behalf-Of"); got != "" {
			t.Errorf("On-Behalf-Of must be absent at the organization level, got %q", got)
		}
	})

	t.Run("POST carries a JSON content type", func(t *testing.T) {
		_ = client.makeRequest(http.MethodPost, "/product/", map[string]string{"name": "x"}, nil)
		h := stub.requests[len(stub.requests)-1].header
		if got := h.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
	})

	t.Run("OnBehalfOf sets the header; zero omits it", func(t *testing.T) {
		_, _ = NewProductService(client).OnBehalfOf(42).GetAll()
		h := stub.requests[len(stub.requests)-1].header
		if got := h.Get("On-Behalf-Of"); got != "42" {
			t.Errorf("On-Behalf-Of = %q, want 42", got)
		}

		_, _ = NewProductService(client).OnBehalfOf(0).GetAll()
		h = stub.requests[len(stub.requests)-1].header
		if got := h.Get("On-Behalf-Of"); got != "" {
			t.Errorf("OnBehalfOf(0) must act at the organization level, got header %q", got)
		}
	})
}

func TestTextRequestReturnsRawBody(t *testing.T) {
	stub := newStubServer(t, "paymentId,type\npay@1,payment\n", 200)
	client, _ := testClient(t, stub.server.URL, Config{})

	body, err := client.makeTextRequest("/accounting/export?format=csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "paymentId,type\npay@1,payment\n" {
		t.Errorf("body = %q", body)
	}
}

func TestNonJSON2xxIsAServerError(t *testing.T) {
	stub := newStubServer(t, "<html>proxy page</html>", 200)
	client, _ := testClient(t, stub.server.URL, Config{})

	var out map[string]any
	err := client.makeRequest(http.MethodGet, "/product/", nil, &out)

	var serverErr *qberrors.ServerError
	if !errors.As(err, &serverErr) {
		t.Fatalf("expected *qberrors.ServerError, got %T: %v", err, err)
	}
	if serverErr.StatusCode != 200 || serverErr.Err == nil {
		t.Errorf("expected status 200 and the decode cause, got %+v", serverErr.QBitFlowError)
	}
}

func TestEmptyBodies(t *testing.T) {
	t.Run("a 204 is nothing to decode", func(t *testing.T) {
		stub := newStubServer(t, "", 204)
		client, _ := testClient(t, stub.server.URL, Config{})

		var out struct{ ID int }
		if err := client.makeRequest(http.MethodGet, "/x", nil, &out); err != nil {
			t.Errorf("a 204 must not error, got %v", err)
		}
	})

	t.Run("an empty 2xx where a result is expected is a ServerError", func(t *testing.T) {
		for _, body := range []string{"", "  \n"} {
			stub := newStubServer(t, body, 200)
			client, _ := testClient(t, stub.server.URL, Config{})

			var out struct{ ID int }
			err := client.makeRequest(http.MethodGet, "/x", nil, &out)

			var serverErr *qberrors.ServerError
			if !errors.As(err, &serverErr) || serverErr.StatusCode != 200 {
				t.Errorf("body %q: expected a ServerError with status 200, got %T: %v", body, err, err)
			}
		}
	})

	t.Run("an empty 2xx is fine when no result is expected", func(t *testing.T) {
		stub := newStubServer(t, "", 200)
		client, _ := testClient(t, stub.server.URL, Config{})

		if err := client.makeRequest(http.MethodDelete, "/product/1", nil, nil); err != nil {
			t.Errorf("a DELETE without a result must not error, got %v", err)
		}
	})
}

func TestWrongJSONTypeIsAServerError(t *testing.T) {
	stub := newStubServer(t, `{"id":"not-a-number"}`, 200)
	client, _ := testClient(t, stub.server.URL, Config{})

	var out struct {
		ID uint64 `json:"id"`
	}
	err := client.makeRequest(http.MethodGet, "/x", nil, &out)

	var serverErr *qberrors.ServerError
	if !errors.As(err, &serverErr) || serverErr.StatusCode != 200 || serverErr.Err == nil {
		t.Fatalf("expected a ServerError carrying status 200 and the decode cause, got %T: %v", err, err)
	}
}
