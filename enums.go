package qbitflow

// Every enum is an open string type: a value the SDK does not know yet is kept as is, never
// rejected. Compare against the constants.

// SubscriptionStatus is a subscription's lifecycle status.
type SubscriptionStatus string

// Subscription statuses.
const (
	SubscriptionStatusTrial        SubscriptionStatus = "trial"        // In its free trial
	SubscriptionStatusTrialExpired SubscriptionStatus = "trialExpired" // The trial ended without its customer confirming it
	SubscriptionStatusActive       SubscriptionStatus = "active"       // Billed normally
	SubscriptionStatusPastDue      SubscriptionStatus = "pastDue"      // A bill failed: retried (dunning) until it is paid or the subscription cancelled
	SubscriptionStatusPaused       SubscriptionStatus = "paused"       // Paused by its customer: not billed until resumed
	SubscriptionStatusStopped      SubscriptionStatus = "stopped"      // Stopped: cancelled at the end of the current period
	SubscriptionStatusCancelled    SubscriptionStatus = "cancelled"    // Cancelled. Final
)

// ActionRequired is what a subscription's customer must do ("" = nothing).
type ActionRequired string

// Actions required from a subscription's customer.
const (
	ActionRequiredTopUpAllowance ActionRequired = "topUpAllowance" // The allowance is too low for the next bill
	ActionRequiredRaiseMaximum   ActionRequired = "raiseMaximum"   // The bill is above the customer's maximum per period
	ActionRequiredConfirmTrial   ActionRequired = "confirmTrial"   // The trial must be confirmed to be billed
)

// CancellationReason says why a subscription stopped or was cancelled.
type CancellationReason string

// Cancellation reasons.
const (
	CancellationReasonCustomer          CancellationReason = "customer"
	CancellationReasonMerchant          CancellationReason = "merchant"
	CancellationReasonBillingFailed     CancellationReason = "billingFailed"
	CancellationReasonTrialNotConverted CancellationReason = "trialNotConverted"
	CancellationReasonMerchantClosed    CancellationReason = "merchantClosed"
	CancellationReasonInactiveOnChain   CancellationReason = "inactiveOnChain"
)

// BillingFailureReason says why a bill failed (subscription.billingFailed).
type BillingFailureReason string

// Billing failure reasons.
const (
	BillingFailureReasonInsufficientBalance BillingFailureReason = "insufficientBalance"
	BillingFailureReasonAllowanceExhausted  BillingFailureReason = "allowanceExhausted"
	BillingFailureReasonApprovalRevoked     BillingFailureReason = "approvalRevoked"
	BillingFailureReasonMaxAmountExceeded   BillingFailureReason = "maxAmountExceeded"
	BillingFailureReasonOther               BillingFailureReason = "other"
)

// BillingStage is where a bill stands (TestBill.Stage).
type BillingStage string

// Billing stages.
const (
	BillingStageRunning BillingStage = "running" // Being planned, charged or recorded
	BillingStageWaiting BillingStage = "waiting" // Waiting for its next attempt, its due date, or the end of a trial's grace period
	BillingStagePending BillingStage = "pending" // Over the customer's maximum: waiting for their new one
	BillingStageDone    BillingStage = "done"
)

// BillingOutcome is how a bill ended (TestBill.Outcome).
type BillingOutcome string

// Billing outcomes.
const (
	BillingOutcomePaid      BillingOutcome = "paid"
	BillingOutcomeNotDue    BillingOutcome = "notDue"    // Nothing to bill: paid already, cancelled, or gone
	BillingOutcomeSkipped   BillingOutcome = "skipped"   // Below the minimum amount: the period moved on without a charge
	BillingOutcomeCancelled BillingOutcome = "cancelled" // The subscription was cancelled (stopped, its trial over, its merchant closed, inactive on-chain, or its last attempt failed)
	BillingOutcomePaused    BillingOutcome = "paused"    // Its customer paused the subscription: not charged
)

// CheckoutSessionStatusValue is a checkout session's (or a refund approval's) status.
type CheckoutSessionStatusValue string

