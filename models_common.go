package qbitflow

import (
	"net/url"
	"strconv"
	"time"
)

// Typing rule (types.md §1): a field the API always sends is a value; an optional string is a
// string ("" = absent); any other optional field is a pointer (nil = absent); lists are slices
// and merchant metadata a map[string]string (both nil-safe). The API never sends null: an
// optional field without a value is absent. Money, token amounts and fee rates are decimal
// strings ("12.5", "10004200", "0.99"), never parsed into floats: convert token amounts with
// FormatAmount and ParseAmount. Timestamps are RFC 3339 instants in UTC.

// Duration is a length of time as the API writes it: a value in the largest exact unit
// ({1, months} = 30 days). Value 0 means none (a unit is then optional).
type Duration struct {
	// Value is the number of units; 0 means no duration.
	Value uint32 `json:"value"`
	// Unit is required when Value > 0: seconds, minutes, hours, days, weeks, months (30 days)
	// or years (365 days).
	Unit DurationUnit `json:"unit,omitempty"`
}

// Currency is a currency of the catalog: a chain's native coin (ETH, SOL; its chain is itself)
// or a token on a chain (USDC on Base). It is the one currency object of every response
// (Currencies.List, Payment.Currency, Member.AcceptedCurrencies, …).
type Currency struct {
	// ID is the currency's id (Currencies.Get): a public catalog, unique across test and live.
	ID uint64 `json:"id"`
	// Code is what requests name it by: CHAIN:TOKEN for a token ("BASE:USDC"), the chain's
	// symbol for its native coin ("ETH").
	Code string `json:"code"`
	// Symbol is its symbol ("USDC", "ETH").
	Symbol string `json:"symbol"`
	// Name is its full name ("USD Coin", "Ethereum").
	Name string `json:"name"`
	// Chain is the chain it is on, by its native coin's symbol: ETH, BASE or SOL.
	Chain Chain `json:"chain"`
	// Decimals is the decimals of its on-chain amounts: min units = amount × 10^Decimals (6 for
	// USDC). FormatAmount converts with it.
	Decimals uint8 `json:"decimals"`
	// Address is a token's contract address (EVM) or mint (Solana); "" (absent) on a native coin.
	Address string `json:"address,omitempty"`
	// MainCurrencyID is a token's chain, by its native coin's id; nil (absent) on a native coin.
	MainCurrencyID *uint64 `json:"mainCurrencyId,omitempty"`
	// Test is true for a test network's currency, false for a live one.
	Test bool `json:"test"`
}

// TokenAmount is an amount of a token (or of a chain's native coin, when its field says so) as
// responses write it: the exact amount in the token's units and its on-chain integer, always
// together, and its USD value when a price is known. Its currency is the resource's.
//
//	{"amount": "12.5", "amountMinUnits": "12500000", "amountUsd": "12.5"}
type TokenAmount struct {
	// Amount is the amount in the token's units, exact: AmountMinUnits / 10^decimals, a decimal
	// string ("12.5").
	Amount string `json:"amount"`
	// AmountMinUnits is the on-chain integer, in the token's smallest unit, a string of digits
	// ("12500000").
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is its value in USD, a decimal string; "" (absent) when no price was known.
	AmountUSD string `json:"amountUsd,omitempty"`
}

// CustomerSummary is a customer as a transaction row names it (a payment, a bill, a
// subscription, a refund): who paid, even once the merchant deleted the customer. Merchant API
// reads only: absent from webhooks.
type CustomerSummary struct {
	// UUID is the customer (Customers.Get).
	UUID string `json:"uuid"`
	// FirstName is the first name.
	FirstName string `json:"firstName"`
	// LastName is the last name; "" (absent) when not given.
	LastName string `json:"lastName,omitempty"`
	// Email is the email, lowercase.
	Email string `json:"email"`
	// Reference is the merchant's reference of the customer; "" (absent) without one.
	Reference string `json:"reference,omitempty"`
	// Deleted is true when the merchant deleted the customer: it still names its rows, but
	// Customers.Get answers 404.
	Deleted bool `json:"deleted"`
}

