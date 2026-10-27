package qbitflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// transport is the configuration a Client and its OnBehalfOf copies share.
type transport struct {
	// apiKey is behind a pointer so that no fmt verb ever prints it: a pointer nested in a
	// value fmt prints is shown as an address (a string field would be shown as is, even
	// through the Client with %s).
	apiKey     *string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration // per attempt
	maxRetries int

	// Test hooks.
	sleep  func(ctx context.Context, d time.Duration) error // waits between attempts
	now    func() time.Time                                 // the clock (Retry-After HTTP-dates)
	newKey func() string                                    // generates an Idempotency-Key
}

// endpoint describes one API call for Client.send and the call helpers.
type endpoint struct {
	method string
	// path is the route below the base URL, starting with "/"; build it with pathf so every
	// caller-supplied segment is escaped.
	path string
	// query is sent as the query string (nil: none). Build it with a params type's values().
	query url.Values
	// body is encoded as JSON (nil: no body, no Content-Type).
	body any
	// idempotent marks the 6 idempotent creates: they send an Idempotency-Key (generated per
	// call, or WithIdempotencyKey's) and are retried like reads. Other writes never are.
	idempotent bool
	// accept overrides the Accept header (default application/json).
	accept string
}

// response is one HTTP exchange, its body read in full.
type response struct {
	status int
	header http.Header
	body   []byte
}

// pathf formats an API path, escaping every segment: a "/" inside an id is sent as %2F, never
// as a path separator, and "." / ".." segments are escaped too.
//
//	pathf("/products/%s", uuid)
func pathf(format string, segments ...string) string {
	args := make([]any, len(segments))
	for i, segment := range segments {
		escaped := url.PathEscape(segment)
		if strings.Trim(escaped, ".") == "" {
			escaped = strings.ReplaceAll(escaped, ".", "%2E")
		}
		args[i] = escaped
	}
	return fmt.Sprintf(format, args...)
}

// call sends e and decodes its JSON answer into a T. The workhorse of the service methods:
//
//	page, err := s.client.call[Page[Customer]](ctx, endpoint{method: http.MethodGet, path: "/customers", query: params.values()}, opts)
func (c *Client) call[T any](ctx context.Context, e endpoint, opts []RequestOption) (T, error) {
	v, _, err := c.callStatus[T](ctx, e, opts)
	return v, err
}

