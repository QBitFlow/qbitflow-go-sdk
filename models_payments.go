package qbitflow

import (
	"net/url"
	"time"
)

// Payment is a confirmed one-time payment.
type Payment struct {
	// UUID is the payment's id (pay@…), also its checkout session's.
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet.
	From string `json:"from"`
	// To is the wallet that received it (the merchant's, or the organization's while it holds
	// the member's funds).
	To string `json:"to"`
	// Amount is what the customer paid, in USD: Price plus every line of Fees. It is what the
	// contracts split (QBitFlow's fee is taken on it). The network fee paid on top is not in it.
	Amount float64 `json:"amount"`
	// AmountMinUnits is the amount in the token's min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// CurrencyID is the currency paid in.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that currency, as Currencies.Get returns it.
	Currency *Currency `json:"currency"`
	// TxHash is the transaction's hash.
	TxHash string `json:"txHash"`
	// Chain is the chain it was paid on (ETH, BASE, SOL; the testnet's in test mode).
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the transaction on the chain's block explorer.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`

	// Reference is the merchant's reference for the payment, set when creating its checkout.
	Reference string `json:"reference,omitempty"`
	// Price is the product's price in USD, as the checkout had it. Amount = Price + the sum of
	// Fees' AmountUSD. On payments recorded before checkout fees existed, Price is Amount.
	Price float64 `json:"price"`
	// Fees is what the checkout added to the price, line by line, as the customer saw them (a
	// tax, shipping, the processing fee when the customer paid it); empty without any.
	Fees []FeeLine `json:"fees,omitempty"`
	// Name is what was paid for: the checkout's product name when the checkout was created.
	Name string `json:"name"`
	// Description is the same, for the product's description.
	Description string `json:"description"`
	// ProductUUID is the product paid for; "" for an inline product.
	ProductUUID string `json:"productUuid,omitempty"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is the merchant's reference of the customer, as the checkout was given it.
	CustomerReference string `json:"customerReference,omitempty"`
	// Note is a qbf.cash tipper's message to the handle's owner.
	Note string `json:"note,omitempty"`
	// Customer names the customer (API reads only, never in webhooks).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Metadata is the payment's fees and split.
	Metadata PaymentMetadata `json:"metadata"`

	// ConfirmedAt is when the chain confirmed it (its block's time), when known.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	// PaidMinUnits is everything the customer's wallet sent (amount plus the network fee paid on
	// top), in min units: a refund's base. "" when the split wasn't recorded.
	PaidMinUnits string `json:"paidMinUnits,omitempty"`
	// PaidUSD is the same in USD.
	PaidUSD *float64 `json:"paidUsd,omitempty"`

	// Refund is its refund, if any (API reads only).
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now (API reads only).
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not (refundExists, heldFundsReleased).
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`

	// CheckoutOpenedAt is when its checkout session was created.
	CheckoutOpenedAt *time.Time `json:"checkoutOpenedAt,omitempty"`
}

// FeeLine is an amount a checkout adds to its product's price, shown to the customer and paid
// with it, a subscription's with every bill (Payment.Fees, PaymentSessionData.Fees,
// Subscription.Fees).
type FeeLine struct {
	// Type is custom (the merchant's line) or processingFee (QBitFlow's fee, which the merchant
	// has the customer pay).
	Type FeeLineType `json:"type"`
	// Label is the line's name, as the checkout shows it ("Processing fee" for the processing fee).
	Label string `json:"label"`
	// Description is more about the line, as the merchant wrote it; "" without one.
	Description string `json:"description,omitempty"`
	// AmountUSD is the line's amount in USD, a decimal string with at most 2 decimals ("4.99").
	AmountUSD string `json:"amountUsd"`
}

