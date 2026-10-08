package utils

import (
	"fmt"
	"net/url"
)

// CursorQueryBuilder builds a URL with optional limit and cursor query parameters
func CursorQueryBuilder(baseEndpoint string, limit *uint16, cursor *string) string {
	endpoint, err := url.Parse(baseEndpoint)
	if err != nil {
		return baseEndpoint
	}

	q := endpoint.Query()

	// Build query parameters
	if limit != nil {
		q.Set("limit", fmt.Sprintf("%d", *limit))
	}

	if cursor != nil {
		q.Set("cursor", *cursor)
	}
	endpoint.RawQuery = q.Encode()

	return endpoint.String()
}
