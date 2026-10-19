package qbitflow

import (
	"net/url"
	"time"
)

// Wallet is a space's receiving address on one chain (Currency, its native coin), with the
// currencies it accepts on that chain (TokenWallets). Its mode is the space's. Wallets are added
// and removed in the dashboard.
type Wallet struct {
	// UUID is the wallet's id.
	UUID string `json:"uuid"`
	// CreatedAt is when it was added.
	CreatedAt time.Time `json:"createdAt"`
	// Address is the wallet's address on its chain (EVM: 0x…, Solana: base58).
	Address string `json:"address"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose wallet it is; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// TokenWallets are the currencies it accepts on its chain; empty when none.
	TokenWallets []TokenWallet `json:"tokenWallets"`
	// Currency is the chain, by its native coin.
	Currency Currency `json:"currency"`
}

// TokenWallet is a currency a wallet accepts on its chain: a token (USDC on Base) or the chain's
// native coin.
type TokenWallet struct {
	// UUID is the token wallet's id.
	UUID string `json:"uuid"`
	// CreatedAt is when the wallet started accepting it.
	CreatedAt time.Time `json:"createdAt"`
	// Currency is the currency accepted.
	Currency Currency `json:"currency"`
	// WalletUUID is the wallet it belongs to.
	WalletUUID string `json:"walletUuid"`
	// Balance is what the wallet holds of the currency, read on its chain now: set only by
	// Wallets.List with WithBalances, and nil when the chain couldn't be read.
	Balance *Balance `json:"balance,omitempty"`
}

// Balance is what a wallet holds of a currency, read on its chain now. Amount, AmountMinUnits
// and AmountUSD (its value at today's price; "" when QBitFlow has no price it can trust right
// now) come from TokenAmount.
//
//	{"currency": {…}, "amount": "12.5", "amountMinUnits": "12500000", "amountUsd": "12.5"}
type Balance struct {
	// Currency is the currency held.
	Currency Currency `json:"currency"`

	TokenAmount
}

// WalletListParams are Wallets.List's options.
type WalletListParams struct {
	// WithBalances adds each token wallet's balance (read on-chain: slower).
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

// AcceptedCurrenciesParams are Wallets.ListAcceptedCurrencies' options.
type AcceptedCurrenciesParams struct {
	// UserUUID reads a member's (an organization key; 403 otherwise, also with On-Behalf-Of); ""
	// for the request's space.
	UserUUID string
}

// Validate checks the params.
func (p *AcceptedCurrenciesParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.uuid("userUuid", p.UserUUID)
	return v.err()
}

func (p *AcceptedCurrenciesParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setString(q, "userUuid", p.UserUUID)
	return q
}

// CurrencyListParams are Currencies.List's options.
type CurrencyListParams struct {
	// Test lists the test networks' currencies (whatever the key's mode); false: mainnet.
	Test bool
	// Main lists only the chains' native coins (ETH, BASE, SOL), no tokens.
	Main bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *CurrencyListParams) Validate() error { return nil }

func (p *CurrencyListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBool(q, "test", p.Test)
	setBool(q, "main", p.Main)
	return q
}
