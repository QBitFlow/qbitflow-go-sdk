package qbitflow

import (
	"bytes"
	"encoding/json"
	"time"
)

// Event is a webhook event (a v2 delivery's body, or an event of the log). Data depends on
// Type: decode it with the typed accessor of its type (e.PaymentCompleted(), …) or with
// DecodeData. An event of a type this SDK does not know keeps its raw Data.
//
// Deliveries are at least once: deduplicate on ID, and answer 2xx fast (also to the events
// you ignore). UserUUID names the member whose space the event happened in: pass it to
// WithOnBehalfOf for follow-up reads.
type Event struct {
	// ID is the event's id (evt_…): the same on every retry, deduplicate on it.
	ID string `json:"id"`
	// Type is the event's type.
	Type EventType `json:"type"`
	// Version is the payload version: v2, the only one.
	Version WebhookPayloadVersion `json:"version"`
	// CreatedAt is when it happened (at most millisecond precision).
	CreatedAt time.Time `json:"createdAt"`
	// Test is true when it happened in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it happened in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// Data is the event's object, raw (see the typed accessors).
	Data json.RawMessage `json:"data"`
}

// DecodeData decodes the event's Data into a T, whatever the event's type. Absent or null
// data gives T's zero value; data that does not fit T is a *ValidationError.
func (e *Event) DecodeData[T any]() (T, error) {
	var zero T
	data := bytes.TrimSpace(e.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return zero, nil
	}
	v, err := decodeJSON[T](data)
	if err != nil {
		ve := newFieldError("data", "does not match the "+string(e.Type)+" event's data")
		ve.Err = err
		return zero, ve
	}
	return v, nil
}