// Bill is a subscription's paid bill (a subscription history entry, sub-hist@…).
//
// On Subscriptions.GetPublicHistory the fields the API only returns to the merchant
// (CustomerUUID, CustomerReference, Customer, Metadata, UserUUID, PaidMinUnits, PaidUSD,
// Refund…) are empty.
type Bill struct {
	// UUID is the bill's id (sub-hist@…).
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet.
	From string `json:"from"`
	// To is the wallet that received it.
	To string `json:"to"`
	// Amount is what the customer paid, in USD.
	Amount float64 `json:"amount"`
	// AmountMinUnits is the amount in the token's min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// CurrencyID is the currency paid in.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that currency.
	Currency *Currency `json:"currency"`
	// TxHash is the transaction's hash.
	TxHash string `json:"txHash"`
	// Chain is the chain it was paid on.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the transaction on the chain's block explorer.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`

	// Name is the subscription's product name when the bill was paid.
	Name string `json:"name"`
	// Description is the same, for the product's description.
	Description string `json:"description"`
	// ProductUUID is the subscription's product.
	ProductUUID string `json:"productUuid,omitempty"`
	// SubscriptionUUID is the bill's subscription (sub@…).
	SubscriptionUUID string `json:"subscriptionUuid"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is the merchant's reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`
	// Customer names the customer (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Metadata is the bill's fees and split.
	Metadata PaymentMetadata `json:"metadata"`
	// PeriodStart is the start of the period the bill paid (its due date).
	PeriodStart *time.Time `json:"periodStart,omitempty"`
	// PeriodEnd is the end of the period the bill paid: paid until then.
	PeriodEnd *time.Time `json:"periodEnd,omitempty"`

	// ConfirmedAt is when the chain confirmed it, when known.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	// PaidMinUnits is everything the customer's wallet sent, in min units.
	PaidMinUnits string `json:"paidMinUnits,omitempty"`
	// PaidUSD is the same in USD.
	PaidUSD *float64 `json:"paidUsd,omitempty"`

	// Refund is its refund, if any (API reads only).
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now (API reads only).
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not.
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`
}

// CombinedPayment is a row of the combined feed: a one-time payment or a subscription's bill.
type CombinedPayment struct {
	// Source is payment or subscriptionHistory.
	Source CombinedPaymentSource `json:"source"`
	// UUID is the payment's (pay@…) or the bill's (sub-hist@…) id.
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet.
	From string `json:"from"`
	// To is the wallet that received it.
	To string `json:"to"`
	// Name is what was paid for.
	Name string `json:"name"`
	// Description is the same, for the description.
	Description string `json:"description"`
	// Amount is what the customer paid, in USD (on a payment row, the price plus its checkout's
	// fees).
	Amount float64 `json:"amount"`
	// AmountMinUnits is the amount in the currency's min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// CurrencyID is the currency paid in.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that currency.
	Currency *Currency `json:"currency"`
	// ProductUUID is the product paid for; "" for an inline product.
	ProductUUID string `json:"productUuid,omitempty"`
	// TxHash is the transaction's hash.
	TxHash string `json:"txHash"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// Customer names the customer (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// CustomerReference is the merchant's reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`
	// SubscriptionUUID is the bill's subscription (sub@…); "" on a payment.
	SubscriptionUUID string `json:"subscriptionUuid,omitempty"`
	// Reference is the payment's own reference (payment rows).
	Reference string `json:"reference,omitempty"`
	// SubscriptionReference is the reference of the bill's subscription (bill rows).
	SubscriptionReference string `json:"subscriptionReference,omitempty"`
	// Chain is the chain it was paid on.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the transaction on the chain's block explorer.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space the row is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// Refund is its refund, if any.
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now.
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not.
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`
	// Metadata is its fees and split; nil only on payments recorded before fees were split (v1).
	Metadata *PaymentMetadata `json:"metadata"`
}

// Failure is a failed attempt to pay a checkout or a bill (the failures log).
type Failure struct {
	// UUID is the failure's id.
	UUID string `json:"uuid"`
	// Kind is payment, subscriptionCheckout or bill.
	Kind FailureKind `json:"kind"`
	// TxUUID is what failed: the checkout (pay@…, sub@…) or the bill (sub-hist@…).
	TxUUID string `json:"txUuid"`
	// Attempt is its number among its transaction's failed attempts, from 1.
	Attempt int `json:"attempt"`
	// SubscriptionUUID is a bill's subscription (sub@…).
	SubscriptionUUID string `json:"subscriptionUuid,omitempty"`
	// Code is the error code (e.g. insufficient_funds).
	Code string `json:"code"`
	// Category is what it means, from the code.
	Category FailureCategory `json:"category"`
	// Message is what the customer was told.
	Message string `json:"message,omitempty"`
	// AttemptedUSD is what the customer tried to pay, in USD: never received.
	AttemptedUSD float64 `json:"attemptedUsd"`
	// AttemptedMinUnits is the same in the currency's min units (a decimal string).
	AttemptedMinUnits string `json:"attemptedMinUnits"`
	// CurrencyID is the currency it was tried in.
	CurrencyID uint64 `json:"currencyId"`
	// From is the customer's wallet.
	From string `json:"from"`
	// TxHash is the transaction, when one was sent (a revert, a timeout).
	TxHash string `json:"txHash,omitempty"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// ProductUUID is the product, when it had one.
	ProductUUID string `json:"productUuid,omitempty"`
	// CreatedAt is when it failed.
	CreatedAt time.Time `json:"createdAt"`
	// Customer names the customer (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// PaymentListParams filters Payments.List.
type PaymentListParams struct {
	// Limit is the page size (server default 10, max 50; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
	// CustomerUUID keeps only this customer's.
	CustomerUUID string
	// ProductUUID keeps only this product's.
	ProductUUID string
	// CreatedAfter keeps only those created after this instant (excluded).
	CreatedAfter *time.Time
	// CreatedBefore keeps only those created before this instant (excluded).
	CreatedBefore *time.Time
	// IncludeMembers adds the members' rows (organization space only); not with UserUUID.
	IncludeMembers bool
	// UserUUID reads one member's rows (organization space only); not with IncludeMembers.
	UserUUID string
	// Refunded keeps only the payments with (true) or without (false) an approved refund.
	Refunded *bool
}

func (p *PaymentListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters (UUIDs, includeMembers/userUuid exclusivity).
func (p *PaymentListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	p.filters().validate(&v)
	return v.err()
}

func (p *PaymentListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setBoolPtr(q, "refunded", p.Refunded)
	return q
}

// CombinedPaymentListParams filters Payments.ListCombined.
type CombinedPaymentListParams struct {
	// Limit is the page size (server default 10, max 50; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
	// CustomerUUID keeps only this customer's.
	CustomerUUID string
	// ProductUUID keeps only this product's.
	ProductUUID string
	// CreatedAfter keeps only those created after this instant (excluded).
	CreatedAfter *time.Time
	// CreatedBefore keeps only those created before this instant (excluded).
	CreatedBefore *time.Time
	// IncludeMembers adds the members' rows (organization space only); not with UserUUID.
	IncludeMembers bool
	// UserUUID reads one member's rows (organization space only); not with IncludeMembers.
	UserUUID string
	// Source keeps only the payments (payment) or only the bills (subscriptionHistory).
	Source CombinedPaymentSource
	// SubscriptionUUID keeps only this subscription's bills (sub@…).
	SubscriptionUUID string
	// Refunded keeps only the rows with (true) or without (false) an approved refund.
	Refunded *bool
}

func (p *CombinedPaymentListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters.
func (p *CombinedPaymentListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	p.filters().validate(&v)
	oneOf(&v, "source", p.Source, CombinedPaymentSourcePayment, CombinedPaymentSourceSubscriptionHistory)
	v.txID("subscriptionUuid", p.SubscriptionUUID)
	return v.err()
}

func (p *CombinedPaymentListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setString(q, "source", string(p.Source))
	setString(q, "subscriptionUuid", p.SubscriptionUUID)
	setBoolPtr(q, "refunded", p.Refunded)
	return q
}

// FailureListParams filters Failures.List.
type FailureListParams struct {
	// Limit is the page size (server default 10, max 50; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
	// CustomerUUID keeps only this customer's.
	CustomerUUID string
	// ProductUUID keeps only this product's.
	ProductUUID string
	// CreatedAfter keeps only those created after this instant (excluded).
	CreatedAfter *time.Time
	// CreatedBefore keeps only those created before this instant (excluded).
	CreatedBefore *time.Time
	// IncludeMembers adds the members' rows (organization space only); not with UserUUID.
	IncludeMembers bool
	// UserUUID reads one member's rows (organization space only); not with IncludeMembers.
	UserUUID string
	// Kind keeps only one kind: payment, subscriptionCheckout or bill.
	Kind FailureKind
	// Category keeps only one category.
	Category FailureCategory
	// SubscriptionUUID keeps only this subscription's bills (sub@…).
	SubscriptionUUID string
}

func (p *FailureListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters.
func (p *FailureListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	p.filters().validate(&v)
	oneOf(&v, "kind", p.Kind, FailureKindPayment, FailureKindSubscriptionCheckout, FailureKindBill)
	oneOf(&v, "category", p.Category, FailureCategoryInsufficientBalance, FailureCategoryAllowanceExhausted,
		FailureCategoryApprovalRevoked, FailureCategoryMaxAmountExceeded, FailureCategoryReverted,
		FailureCategoryNotConfirmed, FailureCategorySignatureRejected, FailureCategoryOther)
	v.txID("subscriptionUuid", p.SubscriptionUUID)
	return v.err()
}

func (p *FailureListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setString(q, "kind", string(p.Kind))
	setString(q, "category", string(p.Category))
	setString(q, "subscriptionUuid", p.SubscriptionUUID)
	return q
}