// RefundSummary is a payment's or bill's refund, as the payment shows it (Refunds.Get has the
// rest of the refund with the same id).
type RefundSummary struct {
	// UUID is the refund (refund@…).
	UUID string `json:"uuid"`
	// Status is pending, approved or rejected.
	Status RefundStatus `json:"status"`
	// InitiatedBy is customer (a request) or merchant.
	InitiatedBy RefundInitiator `json:"initiatedBy"`
	// RefundPercent is the share of what the customer paid it sends back, a percent string
	// ("100": all of it).
	RefundPercent string `json:"refundPercent"`
	// Amount is what it sends back, in the currency's units (a decimal string); "" (absent) when
	// its currency isn't known (a refund requested before refunds named it).
	Amount string `json:"amount,omitempty"`
	// AmountMinUnits is the same in the currency's smallest unit (a string of digits).
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is the same in USD, at the payment's time (a decimal string); "" (absent) when
	// unknown.
	AmountUSD string `json:"amountUsd,omitempty"`
	// CreatedAt is when it was requested or prepared.
	CreatedAt time.Time `json:"createdAt"`
	// RespondedAt is when it was approved (confirmed) or rejected; nil while pending.
	RespondedAt *time.Time `json:"respondedAt,omitempty"`
}

// Attempt is how a session's (or a refund approval's) last attempt failed, while it waits for
// another: never final.
type Attempt struct {
	// Status is failed.
	Status AttemptStatus `json:"status"`
	// Code is the error code (e.g. insufficient_funds, token_not_approved): what to do about it.
	Code string `json:"code,omitempty"`
	// Message is the reason, for the customer.
	Message string `json:"message,omitempty"`
	// TxHash is the attempt's transaction, if it was sent.
	TxHash string `json:"txHash,omitempty"`
	// At is when it failed.
	At time.Time `json:"at"`
}

// Settlement is QBitFlow's record of how a payment or a bill settled, made once at confirmation:
// its fee terms, how the contracts split what the payer paid, and the transaction that paid it.
// Every fee record, export, held-funds total and refund reads it. Merchant only (API reads and
// webhooks).
type Settlement struct {
	// FeePercent is QBitFlow's fee on the payment, a percent string ("0.99" = 0.99 %), deducted
	// from the amount paid.
	FeePercent string `json:"feePercent"`
	// OrganizationFee is the organization's fee on its member's payment (a marketplace's
	// commission), taken from what remains after QBitFlow's fee; nil (absent) when none.
	OrganizationFee *OrganizationFee `json:"organizationFee,omitempty"`
	// ReferralFee is the referrer's share of QBitFlow's fee; nil (absent) when none.
	ReferralFee *ReferralFee `json:"referralFee,omitempty"`
	// Split is how the contracts split what the payer paid; nil (absent) on records made before
	// the split was recorded. For a payment the organization held for its member, it is the split
	// a direct payment would have had (the organization's fee withheld, not paid on-chain).
	Split *Split `json:"split,omitempty"`
	// Transaction is the transaction that paid it; nil (absent) before its confirmation.
	Transaction *SettlementTransaction `json:"transaction,omitempty"`
}

// OrganizationFee is the organization's fee on its member's payment or bill (a marketplace's
// commission).
type OrganizationFee struct {
	// Address is the organization's wallet that receives the fee.
	Address string `json:"address"`
	// FeePercent is the fee, a percent string, taken from what remains after QBitFlow's fee.
	FeePercent string `json:"feePercent"`
}

// ReferralFee is the referrer's share of QBitFlow's fee on a payment or bill.
type ReferralFee struct {
	// Address is the referrer's wallet that receives the fee.
	Address string `json:"address"`
	// FeePercent is the referrer's share of the platform fee, a percent string ("20" = 20 % of it).
	FeePercent string `json:"feePercent"`
}

// Split is how a payment or a bill was split, as the contracts paid it. Each part is an amount
// of the payment's currency (USD values in proportion to the payment's, to 6 decimals):
// Platform + Organization + Referral + Merchant = the payment's amount, exactly, in each unit.
type Split struct {
	// Platform is QBitFlow's fee (the referrer's share not included).
	Platform TokenAmount `json:"platform"`
	// Organization is the organization's fee; nil (absent) without one.
	Organization *TokenAmount `json:"organization,omitempty"`
	// Referral is the referrer's share of QBitFlow's fee; nil (absent) without one.
	Referral *TokenAmount `json:"referral,omitempty"`
	// Merchant is what the merchant (the member, for a member's payment) received, net of every
	// fee.
	Merchant TokenAmount `json:"merchant"`
	// NetworkFee is the network fee the payer paid on top of the amount, in the payment's
	// currency; nil (absent) on records made before it was recorded ("0" once recorded as none).
	NetworkFee *TokenAmount `json:"networkFee,omitempty"`
}

