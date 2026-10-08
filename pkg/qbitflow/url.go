package qbf

import "net/url"

// escapePath percent-encodes a value for safe interpolation into a URL path segment.
//
// References, emails and customer identifiers come from the caller's own systems and
// routinely contain characters such as '/', '#' and '?' that would otherwise change the
// shape of the request: a product reference of "ORD/2026/17" interpolated raw addresses
// a different route entirely, so the lookup either 404s or returns the wrong resource.
//
// This is a no-op for the API's prefixed identifiers (pay@<uuid>, sub@<uuid>) because
// '@' is legal in a path segment, so existing calls are unaffected.
func escapePath(segment string) string {
	return url.PathEscape(segment)
}

// escapeQuery percent-encodes a value for safe interpolation into a query-string value.
//
// Unlike a path segment, '&', '=' and '+' are all significant here, so a value carrying
// any of them would otherwise be parsed as extra parameters rather than as the value.
func escapeQuery(value string) string {
	return url.QueryEscape(value)
}