// decodeEventData is the typed accessors' body: it checks the event's type, then decodes Data.
func decodeEventData[T any](e *Event, want EventType) (*T, error) {
	if e.Type != want {
		return nil, newFieldError("type", "is "+string(e.Type)+", not "+string(want))
	}
	v, err := e.DecodeData[T]()
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// PaymentCompleted is payment.completed's data: the payment, confirmed, as Payments.Get returns
// it to its owner (its refund state excepted: API reads only), and its customer page. Fulfil the
// order: UUID is the checkout session's pay@…, Reference and Metadata yours.
type PaymentCompleted struct {
	Payment

	// ManagementPageLink is the payment's page for its customer.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionBilled is subscription.billed's data: the bill paid, and its subscription as the
// bill leaves it. Access is paid until Bill.PeriodEnd, unless Subscription.Status is cancelled
// (a charge that landed just after a cancellation: record it, grant nothing).
type SubscriptionBilled struct {
	// Bill is the bill paid (as Subscriptions.GetBill returns it): its currency, settlement, and
	// the period it pays, PeriodStart → PeriodEnd.
	Bill Bill `json:"bill"`
	// Subscription is the subscription once the bill is paid: active (stopped, paused or
	// cancelled if that happened while the charge was on its way), its next bill, its allowance,
	// its reference and its metadata.
	Subscription Subscription `json:"subscription"`
}

// SubscriptionStatusChanged is subscription.statusChanged's data: the subscription with its new
// Status, and the one before. Every status change sends it, once.
type SubscriptionStatusChanged struct {
	Subscription

	// PreviousStatus is the status before the change.
	PreviousStatus SubscriptionStatus `json:"previousStatus"`
}

// SubscriptionActionRequiredChanged is subscription.actionRequiredChanged's data: the
// subscription with its new ActionRequired ("" = nothing to do anymore), and the one before.
// Its ManagementPageLink is where the customer acts.
type SubscriptionActionRequiredChanged struct {
	Subscription

	// PreviousActionRequired is what its customer had to do before ("" = nothing).
	PreviousActionRequired ActionRequired `json:"previousActionRequired,omitempty"`
}

// SubscriptionBillingFailed is subscription.billingFailed's data: the subscription as a failed
// bill attempt leaves it, and the failure. Its ManagementPageLink is where the customer fixes
// it. Sent once per failed attempt (pastDue), and once when a bill above the customer's maximum
// starts waiting for a new one (maxAmountExceeded: no attempt spent).
type SubscriptionBillingFailed struct {
	Subscription

	// Reason is what the customer does: insufficientBalance (add funds to the wallet),
	// allowanceExhausted (top up the allowance), approvalRevoked (approve the token again),
	// maxAmountExceeded (sign a new maximum) or other (the network refused the charge).
	Reason BillingFailureReason `json:"reason"`
	// FailureCode is the charge's error code (e.g. insufficient_funds, insufficient_allowance).
	FailureCode string `json:"failureCode,omitempty"`
	// BillUUID is the bill that failed: the sub-hist@… its subscription.billed has once it is paid.
	BillUUID string `json:"billUuid"`
	// AmountUSD is the bill, in USD, a decimal string: its PriceUSD plus its Fees lines.
	AmountUSD string `json:"amountUsd"`
	// Attempt is this bill's failed attempts so far, this one included (maxAmountExceeded spends
	// none).
	Attempt uint16 `json:"attempt"`
	// RemainingAttempts is the attempts left (retried 1, 3, 5 and 7 days after the first failed
	// attempt). 0: this was the last; its cancellation follows (subscription.statusChanged).
	RemainingAttempts uint16 `json:"remainingAttempts"`
	// NextAttemptAt is when the bill is tried again at the latest (maxAmountExceeded: when it
	// stops waiting for a new maximum); nil after the last attempt.
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
}

// SubscriptionUpcomingBill is subscription.upcomingBill's data: a bill (or a trial's end) is
// near, sent once per bill some days before (3 by default). Remind the customer when
// BalanceSufficient or AllowanceSufficient is false, or when TrialEnding.
type SubscriptionUpcomingBill struct {
	Subscription

	// BillingAt is when it is billed (for a trial: when the trial ends).
	BillingAt time.Time `json:"billingAt"`
	// AmountUSD is what the bill charges, in USD, a decimal string: its PriceUSD plus its Fees
	// lines.
	AmountUSD string `json:"amountUsd"`
	// TrialEnding is true when the trial ends then: the customer confirms the subscription (its
	// ManagementPageLink) to be billed.
	TrialEnding bool `json:"trialEnding"`
	// BalanceSufficient says whether the wallet holds the bill at today's price; nil for a trial,
	// or when it couldn't be checked.
	BalanceSufficient *bool `json:"balanceSufficient,omitempty"`
	// AllowanceSufficient says whether the remaining allowance covers the bill; nil for a trial,
	// or without a price.
	AllowanceSufficient *bool `json:"allowanceSufficient,omitempty"`
}

// MemberJoined is member.joined's data: the member, in the mode they joined, and the invitation
// they accepted. Store UserUUID for On-Behalf-Of.
type MemberJoined struct {
	Member

	// InvitationUUID is the invitation they accepted (Invitations.Create's UUID): match the event
	// to it.
	InvitationUUID string `json:"invitationUuid"`
}

// HeldFundsReleased is heldFunds.released's data: the transfer paying a member the funds the
// organization held for them (in the member's space: the envelope's UserUUID), and the lines it
// settled.
type HeldFundsReleased struct {
	// UUID is the transfer's id (transfer@…).
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded.
	CreatedAt time.Time `json:"createdAt"`
	// From is the organization's wallet.
	From string `json:"from"`
	// To is the member's wallet.
	To string `json:"to"`
	// Amount is what the member received, in the currency's units, exact: the sum of the lines'
	// Owed.
	Amount string `json:"amount"`
	// AmountMinUnits is the same in the currency's smallest unit.
	AmountMinUnits string `json:"amountMinUnits"`
	// AmountUSD is the same in USD, at the release's price.
	AmountUSD string `json:"amountUsd"`
	// Currency is the currency paid in.
	Currency Currency `json:"currency"`
	// TxHash is the transaction's hash.
	TxHash string `json:"txHash"`
	// Chain is the chain.
	Chain Chain `json:"chain,omitempty"`
	// ExplorerURL is the transaction on the chain's block explorer.
	ExplorerURL string `json:"explorerUrl,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member paid.
	UserUUID string `json:"userUuid,omitempty"`
	// Received is true once the recipient received it.
	Received bool `json:"received"`
	// Type is heldFundsRelease.
	Type TransferType `json:"type"`
	// Settlement is the transfer's transaction: its block and what it cost its sender.
	Settlement TransferSettlement `json:"settlement"`
	// Ledgers are the lines it settled, oldest first: the payments and bills it paid (positive)
	// and the refunds of them it took off (negative, type refund), each with what it counted for
	// (Owed, OwedMinUnits, OwedUSD).
	Ledgers []LedgerEntry `json:"ledgers"`
}

// CheckoutExpired is checkout.expired's data: the expired session, a payment's or a
// subscription's (chosen by its txType; an unknown txType decodes as a payment's, and the
// event's Data keeps the raw object).
type CheckoutExpired struct {
	// Payment is set for a payment session (txType payment, or unknown).
	Payment *PaymentSessionData
	// Subscription is set for a subscription session (txType createSubscription).
	Subscription *SubscriptionSessionData
}

// WebhookTest is webhook.test's data (the dashboard's test delivery).
type WebhookTest struct {
	// EndpointUUID is the endpoint tested.
	EndpointUUID string `json:"endpointUuid"`
	// Message is the test's message.
	Message string `json:"message"`
}

// PaymentCompleted decodes a payment.completed event's data (a *ValidationError for another type).
func (e *Event) PaymentCompleted() (*PaymentCompleted, error) {
	return decodeEventData[PaymentCompleted](e, EventTypePaymentCompleted)
}

// SubscriptionCreated decodes a subscription.created event's data: the subscription (status
// active, or trial), as Subscriptions.Get returns it, its ManagementPageLink included.
func (e *Event) SubscriptionCreated() (*Subscription, error) {
	return decodeEventData[Subscription](e, EventTypeSubscriptionCreated)
}

// SubscriptionBilled decodes a subscription.billed event's data.
func (e *Event) SubscriptionBilled() (*SubscriptionBilled, error) {
	return decodeEventData[SubscriptionBilled](e, EventTypeSubscriptionBilled)
}

// SubscriptionStatusChanged decodes a subscription.statusChanged event's data.
func (e *Event) SubscriptionStatusChanged() (*SubscriptionStatusChanged, error) {
	return decodeEventData[SubscriptionStatusChanged](e, EventTypeSubscriptionStatusChanged)
}

// SubscriptionActionRequiredChanged decodes a subscription.actionRequiredChanged event's data.
func (e *Event) SubscriptionActionRequiredChanged() (*SubscriptionActionRequiredChanged, error) {
	return decodeEventData[SubscriptionActionRequiredChanged](e, EventTypeSubscriptionActionRequiredChanged)
}

// SubscriptionBillingFailed decodes a subscription.billingFailed event's data.
func (e *Event) SubscriptionBillingFailed() (*SubscriptionBillingFailed, error) {
	return decodeEventData[SubscriptionBillingFailed](e, EventTypeSubscriptionBillingFailed)
}

// SubscriptionUpcomingBill decodes a subscription.upcomingBill event's data.
func (e *Event) SubscriptionUpcomingBill() (*SubscriptionUpcomingBill, error) {
	return decodeEventData[SubscriptionUpcomingBill](e, EventTypeSubscriptionUpcomingBill)
}

// RefundRequested decodes a refund.requested event's data: a new pending refund, a customer's
// request (InitiatedBy customer) or a merchant's own (Refunds.Create).
func (e *Event) RefundRequested() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundRequested)
}

// RefundApproved decodes a refund.approved event's data (was refund.completed): the refund,
// approved, its transfer confirmed (TxHash, Settlement.Transaction); RefundPercent and Amount
// are the share actually paid back.
func (e *Event) RefundApproved() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundApproved)
}

// RefundRejected decodes a refund.rejected event's data (was refund.denied): a customer's
// request rejected by whoever decides it. A decision, final.
func (e *Event) RefundRejected() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundRejected)
}

// MemberJoined decodes a member.joined event's data.
func (e *Event) MemberJoined() (*MemberJoined, error) {
	return decodeEventData[MemberJoined](e, EventTypeMemberJoined)
}

// MemberRemoved decodes a member.removed event's data.
func (e *Event) MemberRemoved() (*Member, error) {
	return decodeEventData[Member](e, EventTypeMemberRemoved)
}

// HeldFundsReleased decodes a heldFunds.released event's data.
func (e *Event) HeldFundsReleased() (*HeldFundsReleased, error) {
	return decodeEventData[HeldFundsReleased](e, EventTypeHeldFundsReleased)
}

// CheckoutExpired decodes a checkout.expired event's data, a payment's or a subscription's
// session by its txType.
func (e *Event) CheckoutExpired() (*CheckoutExpired, error) {
	probe, err := decodeEventData[struct {
		TxType TransactionType `json:"txType"`
	}](e, EventTypeCheckoutExpired)
	if err != nil {
		return nil, err
	}
	if probe.TxType == TransactionTypeCreateSubscription {
		sub, err := e.DecodeData[SubscriptionSessionData]()
		if err != nil {
			return nil, err
		}
		return &CheckoutExpired{Subscription: &sub}, nil
	}
	pay, err := e.DecodeData[PaymentSessionData]()
	if err != nil {
		return nil, err
	}
	return &CheckoutExpired{Payment: &pay}, nil
}

// WebhookTest decodes a webhook.test event's data.
func (e *Event) WebhookTest() (*WebhookTest, error) {
	return decodeEventData[WebhookTest](e, EventTypeWebhookTest)
}

// validateEventData decodes the data of a known event type, so data that does not match its type is
// refused whoever handles it; an unknown type is never checked. Used by WebhookRouter.
func validateEventData(e *Event) error {
	var err error
	switch e.Type {
	case EventTypePaymentCompleted:
		_, err = e.PaymentCompleted()
	case EventTypeSubscriptionCreated:
		_, err = e.SubscriptionCreated()
	case EventTypeSubscriptionBilled:
		_, err = e.SubscriptionBilled()
	case EventTypeSubscriptionStatusChanged:
		_, err = e.SubscriptionStatusChanged()
	case EventTypeSubscriptionActionRequiredChanged:
		_, err = e.SubscriptionActionRequiredChanged()
	case EventTypeSubscriptionBillingFailed:
		_, err = e.SubscriptionBillingFailed()
	case EventTypeSubscriptionUpcomingBill:
		_, err = e.SubscriptionUpcomingBill()
	case EventTypeRefundRequested:
		_, err = e.RefundRequested()
	case EventTypeRefundApproved:
		_, err = e.RefundApproved()
	case EventTypeRefundRejected:
		_, err = e.RefundRejected()
	case EventTypeMemberJoined:
		_, err = e.MemberJoined()
	case EventTypeMemberRemoved:
		_, err = e.MemberRemoved()
	case EventTypeHeldFundsReleased:
		_, err = e.HeldFundsReleased()
	case EventTypeCheckoutExpired:
		_, err = e.CheckoutExpired()
	case EventTypeWebhookTest:
		_, err = e.WebhookTest()
	}
	return err
}
