package qbitflow

import (
	"net/url"
	"time"
)

// AccountingEvent is one row of the accounting export: a payment, a bill, a refund, or a fee.
//
// The fields are in the CSV's column order, laid out to be read left to right: what happened
// and when, the USD amounts (gross, the fees, net), the same amounts in the currency paid, the
// network fees, the rates, the on-chain proof, then the ids. Read CSV columns by header name, not
// position. Amounts are in the currency's units (7.5 USDC, not 7500000), decimal strings, with
// the currency's decimals beside them. A referralFee row only has Type, TxTimeUTC, Chain,
// BlockNumberOrSlot, Currency, CurrencyDecimals, CurrencyPriceUSD, TokenContractOrMint and the
// referral fee: every other field is empty.
type AccountingEvent struct {
	// TxTimeUTC is when the transaction's block was mined (UTC): an event is in the export of
	// that day.
	TxTimeUTC time.Time `json:"txTimeUtc"`
	// Type is payment, subscriptionHistory, refund, organizationFee or referralFee.
	Type AccountingEventType `json:"type"`
	// PaymentReference is the payment's reference, if any.
	PaymentReference string `json:"paymentReference,omitempty"`
	// RelatedPaymentReference is, for a refund of a payment, the payment's reference (a bill has
	// none).
	RelatedPaymentReference string `json:"relatedPaymentReference,omitempty"`
	// ProductName is the product's name.
	ProductName string `json:"productName,omitempty"`
	// CustomerReference is the merchant's reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`

	// GrossAmountUSD is the gross amount in USD: what was charged (a refund: what was sent back,
	// negative).
	GrossAmountUSD *float64 `json:"grossAmountUsd,omitempty"`
	// PlatformFeeUSD is QBitFlow's fee in USD, the referrer's share included.
	PlatformFeeUSD *float64 `json:"platformFeeUsd,omitempty"`
	// ReferralFeeUSD is the referral fee in USD.
	ReferralFeeUSD float64 `json:"referralFeeUsd"`
	// OrganizationFeeUSD is the organization's fee in USD.
	OrganizationFeeUSD *float64 `json:"organizationFeeUsd,omitempty"`
	// NetAmountUSD is what the merchant received (a payment or a bill: after the fees) or sent
	// back (a refund: the refunded amount, negative), in USD. Network fees are never part of it.
	NetAmountUSD *float64 `json:"netAmountUsd,omitempty"`

	// Chain is the chain.
	Chain Chain `json:"chain"`
	// Currency is what was paid, as markets name it: the token (USDC) or the chain's native
	// currency (ETH, SOL; Base's is ETH). Always set.
	Currency string `json:"currency"`
	// CurrencyDecimals is the currency's decimals: an amount × 10^CurrencyDecimals is its min
	// units on-chain.
	CurrencyDecimals uint8 `json:"currencyDecimals"`
	// CurrencyPriceUSD is the currency's USD price the amounts were converted at
	// (GrossAmountUSD / GrossAmount; a refund: its refunded transaction's). nil without a gross
	// amount.
	CurrencyPriceUSD *float64 `json:"currencyPriceUsd,omitempty"`
	// GrossAmount is the gross amount in the currency's units (a decimal string).
	GrossAmount string `json:"grossAmount,omitempty"`
	// PlatformFee is QBitFlow's fee in the currency's units, the referrer's share included.
	PlatformFee string `json:"platformFee,omitempty"`
	// ReferralFee is the referrer's share, paid out of the platform fee, in the currency's units:
	// what a referralFee event earned. "0" when no referral applied.
	ReferralFee string `json:"referralFee"`
	// OrganizationFee is the organization's fee in the currency's units ("0" when none).
	OrganizationFee string `json:"organizationFee,omitempty"`
	// NetAmount is the net amount in the currency's units (a refund: the refunded amount,
	// negative). GrossAmount = PlatformFee + OrganizationFee + NetAmount exactly.
	NetAmount string `json:"netAmount,omitempty"`

	// NetworkFeesPaidBy is whose wallet paid the transaction's network fees, never part of the
	// net: payer (a payment or a bill: on top of the amount), merchant (a refund the space sent:
	// book it apart) or organization (a refund of a payment the organization held for its
	// member). "" on referralFee rows.
	NetworkFeesPaidBy NetworkFeesPayer `json:"networkFeesPaidBy,omitempty"`
	// NetworkFees is the network fees in NetworkFeesCurrency's units (a decimal string); "" when
	// unknown (a refund recorded before they were).
	NetworkFees string `json:"networkFees,omitempty"`
	// NetworkFeesCurrency is what paid them: the event's currency on a payment or a bill (the
	// payer's gas refund, taken in it), the chain's native currency (ETH, SOL) on a refund.
	NetworkFeesCurrency string `json:"networkFeesCurrency,omitempty"`
	// NetworkFeesCurrencyDecimals is NetworkFeesCurrency's decimals.
	NetworkFeesCurrencyDecimals *uint8 `json:"networkFeesCurrencyDecimals,omitempty"`
	// NetworkFeesCurrencyPriceUSD is NetworkFeesCurrency's USD price NetworkFeesUSD was
	// converted at; nil when unknown (a refund recorded before it was).
	NetworkFeesCurrencyPriceUSD *float64 `json:"networkFeesCurrencyPriceUsd,omitempty"`
	// NetworkFeesUSD is the network fees in USD; nil when unknown.
	NetworkFeesUSD *float64 `json:"networkFeesUsd,omitempty"`

	// PlatformFeePercent is QBitFlow's fee rate on the event, in percent (0.99 = 0.99 %).
	PlatformFeePercent *float64 `json:"platformFeePercent,omitempty"`
	// ReferralFeePercent is the referrer's share of the platform fee, in percent (20 = 20 % of
	// it).
	ReferralFeePercent float64 `json:"referralFeePercent"`
	// OrganizationFeePercent is the organization's fee rate on the event, in percent.
	OrganizationFeePercent *float64 `json:"organizationFeePercent,omitempty"`

	// FromAddress is the sender; "" on referralFee rows.
	FromAddress string `json:"fromAddress,omitempty"`
	// ToAddress is the receiver; "" on referralFee rows.
	ToAddress string `json:"toAddress,omitempty"`
	// TxHash is the transaction's hash; "" on referralFee rows.
	TxHash string `json:"txHash,omitempty"`
	// BlockNumberOrSlot is the block (or slot).
	BlockNumberOrSlot string `json:"blockNumberOrSlot"`
	// TokenContractOrMint is the token's contract (EVM) or mint (Solana); "" for a native
	// currency.
	TokenContractOrMint string `json:"tokenContractOrMint"`
	// ExplorerURL is the transaction on the chain's explorer; "" on referralFee rows.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// ReceiptURL is the transaction's page for its customer; "" on referralFee rows.
	ReceiptURL string `json:"receiptUrl,omitempty"`

	// PaymentUUID is the event's transaction, prefixed by its type: pay@ (a payment, an
	// organization fee's), sub-hist@ (a bill), refund@ (a refund); "" on referralFee rows.
	PaymentUUID string `json:"paymentUuid,omitempty"`
	// RelatedPaymentUUID is, for a refund, the refunded payment's (pay@…) or bill's (sub-hist@…)
	// id.
	RelatedPaymentUUID string `json:"relatedPaymentUuid,omitempty"`
	// ProductUUID is the product; "" when none.
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
// the 95-day maximum) and encodes it with the format (json or csv).
func accountingQuery(from, to, format string) (url.Values, error) {
	var v validator
	v.dateRange("from", from, "to", to)
	if err := v.err(); err != nil {
		return nil, err
	}
	return url.Values{"from": {from}, "to": {to}, "format": {format}}, nil
}
