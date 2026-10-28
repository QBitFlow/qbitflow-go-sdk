package qbitflow

import (
	"context"
	"net/http"
)

// List returns the request's space's wallets (GET /wallets; a bounded list), each with the
// currencies it accepts. params.WithBalances adds each token wallet's balance, read on-chain.
// params may be nil. Wallets are added and removed in the dashboard.
func (s *WalletsService) List(ctx context.Context, params *WalletListParams, opts ...RequestOption) ([]Wallet, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callItems[Wallet](ctx, endpoint{method: http.MethodGet, path: "/wallets", query: params.values()}, opts)
}

// ListForMember returns a member's wallets in the key's mode (GET /members/:userUuid/wallets; a
// bounded list). Organization key only (403 otherwise, also with On-Behalf-Of); someone who is
// not a member is a *NotFoundError.
func (s *WalletsService) ListForMember(ctx context.Context, userUUID string, opts ...RequestOption) ([]Wallet, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	return s.client.callItems[Wallet](ctx, endpoint{method: http.MethodGet, path: pathf("/members/%s/wallets", userUUID)}, opts)
}

// ListAcceptedCurrencies returns the currencies the space's checkouts accept now
// (GET /wallets/accepted-currencies; a bounded list): check a seller is ready before opening
// their checkouts (none: their checkouts answer 409 merchant_not_ready). params.UserUUID reads a
// member's (organization key; 403 otherwise). params may be nil.
func (s *WalletsService) ListAcceptedCurrencies(ctx context.Context, params *AcceptedCurrenciesParams, opts ...RequestOption) ([]Currency, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callItems[Currency](ctx, endpoint{method: http.MethodGet, path: "/wallets/accepted-currencies", query: params.values()}, opts)
}
