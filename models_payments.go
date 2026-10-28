package qbitflow

import (
	"net/url"
	"time"
)

// Payment is a confirmed one-time payment (pay@…).
type Payment struct {
	// UUID is the payment's id (pay@…), also its checkout session's.
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet.
	From string `json:"from"`
	// To is the wallet that received it (the merchant's, or the organization's while it holds the
	// member's funds).
	To string `json:"to"`
	// Amount is what the customer paid, in the currency's units, exact (a decimal string: "10" for
	// 10 USDC): AmountMinUnits / 10^decimals. It is what the contracts split; the network fee
	// paid on top is not in it (Settlement.Split.NetworkFee).
	Amount string `json:"amount"`
	// AmountMinUnits is the same in the currency's smallest unit, a string of digits
	// ("10000000" for 10 USDC).
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is what the customer paid in USD, a decimal string: PriceUSD plus every line of
	// Fees.
	AmountUSD string `json:"amountUsd"`
	// Currency is the currency paid in.
	Currency Currency `json:"currency"`
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

	// Reference is your reference for the payment (an order or invoice id), set when creating its
	// checkout.
	Reference string `json:"reference,omitempty"`
	// PriceUSD is the product's price in USD, as the checkout had it, a decimal string: AmountUSD
	// is the price plus the fees' lines.
	PriceUSD string `json:"priceUsd"`
	// Fees is what the checkout added to the price, line by line, as the customer saw them (a tax,
	// shipping, the processing fee when the customer paid it); empty without any.
	Fees []FeeLine `json:"fees"`
	// ProductName is what was paid for: the checkout's product name when the checkout was created
	// (later product edits don't change it).
	ProductName string `json:"productName"`
	// ProductDescription is the same, for the product's description; "" without one.
	ProductDescription string `json:"productDescription,omitempty"`
	// Metadata is your own data, copied (read-only) from the checkout session that created the
	// payment; empty without any.
	Metadata map[string]string `json:"metadata"`
	// ProductUUID is the product paid for; "" for an inline product.
	ProductUUID string `json:"productUuid,omitempty"`
	// CustomerUUID is the customer, when QBitFlow has one (named by the checkout, found by its
	// customerReference, or recorded from the details the customer gave).
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer, as the checkout was given it, whether
	// or not QBitFlow has a customer with it.
	CustomerReference string `json:"customerReference,omitempty"`
	// Customer names the customer, even once deleted (API reads only, never in webhooks).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Settlement is how it settled: its fees, its split and its transaction.
	Settlement Settlement `json:"settlement"`

	// ConfirmedAt is when the chain confirmed it (its block's time); nil only on records made
	// before it was recorded.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	// Paid is everything the customer's wallet sent, in the currency's units: Amount plus the
	// network fee paid on top. A refund's base. "" when the split wasn't recorded.
	Paid string `json:"paid,omitempty"`
	// PaidMinUnits is the same in the currency's smallest unit.
	PaidMinUnits string `json:"paidMinUnits,omitempty"`
	// PaidUSD is the same in USD, at the payment's time.
	PaidUSD string `json:"paidUsd,omitempty"`

	// Refund is its refund (one per transaction), if any (API reads only). The merchant's own
	// refund appears once its transfer confirmed (approved).
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now (API reads only).
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not (refundExists, heldFundsReleased); "" when refundable.
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`

	// CheckoutOpenedAt is when its checkout session was created.
	CheckoutOpenedAt *time.Time `json:"checkoutOpenedAt,omitempty"`
}

// Bill is one paid bill of a subscription (sub-hist@…): what its customer paid for one period
// (PeriodStart → PeriodEnd), in the subscription's token, and how it settled. A failed attempt
// is no bill (Failures.List, subscription.billingFailed). Bills carry no merchant metadata: read
// the subscription's.
type Bill struct {
	// UUID is the bill's id (sub-hist@…).
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet.
	From string `json:"from"`
	// To is the wallet that received it.
	To string `json:"to"`
	// Amount is what the customer paid, in the currency's units, exact (a decimal string).
	Amount string `json:"amount"`
	// AmountMinUnits is the same in the currency's smallest unit, a string of digits.
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is what the customer paid in USD, a decimal string: the subscription's PriceUSD
	// plus its Fees lines.
	AmountUSD string `json:"amountUsd"`
	// Currency is the currency paid in.
	Currency Currency `json:"currency"`
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

	// ProductName is the subscription's product name when the bill was paid (later product edits
	// don't change it).
	ProductName string `json:"productName"`
	// ProductDescription is the same, for the product's description; "" without one.
	ProductDescription string `json:"productDescription,omitempty"`
	// ProductUUID is the subscription's product.
	ProductUUID string `json:"productUuid,omitempty"`
	// SubscriptionUUID is the subscription it bills (sub@…).
	SubscriptionUUID string `json:"subscriptionUuid"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer, as the checkout was given it.
	CustomerReference string `json:"customerReference,omitempty"`
	// Customer names the customer, even once deleted (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Settlement is how it settled: its fees, its split and its transaction.
	Settlement Settlement `json:"settlement"`
	// PeriodStart is the start of the period the bill paid: its due date (the subscription's start
	// for its first bill); nil on bills recorded before it was.
	PeriodStart *time.Time `json:"periodStart,omitempty"`
	// PeriodEnd is the end of the period the bill paid (the subscription's next bill): paid until
	// then; nil on bills recorded before it was.
	PeriodEnd *time.Time `json:"periodEnd,omitempty"`

	// ConfirmedAt is when the chain confirmed it (its block's time); nil only on records made
	// before it was recorded.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	// Paid is everything the customer's wallet sent, in the currency's units (Amount plus the
	// network fee): a refund's base.
	Paid string `json:"paid,omitempty"`
	// PaidMinUnits is the same in the currency's smallest unit.
	PaidMinUnits string `json:"paidMinUnits,omitempty"`
	// PaidUSD is the same in USD.
	PaidUSD string `json:"paidUsd,omitempty"`

	// Refund is its refund, if any (API reads only).
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now (API reads only).
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not.
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`
}

// Transaction is a row of the transactions feed (Transactions.List): a one-time payment or a
// subscription's bill, with the fields of its resource (a Payment or a Bill). Open the row's
// detail by its Type and UUID (Payments.Get, Subscriptions.GetBill).
type Transaction struct {
	// Type is payment (pay@…) or bill (sub-hist@…).
	Type TransactionKind `json:"type"`
	// UUID is the payment's (pay@…) or the bill's (sub-hist@…) id.
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded, once confirmed on-chain.
	CreatedAt time.Time `json:"createdAt"`
	// From is the customer's wallet that paid.
	From string `json:"from"`
	// To is the wallet that received it.
	To string `json:"to"`
	// ProductName is what was paid for (a payment: its checkout's product name; a bill: its
	// subscription's product name when it was billed).
	ProductName string `json:"productName"`
	// ProductDescription is the same, for the description; "" without one.
	ProductDescription string `json:"productDescription,omitempty"`
	// PriceUSD is the price in USD, a decimal string: a payment's checkout price, a bill's
	// subscription's locked price; "" on a bill of a subscription without a locked price.
	PriceUSD string `json:"priceUsd,omitempty"`
	// Fees is what was added to the price, line by line; empty without any.
	Fees []FeeLine `json:"fees"`
	// Amount is what the customer paid, in the currency's units, exact (a decimal string).
	Amount string `json:"amount"`
	// AmountMinUnits is the same in the currency's smallest unit, a string of digits.
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is what the customer paid in USD: the price and the fees' lines.
	AmountUSD string `json:"amountUsd"`
	// Currency is the currency paid in.
	Currency Currency `json:"currency"`
	// ProductUUID is the product paid for; "" for an inline product.
	ProductUUID string `json:"productUuid,omitempty"`
	// TxHash is the transaction that paid it.
	TxHash string `json:"txHash"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// Customer names the customer, even once deleted (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// CustomerReference is your reference of the customer, as the checkout was given it.
	CustomerReference string `json:"customerReference,omitempty"`
	// SubscriptionUUID is a bill's subscription (sub@…); "" on a payment.
	SubscriptionUUID string `json:"subscriptionUuid,omitempty"`
	// PeriodStart is a bill's period's start (its due date); nil on a payment.
	PeriodStart *time.Time `json:"periodStart,omitempty"`
	// PeriodEnd is a bill's period's end: paid until then; nil on a payment.
	PeriodEnd *time.Time `json:"periodEnd,omitempty"`
	// Reference is a payment's own reference (a bill has none of its own).
	Reference string `json:"reference,omitempty"`
	// SubscriptionReference is the reference of a bill's subscription.
	SubscriptionReference string `json:"subscriptionReference,omitempty"`
	// Metadata is a payment's merchant data (copied from its checkout session); empty for a bill
	// (its subscription's is on the subscription).
	Metadata map[string]string `json:"metadata"`
	// Chain is the chain it was paid on.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the transaction on the chain's block explorer.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space the row is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// Settlement is its fees, split and transaction, as on the Payment or the Bill.
	Settlement Settlement `json:"settlement"`

	// ConfirmedAt is when the chain confirmed it (its block's time); nil only on records made
	// before it was recorded.
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	// Paid is everything the customer's wallet sent, in the currency's units.
	Paid string `json:"paid,omitempty"`
	// PaidMinUnits is the same in the currency's smallest unit.
	PaidMinUnits string `json:"paidMinUnits,omitempty"`
	// PaidUSD is the same in USD.
	PaidUSD string `json:"paidUsd,omitempty"`

	// Refund is its refund, if any.
	Refund *RefundSummary `json:"refund,omitempty"`
	// Refundable says whether it can be refunded now.
	Refundable *bool `json:"refundable,omitempty"`
	// NotRefundableReason says why not.
	NotRefundableReason NotRefundableReason `json:"notRefundableReason,omitempty"`
}