// callRef is call returning a pointer, for the methods that answer one resource.
func (c *Client) callRef[T any](ctx context.Context, e endpoint, opts []RequestOption) (*T, error) {
	v, _, err := c.callStatus[T](ctx, e, opts)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// callStatus is call that also returns the HTTP status, for the routes whose 2xx status
// means something (a subscription's cancel: 202 = still confirming on-chain).
func (c *Client) callStatus[T any](ctx context.Context, e endpoint, opts []RequestOption) (T, int, error) {
	var zero T
	resp, err := c.send(ctx, e, opts)
	if err != nil {
		return zero, 0, err
	}
	v, err := decodeResponse[T](resp)
	if err != nil {
		return zero, resp.status, err
	}
	return v, resp.status, nil
}

// callVoid sends e and ignores the answer's body (every DELETE answers 204, no body).
func (c *Client) callVoid(ctx context.Context, e endpoint, opts []RequestOption) error {
	_, err := c.send(ctx, e, opts)
	return err
}

// callItems sends e, a bounded list (one page, hasMore false: currencies, wallets, held funds,
// the accounting export), and returns its items, never nil. Defensively, a page that claims more
// is followed with its nextCursor (e's query carrying it) until hasMore is false, so a list the
// server ever splits is never cut short silently; the walk stops on the same guards as
// iteratePages (no next cursor, a cursor that did not move, an empty page).
func (c *Client) callItems[T any](ctx context.Context, e endpoint, opts []RequestOption) ([]T, error) {
	items := []T{}
	cursor := ""
	for {
		page, err := c.call[Page[T]](ctx, e, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if !page.HasMore || page.NextCursor == "" || page.NextCursor == cursor || len(page.Items) == 0 {
			return items, nil
		}
		cursor = page.NextCursor
		query := url.Values{}
		for k, v := range e.query {
			query[k] = v
		}
		query.Set("cursor", cursor)
		e.query = query
	}
}

// getByReference runs a list's ?reference= filter (a page of 0 or 1 item) and returns the item
// whose reference (read with ref) is exactly the one asked for. No such item is a
// *NotFoundError, like a 404 (status 404, code not_found), carrying the list answer's request id;
// checking the reference means a server that ignored the filter can never hand back another
// resource. what names the resource in the error's message ("product").
func (c *Client) getByReference[T any](ctx context.Context, path, what, reference string, ref func(*T) string, opts []RequestOption) (*T, error) {
	var v validator
	if v.required("reference", reference) {
		v.reference("reference", reference)
	}
	if err := v.err(); err != nil {
		return nil, err
	}
	resp, err := c.send(ctx, endpoint{method: http.MethodGet, path: path, query: url.Values{"reference": {reference}}}, opts)
	if err != nil {
		return nil, err
	}
	page, err := decodeResponse[Page[T]](resp)
	if err != nil {
		return nil, err
	}
	for i := range page.Items {
		if ref(&page.Items[i]) == reference {
			return &page.Items[i], nil
		}
	}
	return nil, &NotFoundError{&APIError{
		StatusCode: http.StatusNotFound,
		Code:       codeNotFound,
		Message:    "no " + what + " with this reference",
		Details:    map[string]any{},
		RequestID:  resp.header.Get("X-Request-Id"),
	}}
}

// callText sends e and returns the answer's body as text (the CSV accounting export).
// Error answers are still parsed as JSON errors.
func (c *Client) callText(ctx context.Context, e endpoint, opts []RequestOption) (string, error) {
	if e.accept == "" {
		e.accept = "text/csv, application/json"
	}
	resp, err := c.send(ctx, e, opts)
	if err != nil {
		return "", err
	}
	return string(resp.body), nil
}

// send performs e: it applies the request options, builds the headers, encodes the body, and
// runs the retry policy (behaviour §3). It returns any 2xx response, else a typed error.
func (c *Client) send(ctx context.Context, e endpoint, opts []RequestOption) (*response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var rc requestConfig
	for _, opt := range opts {
		if opt != nil {
			opt.applyRequest(&rc)
		}
	}

	if c.onBehalfOfErr != nil {
		return nil, c.onBehalfOfErr
	}
	onBehalfOf := c.onBehalfOf
	if rc.onBehalfOf != nil {
		if err := checkOnBehalfOf(*rc.onBehalfOf); err != nil {
			return nil, err
		}
		onBehalfOf = *rc.onBehalfOf
	}
	if rc.requestID != "" && !isRequestID(rc.requestID) {
		return nil, newFieldError("X-Request-Id", "must be 1 to 128 characters among A-Z a-z 0-9 - _ . :")
	}

	header := http.Header{}
	header.Set("X-API-Key", *c.t.apiKey)
	header.Set("User-Agent", userAgent)
	header.Set("Accept", "application/json")
	if e.accept != "" {
		header.Set("Accept", e.accept)
	}
	if onBehalfOf != "" {
		header.Set("On-Behalf-Of", onBehalfOf)
	}
	if rc.requestID != "" {
		header.Set("X-Request-Id", rc.requestID)
	}
	if e.idempotent {
		key := rc.idempotencyKey
		if key == "" {
			key = c.t.newKey() // one key per call, reused by every retry
		} else if !isIdempotencyKey(key) {
			return nil, newFieldError("Idempotency-Key", "must be 1 to 255 printable ASCII characters without spaces")
		}
		header.Set("Idempotency-Key", key)
	}

	var payload []byte
	if e.body != nil {
		var err error
		if payload, err = encodeBody(e.body); err != nil {
			return nil, err
		}
		header.Set("Content-Type", "application/json")
	}

	target := c.t.baseURL + e.path
	if len(e.query) > 0 {
		target += "?" + e.query.Encode()
	}

	maxRetries := 0
	if e.method == http.MethodGet || e.idempotent {
		maxRetries = c.t.maxRetries
	}

	for attempt := 0; ; attempt++ {
		resp, err := c.t.roundTrip(ctx, e.method, target, payload, header)
		if err == nil && resp.status >= 200 && resp.status < 300 {
			return resp, nil
		}
		if err == nil {
			err = errorFromResponse(resp, c.t.now())
		}

		if attempt >= maxRetries || !shouldRetry(err, e.idempotent) {
			return nil, err
		}
		// A retry is due but the caller gave up: report the cancellation, exactly as when it
		// happens during the back-off below, whichever moment it raced with.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, newNetworkError("request cancelled while waiting to retry", ctxErr)
		}
		delay, ok := retryDelay(err, attempt)
		if !ok {
			return nil, err
		}
		if sleepErr := c.t.sleep(ctx, delay); sleepErr != nil {
			return nil, newNetworkError("request cancelled while waiting to retry", sleepErr)
		}
	}
}

// roundTrip performs one HTTP attempt, bounded by the per-attempt timeout.
func (t *transport) roundTrip(ctx context.Context, method, target string, payload []byte, header http.Header) (*response, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(attemptCtx, method, target, body)
	if err != nil {
		v := newValidationError("cannot build the request")
		v.Err = err
		return nil, v
	}
	req.Header = header.Clone()

	res, err := t.httpClient.Do(req)
	if err != nil {
		return nil, t.attemptError(ctx, attemptCtx, "request failed", err)
	}
	defer func() { _ = res.Body.Close() }()

	// The body is read under attemptCtx too: the timeout bounds the attempt through its last
	// byte, so a server trickling its body cannot hold the call past it.
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, t.attemptError(ctx, attemptCtx, "failed to read the response", err)
	}
	return &response{status: res.StatusCode, header: res.Header, body: data}, nil
}

