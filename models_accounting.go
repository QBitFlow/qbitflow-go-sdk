package qbitflow

import (
	"net/url"
	"time"
)

// AccountingEvent is one row of the accounting export (Accounting.ExportJSON): a payment, a
// bill or a refund of the space, or a fee it earned on another space's payment or bill.
//
// The export is flat on purpose, for spreadsheets: one column per key, the currency named by its
// symbol, chain and decimals (no nested currency object). Every amount is a decimal string, in
// the currency's units (7.5 USDC, not 7500000) with its on-chain twin …MinUnits beside it; a
// value the row doesn't have (a fee that doesn't apply, an amount not known) is "" (absent),
// never "0". The fields are in the CSV's column order, laid out to be read left to right: what
// happened and when, the USD amounts (gross, the fees, net), the same amounts in the currency
// paid, the network fees, the rates, the on-chain proof, then the ids. Read CSV columns by header
// name, not position. Rows are ordered by TxTimeUTC, then Type, then TxUUID.
//
// GrossAmount = PlatformFee + OrganizationFee + NetAmount exactly (an absent fee counts 0); a
// refund's net is its gross, both negative. A referralFee row only tells what the referrer
// earned, when and on which chain: Type, TxTimeUTC, Chain, BlockNumberOrSlot, Currency,
// CurrencyDecimals, CurrencyPriceUSD, TokenContractOrMint, ReferralFeePercent, ReferralFeeUSD,
// ReferralFee and ReferralFeeMinUnits; every other field is absent.
type AccountingEvent struct {
	// TxTimeUTC is when the transaction's block was mined (UTC): an event is in the export of
	// that day.
	TxTimeUTC time.Time `json:"txTimeUtc"`
	// Type is payment, bill, refund, organizationFee or referralFee.
	Type AccountingEventType `json:"type"`
	// Reference is the payment's own reference (yours), when it has one.
	Reference string `json:"reference,omitempty"`
	// RelatedReference is, for a refund of a payment, the payment's reference (a bill has none).
	RelatedReference string `json:"relatedReference,omitempty"`
	// ProductName is the product's name.
	ProductName string `json:"productName,omitempty"`
	// CustomerReference is your reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`

	// GrossAmountUSD is what the customer paid in USD (a refund: what went back, negative).
	GrossAmountUSD string `json:"grossAmountUsd,omitempty"`
	// PlatformFeeUSD is QBitFlow's fee in USD, the referrer's share included; "" when none (a refund).
	PlatformFeeUSD string `json:"platformFeeUsd,omitempty"`
	// ReferralFeeUSD is the referrer's share of the platform fee in USD; "" when no referral applied.
	ReferralFeeUSD string `json:"referralFeeUsd,omitempty"`
	// OrganizationFeeUSD is the organization's fee on a member's payment in USD; "" when none.
	OrganizationFeeUSD string `json:"organizationFeeUsd,omitempty"`
	// NetAmountUSD is what the merchant received (a payment or a bill: after the fees) or sent
	// back (a refund: the refunded amount, negative), in USD. Network fees are never part of it.
	NetAmountUSD string `json:"netAmountUsd,omitempty"`

	// Chain is the chain.
	Chain Chain `json:"chain"`
	// Currency is what was paid, as markets name it: the token (USDC) or the chain's native
	// currency (ETH, SOL; Base's is ETH). Always set.
	Currency string `json:"currency"`
	// CurrencyDecimals is the currency's decimals: an amount × 10^CurrencyDecimals is its min
	// units on-chain.
	CurrencyDecimals uint8 `json:"currencyDecimals"`
	// CurrencyPriceUSD is the currency's USD price the amounts were converted at: GrossAmountUSD /
	// GrossAmount, to 8 decimals (a refund: its refunded transaction's). "" without a gross amount.
	CurrencyPriceUSD string `json:"currencyPriceUsd,omitempty"`
	// GrossAmount is the gross amount in the currency's units.
	GrossAmount string `json:"grossAmount,omitempty"`
	// GrossAmountMinUnits is the same on-chain: an integer string.
	GrossAmountMinUnits string `json:"grossAmountMinUnits,omitempty"`
	// PlatformFee is QBitFlow's fee in the currency's units, the referrer's share included.
	PlatformFee string `json:"platformFee,omitempty"`
	// PlatformFeeMinUnits is the same on-chain.
	PlatformFeeMinUnits string `json:"platformFeeMinUnits,omitempty"`
	// ReferralFee is the referrer's share, paid out of the platform fee, in the currency's units:
	// what a referralFee event earned. "" when no referral applied (none, or past its deadline).
	ReferralFee string `json:"referralFee,omitempty"`
	// ReferralFeeMinUnits is the same on-chain.
	ReferralFeeMinUnits string `json:"referralFeeMinUnits,omitempty"`
	// OrganizationFee is the organization's fee in the currency's units; "" when none.
	OrganizationFee string `json:"organizationFee,omitempty"`
	// OrganizationFeeMinUnits is the same on-chain.
	OrganizationFeeMinUnits string `json:"organizationFeeMinUnits,omitempty"`
	// NetAmount is the net amount in the currency's units (a refund: the refunded amount, negative).
	NetAmount string `json:"netAmount,omitempty"`
	// NetAmountMinUnits is the same on-chain.
	NetAmountMinUnits string `json:"netAmountMinUnits,omitempty"`

	// NetworkFeesPaidBy is whose wallet paid the transaction's network fees, never part of the
	// net: payer (a payment or a bill: on top of the amount), merchant (a refund the space sent:
	// book it apart) or organization (a refund of a payment the organization held for its
	// member). "" on referralFee rows.
	NetworkFeesPaidBy NetworkFeesPayer `json:"networkFeesPaidBy,omitempty"`
	// NetworkFees is the network fees in NetworkFeesCurrency's units; "" when not recorded.
	NetworkFees string `json:"networkFees,omitempty"`
	// NetworkFeesMinUnits is the same in NetworkFeesCurrency's min units.
	NetworkFeesMinUnits string `json:"networkFeesMinUnits,omitempty"`
	// NetworkFeesCurrency is what paid them: the event's currency on a payment or a bill (the
	// payer's gas refund, taken in it), the chain's native currency (ETH, SOL) on a refund.
	NetworkFeesCurrency string `json:"networkFeesCurrency,omitempty"`
	// NetworkFeesCurrencyDecimals is NetworkFeesCurrency's decimals; nil when absent.
	NetworkFeesCurrencyDecimals *uint8 `json:"networkFeesCurrencyDecimals,omitempty"`
	// NetworkFeesCurrencyPriceUSD is NetworkFeesCurrency's USD price NetworkFeesUSD was
	// converted at; "" when unknown.
	NetworkFeesCurrencyPriceUSD string `json:"networkFeesCurrencyPriceUsd,omitempty"`
	// NetworkFeesUSD is the network fees in USD; "" when unknown.
	NetworkFeesUSD string `json:"networkFeesUsd,omitempty"`

	// PlatformFeePercent is QBitFlow's fee rate on the event, a percent string ("0.99" = 0.99 %).
	PlatformFeePercent string `json:"platformFeePercent,omitempty"`
	// ReferralFeePercent is the referrer's share of the platform fee ("20" = 20 % of it), when a
	// referral applied.
	ReferralFeePercent string `json:"referralFeePercent,omitempty"`
	// OrganizationFeePercent is the organization's fee rate on the event, when it took one.
	OrganizationFeePercent string `json:"organizationFeePercent,omitempty"`

	// FromAddress is the sender; "" on referralFee rows.
	FromAddress string `json:"fromAddress,omitempty"`
	// ToAddress is the receiver; "" on referralFee rows.
	ToAddress string `json:"toAddress,omitempty"`
	// TxHash is the transaction's hash; "" on referralFee rows.
	TxHash string `json:"txHash,omitempty"`
	// BlockNumberOrSlot is the block (or slot).
	BlockNumberOrSlot string `json:"blockNumberOrSlot,omitempty"`
	// TokenContractOrMint is the token's contract (EVM) or mint (Solana); "" for a native currency.
	TokenContractOrMint string `json:"tokenContractOrMint,omitempty"`
	// ExplorerURL is the transaction on the chain's explorer; "" on referralFee rows.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// ReceiptURL is the transaction's public page for its customer; "" on referralFee rows.
	ReceiptURL string `json:"receiptUrl,omitempty"`

	// TxUUID is the event's transaction, prefixed by its type: pay@ (a payment), sub-hist@ (a
	// bill), refund@ (a refund); "" on referralFee rows.
	TxUUID string `json:"txUuid,omitempty"`
	// RelatedTxUUID is, for a refund, the refunded payment's (pay@…) or bill's (sub-hist@…) id.
	RelatedTxUUID string `json:"relatedTxUuid,omitempty"`
	// ProductUUID is the product; "" when the event has none.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference is the product's reference.
	ProductReference string `json:"productReference,omitempty"`
	// ProductDescription is the product's description.
	ProductDescription string `json:"productDescription,omitempty"`
	// CustomerUUID is the customer.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// UserUUID is the member whose space the event is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// accountingQuery checks an export window (two YYYY-MM-DD dates, from <= to; the API enforces
// the 92-day maximum, both days included) and encodes it with the format (json or csv).
func accountingQuery(from, to, format string) (url.Values, error) {
	var v validator
	v.dateRange("from", from, "to", to)
	if err := v.err(); err != nil {
		return nil, err
	}
	return url.Values{"from": {from}, "to": {to}, "format": {format}}, nil
}
