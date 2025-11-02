package utils

import (
	"fmt"
	"net/url"
)

// Helper functions for pointer conversions
func IntPtr(i int) *int             { return &i }
func Uint16Ptr(i uint16) *uint16    { return &i }
func Uint64Ptr(u uint64) *uint64    { return &u }
func StringPtr(s string) *string    { return &s }
func Float64Ptr(f float64) *float64 { return &f }

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