// SettlementTransaction is the transaction that paid (or refunded, or released): where and when
// it landed, and what it cost its sender in the chain's native coin.
type SettlementTransaction struct {
	// BlockNumber is the block (Ethereum, Base) or slot (Solana) it landed in, a string of digits.
	BlockNumber string `json:"blockNumber"`
	// BlockTime is when its block was produced (UTC); nil (absent) only on records made before it
	// was recorded.
	BlockTime *time.Time `json:"blockTime,omitempty"`
	// GasUsed is the gas it consumed (Ethereum, Base); nil (absent) on Solana.
	GasUsed *uint64 `json:"gasUsed,omitempty"`
	// Fee is what the transaction cost its sender, in the chain's native coin (ETH, SOL): its gas
	// and, on Base, its L1 data fee; its AmountUSD at NativePriceUSD.
	Fee TokenAmount `json:"fee"`
	// L1Fee is the part of Fee that paid Base's L1 data fee, in the native coin; nil (absent) off
	// Base.
	L1Fee *TokenAmount `json:"l1Fee,omitempty"`
	// NativePriceUSD is the USD price of the chain's native coin when the transaction was
	// recorded (a decimal string); "" (absent) when no price was available.
	NativePriceUSD string `json:"nativePriceUsd,omitempty"`
}

// TransferSettlement is what a refund or a held-funds release records of its transfer: the
// transaction only (a transfer pays no fee and has no split).
type TransferSettlement struct {
	// Transaction is the transfer's transaction; nil (absent) before it was confirmed.
	Transaction *SettlementTransaction `json:"transaction,omitempty"`
}

// FeeLine is an amount a checkout adds to its product's price, shown to the customer and paid
// with it, a subscription's with every bill (Payment.Fees, Subscription.Fees,
// Transaction.Fees, PaymentSessionData.Fees).
type FeeLine struct {
	// Type is custom (the merchant's line) or processingFee (QBitFlow's fee, which the merchant
	// has the customer pay).
	Type FeeLineType `json:"type"`
	// Label is the line's name, as the checkout shows it ("Processing fee" for the processing fee).
	Label string `json:"label"`
	// Description is more about the line, as the merchant wrote it; "" (absent) without one.
	Description string `json:"description,omitempty"`
	// AmountUSD is the line's amount in USD, a decimal string with at most 2 decimals ("4.99").
	AmountUSD string `json:"amountUsd"`
}

// ReadParams are the options of the reads by id that can reach a member's row (Payments.Get,
// Subscriptions.Get, Subscriptions.GetBill).
type ReadParams struct {
	// IncludeMembers reads a member's row from the organization's space (organization key
	// without On-Behalf-Of; 400 from a member's space).
	IncludeMembers bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *ReadParams) Validate() error { return nil }

func (p *ReadParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBool(q, "includeMembers", p.IncludeMembers)
	return q
}

// Query helpers: an unset value is omitted; booleans are true/false; times are RFC 3339
// (url.Values.Encode escapes a "+" offset as %2B).

func setString(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func setBool(q url.Values, key string, value bool) {
	if value {
		q.Set(key, "true")
	}
}

func setBoolPtr(q url.Values, key string, value *bool) {
	if value != nil {
		q.Set(key, strconv.FormatBool(*value))
	}
}

func setInt(q url.Values, key string, value int) {
	if value != 0 {
		q.Set(key, strconv.Itoa(value))
	}
}

func setTime(q url.Values, key string, value *time.Time) {
	if value != nil {
		q.Set(key, value.Format(time.RFC3339Nano))
	}
}

// setPage encodes a cursor page's limit and cursor.
func setPage(q url.Values, limit int, cursor string) {
	setInt(q, "limit", limit)
	setString(q, "cursor", cursor)
}

// listFilters are the filters every transaction list shares (txtypes.ListFilters).
type listFilters struct {
	customerUUID, productUUID   string
	createdAfter, createdBefore *time.Time
	includeMembers              bool
	userUUID                    string
}

func (f listFilters) encode(q url.Values) {
	setString(q, "customerUuid", f.customerUUID)
	setString(q, "productUuid", f.productUUID)
	setTime(q, "createdAfter", f.createdAfter)
	setTime(q, "createdBefore", f.createdBefore)
	setBool(q, "includeMembers", f.includeMembers)
	setString(q, "userUuid", f.userUUID)
}

func (f listFilters) validate(v *validator) {
	v.uuid("customerUuid", f.customerUUID)
	v.uuid("productUuid", f.productUUID)
	v.uuid("userUuid", f.userUUID)
	v.exclusive("includeMembers", f.includeMembers, "userUuid", f.userUUID != "")
}
