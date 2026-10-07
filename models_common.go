package qbitflow

import (
	"net/url"
	"strconv"
	"time"
)

// Typing rule (types.md §1): a field the API always sends is a value; an optional string is
// a string ("" = absent); any other optional or nullable field is a pointer (nil = absent or
// null); lists are slices (nil when absent). Money amounts in a token's min units, and a few
// USD prices, are decimal strings, never parsed. Timestamps keep the offset the API sends.

// Duration is a length of time as the API writes it: a value in the largest exact unit
// ({1, months} = 30 days). Value 0 means none (a unit is then optional).
type Duration struct {
	// Value is the number of units; 0 means no duration.
	Value uint32 `json:"value"`
	// Unit is required when Value > 0: seconds, minutes, hours, days, weeks, months (30 days)
	// or years (365 days).
	Unit DurationUnit `json:"unit,omitempty"`
}

// Currency is a currency (token or native coin) of the catalog.
type Currency struct {
	// ID is the currency's id (GET /utils/currency/id/:id).
	ID uint64 `json:"id"`
	// Name is its full name (e.g. "USD Coin").
	Name string `json:"name"`
	// Symbol is its symbol (e.g. "USDC").
	Symbol string `json:"symbol"`
	// Decimals is its number of decimals (6 for USDC).
	Decimals uint8 `json:"decimals"`
	// Address is the token's contract address (or mint); "" for a chain's native coin.
	Address string `json:"address"`
	// MainCurrencyID is the chain's native coin, for a token; nil for a native coin.
	MainCurrencyID *uint64 `json:"mainCurrencyId,omitempty"`
	// MainCurrency is that native coin; nil for a native coin.
	MainCurrency *Currency `json:"mainCurrency"`
	// Test is true for a testnet currency.
	Test bool `json:"test"`
}

// CustomerSummary is a customer as a payment, bill or refund names it (merchant API reads only;
// absent from webhooks).
type CustomerSummary struct {
	// UUID is the customer (Customers.Get).
	UUID string `json:"uuid"`
	// Name is the first name.
	Name string `json:"name"`
	// LastName is the last name, when given.
	LastName string `json:"lastName,omitempty"`
	// Email is the email, lowercase.
	Email string `json:"email"`
	// Reference is the merchant's reference of the customer, when it has one.
	Reference string `json:"reference,omitempty"`
	// Deleted is true when the merchant deleted the customer (it still names its rows).
	Deleted *bool `json:"deleted,omitempty"`
}

// RefundSummary is a transaction's refund, as its payment or bill shows it.
type RefundSummary struct {
	// UUID is the refund (refund@…).
	UUID string `json:"uuid"`
	// Status is pending, approved or rejected.
	Status RefundStatus `json:"status"`
	// InitiatedBy is customer (a request) or merchant.
	InitiatedBy RefundInitiator `json:"initiatedBy"`
	// RefundPercent is the share of what the customer paid it sends back, in percent.
	RefundPercent float64 `json:"refundPercent"`
	// AmountMinUnits is what it sends back, in the currency's min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is the same in USD, at the payment's time.
	AmountUSD float64 `json:"amountUsd"`
	// CreatedAt is when it was requested or started.
	CreatedAt time.Time `json:"createdAt"`
	// RespondedAt is when it was approved (confirmed) or denied; nil while pending.
	RespondedAt *time.Time `json:"respondedAt"`
}

// Attempt is a failed attempt to pay (or to send a refund): never final.
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

// PaymentMetadata is a payment's (or bill's) fees and split.
type PaymentMetadata struct {
	// FeePercent is QBitFlow's fee on the payment, in percent (1.5 = 1.5 %).
	FeePercent float64 `json:"feePercent"`
	// OrganizationFee is the organization's fee on its member's payment; nil when none.
	OrganizationFee *OrganizationFee `json:"organizationFee,omitempty"`
	// ReferralFee is the referrer's share of QBitFlow's fee; nil when none.
	ReferralFee *ReferralFee `json:"referralFee,omitempty"`
	// TxMetadata is the transaction's network fees and block.
	TxMetadata TxMetadata `json:"txMetadata"`
	// TxAmounts is how the payment was split (QBitFlow, referrer, organization, merchant).
	TxAmounts TxAmounts `json:"txAmounts"`
}

