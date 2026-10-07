package qbitflow

import (
	"net/url"
	"time"
)

// AccountingEvent is one row of the accounting export: a payment, a bill, a refund, or a fee.
type AccountingEvent struct {
	// PaymentUUID is the event's transaction (prefixed); "" on referralFee rows.
	PaymentUUID string `json:"paymentUuid,omitempty"`
	// PaymentReference is the payment's reference, if any.
	PaymentReference string `json:"paymentReference,omitempty"`
	// Type is payment, subscriptionHistory, refund, organizationFee or referralFee.
	Type AccountingEventType `json:"type"`
	// TxTimeUTC is the transaction's time (the API may send a local offset).
	TxTimeUTC time.Time `json:"txTimeUtc"`
	// ReceiptURL is the transaction's page for its customer.
	ReceiptURL string `json:"receiptUrl,omitempty"`
	// RelatedPaymentUUID is, for a refund, the refunded payment.
	RelatedPaymentUUID string `json:"relatedPaymentUuid,omitempty"`
	// RelatedPaymentReference is, for a refund, the refunded payment's reference.
	RelatedPaymentReference string `json:"relatedPaymentReference,omitempty"`
	// UserUUID is the member whose space the event is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// ProductUUID is the product; "" when none.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference is the product's reference.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is the product's name.
	ProductName string `json:"productName,omitempty"`
	// ProductDescription is the product's description.
	ProductDescription string `json:"productDescription,omitempty"`
	// CustomerUUID is the customer.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is the customer's reference.
	CustomerReference string `json:"customerReference,omitempty"`
	// Chain is the chain.
	Chain Chain `json:"chain"`
	// BlockNumberOrSlot is the block (or slot).
	BlockNumberOrSlot string `json:"blockNumberOrSlot"`
	// TxHash is the transaction's hash; "" on referralFee rows.
	TxHash string `json:"txHash,omitempty"`
	// FromAddress is the sender; "" on referralFee rows.
	FromAddress string `json:"fromAddress,omitempty"`
	// ToAddress is the receiver; "" on referralFee rows.
	ToAddress string `json:"toAddress,omitempty"`
	// TokenSymbol is the token's symbol.
	TokenSymbol string `json:"tokenSymbol"`
	// CurrencyDecimals is the token's decimals.
	CurrencyDecimals uint8 `json:"currencyDecimals"`
	// TokenContractOrMint is the token's contract (or mint).
	TokenContractOrMint string `json:"tokenContractOrMint"`
	// ExplorerURL is the transaction on the chain's explorer; "" on referralFee rows.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// GrossAmount is the gross amount in min units (a decimal string).
	GrossAmount string `json:"grossAmount,omitempty"`
	// GrossAmountUSD is the gross amount in USD.
	GrossAmountUSD *float64 `json:"grossAmountUsd,omitempty"`
	// PlatformFeePercent is QBitFlow's fee rate, in percent (1.5 = 1.5 %).
	PlatformFeePercent *float64 `json:"platformFeePercent,omitempty"`
	// PlatformFeeUSD is QBitFlow's fee in USD.
	PlatformFeeUSD *float64 `json:"platformFeeUsd,omitempty"`
	// PlatformFee is QBitFlow's fee in min units.
	PlatformFee string `json:"platformFee,omitempty"`
	// OrganizationFeePercent is the organization's fee rate, in percent.
	OrganizationFeePercent *float64 `json:"organizationFeePercent,omitempty"`
	// OrganizationFeeUSD is the organization's fee in USD.
	OrganizationFeeUSD *float64 `json:"organizationFeeUsd,omitempty"`
	// OrganizationFee is the organization's fee in min units.
	OrganizationFee string `json:"organizationFee,omitempty"`
	// ReferralFeePercent is the referrer's share of the platform fee, in percent.
	ReferralFeePercent float64 `json:"referralFeePercent"`
	// ReferralFeeUSD is the referral fee in USD.
	ReferralFeeUSD float64 `json:"referralFeeUsd"`
	// ReferralFee is the referral fee in min units.
	ReferralFee string `json:"referralFee"`
	// NetworkFeesUSD is the network fees in USD.
	NetworkFeesUSD *float64 `json:"networkFeesUsd,omitempty"`
	// NetworkFees is the network fees in min units.
	NetworkFees string `json:"networkFees,omitempty"`
	// NetAmountUSD is the net amount in USD.
	NetAmountUSD *float64 `json:"netAmountUsd,omitempty"`
	// NetAmount is the net amount in min units.
	NetAmount string `json:"netAmount,omitempty"`
	// UserName is the member's first name.
	UserName string `json:"userName,omitempty"`
	// UserLastName is the member's last name.
	UserLastName string `json:"userLastName,omitempty"`
	// CustomerName is the customer's first name.
	CustomerName string `json:"customerName,omitempty"`
	// CustomerLastName is the customer's last name.
	CustomerLastName string `json:"customerLastName,omitempty"`
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
