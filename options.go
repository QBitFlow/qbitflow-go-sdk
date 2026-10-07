package qbitflow

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the API root used when WithBaseURL is not given.
	DefaultBaseURL = "https://api.qbitflow.app/v2"
	// DefaultTimeout bounds each HTTP attempt when WithTimeout is not given.
	DefaultTimeout = 30 * time.Second
	// DefaultMaxRetries is the number of retries when WithMaxRetries is not given.
	DefaultMaxRetries = 3
)

// clientConfig is what the ClientOptions set.
type clientConfig struct {
	baseURL    string
	timeout    time.Duration
	maxRetries int
	httpClient *http.Client
	onBehalfOf string
}

// ClientOption configures NewClient: WithBaseURL, WithTimeout, WithMaxRetries,
// WithHTTPClient, WithOnBehalfOf.
type ClientOption interface {
	applyClient(*clientConfig) error
}

// RequestOption configures one call: WithOnBehalfOf, WithIdempotencyKey, WithRequestID.
type RequestOption interface {
	applyRequest(*requestConfig)
}

type clientOptionFunc func(*clientConfig) error

func (f clientOptionFunc) applyClient(c *clientConfig) error { return f(c) }

type requestOptionFunc func(*requestConfig)

func (f requestOptionFunc) applyRequest(c *requestConfig) { f(c) }

// WithBaseURL sets the API root (default https://api.qbitflow.app/v2). It must be an absolute
// http(s) URL; a trailing slash is stripped.
func WithBaseURL(baseURL string) ClientOption {
	return clientOptionFunc(func(c *clientConfig) error {
		trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
		u, err := url.Parse(trimmed)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return newFieldError("baseUrl", "must be an absolute http or https URL")
		}
		c.baseURL = trimmed
		return nil
	})
}

// WithTimeout bounds each HTTP attempt (default 30 s); a retried call may take longer in
// total. Use the context for an overall deadline. It must be positive.
func WithTimeout(timeout time.Duration) ClientOption {
	return clientOptionFunc(func(c *clientConfig) error {
		if timeout <= 0 {
			return newFieldError("timeout", "must be positive")
		}
		c.timeout = timeout
		return nil
	})
}

// WithMaxRetries sets how many times a retryable call (a read, or one of the idempotent
// creates) is re-sent after a transient failure (default 3). 0 disables retries; a negative
// value is refused.
func WithMaxRetries(maxRetries int) ClientOption {
	return clientOptionFunc(func(c *clientConfig) error {
		if maxRetries < 0 {
			return newFieldError("maxRetries", "must not be negative")
		}
		c.maxRetries = maxRetries
		return nil
	})
}

// WithHTTPClient sets the http.Client used for every request (proxies, custom transports,
// instrumentation). The SDK works on a copy that never follows redirects; its Timeout is
// left as is (the SDK bounds each attempt with WithTimeout). nil keeps the default.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return clientOptionFunc(func(c *clientConfig) error {
		c.httpClient = httpClient
		return nil
	})
}

// OnBehalfOfOption is returned by WithOnBehalfOf; it is both a ClientOption and a RequestOption.
type OnBehalfOfOption struct {
	userUUID string
}

// WithOnBehalfOf acts in a member's space: the request sends On-Behalf-Of: userUUID (a
// member's userUuid; organization key only).
//
// As a ClientOption it is the client's default (checked by NewClient). As a RequestOption it
// overrides the client's for one call, and "" forces the organization level (no header).
func WithOnBehalfOf(userUUID string) OnBehalfOfOption {
	return OnBehalfOfOption{userUUID: userUUID}
}

func (o OnBehalfOfOption) applyClient(c *clientConfig) error {
	if err := checkOnBehalfOf(o.userUUID); err != nil {
		return err
	}
	c.onBehalfOf = o.userUUID
	return nil
}

func (o OnBehalfOfOption) applyRequest(c *requestConfig) {
	value := o.userUUID
	c.onBehalfOf = &value
}

// requestConfig is what the RequestOptions set.
type requestConfig struct {
	onBehalfOf     *string // nil: the client's
	idempotencyKey string
	requestID      string
}

// WithIdempotencyKey sets the Idempotency-Key of one of the 7 idempotent creates (the SDK
// otherwise generates a UUID v4 per call), e.g. to retry a create across processes: 1 to 255
// printable ASCII characters without spaces. Other methods ignore it.
func WithIdempotencyKey(key string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) { c.idempotencyKey = key })
}

// WithRequestID sends X-Request-Id: id (1 to 128 of A-Z a-z 0-9 - _ . :); the API echoes it
// and errors carry it as RequestID.
func WithRequestID(id string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) { c.requestID = id })
}

// checkOnBehalfOf validates an On-Behalf-Of value: "" (none) or a non-nil UUID.
func checkOnBehalfOf(userUUID string) error {
	if userUUID == "" {
		return nil
	}
	if !isUUID(userUUID) || isNilUUID(userUUID) {
		return newFieldError("onBehalfOf", "must be a member's userUuid (a UUID, not the nil UUID)")
	}
	return nil
}
