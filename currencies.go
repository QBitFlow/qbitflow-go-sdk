package qbitflow

import (
	"context"
	"net/http"
	"strconv"
)

// List returns the currency catalog (GET /currencies, public; a bounded list): the live
// currencies, or the test networks' with params.Test (whatever the key's mode); params.Main
// keeps only the chains' native coins (ETH, BASE, SOL). Rate limited to 60 requests a minute per
// IP: cache the list. params may be nil.
func (s *CurrenciesService) List(ctx context.Context, params *CurrencyListParams, opts ...RequestOption) ([]Currency, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callItems[Currency](ctx, endpoint{method: http.MethodGet, path: "/currencies", query: params.values()}, opts)
}

// Get returns a currency by its id (GET /currencies/:id, public), e.g. a token's
// MainCurrencyID. Ids are unique across test and live.
func (s *CurrenciesService) Get(ctx context.Context, id uint64, opts ...RequestOption) (*Currency, error) {
	if id == 0 {
		return nil, newFieldError("id", "must be a currency id (above 0)")
	}
	return s.client.callRef[Currency](ctx, endpoint{method: http.MethodGet, path: "/currencies/" + strconv.FormatUint(id, 10)}, opts)
}
