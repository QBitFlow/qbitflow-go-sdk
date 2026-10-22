package qbitflow

import (
	"fmt"
	"net/url"
	"time"
)

// Refund is a refund of a payment (pay@…) or a subscription bill (sub-hist@…): a customer's
// request the merchant approves or rejects, or a refund the merchant sent from the dashboard
// (recorded once its transfer confirmed: approved, nothing before). Refunds are read only here.
// It is also the data of the refund.requested, refund.approved and refund.rejected events.
type Refund struct {
	// UUID is the refund's id (refund@…).
	UUID string `json:"uuid"`
	// TxUUID is the refunded transaction: a payment (pay@…) or a bill (sub-hist@…).
	TxUUID string `json:"txUuid"`
	// Reason is why the refund: the customer's reason, or the merchant's when it started the
	// refund; "" (absent) when it gave none.
	Reason string `json:"reason,omitempty"`
	// Status is pending (a customer's request not answered yet), approved (its transfer
	// confirmed) or rejected. failed is declared but never set.
	Status RefundStatus `json:"status"`
	// CreatedAt is when it was requested (or prepared by the merchant).
	CreatedAt time.Time `json:"createdAt"`
	// MerchantMessage is the merchant's message to the customer (why it rejected the request, or
	// a note on a refund it started); "" (absent) when none.
	MerchantMessage string `json:"merchantMessage,omitempty"`
	// RespondedAt is when the refund was answered (rejected, or its transfer confirmed); nil while
	// pending.
	RespondedAt *time.Time `json:"respondedAt,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space the refunded transaction is in; "" for the
	// organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// TxHash is the refund transfer's transaction, once approved; "" before.
	TxHash string `json:"txHash,omitempty"`
	// InitiatedBy is customer (requested by the customer, the merchant answers) or merchant (sent
	// by the merchant without a request, from the dashboard: recorded once confirmed, approved).
	InitiatedBy RefundInitiator `json:"initiatedBy"`
	// Held is true when the refunded payment was paid to the organization's wallet, which
	// holds the seller's funds: the organization decides the refund and sends it from its wallet.
	// Once sent, it is a negative line of the seller's held funds.
	Held bool `json:"held"`
	// Paid is what the customer paid for the refunded transaction (its amount, fees included, and
	// the network fee they paid on top), in the currency's units: the base of RefundPercent.
	Paid string `json:"paid"`
	// PaidMinUnits is the same in the currency's smallest unit, a string of digits.
	PaidMinUnits string `json:"paidMinUnits"`
	// PaidUSD is the same in USD at the refunded transaction's time; "" (absent) when unknown (a
	// request made before it was recorded).
	PaidUSD string `json:"paidUsd,omitempty"`
	// RefundPercent is the share of Paid the refund sends back, a percent string ("100": all of
	// it). A customer requests "100"; the merchant sets it when it approves, or when it starts the
	// refund.
	RefundPercent string `json:"refundPercent"`
	// Amount is what the refund sends back: RefundPercent of Paid, rounded down to a whole min
	// unit, in the currency's units.
	Amount string `json:"amount"`
	// AmountMinUnits is the same in the currency's smallest unit, a string of digits.
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is the same in USD at the refunded transaction's time; "" (absent) when unknown.
	AmountUSD string `json:"amountUsd,omitempty"`
	// Currency is the currency paid, and sent back.
	Currency Currency `json:"currency"`
	// Settlement is the refund transfer's transaction (its block, what it cost), once approved;
	// nil before.
	Settlement *TransferSettlement `json:"settlement,omitempty"`
	// Chain is the currency's chain.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the refund transfer on the chain's explorer, once sent.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Approval is where the refund transfer stands while the request is pending and it was sent
	// (approved, or answered by the merchant's own refund): being confirmed, or its last attempt
	// failed (API reads only, never in webhooks).
	Approval *RefundApproval `json:"approval,omitempty"`
	// CustomerUUID is the refunded transaction's customer (API reads only).
	CustomerUUID string `json:"customerUuid,omitempty"`
	// ProductName is what the refunded transaction paid for (API reads only).
	ProductName string `json:"productName,omitempty"`
	// Customer names the refunded transaction's customer, even once deleted (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
}

// RefundApproval is where a pending request's refund transfer stands once sent.
type RefundApproval struct {
	// Status is waitingConfirmation (sent, being confirmed on-chain) or created (its last attempt
	// failed: LastAttempt; send it again from the dashboard).
	Status CheckoutSessionStatusValue `json:"status"`
	// TxHash is the transfer being confirmed, once known.
	TxHash string `json:"txHash,omitempty"`
	// LastAttempt is why the last attempt failed.
	LastAttempt *Attempt `json:"lastAttempt,omitempty"`
}

// maxRefundStatuses is the most statuses RefundListParams.Status may name.
const maxRefundStatuses = 5

// RefundListParams filters Refunds.List.
type RefundListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
	// Status keeps only the refunds in one of these statuses (at most 5 of pending, approved,
	// rejected, failed); empty: every status. RefundStatusPending lists the ones
	// awaiting an answer.
	Status []RefundStatus
	// TxUUID keeps only the refund of this transaction (pay@…, sub-hist@…, or the bare UUID).
	TxUUID string
	// Held keeps only the refunds of payments held for a member (true: the ones the organization
	// decides and sends), or only the others (false).
	Held *bool
	// IncludeMembers adds the members' refunds, each naming its member: the API's default is true
	// from the organization's space (false from a member's, where true is refused). Not with
	// UserUUID.
	IncludeMembers *bool
	// UserUUID reads one member's refunds (organization space only).
	UserUUID string
}

// Validate checks the filters.
func (p *RefundListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	if len(p.Status) > maxRefundStatuses {
		v.add("status", "must name at most %d statuses", maxRefundStatuses)
	}
	for i, status := range p.Status {
		field := fmt.Sprintf("status[%d]", i)
		if status == "" {
			v.add(field, "is required")
			continue
		}
		oneOf(&v, field, status, RefundStatusPending, RefundStatusApproved, RefundStatusRejected, RefundStatusFailed)
	}
	v.txID("txUuid", p.TxUUID)
	v.uuid("userUuid", p.UserUUID)
	v.exclusive("includeMembers", p.IncludeMembers != nil && *p.IncludeMembers, "userUuid", p.UserUUID != "")
	return v.err()
}

func (p *RefundListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	for _, status := range p.Status {
		q.Add("status", string(status))
	}
	setString(q, "txUuid", p.TxUUID)
	setBoolPtr(q, "held", p.Held)
	setBoolPtr(q, "includeMembers", p.IncludeMembers)
	setString(q, "userUuid", p.UserUUID)
	return q
}
