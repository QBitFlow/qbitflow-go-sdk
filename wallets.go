package qbitflow

import (
	"context"
	"net/http"
)

// List returns the request's space's wallets (GET /wallet/user). params.WithBalances adds each
// token wallet's balance. params may be nil. Wallets are added and removed in the dashboard.
func (s *WalletsService) List(ctx context.Context, params *WalletListParams, opts ...RequestOption) ([]Wallet, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Wallet](ctx, endpoint{method: http.MethodGet, path: "/wallet/user", query: params.values()}, opts)
}

// ListForMember returns a member's wallets in the key's mode (GET /wallet/user/:userUuid).
// Organization key only; someone who is not a member is a *NotFoundError.
func (s *WalletsService) ListForMember(ctx context.Context, userUUID string, opts ...RequestOption) ([]Wallet, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	return s.client.call[[]Wallet](ctx, endpoint{method: http.MethodGet, path: pathf("/wallet/user/%s", userUUID)}, opts)
}

// ListSupportedCurrencies returns the currencies the space's checkouts accept
// (GET /wallet/supported-currencies): check a seller is ready before opening their checkouts
// (none: 409 merchant_not_ready). params.UserUUID reads a member's (organization key; 403
// otherwise). params may be nil.
func (s *WalletsService) ListSupportedCurrencies(ctx context.Context, params *SupportedCurrenciesParams, opts ...RequestOption) ([]Currency, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Currency](ctx, endpoint{method: http.MethodGet, path: "/wallet/supported-currencies", query: params.values()}, opts)
}