// Checkout session statuses.
const (
	CheckoutSessionStatusCreated             CheckoutSessionStatusValue = "created"             // Waiting for the customer: nothing sent yet, or the last attempt failed (lastAttempt)
	CheckoutSessionStatusWaitingConfirmation CheckoutSessionStatusValue = "waitingConfirmation" // A transaction was sent: waiting for the network and its record
	CheckoutSessionStatusCompleted           CheckoutSessionStatusValue = "completed"           // Confirmed and recorded. Final
	CheckoutSessionStatusExpired             CheckoutSessionStatusValue = "expired"             // Expired unpaid (at its expiresAt, or expired by its merchant). Final
)

// FeeLineType tells who added a line to a checkout's price (FeeLine.Type).
type FeeLineType string

// Fee line types.
const (
	FeeLineTypeCustom        FeeLineType = "custom"        // The merchant's own line: a tax, shipping, a service fee
	FeeLineTypeProcessingFee FeeLineType = "processingFee" // QBitFlow's processing fee, which the merchant has the customer pay (computed by QBitFlow)
)

// AttemptStatus is a failed attempt's status.
type AttemptStatus string

// Attempt statuses.
const (
	AttemptStatusFailed AttemptStatus = "failed" // Not sent, not confirmed, or failed on the network: nothing was paid
)

// RefundStatus is where a refund stands. An approval and a rejection are final: the transaction
// never gets another refund.
type RefundStatus string

// Refund statuses.
const (
	RefundStatusPending  RefundStatus = "pending"  // A customer's request not answered yet
	RefundStatusApproved RefundStatus = "approved" // The refund transfer is confirmed (refund.approved); the merchant's own refund is recorded only then
	RefundStatusRejected RefundStatus = "rejected" // Rejected by whoever decides it (refund.rejected)
	RefundStatusFailed   RefundStatus = "failed"   // Declared, never set: a failed approval leaves the request pending
)

// RefundInitiator says who started a refund.
type RefundInitiator string

// Refund initiators.
const (
	RefundInitiatorCustomer RefundInitiator = "customer" // Requested by the customer; the merchant answers
	RefundInitiatorMerchant RefundInitiator = "merchant" // Sent by the merchant without a request
)

// NotRefundableReason says why a transaction cannot be refunded now.
type NotRefundableReason string

// Reasons a transaction is not refundable.
const (
	NotRefundableReasonRefundExists      NotRefundableReason = "refundExists"
	NotRefundableReasonHeldFundsReleased NotRefundableReason = "heldFundsReleased"
)

// TransactionKind says what a row of the transactions feed is (Transaction.Type). Not
// TransactionType, which is a checkout's txType.
type TransactionKind string

// Transactions feed row kinds.
const (
	TransactionKindPayment TransactionKind = "payment" // A one-time payment (pay@…): Payments.Get
	TransactionKindBill    TransactionKind = "bill"    // A subscription's bill (sub-hist@…): Subscriptions.GetBill
)

// FailureKind says what a failed attempt was paying.
type FailureKind string

// Failure kinds.
const (
	FailureKindPayment              FailureKind = "payment"              // A one-time payment's attempt (pay@…)
	FailureKindSubscriptionCheckout FailureKind = "subscriptionCheckout" // A subscription checkout's: its creation and first bill (sub@…)
	FailureKindBill                 FailureKind = "bill"                 // A subscription's bill (sub-hist@…)
)

// FailureCategory says what a failure means, from its code.
type FailureCategory string

// Failure categories.
const (
	FailureCategoryInsufficientBalance FailureCategory = "insufficientBalance" // The customer's wallet couldn't pay
	FailureCategoryAllowanceExhausted  FailureCategory = "allowanceExhausted"  // A subscription's allowance is used up or expired
	FailureCategoryApprovalRevoked     FailureCategory = "approvalRevoked"     // The wallet no longer lets the subscription pull the token
	FailureCategoryMaxAmountExceeded   FailureCategory = "maxAmountExceeded"   // A bill above the customer's maximum per period
	FailureCategoryReverted            FailureCategory = "reverted"            // Sent and reverted on-chain
	FailureCategoryNotConfirmed        FailureCategory = "notConfirmed"        // Sent, never taken or confirmed in time
	FailureCategorySignatureRejected   FailureCategory = "signatureRejected"   // The customer's signature was refused
	FailureCategoryOther               FailureCategory = "other"
)

// Chain is a blockchain, by its native coin's symbol (Currency.Chain; the testnet's in test mode).
type Chain string