// attemptError wraps a failed attempt's error in a *NetworkError naming why: the caller's
// context ended, the per-attempt timeout elapsed, or what (the fallback message).
func (t *transport) attemptError(ctx, attemptCtx context.Context, fallback string, err error) *NetworkError {
	switch {
	case ctx.Err() != nil:
		return newNetworkError("request cancelled", err)
	case errors.Is(attemptCtx.Err(), context.DeadlineExceeded):
		return newNetworkError("request timed out after "+t.timeout.String(), err)
	}
	return newNetworkError(fallback, err)
}

// encodeBody encodes a request body. A value JSON cannot represent (NaN, ±Inf, invalid
// UTF-8, which encoding/json would silently replace) is the caller's: a *ValidationError, and
// nothing is sent.
func encodeBody(body any) ([]byte, error) {
	if field, ok := findInvalidUTF8(reflect.ValueOf(body), "", 0); ok {
		if field == "" {
			field = "body"
		}
		return nil, newFieldError(field, "must be valid UTF-8")
	}
	data, err := json.Marshal(body)
	if err != nil {
		v := newValidationError("request body cannot be encoded as JSON")
		v.Err = err
		return nil, v
	}
	return data, nil
}

// findInvalidUTF8 walks a request value and reports the wire name of the first string that is
// not valid UTF-8.
func findInvalidUTF8(v reflect.Value, path string, depth int) (string, bool) {
	if !v.IsValid() || depth > 32 {
		return "", false
	}
	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return path, true
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return findInvalidUTF8(v.Elem(), path, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if field, ok := findInvalidUTF8(v.Index(i), path+"["+strconv.Itoa(i)+"]", depth+1); ok {
				return field, ok
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			if !utf8.ValidString(key) {
				return path, true
			}
			if field, ok := findInvalidUTF8(iter.Value(), joinPath(path, key), depth+1); ok {
				return field, ok
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			sf := t.Field(i)
			if !sf.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}
			sub := path
			if !sf.Anonymous || name != "" {
				if name == "" {
					name = sf.Name
				}
				sub = joinPath(path, name)
			}
			if field, ok := findInvalidUTF8(v.Field(i), sub, depth+1); ok {
				return field, ok
			}
		}
	}
	return "", false
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// decodeResponse decodes a 2xx JSON answer (behaviour §6): an absent or null value leaves the
// field's zero value; a value of the wrong JSON type, an empty body or a non-JSON body is a
// *ServerError. Integral floats are accepted into integer fields (1.0 → 1).
func decodeResponse[T any](resp *response) (T, error) {
	var zero T
	if len(bytes.TrimSpace(resp.body)) == 0 {
		return zero, newServerError("empty response body where JSON was expected", resp, nil)
	}
	// Every answer is a JSON object (lists are pages): null is no answer at all.
	if bytes.Equal(bytes.TrimSpace(resp.body), []byte("null")) {
		return zero, newServerError("null response body where a JSON object was expected", resp, nil)
	}
	v, err := decodeJSON[T](resp.body)
	if err != nil {
		return zero, newServerError("unexpected response body", resp, err)
	}
	return v, nil
}

// decodeJSON unmarshals data into a T, retrying once with integral floats rewritten as integers
// when the first attempt failed on a number (encoding/json refuses 1.0 for an int).
func decodeJSON[T any](data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && strings.HasPrefix(typeErr.Value, "number") {
		if normalized, changed := normalizeIntegralFloats(data); changed {
			var retry T
			if retryErr := json.Unmarshal(normalized, &retry); retryErr == nil {
				return retry, nil
			}
		}
	}
	return v, err
}

// normalizeIntegralFloats rewrites every JSON number written as an integral float (1.0, 2e3)
// as an integer. It reports whether anything changed.
func normalizeIntegralFloats(data []byte) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var tree any
	if dec.Decode(&tree) != nil {
		return nil, false
	}
	changed := false
	var walk func(any) any
	walk = func(node any) any {
		switch n := node.(type) {
		case map[string]any:
			for k, v := range n {
				n[k] = walk(v)
			}
		case []any:
			for i, v := range n {
				n[i] = walk(v)
			}
		case json.Number:
			s := string(n)
			if !strings.ContainsAny(s, ".eE") {
				return n
			}
			f, err := strconv.ParseFloat(s, 64)
			if err != nil || f != math.Trunc(f) || math.Abs(f) >= 1<<63 {
				return n
			}
			changed = true
			return json.Number(strconv.FormatInt(int64(f), 10))
		}
		return node
	}
	tree = walk(tree)
	if !changed {
		return nil, false
	}
	out, err := json.Marshal(tree)
	if err != nil {
		return nil, false
	}
	return out, true
}