// Failure is a failed payment attempt (the failures log): what the customer tried to pay, and
// why it failed. Recorded once per attempt, never changed; it never moved money.
type Failure struct {
	// UUID is the failure's own id, a plain UUID (a failed attempt is no transaction: no prefix).
	UUID string `json:"uuid"`
	// Kind is payment, subscriptionCheckout or bill.
	Kind FailureKind `json:"kind"`
	// TxUUID is what failed: the checkout (pay@…, sub@…: the id its payment or subscription would
	// have had) or the bill (sub-hist@…: its id once paid).
	TxUUID string `json:"txUuid"`
	// Attempt is its number among its transaction's failed attempts, from 1.
	Attempt int `json:"attempt"`
	// SubscriptionUUID is a bill's subscription (sub@…).
	SubscriptionUUID string `json:"subscriptionUuid,omitempty"`
	// Code is the error code (e.g. insufficient_funds).
	Code string `json:"code"`
	// Category is what it means, from the code.
	Category FailureCategory `json:"category"`
	// Message is what the customer was told; "" without a message.
	Message string `json:"message,omitempty"`
	// Attempted is what the customer tried to pay, in the currency's units, exact (a decimal
	// string): never received.
	Attempted string `json:"attempted"`
	// AttemptedMinUnits is the same in the currency's smallest unit, a string of digits.
	AttemptedMinUnits string `json:"attemptedMinUnits"`
	// AttemptedUSD is the same in USD, a decimal string.
	AttemptedUSD string `json:"attemptedUsd"`
	// Currency is the currency it was tried in.
	Currency Currency `json:"currency"`
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
	// UserUUID is the member whose space it was paid to; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// PaymentListParams filters Payments.List.
type PaymentListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
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
	// Reference keeps only the payment with this reference (set on its checkout): a page of 0 or
	// 1 payment per space read (Payments.GetByReference reads it).
	Reference string
	// Refunded keeps only the payments with (true) or without (false) an approved refund.
	Refunded *bool
}