// Chains.
const (
	ChainETH  Chain = "ETH"  // Ethereum
	ChainBASE Chain = "BASE" // Base
	ChainSOL  Chain = "SOL"  // Solana
)

// TransactionType is a transaction's kind (checkout.expired's txType: payment or
// createSubscription).
type TransactionType string

// Transaction types.
const (
	TransactionTypePayment                 TransactionType = "payment"
	TransactionTypeTransfer                TransactionType = "transfer"
	TransactionTypeTokenTransfer           TransactionType = "tokenTransfer"
	TransactionTypeCreateSubscription      TransactionType = "createSubscription"
	TransactionTypeCancelSubscription      TransactionType = "cancelSubscription"
	TransactionTypeForceCancelSubscription TransactionType = "forceCancelSubscription"
	TransactionTypeExecuteSubscription     TransactionType = "executeSubscription"
	TransactionTypeCreatePaygSubscription  TransactionType = "createPaygSubscription"
	TransactionTypeCancelPaygSubscription  TransactionType = "cancelPaygSubscription"
	TransactionTypeIncreaseAllowance       TransactionType = "increaseAllowance"
	TransactionTypeUpdateMaxAmount         TransactionType = "updateMaxAmount"
	TransactionTypeRefund                  TransactionType = "refund"
	TransactionTypeFaucet                  TransactionType = "faucet"
	TransactionTypeClaimFunds              TransactionType = "claimFunds"
	TransactionTypeReleaseHeldFunds        TransactionType = "releaseHeldFunds"
)

// TransferType is a transfer's kind (HeldFundsReleased.Type: heldFundsRelease).
type TransferType string

// Transfer types.
const (
	TransferTypeAccountClaim     TransferType = "accountClaim"     // v1: funds held for a user, paid when they claimed their account
	TransferTypeHeldFundsRelease TransferType = "heldFundsRelease" // The funds an organization held for a member, paid out to them
	TransferTypeInternal         TransferType = "internal"
	TransferTypeExternal         TransferType = "external"
)

// LedgerEntryType is a held-funds line's kind. A held member sells no subscription, so no bill
// is ever a line.
type LedgerEntryType string

// Ledger entry types.
const (
	LedgerEntryTypePayment LedgerEntryType = "payment" // A payment made to the member while the organization held their funds
	LedgerEntryTypeRefund  LedgerEntryType = "refund"  // A refund the organization sent of one of them
)

// AccountingEventType is an accounting export row's kind.
type AccountingEventType string

// Accounting event types.
const (
	AccountingEventTypePayment         AccountingEventType = "payment"         // A one-time payment (pay@…)
	AccountingEventTypeBill            AccountingEventType = "bill"            // A subscription's bill (sub-hist@…)
	AccountingEventTypeRefund          AccountingEventType = "refund"          // A refund the space sent (refund@…)
	AccountingEventTypeOrganizationFee AccountingEventType = "organizationFee" // The fee the organization earned on a member's payment or bill
	AccountingEventTypeReferralFee     AccountingEventType = "referralFee"     // The referrer's share earned on a referred organization's payment or bill
)

// NetworkFeesPayer is whose wallet paid an accounting event's network fees
// (AccountingEvent.NetworkFeesPaidBy).
type NetworkFeesPayer string

// Network fees payers.
const (
	NetworkFeesPaidByPayer        NetworkFeesPayer = "payer"        // A payment or a bill: the payer, on top of the amount
	NetworkFeesPaidByMerchant     NetworkFeesPayer = "merchant"     // A refund the space sent from its wallet
	NetworkFeesPaidByOrganization NetworkFeesPayer = "organization" // A refund of a payment the organization held for its member, sent from the organization's wallet
)

// InvitationStatus is an invitation's status (computed when read).
type InvitationStatus string

// Invitation statuses.
const (
	InvitationStatusPending  InvitationStatus = "pending"  // Sent, waiting for the person
	InvitationStatusAccepted InvitationStatus = "accepted" // The person joined
	InvitationStatusRevoked  InvitationStatus = "revoked"  // Revoked by the organization (or replaced by a newer one)
	InvitationStatusExpired  InvitationStatus = "expired"  // Not accepted in time
)

// EventType is a webhook event's type.
type EventType string