// OrganizationFee is the organization's fee on a member's payment (a marketplace commission).
type OrganizationFee struct {
	// Organization is the address receiving the fee.
	Organization string `json:"organization"`
	// FeePercent is the fee, in percent, taken from what remains after QBitFlow's fee.
	FeePercent float64 `json:"feePercent"`
}

// ReferralFee is a referrer's share of QBitFlow's fee.
type ReferralFee struct {
	// Referrer is the address receiving the fee.
	Referrer string `json:"referrer"`
	// FeePercent is the referrer's share of the platform fee, in percent (20 = 20 % of it).
	FeePercent float64 `json:"feePercent"`
	// Deadline is when the referral stops being paid.
	Deadline time.Time `json:"deadline"`
}

// TxMetadata is a transaction's network fees and block.
type TxMetadata struct {
	// NetworkFees are the fees the transaction's sender paid, in the native coin's min units.
	NetworkFees NetworkFees `json:"networkFees"`
	// BlockData is the block that included the transaction.
	BlockData BlockData `json:"blockData"`
	// MainCurrencyPriceUSD is the native coin's USD price at the transaction's time, when recorded.
	MainCurrencyPriceUSD *float64 `json:"mainCurrencyPriceUsd,omitempty"`
}

// BlockData is the block that included a transaction.
type BlockData struct {
	// Number is the block number (or slot).
	Number string `json:"number"`
	// Timestamp is the block's time, in unix seconds.
	Timestamp uint64 `json:"timestamp"`
}

// NetworkFees are a transaction's network fees, in the native coin's min units.
type NetworkFees struct {
	// Amount is what the sender paid: its gas and, on Base, its L1 data fee (a decimal string).
	Amount string `json:"amount"`
	// UnitsConsumed is the gas used.
	UnitsConsumed uint64 `json:"unitsConsumed"`
	// L1Fee is the part of Amount that paid Base's L1 data fee; "" on other networks.
	L1Fee string `json:"l1Fee,omitempty"`
}

// TxAmounts is how a payment was split, in USD and in the token's min units.
type TxAmounts struct {
	// USD is the split in USD at the transaction's time.
	USD TxAmountsUSD `json:"usd"`
	// MinUnits is the split in the token's min units, exactly as paid.
	MinUnits TxAmountsMinUnits `json:"minUnits"`
}

// TxAmountsMinUnits is a payment's split in the token's min units (decimal strings).
type TxAmountsMinUnits struct {
	// Platform is QBitFlow's fee (the referrer's share included).
	Platform string `json:"platform"`
	// Organization is the organization's fee.
	Organization string `json:"organization"`
	// Referral is the referrer's share.
	Referral string `json:"referral"`
	// Merchant is what the merchant received.
	Merchant string `json:"merchant"`
	// NetworkFee is the network fee the payer paid on top; "" when not recorded.
	NetworkFee string `json:"networkFee,omitempty"`
}

// TxAmountsUSD is a payment's split in USD.
type TxAmountsUSD struct {
	// Platform is QBitFlow's fee.
	Platform float64 `json:"platform"`
	// Organization is the organization's fee; nil when none.
	Organization *float64 `json:"organization,omitempty"`
	// Referral is the referrer's share; nil when none.
	Referral *float64 `json:"referral,omitempty"`
	// Merchant is what the merchant received.
	Merchant float64 `json:"merchant"`
	// NetworkFee is the network fee the payer paid on top; nil when not recorded.
	NetworkFee *float64 `json:"networkFee,omitempty"`
}

// ReadParams are the options of the reads by id that can reach a member's row (Payments.Get,
// Subscriptions.Get, Subscriptions.GetBill).
type ReadParams struct {
	// IncludeMembers reads a member's row from the organization's space (organization key
	// without On-Behalf-Of).
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
