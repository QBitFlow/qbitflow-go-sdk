package qbitflow

import (
	"net/url"
	"time"
)

// Subscription is a subscription (sub@…). Grant access while now < AccessUntil, whatever the
// status (HasAccess): the status says why, ActionRequired what its customer must do.
type Subscription struct {
	// UUID is the subscription's id (sub@…), its checkout session's for life.
	UUID string `json:"uuid"`
	// Reference is your reference for the subscription (an order or customer id), set when
	// creating its checkout.
	Reference string `json:"reference,omitempty"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is its last change (a bill, a status, its metadata).
	UpdatedAt time.Time `json:"updatedAt"`
	// From is the subscriber's wallet, the one it bills; "" (absent) when unknown (a trial not
	// confirmed yet may have none).
	From string `json:"from,omitempty"`
	// To is the wallet its bills pay: the merchant's (or, while the organization holds a member's
	// funds, the organization's).
	To string `json:"to"`
	// ProductUUID is its product (Products.Get); "" (absent) without one.
	ProductUUID string `json:"productUuid,omitempty"`
	// SubscriptionHash is the transaction that created it on-chain; "" (absent) until then (a
	// trial, until its customer confirms it).
	SubscriptionHash string `json:"subscriptionHash,omitempty"`
	// Currency is the currency (token) billed.
	Currency Currency `json:"currency"`
	// Frequency is the billing interval.
	Frequency Duration `json:"frequency"`
	// Status is trial, trialExpired, active, pastDue, paused, stopped or cancelled. Every change
	// sends subscription.statusChanged.
	Status SubscriptionStatus `json:"status"`
	// ActionRequired is what its customer must do (topUpAllowance, raiseMaximum, confirmTrial);
	// "" (absent) for nothing. Every change sends subscription.actionRequiredChanged.
	ActionRequired ActionRequired `json:"actionRequired,omitempty"`
	// CurrentPeriodEnd is the paid-through date, to display ("paid through"). A trial: its end;
	// active, paused, stopped: the end of the period paid for; pastDue: the start of the unpaid
	// period (past); cancelled: when it was cancelled, or the end of its last paid period if that
	// came first. For access, read AccessUntil: an active subscription is renewed a few minutes
	// after this date. nil when absent.
	CurrentPeriodEnd *time.Time `json:"currentPeriodEnd,omitempty"`
	// AccessUntil is when access ends: grant it while now < AccessUntil, whatever the status
	// (HasAccess). An active subscription: CurrentPeriodEnd plus 1 hour, since its renewal is
	// billed within minutes of its due date (a subscriber whose bill is on its way keeps access;
	// a failed bill makes it pastDue). Every other status: CurrentPeriodEnd. nil when absent
	// (a subscription stored before the API sent it: HasAccess then reads CurrentPeriodEnd).
	AccessUntil *time.Time `json:"accessUntil,omitempty"`
	// PriceUSD is the price per period the customer subscribed at, in USD (a decimal string,
	// "9.99"), locked at signup (a free trial's too); never "0". Every bill charges it plus Fees,
	// converted to the token at the bill's price.
	PriceUSD string `json:"priceUsd"`
	// Fees is what its checkout added to the price, line by line, as the customer saw them (a
	// tax, shipping, the processing fee when the customer pays it), locked at signup with the
	// price: every bill charges PriceUSD plus these lines. Empty without any.
	Fees []FeeLine `json:"fees"`
	// Allowance is what the subscriber's wallet still lets QBitFlow bill, and the most one bill
	// may charge, at today's price; nil (absent) during a free trial (nothing signed yet).
	Allowance *Allowance `json:"allowance,omitempty"`
	// LastBilledAt is when it was last billed; nil (absent) before its first bill (a trial).
	LastBilledAt *time.Time `json:"lastBilledAt,omitempty"`
	// NextBillingAt is when its next bill is due (a trial: when it ends; stopped: when it is
	// cancelled); nil (absent) once cancelled.
	NextBillingAt *time.Time `json:"nextBillingAt,omitempty"`
	// CancelledAt is when it was cancelled; nil until then.
	CancelledAt *time.Time `json:"cancelledAt,omitempty"`
	// CancellationReason is why it stopped or was cancelled; "" while it runs.
	CancellationReason CancellationReason `json:"cancellationReason,omitempty"`
	// CancellableAt is the earliest its customer may cancel it (the checkout's minPeriods); nil
	// when they may cancel it any time.
	CancellableAt *time.Time `json:"cancellableAt,omitempty"`
	// Metadata is your own data: its checkout session's, copied when the subscription was
	// created, then changed with Subscriptions.Update (merged). Empty without any.
	Metadata map[string]string `json:"metadata"`
	// ManagementPageLink is the subscription's page for its customer: where they top up its
	// allowance, raise its maximum, pause, resume or cancel it, and confirm a trial. Share it with
	// them.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer, as the checkout was given it.
	CustomerReference string `json:"customerReference,omitempty"`
	// Customer names the customer, even once deleted (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Dunning is where its failing bill's retries stand (API reads only: webhooks tell each
	// failure with subscription.billingFailed); nil while no bill is failing.
	Dunning *DunningStatus `json:"dunning,omitempty"`
}

// Allowance is what a subscriber's wallet still lets QBitFlow bill for their subscription (the
// allowance left on-chain), in the subscription's token, how many bills that covers at today's
// price, and the most one bill may charge, as they signed it. Amount, AmountMinUnits and
// AmountUSD (its value at today's price; "" without one) come from TokenAmount.
type Allowance struct {
	TokenAmount

	// RemainingPeriods is how many whole bills the allowance still covers at today's price
	// (rounded down, before each bill's network fee). 0 is a value: the customer must top it up
	// (ActionRequired topUpAllowance) before the next bill. nil without a price.
	RemainingPeriods *uint64 `json:"remainingPeriods,omitempty"`
	// MaxPerPeriod is the most one bill may charge, as the subscriber signed it, in the token's
	// units: a bill above it waits for a new maximum (ActionRequired raiseMaximum). "" when none
	// is recorded.
	MaxPerPeriod string `json:"maxPerPeriod,omitempty"`
	// MaxPerPeriodMinUnits is the same as an on-chain integer.
	MaxPerPeriodMinUnits string `json:"maxPerPeriodMinUnits,omitempty"`
}

// HasAccess reports whether the subscription grants access at at: at is before AccessUntil,
// whatever the status (a cancelled subscription keeps access to the end of the period paid
// for; an active one an hour past CurrentPeriodEnd, while its renewal is billed). Without
// AccessUntil (a subscription stored before the API sent it), it reads CurrentPeriodEnd. False
// when neither is set. The zero time means now; a nil subscription has no access.
func (s *Subscription) HasAccess(at time.Time) bool {
	if s == nil {
		return false
	}
	until := s.AccessUntil
	if until == nil {
		until = s.CurrentPeriodEnd
	}
	if until == nil {
		return false
	}
	if at.IsZero() {
		at = time.Now()
	}
	return at.Before(*until)
}

// DunningStatus is a past-due subscription's retries.
type DunningStatus struct {
	// FailedAttempts is the bill's failed attempts so far.
	FailedAttempts uint16 `json:"failedAttempts"`
	// RemainingAttempts is the attempts left: after the last one fails, it is cancelled.
	RemainingAttempts uint16 `json:"remainingAttempts"`
	// FailingSince is the bill's first failed attempt.
	FailingSince *time.Time `json:"failingSince,omitempty"`
	// NextAttemptAt is when the bill is tried again at the latest.
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	// AwaitingMaximumSince is when it started waiting for the customer to raise their maximum.
	AwaitingMaximumSince *time.Time `json:"awaitingMaximumSince,omitempty"`
}

// SubscriptionCancellation is Subscriptions.Cancel's answer.
type SubscriptionCancellation struct {
	// Subscription is the subscription as the API answered it.
	Subscription Subscription `json:"subscription"`
	// Pending is true when the API answered 202: the cancellation is still confirming on-chain
	// and Subscription's status is not updated yet.
	Pending bool `json:"pending"`
}

// TestBill is where a test bill stands once its attempt settled (Subscriptions.TestBill): done
// (its Outcome), waiting for its next attempt after a failure (FailureCode, NextAttemptAt), or
// still running after 2 minutes. Its events are a live bill's (subscription.billed,
// subscription.billingFailed, subscription.statusChanged).
type TestBill struct {
	// BillUUID is the bill (sub-hist@…): the id its subscription.billed and Subscriptions.GetBill
	// carry once it is paid.
	BillUUID string `json:"billUuid"`
	// Stage is running (being charged), waiting (for its next attempt, its due date or a trial's
	// grace period), pending (over the customer's maximum: waiting for a new one) or done.
	Stage BillingStage `json:"stage"`
	// Outcome is how it ended, once done: paid, notDue, skipped, cancelled or paused; "" before.
	Outcome BillingOutcome `json:"outcome,omitempty"`
	// Attempts is its failed attempts so far.
	Attempts uint16 `json:"attempts"`
	// NextAttemptAt is when it is tried again at the latest, while waiting or pending; nil
	// otherwise.
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	// TxHash is the charge's transaction, once paid.
	TxHash string `json:"txHash,omitempty"`
	// FailureCode is its last failure's code (e.g. insufficient_allowance).
	FailureCode string `json:"failureCode,omitempty"`
}

// UpdateSubscriptionParams changes a subscription (Subscriptions.Update): its metadata only. Its
// price and terms are locked at signup; its status changes through its actions (Cancel, its
// customer's pause and resume).
type UpdateSubscriptionParams struct {
	// Metadata is merged into the subscription's: a key set to a value sets it, a key set to ""
	// deletes it, keys not named are kept (nil: unchanged). At most 50 keys once merged; each key
	// 1 to 40 of A-Z a-z 0-9 _ - . :, each value at most 500 characters without control or
	// bidirectional characters.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the params: the metadata sent (the 50-key cap on the merged
// result is the API's).
func (p *UpdateSubscriptionParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.metadata("metadata", p.Metadata)
	return v.err()
}

// SubscriptionListParams filters Subscriptions.List.
type SubscriptionListParams struct {
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
	// Status keeps only those in this status (cancelled ones are listed unless filtered out). Not
	// checked against the known statuses: a newer one is sent as is.
	Status SubscriptionStatus
	// Reference keeps only the one with this reference (a page of 0 or 1 subscription;
	// Subscriptions.GetByReference reads it).
	Reference string
}

func (p *SubscriptionListParams) filters() listFilters {
	return listFilters{p.CustomerUUID, p.ProductUUID, p.CreatedAfter, p.CreatedBefore, p.IncludeMembers, p.UserUUID}
}

// Validate checks the filters.
func (p *SubscriptionListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	p.filters().validate(&v)
	v.reference("reference", p.Reference)
	return v.err()
}

func (p *SubscriptionListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	p.filters().encode(q)
	setString(q, "status", string(p.Status))
	setString(q, "reference", p.Reference)
	return q
}

// BillListParams pages Subscriptions.ListBills.
type BillListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
}

// Validate checks the params.
func (p *BillListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	return v.err()
}

func (p *BillListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	return q
}

// CancelSubscriptionParams are Subscriptions.Cancel's options.
type CancelSubscriptionParams struct {
	// Immediate cancels now (nil or true, the default: status cancelled); false stops it now
	// (status stopped, never billed again) and cancels it when its paid period ends
	// (NextBillingAt). A trial is cancelled immediately only (false: 400).
	Immediate *bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *CancelSubscriptionParams) Validate() error { return nil }

func (p *CancelSubscriptionParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBoolPtr(q, "immediate", p.Immediate)
	return q
}