func (p *PaymentListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters (cursor, UUIDs, includeMembers/userUuid exclusivity, reference).
func (p *PaymentListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	p.filters().validate(&v)
	v.reference("reference", p.Reference)
	return v.err()
}

func (p *PaymentListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setString(q, "reference", p.Reference)
	setBoolPtr(q, "refunded", p.Refunded)
	return q
}

// TransactionListParams filters Transactions.List.
type TransactionListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
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
	// Type keeps only the one-time payments (payment) or only the subscriptions' bills (bill).
	Type TransactionKind
	// SubscriptionUUID keeps only this subscription's bills (sub@… or the bare UUID).
	SubscriptionUUID string
	// Refunded keeps only the rows with (true) or without (false) an approved refund.
	Refunded *bool
}

func (p *TransactionListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters.
func (p *TransactionListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	p.filters().validate(&v)
	oneOf(&v, "type", p.Type, TransactionKindPayment, TransactionKindBill)
	v.txID("subscriptionUuid", p.SubscriptionUUID)
	return v.err()
}

func (p *TransactionListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setString(q, "type", string(p.Type))
	setString(q, "subscriptionUuid", p.SubscriptionUUID)
	setBoolPtr(q, "refunded", p.Refunded)
	return q
}

// FailureListParams filters Failures.List.
type FailureListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
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
	v.cursor(p.Cursor)
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
