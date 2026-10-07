package qbitflow

import (
	"net/url"
	"time"
)

// Refund is a refund of a payment or a bill.
type Refund struct {
	// UUID is the refund's id (refund@…).
	UUID string `json:"uuid"`
	// TxUUID is the refunded transaction: a payment (pay@…) or a bill (sub-hist@…).
	TxUUID string `json:"txUuid"`
	// Reason is the customer's reason, or the merchant's when it started the refund.
	Reason string `json:"reason"`
	// Status is pending, approved or rejected.
	Status RefundStatus `json:"status"`
	// CreatedAt is when it was requested or started.
	CreatedAt time.Time `json:"createdAt"`
	// MerchantMessage is the merchant's message to the customer ("" until set).
	MerchantMessage string `json:"merchantMessage"`
	// RespondedAt is when the merchant answered; nil while pending.
	RespondedAt *time.Time `json:"respondedAt"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// TxHash is the refund transfer's hash ("" until sent).
	TxHash string `json:"txHash"`
	// InitiatedBy is customer (a request the merchant answers) or merchant.
	InitiatedBy RefundInitiator `json:"initiatedBy"`
	// Held is true when the refunded transaction's funds are held by the organization.
	Held bool `json:"held"`
	// PaidMinUnits is what the customer paid for the refunded transaction, in min units.
	PaidMinUnits string `json:"paidMinUnits"`
	// PaidUSD is the same in USD.
	PaidUSD float64 `json:"paidUsd"`
	// RefundPercent is the share of what the customer paid it sends back, in percent.
	RefundPercent float64 `json:"refundPercent"`
	// AmountMinUnits is what it sends back, in min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is the same in USD, at the refunded transaction's time.
	AmountUSD float64 `json:"amountUsd"`
	// CurrencyID is the currency paid, and sent back.
	CurrencyID uint64 `json:"currencyId"`
	// Metadata is the refund transfer's network fees and block, once approved; nil before.
	Metadata *TxMetadata `json:"metadata"`
	// Chain is the currency's chain.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the refund transfer on the chain's explorer, once sent.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Approval is the approval being confirmed, or its failed attempt (API reads only).
	Approval *RefundApproval `json:"approval,omitempty"`
	// ProductName is what the refunded transaction paid for (API reads only).
	ProductName string `json:"productName,omitempty"`
	// Customer names the refunded transaction's customer (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
}

// RefundApproval is a refund's approval in progress.
type RefundApproval struct {
	// Status is waitingConfirmation (sent, being confirmed) or created (its last attempt failed).
	Status CheckoutSessionStatusValue `json:"status"`
	// TxHash is the transfer being confirmed, once known.
	TxHash string `json:"txHash,omitempty"`
	// LastAttempt is why the last attempt failed.
	LastAttempt *Attempt `json:"lastAttempt,omitempty"`
}

// InitiateRefundParams starts a refund (Refunds.Initiate): a pending refund the merchant
// signs in the dashboard.
type InitiateRefundParams struct {
	// TxUUID is required: the payment (pay@…) or bill (sub-hist@…) to refund.
	TxUUID string `json:"txUuid"`
	// RefundPercent is the share of what the customer paid to send back: above 0, at most 100,
	// at most 2 decimals (nil = 100).
	RefundPercent *float64 `json:"refundPercent,omitempty"`
	// Reason is why the merchant refunds (optional, at most 500 characters).
	Reason string `json:"reason,omitempty"`
	// MerchantMessage is a note to the customer (optional, at most 500 characters).
	MerchantMessage string `json:"merchantMessage,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *InitiateRefundParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("txUuid", p.TxUUID) {
		v.txID("txUuid", p.TxUUID)
	}
	if p.RefundPercent != nil {
		v.percent("refundPercent", *p.RefundPercent, 100, true)
	}
	v.text("reason", p.Reason, 0, 500)
	v.text("merchantMessage", p.MerchantMessage, 0, 500)
	return v.err()
}

// RefundListParams filters Refunds.List and Refunds.ListInactive (Limit and Cursor page the
// latter; Refunds.List does not send them).
type RefundListParams struct {
	// Limit is ListInactive's page size (server default 10, max 50; 0 = default).
	Limit int
	// Cursor is ListInactive's previous page's NextCursor; "" for the first page.
	Cursor string
	// IncludeMembers adds the members' refunds: the API's default is true from the
	// organization's space (false from a member's, where true is refused). Not with UserUUID.
	IncludeMembers *bool
	// UserUUID reads one member's refunds (organization space only).
	UserUUID string
	// Held keeps only the refunds of held (true) or not held (false) transactions.
	Held *bool
}

// Validate checks the filters.
func (p *RefundListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.uuid("userUuid", p.UserUUID)
	v.exclusive("includeMembers", p.IncludeMembers != nil && *p.IncludeMembers, "userUuid", p.UserUUID != "")
	return v.err()
}

// values encodes the filters; paged adds Limit and Cursor (ListInactive).
func (p *RefundListParams) values(paged bool) url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	if paged {
		setPage(q, p.Limit, p.Cursor)
	}
	setBoolPtr(q, "includeMembers", p.IncludeMembers)
	setString(q, "userUuid", p.UserUUID)
	setBoolPtr(q, "held", p.Held)
	return q
}
