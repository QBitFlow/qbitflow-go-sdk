package qbitflow

import (
	"net/url"
	"time"
)

// Subscription is a subscription. Grant access while now < CurrentPeriodEnd.
type Subscription struct {
	// UUID is the subscription's id (sub@…), its checkout session's for life.
	UUID string `json:"uuid"`
	// Reference is the merchant's reference, set when creating its checkout.
	Reference string `json:"reference,omitempty"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when it last changed.
	UpdatedAt time.Time `json:"updatedAt"`
	// From is the subscriber's wallet.
	From string `json:"from"`
	// To is the wallet that receives the bills.
	To string `json:"to"`
	// ProductUUID is the subscription's product.
	ProductUUID string `json:"productUuid"`
	// SubscriptionHash is the transaction that created it on-chain ("" on a trial until confirmed).
	SubscriptionHash string `json:"subscriptionHash"`
	// CurrencyID is the currency billed.
	CurrencyID uint64 `json:"currencyId"`
	// Currency is that currency.
	Currency *Currency `json:"currency"`
	// Frequency is the billing interval.
	Frequency Duration `json:"frequency"`
	// Allowance is what remains of the allowance the subscriber granted, in the token's min
	// units (a decimal string).
	Allowance string `json:"allowance"`
	// Status is trial, trialExpired, active, pastDue, paused, stopped or cancelled.
	Status SubscriptionStatus `json:"status"`
	// ActionRequired is what its customer must do (topUpAllowance, raiseMaximum, confirmTrial);
	// "" for nothing.
	ActionRequired ActionRequired `json:"actionRequired,omitempty"`
	// CurrentPeriodEnd is the end of the period paid for (or of the trial): grant access while
	// now < CurrentPeriodEnd.
	CurrentPeriodEnd *time.Time `json:"currentPeriodEnd,omitempty"`
	// PriceUSD is the price per period the customer subscribed at, in USD (a decimal string;
	// "0" for old v1 subscriptions). Every bill charges it plus Fees.
	PriceUSD string `json:"priceUsd"`
	// Fees is what its checkout added to the price, line by line, as the customer saw them (a
	// tax, shipping, the processing fee when the customer pays it), locked at signup with the
	// price: every bill charges PriceUSD plus these lines. Empty without any.
	Fees []FeeLine `json:"fees,omitempty"`
	// MaxAmountPerPeriod is the customer's maximum per period, in min units (a decimal string);
	// "" on trials and old subscriptions.
	MaxAmountPerPeriod string `json:"maxAmountPerPeriod,omitempty"`
	// LastBillingDate is the date of the last bill.
	LastBillingDate time.Time `json:"lastBillingDate"`
	// NextBillingDate is when the next bill is due (a trial: when it ends; stopped: when it is
	// cancelled); nil once cancelled.
	NextBillingDate *time.Time `json:"nextBillingDate,omitempty"`
	// CancelledAt is when it was cancelled.
	CancelledAt *time.Time `json:"cancelledAt,omitempty"`
	// CancellationReason is why it stopped or was cancelled.
	CancellationReason CancellationReason `json:"cancellationReason,omitempty"`
	// MinimumCancellationDate is the earliest date it can be cancelled (MinPeriods).
	MinimumCancellationDate *time.Time `json:"minimumCancellationDate,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
	// CustomerUUID is the customer, when QBitFlow has one.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is the merchant's reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`
	// Customer names the customer (API reads only).
	Customer *CustomerSummary `json:"customer,omitempty"`
	// Dunning is the failing bill's retries, while past due (API reads only).
	Dunning *DunningStatus `json:"dunning,omitempty"`
}

// HasAccess reports whether the subscription grants access at at: CurrentPeriodEnd is set and
// at is before it, whatever the status (a cancelled subscription keeps access to the end of the
// period paid for). The zero time means now; a nil subscription has no access.
func (s *Subscription) HasAccess(at time.Time) bool {
	if s == nil || s.CurrentPeriodEnd == nil {
		return false
	}
	if at.IsZero() {
		at = time.Now()
	}
	return at.Before(*s.CurrentPeriodEnd)
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

// BillingState is a test billing run's state (Subscriptions.ExecuteTestBilling).
type BillingState struct {
	// BillUUID is the bill's id, bare (its history entry is sub-hist@<BillUUID>).
	BillUUID string `json:"billUuid"`
	// Stage is running, waiting, pending or done.
	Stage BillingStage `json:"stage"`
	// Outcome is how it ended, once done.
	Outcome BillingOutcome `json:"outcome,omitempty"`
	// Attempts is the failed attempts so far.
	Attempts uint16 `json:"attempts"`
	// NextAttempt is when it tries again at the latest, while waiting or pending.
	NextAttempt *time.Time `json:"nextAttempt,omitempty"`
	// TxHash is the charge's transaction, once paid.
	TxHash string `json:"txHash,omitempty"`
	// FailureCode is the last failure's code (e.g. insufficient_allowance).
	FailureCode string `json:"failureCode,omitempty"`
}

// SubscriptionListParams filters Subscriptions.List.
type SubscriptionListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
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
	// Status keeps only those in this status.
	Status SubscriptionStatus
	// Reference keeps only the one with this reference.
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
	p.filters().validate(&v)
	oneOf(&v, "status", p.Status, SubscriptionStatusTrial, SubscriptionStatusTrialExpired, SubscriptionStatusActive,
		SubscriptionStatusPastDue, SubscriptionStatusPaused, SubscriptionStatusStopped, SubscriptionStatusCancelled)
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
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *BillListParams) Validate() error { return nil }

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
	// Immediate cancels now (nil or true, the default); false stops it at the end of the
	// current period.
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
