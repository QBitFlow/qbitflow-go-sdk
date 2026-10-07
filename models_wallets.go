package qbitflow

import (
	"net/url"
	"time"
)

// Wallet is a wallet of a space on one chain, with its token wallets.
type Wallet struct {
	// UUID is the wallet's id.
	UUID string `json:"uuid"`
	// CreatedAt is when it was added.
	CreatedAt time.Time `json:"createdAt"`
	// PublicKey is the wallet's address.
	PublicKey string `json:"publicKey"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose wallet it is; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// TokenWallets are the tokens enabled on it.
	TokenWallets []TokenWallet `json:"tokenWallets"`
	// CurrencyID is the chain's native coin.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that native coin.
	Currency Currency `json:"currency"`
}

// TokenWallet is a token enabled on a wallet.
type TokenWallet struct {
	// UUID is the token wallet's id.
	UUID string `json:"uuid"`
	// CreatedAt is when it was enabled.
	CreatedAt time.Time `json:"createdAt"`
	// TokenID is the token's currency id.
	TokenID uint64 `json:"tokenId"`
	// Token is the token's currency.
	Token Currency `json:"token"`
	// WalletUUID is the wallet it belongs to.
	WalletUUID string `json:"walletUuid"`
	// Balance is its balance; set only by Wallets.List with WithBalances.
	Balance *Balance `json:"balance,omitempty"`
}

// Balance is a token wallet's balance.
type Balance struct {
	// Currency is the token (e.g. "SOL:USDC").
	Currency string `json:"currency"`
	// Balance is the balance in the token (a decimal string).
	Balance string `json:"balance"`
	// BalanceUSD is the same in USD.
	BalanceUSD float64 `json:"balanceUsd"`
}

// WalletListParams are Wallets.List's options.
type WalletListParams struct {
	// WithBalances adds each token wallet's balance.
	WithBalances bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *WalletListParams) Validate() error { return nil }

func (p *WalletListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBool(q, "withBalances", p.WithBalances)
	return q
}

// SupportedCurrenciesParams are Wallets.ListSupportedCurrencies' options.
type SupportedCurrenciesParams struct {
	// UserUUID reads a member's (organization key); "" for the request's space.
	UserUUID string
}

// Validate checks the params.
func (p *SupportedCurrenciesParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.uuid("userUuid", p.UserUUID)
	return v.err()
}

func (p *SupportedCurrenciesParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setString(q, "userUuid", p.UserUUID)
	return q
}

// CurrencyListParams are Currencies.ListAvailable's and ListMain's options.
type CurrencyListParams struct {
	// Test lists the testnet currencies (whatever the key's mode).
	Test bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *CurrencyListParams) Validate() error { return nil }

func (p *CurrencyListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBool(q, "test", p.Test)
	return q
}
