package qbitflow

import (
	"context"
	"net/http"
	"strconv"
)

// ListAvailable returns every currency checkouts can take (GET /utils/all-available-currencies,
// public). params.Test lists the testnet currencies, whatever the key's mode. Rate limited to
// 60 requests a minute per IP: cache the list. params may be nil.
func (s *CurrenciesService) ListAvailable(ctx context.Context, params *CurrencyListParams, opts ...RequestOption) ([]Currency, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Currency](ctx, endpoint{method: http.MethodGet, path: "/utils/all-available-currencies", query: params.values()}, opts)
}

// ListMain returns the chains' main currencies (GET /utils/all-main-currencies, public), with
// ListAvailable's params and rate limit.
func (s *CurrenciesService) ListMain(ctx context.Context, params *CurrencyListParams, opts ...RequestOption) ([]Currency, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Currency](ctx, endpoint{method: http.MethodGet, path: "/utils/all-main-currencies", query: params.values()}, opts)
}

// Get returns a currency by its id (GET /utils/currency/id/:id, public): resolve the
// CurrencyID and AcceptedCurrencyIDs fields with it. Ids are unique across test and live.
func (s *CurrenciesService) Get(ctx context.Context, id uint64, opts ...RequestOption) (*Currency, error) {
	if id == 0 {
		return nil, newFieldError("id", "must be a currency id (above 0)")
	}
	return s.client.callRef[Currency](ctx, endpoint{method: http.MethodGet, path: "/utils/currency/id/" + strconv.FormatUint(id, 10)}, opts)
}
