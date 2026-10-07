package qbf

import "net/url"

// escapePath percent-encodes a value for safe interpolation into a URL path segment.
//
// References, emails and customer identifiers come from the caller's own systems and can
// contain characters such as '?', '#', '%' or spaces that would otherwise change the shape
// of the request (a '?' would start the query string). Escaping keeps the value one path
// segment.
//
// A '/' is escaped too (as %2F), which is the correct encoding, but the API currently
// cannot route a path parameter containing '/': such a lookup answers 404. Avoid '/' in
// references you intend to look up.
//
// This is a no-op for the API's prefixed identifiers (pay@<uuid>, sub@<uuid>) because
// '@' is legal in a path segment.
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
