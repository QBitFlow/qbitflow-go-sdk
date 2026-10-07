package qbf

import (
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/utils"
)

// nonNil returns s, or an empty non-nil slice when s is nil — the API sends an empty Go
// slice as JSON null, and the SDK always hands back a usable list.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// getList fetches a route that answers with a plain JSON array.
func getList[T any](client *Client, endpoint string) ([]T, error) {
	var result []T
	if err := client.makeRequest("GET", endpoint, nil, &result); err != nil {
		return nil, err
	}
	return nonNil(result), nil
}

// getPage fetches one page of a cursor-paginated route. limit, when given, must be
// positive; a nil or empty cursor requests the first page.
func getPage[T any](client *Client, route string, limit *uint16, cursor *string) (*qbmodels.CursorData[T], error) {
	if err := validateLimit(limit); err != nil {
		return nil, err
	}

	endpoint := utils.CursorQueryBuilder(route, limit, omitEmpty(cursor))

	var result qbmodels.CursorData[T]
	if err := client.makeRequest("GET", endpoint, nil, &result); err != nil {
		return nil, err
	}
	result.Items = nonNil(result.Items)
	return &result, nil
}
