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
	// Version is the payload version (v2).
	Version WebhookPayloadVersion `json:"version"`
	// CreatedAt is when it happened.
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

// PaymentCompleted is payment.completed's data: the payment, confirmed.
type PaymentCompleted struct {
	Payment

	// ManagementPageLink is the payment's page for its customer.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionCreated is subscription.created's data (status active, or trial).
type SubscriptionCreated struct {
	Subscription

	// ManagementPageLink is the subscription's management page for its customer.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionBilled is subscription.billed's data: the paid bill.
type SubscriptionBilled struct {
	Bill

	// SubscriptionReference is the subscription's merchant reference, if it has one.
	SubscriptionReference string `json:"subscriptionReference,omitempty"`
	// SubscriptionStatus is the subscription's status once the bill is paid.
	SubscriptionStatus SubscriptionStatus `json:"subscriptionStatus"`
}

// SubscriptionStatusChanged is subscription.statusChanged's data.
type SubscriptionStatusChanged struct {
	Subscription

	// PreviousStatus is the status before the change.
	PreviousStatus SubscriptionStatus `json:"previousStatus"`
	// ManagementPageLink is the subscription's management page for its customer.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionActionRequiredChanged is subscription.actionRequiredChanged's data.
type SubscriptionActionRequiredChanged struct {
	Subscription

	// PreviousActionRequired is what its customer had to do before ("" = nothing).
	PreviousActionRequired ActionRequired `json:"previousActionRequired,omitempty"`
	// ManagementPageLink is the subscription's management page for its customer.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionBillingFailed is subscription.billingFailed's data: a bill attempt failed.
type SubscriptionBillingFailed struct {
	Subscription

	// Reason is insufficientBalance, allowanceExhausted, approvalRevoked, maxAmountExceeded or other.
	Reason BillingFailureReason `json:"reason"`
	// FailureCode is the charge's error code (e.g. insufficient_funds).
	FailureCode string `json:"failureCode,omitempty"`
	// BillUUID is the bill that failed (sub-hist@…: its id once paid).
	BillUUID string `json:"billUuid"`
	// AmountUSD is the bill, in USD (a decimal string, unlike SubscriptionUpcomingBill's).
	AmountUSD string `json:"amountUsd"`
	// Attempt is this bill's failed attempts so far, this one included.
	Attempt uint16 `json:"attempt"`
	// RemainingAttempts is the attempts left.
	RemainingAttempts uint16 `json:"remainingAttempts"`
	// NextAttemptAt is when the bill is tried again at the latest.
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	// ManagementPageLink is the subscription's management page, where its customer fixes it.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionUpcomingBill is subscription.upcomingBill's data: a bill (or a trial's end) is near.
type SubscriptionUpcomingBill struct {
	Subscription

	// BillingDate is when it is billed (for a trial: when the trial ends).
	BillingDate time.Time `json:"billingDate"`
	// AmountUSD is the bill, in USD (a number, unlike SubscriptionBillingFailed's).
	AmountUSD float64 `json:"amountUsd"`
	// TrialEnding is true when the trial ends then: the customer must confirm it to be billed.
	TrialEnding bool `json:"trialEnding"`
	// BalanceSufficient says whether the wallet holds the bill; nil for a trial.
	BalanceSufficient *bool `json:"balanceSufficient,omitempty"`
	// AllowanceSufficient says whether the remaining allowance covers the bill; nil for a trial.
	AllowanceSufficient *bool `json:"allowanceSufficient,omitempty"`
}

// MemberJoined is member.joined's data: the member, and the invitation they accepted.
type MemberJoined struct {
	Member

	// InvitationUUID is the invitation they accepted (Invitations.Create's).
	InvitationUUID string `json:"invitationUuid"`
}

// HeldFundsReleased is heldFunds.released's data: the transfer paying a member the funds the
// organization held for them, and the lines it settled.
type HeldFundsReleased struct {
	// UUID is the transfer's id (transfer@…).
	UUID string `json:"uuid"`
	// CreatedAt is when it was recorded.
	CreatedAt time.Time `json:"createdAt"`
	// From is the organization's wallet.
	From string `json:"from"`
	// To is the member's wallet.
	To string `json:"to"`
	// Amount is the amount, in USD.
	Amount float64 `json:"amount"`
	// AmountMinUnits is the amount in the token's min units (a decimal string).
	AmountMinUnits string `json:"amountMinUnits"`
	// CurrencyID is the currency paid in.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that currency.
	Currency *Currency `json:"currency"`
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
	// TxMetadata is the transaction's network fees and block.
	TxMetadata TxMetadata `json:"txMetadata"`
	// Ledgers are the held-funds lines it settled.
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

// SubscriptionCreated decodes a subscription.created event's data.
func (e *Event) SubscriptionCreated() (*SubscriptionCreated, error) {
	return decodeEventData[SubscriptionCreated](e, EventTypeSubscriptionCreated)
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

// RefundRequested decodes a refund.requested event's data (a pending refund).
func (e *Event) RefundRequested() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundRequested)
}

// RefundCompleted decodes a refund.completed event's data (an approved refund).
func (e *Event) RefundCompleted() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundCompleted)
}

// RefundDenied decodes a refund.denied event's data (a rejected refund).
func (e *Event) RefundDenied() (*Refund, error) {
	return decodeEventData[Refund](e, EventTypeRefundDenied)
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
	case EventTypeRefundCompleted:
		_, err = e.RefundCompleted()
	case EventTypeRefundDenied:
		_, err = e.RefundDenied()
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