// Webhook event types.
const (
	EventTypePaymentCompleted                  EventType = "payment.completed"
	EventTypeSubscriptionCreated               EventType = "subscription.created"
	EventTypeSubscriptionBilled                EventType = "subscription.billed"
	EventTypeSubscriptionStatusChanged         EventType = "subscription.statusChanged"
	EventTypeSubscriptionActionRequiredChanged EventType = "subscription.actionRequiredChanged"
	EventTypeSubscriptionBillingFailed         EventType = "subscription.billingFailed"
	EventTypeSubscriptionUpcomingBill          EventType = "subscription.upcomingBill"
	EventTypeRefundRequested                   EventType = "refund.requested"
	EventTypeRefundApproved                    EventType = "refund.approved" // Was refund.completed
	EventTypeRefundRejected                    EventType = "refund.rejected" // Was refund.denied
	EventTypeMemberJoined                      EventType = "member.joined"
	EventTypeMemberRemoved                     EventType = "member.removed"
	EventTypeHeldFundsReleased                 EventType = "heldFunds.released"
	EventTypeCheckoutExpired                   EventType = "checkout.expired"
	EventTypeWebhookTest                       EventType = "webhook.test" // Sent only by an endpoint's test (the dashboard); no endpoint subscribes to it
)

// knownEventTypes lists every EventType constant (the router's known types).
var knownEventTypes = []EventType{
	EventTypePaymentCompleted, EventTypeSubscriptionCreated, EventTypeSubscriptionBilled,
	EventTypeSubscriptionStatusChanged, EventTypeSubscriptionActionRequiredChanged,
	EventTypeSubscriptionBillingFailed, EventTypeSubscriptionUpcomingBill, EventTypeRefundRequested,
	EventTypeRefundApproved, EventTypeRefundRejected, EventTypeMemberJoined, EventTypeMemberRemoved,
	EventTypeHeldFundsReleased, EventTypeCheckoutExpired, EventTypeWebhookTest,
}

// WebhookPayloadVersion is a webhook endpoint's (and an event's) payload version.
type WebhookPayloadVersion string

// Webhook payload versions.
const (
	WebhookPayloadVersionV2 WebhookPayloadVersion = "v2" // The event envelope {id, type, version, createdAt, test, userUuid, data}: the only version
)

// EndpointDisabledReason says why a webhook endpoint is disabled.
type EndpointDisabledReason string

// Endpoint disabled reasons.
const (
	EndpointDisabledReasonFailing EndpointDisabledReason = "failing" // Every delivery failing for too long
	EndpointDisabledReasonOwner   EndpointDisabledReason = "owner"   // Disabled by its owner
	EndpointDisabledReasonClosed  EndpointDisabledReason = "closed"  // Its member removed, or its organization closed
)

// DurationUnit is a Duration's unit (months = 30 days, years = 365 days).
type DurationUnit string

// Duration units.
const (
	DurationUnitSeconds DurationUnit = "seconds"
	DurationUnitMinutes DurationUnit = "minutes"
	DurationUnitHours   DurationUnit = "hours"
	DurationUnitDays    DurationUnit = "days"
	DurationUnitWeeks   DurationUnit = "weeks"
	DurationUnitMonths  DurationUnit = "months"
	DurationUnitYears   DurationUnit = "years"
)

// Role is what a credential (or an invitation) may do in a space. An API key is admin
// (organization key) or user (a member's key, or On-Behalf-Of).
type Role string

// Roles.
const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleHandle Role = "handle"
)

// Credential is what authenticated a request (Me.Credential).
type Credential string

// Credentials.
const (
	CredentialAPIKey  Credential = "apiKey"  // An API key (X-API-Key)
	CredentialSession Credential = "session" // A signed-in person's access token
)

// MerchantNotReadyReason is details.reason of a 409 merchant_not_ready.
type MerchantNotReadyReason string

// Reasons a merchant cannot be paid yet.
const (
	MerchantNotReadyReasonNoWallet             MerchantNotReadyReason = "noWallet"
	MerchantNotReadyReasonOrganizationNoWallet MerchantNotReadyReason = "organizationNoWallet"
	MerchantNotReadyReasonNoTokenWallet        MerchantNotReadyReason = "noTokenWallet"
	MerchantNotReadyReasonFundsHeld            MerchantNotReadyReason = "fundsHeld" // A subscription checkout of a member whose funds the organization holds: one-time payments only until Members.Trust
)
