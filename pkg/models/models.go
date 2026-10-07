// Package qbmodels holds the request and response types of the QBitFlow API.
//
// # Decoding policy
//
// Every response type mirrors the Go definition the server serializes, field for field:
//
//   - A value field (string, number, bool, nested struct, time.Time) is always usable. When the
//     API omits it — which it does for `omitempty` fields at their zero value — or sends null, it
//     decodes to the zero value (0, "", false, a zero-valued struct, the zero time). You never
//     have to nil-check it.
//   - A pointer field is genuinely nullable on the server: nil means the API sent null or nothing.
//   - A slice is a list; an absent or null list decodes to an empty one. The services return a
//     non-nil empty slice; if you decode a payload yourself, a nil slice behaves the same in Go.
//   - An enum-like field is a string type: a value this SDK does not know yet is kept verbatim,
//     never rejected and never coerced.
//
// A field present with the wrong JSON type (a string where a number is expected, …) is a
// response-shape failure: the service reports it as a *qberrors.ServerError carrying the HTTP
// status.
package qbmodels

import (
	"encoding/json"
	"fmt"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

// TransactionType represents the on-chain transaction type used in webhooks and
// transaction-status lookups.
type TransactionType string

const (
	TransactionTypeOneTimePayment             TransactionType = "payment"
	TransactionTypeTransfer                   TransactionType = "transfer"
	TransactionTypeTokenTransfer              TransactionType = "tokenTransfer"
	TransactionTypeCreateSubscription         TransactionType = "createSubscription"
	TransactionTypeCancelSubscription         TransactionType = "cancelSubscription"
	TransactionTypeExecuteSubscriptionPayment TransactionType = "executeSubscription"
	TransactionTypeCreatePAYGSubscription     TransactionType = "createPAYGSubscription"
	TransactionTypeCancelPAYGSubscription     TransactionType = "cancelPAYGSubscription"
	TransactionTypeIncreaseAllowance          TransactionType = "increaseAllowance"
	TransactionTypeUpdateMaxAmount            TransactionType = "updateMaxAmount"
	TransactionTypeRefund                     TransactionType = "refund"
	TransactionTypeFaucet                     TransactionType = "faucet"
	TransactionTypeClaimFunds                 TransactionType = "claimFunds"
)

// TransactionShortType is the short transaction category encoded in a UUID prefix
// (e.g. "payment" -> pay@, "subscription" -> sub@, "subscriptionHistory" -> sub-hist@,
// "refund" -> refund@).
//
// NOTE: this is NOT the type reported on session data or webhooks — those carry the
// long-form TransactionType (e.g. "createSubscription"). Use TransactionType there.
type TransactionShortType string

const (
	TransactionShortTypePayment      TransactionShortType = "payment"
	TransactionShortTypeSubscription TransactionShortType = "subscription"
	TransactionShortTypePayAsYouGo   TransactionShortType = "payAsYouGo"
	TransactionShortTypeSubHistory   TransactionShortType = "subscriptionHistory"
	TransactionShortTypeRefund       TransactionShortType = "refund"
	TransactionShortTypeTransfer     TransactionShortType = "transfer"
)

// TransactionStatusValue represents the status of a transaction
type TransactionStatusValue string

const (
	TransactionStatusCreated             TransactionStatusValue = "created"
	TransactionStatusWaitingConfirmation TransactionStatusValue = "waitingConfirmation"
	TransactionStatusPending             TransactionStatusValue = "pending"
	TransactionStatusCompleted           TransactionStatusValue = "completed"
	TransactionStatusFailed              TransactionStatusValue = "failed"
	TransactionStatusCancelled           TransactionStatusValue = "cancelled"
	TransactionStatusExpired             TransactionStatusValue = "expired"
)

// TransactionStatus represents the status details of a transaction.
type TransactionStatus struct {
	Status TransactionStatusValue `json:"status"`
	// TxHash is the on-chain hash once known; "" before that.
	TxHash string `json:"txHash,omitempty"`
	// Message is optional context from the API (for example a failure reason); "" when none.
	Message string `json:"message,omitempty"`

	// SettlementDetails holds the finalized payment metadata for a completed,
	// successful transaction (nil until settled).
	SettlementDetails *PaymentMetadata `json:"settlementDetails,omitempty"`
}

// SubscriptionStatus is the lifecycle state of a subscription. Unknown values sent by a
// newer API are preserved as-is (the type is a string), never coerced to a default.
type SubscriptionStatus string

const (
	// SubscriptionStatusActive indicates an active subscription.
	SubscriptionStatusActive SubscriptionStatus = "active"

	// SubscriptionStatusPastDue indicates the last billing cycle was not successful.
	SubscriptionStatusPastDue SubscriptionStatus = "past_due"

	// SubscriptionStatusLowOnFunds indicates a subscription with low allowance, therefore the next billing cycle may fail.
	SubscriptionStatusLowOnFunds SubscriptionStatus = "low_on_funds"

	// SubscriptionStatusCancelled indicates a cancelled/inactive subscription.
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"

	// SubscriptionStatusPending indicates a pending subscription, eg max amount has been reached and the user needs to increase it.
	SubscriptionStatusPending SubscriptionStatus = "pending"

	// SubscriptionStatusTrial indicates a subscription in trial period.
	SubscriptionStatusTrial SubscriptionStatus = "trial"

	// SubscriptionStatusTrialExpired indicates a subscription whose trial period has expired.
	SubscriptionStatusTrialExpired SubscriptionStatus = "trial_expired"
)

// DurationUnit represents the unit of time for durations
type DurationUnit string

const (
	DurationUnitSeconds DurationUnit = "seconds"
	DurationUnitMinutes DurationUnit = "minutes"
	DurationUnitHours   DurationUnit = "hours"
	DurationUnitDays    DurationUnit = "days"
	DurationUnitWeeks   DurationUnit = "weeks"
	DurationUnitMonths  DurationUnit = "months"
	DurationUnitYears   DurationUnit = "years"
)

// Duration represents a time duration with a value and unit (e.g., {Value: 1, Unit: "months"}).
//
// The API stores Value as an unsigned 32-bit integer, so it must lie within 0..4294967295
// (a billing Frequency must be at least 1); the SDK checks this before the request.
type Duration struct {
	Value int          `json:"value"`
	Unit  DurationUnit `json:"unit"`
}

// SubscriptionType represents the type of subscription (used when resolving a
// subscription by reference).
type SubscriptionType string

const (
	SubscriptionTypeSubscription SubscriptionType = "subscription"
)

// Currency represents a cryptocurrency supported by QBitFlow. Tokens reference a
// main (native) currency via MainCurrencyID.
type Currency struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	// Address is the contract address (or mint) for a token; "" for a native currency.
	Address string `json:"address"`
	// MainCurrencyID and MainCurrency are set for a token and nil for a main currency.
	MainCurrencyID *uint64   `json:"mainCurrencyId,omitempty"`
	MainCurrency   *Currency `json:"mainCurrency"`
	Test           bool      `json:"test"`
}

// LinkResponse represents a response containing a payment/subscription checkout link
type LinkResponse struct {
	UUID string `json:"uuid"`
	Link string `json:"link"`
}

// SessionCheckout is the unified response from GET /transaction/session-checkout/:uuid, and
// the `session` object of a transaction webhook.
//
// Payment sessions carry TxType "payment"; subscription sessions carry "createSubscription"
// plus the Frequency/TrialPeriod/MinPeriods fields. Use IsSubscription / IsPayment.
//
// Every field is always usable: the optional ones decode to their zero value ("", 0) when
// the session does not carry them. CustomerUUID is the only nullable field.
type SessionCheckout struct {
	UUID string `json:"uuid"`
	// Reference is your own reference for the transaction, set when the session was created.
	Reference string `json:"reference,omitempty"`
	ProductID uint64 `json:"productId,omitempty"`
	// ProductReference is your own product reference, if the product was selected by reference.
	ProductReference string  `json:"productReference,omitempty"`
	ProductName      string  `json:"productName,omitempty"`
	Description      string  `json:"description,omitempty"`
	Price            float64 `json:"price,omitempty"`
	SuccessURL       string  `json:"successUrl,omitempty"`
	CancelURL        string  `json:"cancelUrl,omitempty"`

	// Set by the server.
	OrganizationID   uint64 `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	// FeeBps is the QBitFlow platform fee applied to this session, in basis points.
	FeeBps uint16 `json:"feeBps"`
	// OrganizationFeeBps is the additional organization fee in basis points (0 when none).
	OrganizationFeeBps uint16 `json:"organizationFeeBps,omitempty"`
	// UserID and UserName identify the user who created the session (0 / "" at the
	// organization level).
	UserID   uint64 `json:"userId,omitempty"`
	UserName string `json:"userName,omitempty"`
	Test     bool   `json:"test"`
	// CustomerUUID is the customer pre-associated with the session; nil when the customer is
	// collected during checkout.
	CustomerUUID *string `json:"customerUUID,omitempty"`
	// CustomerReference is your own customer reference, if the customer was pre-filled by reference.
	CustomerReference string          `json:"customerReference,omitempty"`
	TxType            TransactionType `json:"txType"`
	// AvailableCurrencies lists the currency IDs accepted for this payment.
	// Resolve details via the Currencies service.
	AvailableCurrencies []uint64 `json:"availableCurrencies"`

	// Subscription-specific fields — zero for payment sessions. Frequency, TrialPeriod are
	// in seconds.
	Frequency   uint32 `json:"frequency,omitempty"`
	TrialPeriod uint32 `json:"trialPeriod,omitempty"`
	MinPeriods  uint32 `json:"minPeriods,omitempty"`
	// UpgradingFromTrial is set when the subscription session upgrades an existing trial.
	UpgradingFromTrial bool `json:"upgradingFromTrial,omitempty"`
}

// IsSubscription reports whether the session is a subscription checkout.
//
// It prefers the TxType discriminator ("createSubscription") and falls back to the
// presence of a billing Frequency for payloads that omit txType.
func (s *SessionCheckout) IsSubscription() bool {
	if s.TxType != "" {
		return s.TxType == TransactionTypeCreateSubscription
	}
	return s.Frequency > 0
}

// IsPayment reports whether the session is a one-time payment checkout.
//
// It is true for TxType "payment", or — when txType is absent — for a session without a
// billing Frequency. A session of any other (for example a newer) type is neither a payment
// nor a subscription: both IsPayment and IsSubscription report false.
func (s *SessionCheckout) IsPayment() bool {
	if s.TxType != "" {
		return s.TxType == TransactionTypeOneTimePayment
	}
	return s.Frequency == 0
}

// ----- Payment metadata (typed) -----

// PaymentMetadata is the structured metadata attached to a payment or subscription-billing
// record: the fee breakdown, on-chain transaction metadata, and computed amounts.
type PaymentMetadata struct {
	// FeeBps is the QBitFlow platform fee in basis points (1% = 100 bps), deducted from
	// the amount paid; the merchant receives amount - platform fee - organization fee.
	FeeBps uint16 `json:"feeBps"`
	// OrganizationFee is the optional additional fee kept by the organization (nil when none).
	OrganizationFee *OrganizationFee `json:"organizationFee,omitempty"`
	// ReferralFee is the optional referral fee (nil when none).
	ReferralFee *ReferralFee  `json:"referralFee,omitempty"`
	TxMetadata  TxMetadata    `json:"txMetadata"` // on-chain metadata (populated after confirmation)
	TxAmounts   TxAmountsFull `json:"txAmounts"`  // computed fee/merchant amounts
}

// OrganizationFee is an additional fee kept by the organization, on top of the platform fee.
type OrganizationFee struct {
	OrganizationID uint64 `json:"organizationId"`
	// Organization is the address receiving the fee.
	Organization string `json:"organization"`
	FeeBps       uint16 `json:"feeBps"`
}

// ReferralFee is an optional fee paid to a referrer.
type ReferralFee struct {
	ReferralID uint64 `json:"referralId"`
	// Referrer is the address receiving the fee.
	Referrer string    `json:"referrer"`
	FeeBps   uint16    `json:"feeBps"`
	Deadline time.Time `json:"deadline"`
}

// TxMetadata holds on-chain details parsed from the settled transaction.
type TxMetadata struct {
	NetworkFees NetworkFees `json:"networkFees"`
	BlockData   BlockData   `json:"blockData"`
	// MainCurrencyPriceUSD is the native-currency USD price at transaction time, used for
	// accounting on refunds or when the merchant pays the network fees (0 otherwise).
	MainCurrencyPriceUSD float64 `json:"mainCurrencyPriceUSD,omitempty"`
}

// BlockData identifies the block a transaction was included in.
type BlockData struct {
	Number    string `json:"number"`
	Timestamp uint64 `json:"timestamp"`
}

// NetworkFees describes the on-chain network fees, in the smallest native-currency units.
type NetworkFees struct {
	Amount        string `json:"amount"` // decimal string, min units of the native currency
	UnitsConsumed uint64 `json:"unitsConsumed"`
}

// TxAmountsMinUnits holds the per-party amounts in the smallest currency units (decimal strings).
type TxAmountsMinUnits struct {
	Platform     string `json:"platform"`
	Organization string `json:"organization"`
	Referral     string `json:"referral"`
	Merchant     string `json:"merchant"`
}

// TxAmountsUSD holds the per-party amounts in USD (Organization and Referral are 0 when the
// fee does not apply).
type TxAmountsUSD struct {
	Platform     float64 `json:"platform"`
	Organization float64 `json:"organization,omitempty"`
	Referral     float64 `json:"referral,omitempty"`
	Merchant     float64 `json:"merchant"`
}

// TxAmountsFull holds the computed per-party amounts, in both USD and min units.
type TxAmountsFull struct {
	Usd      TxAmountsUSD      `json:"usd"`
	MinUnits TxAmountsMinUnits `json:"minUnits"`
}

// Payment represents a completed one-time payment.
type Payment struct {
	UUID            string    `json:"uuid"` // pay@-prefixed
	CreatedAt       time.Time `json:"createdAt"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Amount          float64   `json:"amount"`         // USD
	AmountMinUnits  string    `json:"amountMinUnits"` // decimal string, smallest currency units
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"` // the currency the payment was made in
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	OrganizationID  uint64    `json:"organizationId"`
	// UserID is the user the payment belongs to; 0 for an organization-level payment.
	UserID uint64 `json:"userId"`

	// Reference is your own reference for the payment, set when the session was created;
	// nil when none was set.
	Reference   *string `json:"reference,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	// ProductID is the product paid for; 0 when the payment is not linked to a product.
	ProductID uint64 `json:"productId,omitempty"`
	// CustomerUUID is the customer who paid; nil when no customer is attached.
	CustomerUUID *string         `json:"customerUUID,omitempty"`
	Metadata     PaymentMetadata `json:"metadata"`
}

// CombinedPayment represents a single item from the combined payment list
// (may originate from a one-time payment or a subscription billing cycle).
type CombinedPayment struct {
	Source         string    `json:"source"` // "payment" or "subscription_history"
	UUID           string    `json:"uuid"`
	CreatedAt      time.Time `json:"createdAt"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Amount         float64   `json:"amount"`
	AmountMinUnits string    `json:"amountMinUnits"`
	CurrencyID     uint64    `json:"currencyId"`
	// ProductID is nil when the item is not linked to a product.
	ProductID       *uint64 `json:"productId,omitempty"`
	TransactionHash string  `json:"transactionHash"`
	// CustomerUUID is always sent; it is the zero UUID when no customer is attached.
	CustomerUUID string `json:"customerUUID"`
	// SubscriptionUUID is set only for subscription-history rows.
	SubscriptionUUID *string `json:"subscriptionUUID,omitempty"`
	Test             bool    `json:"test"`
	// Metadata is nil when the row carries none.
	Metadata *PaymentMetadata `json:"metadata"`
	Currency Currency         `json:"currency"`
}

// Subscription represents an active or historical subscription
type Subscription struct {
	UUID string `json:"uuid"` // sub@-prefixed (differs from the checkout session UUID)
	// Reference is your own reference for the subscription, set when the session was created;
	// nil when none was set.
	Reference *string   `json:"reference,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	From               string             `json:"from"`
	To                 string             `json:"to"`
	ProductID          uint64             `json:"productId"`
	SubscriptionHash   string             `json:"subscriptionHash"`
	CurrencyID         uint64             `json:"currencyId"`
	Currency           Currency           `json:"currency"`
	Frequency          uint32             `json:"frequency"` // billing interval in seconds
	Allowance          string             `json:"allowance"` // remaining allowance in USD (decimal string)
	SubscriptionStatus SubscriptionStatus `json:"subscriptionStatus"`
	Stopped            bool               `json:"stopped"` // will be cancelled at end of current period

	// LastBillingDate is the zero time until the subscription has been billed.
	LastBillingDate time.Time `json:"lastBillingDate"`
	NextBillingDate time.Time `json:"nextBillingDate"`
	// MinimumCancellationDate is nil when the subscription has no minimum period.
	MinimumCancellationDate *time.Time `json:"minimumCancellationDate,omitempty"`
	Test                    bool       `json:"test"`

	OrganizationID uint64 `json:"organizationId"`
	// UserID is the user the subscription belongs to; 0 at the organization level.
	UserID uint64 `json:"userId"`
	// CustomerUUID is the subscribed customer; nil when no customer is attached.
	CustomerUUID *string `json:"customerUUID,omitempty"`
}

// SubscriptionHistory represents a single billing event for a subscription
type SubscriptionHistory struct {
	UUID            string    `json:"uuid"` // sub-hist@-prefixed
	CreatedAt       time.Time `json:"createdAt"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Amount          float64   `json:"amount"`
	AmountMinUnits  string    `json:"amountMinUnits"`
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"`
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	OrganizationID  uint64    `json:"organizationId"`
	UserID          uint64    `json:"userId"`

	Name             string `json:"name"`
	Description      string `json:"description"`
	ProductID        uint64 `json:"productId"`
	SubscriptionUUID string `json:"subscriptionUUID"` // sub@-prefixed
	// CustomerUUID is the billed customer; nil when no customer is attached.
	CustomerUUID *string         `json:"customerUUID,omitempty"`
	Metadata     PaymentMetadata `json:"metadata"`
}

// CursorData represents paginated cursor-based data
type CursorData[T any] struct {
	// Items is the page; the services always return a non-nil (possibly empty) slice.
	Items []T `json:"items"`
	// NextCursor is the cursor of the next page; nil on the last page.
	NextCursor *string `json:"nextCursor"`
}

// HasMore reports whether another page exists; pass NextCursor to the same method to fetch it.
func (c *CursorData[T]) HasMore() bool {
	return c.NextCursor != nil
}

// SuccessResponse represents a generic success message response (maps to JSONMessage on the server)
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// Customer represents a customer record
type Customer struct {
	UUID     string `json:"uuid"` // bare UUID (no prefix)
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Email    string `json:"email"`

	// PhoneNumber, Address and Reference are "" when not set.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Address     string `json:"address,omitempty"`
	Reference   string `json:"reference,omitempty"`

	CreatedAt      time.Time `json:"createdAt"`
	OrganizationID uint64    `json:"organizationId"`
	// UserID is the user who owns the customer; 0 for an organization-level customer.
	UserID uint64 `json:"userId,omitempty"`
	Test   bool   `json:"test"`
}

// Product represents a product record
type Product struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"` // USD
	CreatedAt   time.Time `json:"createdAt"`
	// IsActive is false for a hidden product created behind the scenes by an inline session
	// checkout; hidden products are excluded from GetAll.
	IsActive       bool   `json:"isActive"`
	Reference      string `json:"reference"`
	Test           bool   `json:"test"`
	OrganizationID uint64 `json:"organizationId"`
	// UserID is the user who owns the product; 0 for an organization-level product.
	UserID uint64 `json:"userId"`
}

// User represents a user within an organization
type User struct {
	ID                 uint64    `json:"id"`
	Name               string    `json:"name"`
	LastName           string    `json:"lastName"`
	Email              string    `json:"email"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	OrganizationID     uint64    `json:"organizationId"`
	Role               UserRole  `json:"role"`
	OrganizationFeeBps uint16    `json:"organizationFeeBps"`
	// ClaimedAt is set once the invited user has claimed their account; nil until then.
	ClaimedAt *time.Time `json:"claimedAt"`
}

// UserRole represents the role of a user.
//
// The hierarchy is RoleHandle < RoleUser < RoleAdmin < RoleOwner. RoleHandle and RoleOwner
// are read-only: the API accepts only "admin" or "user" when creating a user.
type UserRole string

const (
	// RoleHandle is the lowest authenticated tier — a special case of a user, backing the
	// second frontend app.
	RoleHandle UserRole = "handle"
	RoleUser   UserRole = "user"
	RoleAdmin  UserRole = "admin"
	// RoleOwner is the organization owner.
	RoleOwner UserRole = "owner"
)

// ApiKey represents an API key record (the secret key value is never returned).
type ApiKey struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	OrganizationID uint64 `json:"organizationId"`
	// UserID is the user the key belongs to; 0 for an organization-level key.
	UserID    uint64    `json:"userId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is nil when the key never expires.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Role      UserRole   `json:"role"`
	Test      bool       `json:"test"`
}

// Organization represents an organization (merchant account)
type Organization struct {
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	FeePercentage float32   `json:"feePercentage"` // default platform fee %
	CreatedAt     time.Time `json:"createdAt"`
}

// CreateClaimRequestResponse is returned when an admin creates a claim request for a user
type CreateClaimRequestResponse struct {
	Message string `json:"message"`
	Link    string `json:"link"` // the QBitFlow page where the user sets up their password and wallet
}

// ClaimFund represents funds the organization owes a user who has claimed their account
type ClaimFund struct {
	UserID          uint64    `json:"userId"`
	TotalAmountOwed float64   `json:"totalAmountOwed"` // USD
	Funded          bool      `json:"funded"`
	Test            bool      `json:"test"`
	CreatedAt       time.Time `json:"createdAt"`
}

// RefundStatus represents the lifecycle status of a refund
type RefundStatus string

const (
	RefundStatusPending  RefundStatus = "pending"
	RefundStatusApproved RefundStatus = "approved"
	// RefundStatusRefused is the merchant-refused state (the API's value is "refused").
	RefundStatusRefused RefundStatus = "refused"
	RefundStatusFailed  RefundStatus = "failed"
)

// RefundEntry represents a refund record
type RefundEntry struct {
	UUID      string       `json:"uuid"` // refund@-prefixed
	TxID      string       `json:"txId"` // the refunded transaction, e.g. pay@<uuid>
	Test      bool         `json:"test"`
	Reason    string       `json:"reason"`
	Status    RefundStatus `json:"status"`
	CreatedAt time.Time    `json:"createdAt"`
	// MerchantMessage is the merchant's reply; "" until the merchant responds.
	MerchantMessage string `json:"merchantMessage"`
	// RespondedAt is nil until the merchant has handled the refund.
	RespondedAt    *time.Time `json:"respondedAt"`
	UserID         uint64     `json:"userId"`
	OrganizationID uint64     `json:"organizationId"`
	// TxHash is the on-chain hash of the refund transfer; "" until it is processed.
	TxHash         string `json:"txHash"`
	AmountMinUnits string `json:"amountMinUnits"`
	// Metadata is nil until the refund transfer has been processed.
	Metadata *TxMetadata `json:"metadata"`
}

// Known AccountingEvent.Type values. Type is a plain string, so a value not listed here is
// kept verbatim. The subscription-billing value is documented inconsistently by the API
// ("subscriptionHistory" as a constant, "subHistory" in prose); both spellings are listed.
const (
	AccountingEventTypePayment             = "payment"
	AccountingEventTypeSubscriptionHistory = "subscriptionHistory"
	AccountingEventTypeSubHistory          = "subHistory"
	AccountingEventTypeRefund              = "refund"
	AccountingEventTypeOrganizationFee     = "organizationFee"
	AccountingEventTypeReferralFee         = "referralFee"
)

// AccountingEvent represents a single line item in an accounting export
type AccountingEvent struct {
	PaymentID        string `json:"paymentId"`
	PaymentReference string `json:"paymentReference"`
	// Type is one of the AccountingEventType* values (kept verbatim when unknown).
	Type                    string    `json:"type"`
	TxTimeUTC               time.Time `json:"txTimeUtc"`
	ReceiptURL              string    `json:"receiptUrl"`
	RelatedPaymentID        string    `json:"relatedPaymentId"`        // for refunds, the original payment ID
	RelatedPaymentReference string    `json:"relatedPaymentReference"` // for refunds, the original payment reference

	ProductID          uint64 `json:"productId"`
	ProductReference   string `json:"productReference"`
	ProductName        string `json:"productName"`
	ProductDescription string `json:"productDescription"`
	CustomerUUID       string `json:"customerUUID"`
	CustomerReference  string `json:"customerReference"`

	Chain             string `json:"chain"`
	BlockNumberOrSlot string `json:"blockNumberOrSlot"`
	TxHash            string `json:"txHash"`
	FromAddress       string `json:"fromAddress"`
	ToAddress         string `json:"toAddress"`

	TokenSymbol         string `json:"tokenSymbol"`
	CurrencyDecimals    uint8  `json:"currencyDecimals"`
	TokenContractOrMint string `json:"tokenContractOrMint"`

	ExplorerURL string `json:"explorerUrl"`

	// Amounts serialized as decimal strings by the server
	GrossAmount    string  `json:"grossAmount"`
	GrossAmountUSD float64 `json:"grossAmountUsd"`

	PlatformFeePercent float64 `json:"platformFeePercent"`
	PlatformFeeUSD     float64 `json:"platformFeeUsd"`
	PlatformFee        string  `json:"platformFee"`

	OrganizationFeePercent float64 `json:"organizationFeePercent"`
	OrganizationFeeUSD     float64 `json:"organizationFeeUsd"`
	OrganizationFee        string  `json:"organizationFee"`

	// Network fees — only populated when QBitFlow pays them (e.g. refunds)
	NetworkFeesUSD float64 `json:"networkFeesUsd"`
	NetworkFees    string  `json:"networkFees"`

	NetAmountUSD float64 `json:"netAmountUsd"`
	NetAmount    string  `json:"netAmount"`
}

// SessionWebhookResponse is the transaction webhook QBitFlow POSTs to your Transaction
// webhook URL when a checkout completes.
type SessionWebhookResponse struct {
	UUID   string          `json:"uuid"`
	TxType TransactionType `json:"txType"`
	// Status is the transaction state at delivery time; nil if the API sent null.
	Status *TransactionStatus `json:"status"`
	// Session is the checkout session (a subscription session also carries Frequency…).
	Session SessionCheckout `json:"session"`
	// ManagementPageLink links to the QBitFlow page for managing the transaction; "" when absent.
	ManagementPageLink string `json:"managementPageLink,omitempty"`
}

// SubscriptionWebhookType identifies which payload a subscription webhook carries.
type SubscriptionWebhookType string

const (
	// SubscriptionWebhookTypeStatusTransition means Data is a SubscriptionStatusTransition.
	SubscriptionWebhookTypeStatusTransition SubscriptionWebhookType = "status_transition"
	// SubscriptionWebhookTypeBilling means Data is a SubscriptionHistory.
	SubscriptionWebhookTypeBilling SubscriptionWebhookType = "billing"
)

// SubscriptionStatusTransition is the Data payload of a subscription webhook whose Type
// is SubscriptionWebhookTypeStatusTransition.
type SubscriptionStatusTransition struct {
	PreviousStatus SubscriptionStatus `json:"previousStatus"`
	CurrentStatus  SubscriptionStatus `json:"currentStatus"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

// SubscriptionWebhook is the envelope QBitFlow POSTs to your subscription webhook URL.
//
// The subscription identity lives on the envelope and the event-specific payload in Data,
// discriminated by Type. Decode the envelope first, then unwrap Data with StatusTransition
// or Billing:
//
//	var event qbmodels.SubscriptionWebhook
//	if err := json.Unmarshal(body, &event); err != nil { ... }
//
//	switch event.Type {
//	case qbmodels.SubscriptionWebhookTypeStatusTransition:
//		transition, err := event.StatusTransition()
//		// ... react to transition.PreviousStatus -> transition.CurrentStatus
//	case qbmodels.SubscriptionWebhookTypeBilling:
//		billing, err := event.Billing()
//		// ... record billing for event.SubscriptionUUID
//	}
type SubscriptionWebhook struct {
	SubscriptionUUID string `json:"subscriptionUUID"`
	// SubscriptionReference is your own reference for the subscription; "" when none was set.
	SubscriptionReference string                  `json:"subscriptionReference,omitempty"`
	Type                  SubscriptionWebhookType `json:"type"`
	// Data is the event-specific payload; use StatusTransition or Billing to decode it.
	Data json.RawMessage `json:"data"`
}

// StatusTransition decodes Data as a status change.
//
// It returns a *qberrors.ValidationError when Type is not
// SubscriptionWebhookTypeStatusTransition, so a mis-routed payload fails loudly rather than
// yielding a zero-valued transition, and when Data does not decode.
func (w *SubscriptionWebhook) StatusTransition() (*SubscriptionStatusTransition, error) {
	if w.Type != SubscriptionWebhookTypeStatusTransition {
		return nil, qberrors.NewValidationError(fmt.Sprintf("subscription webhook is %q, not %q",
			w.Type, SubscriptionWebhookTypeStatusTransition))
	}

	var transition SubscriptionStatusTransition
	if err := decodeData(w.Data, &transition); err != nil {
		return nil, qberrors.NewValidationError(fmt.Sprintf("decoding subscription status transition: %v", err))
	}

	return &transition, nil
}

// Billing decodes Data as the billing record for a renewed period.
//
// It returns a *qberrors.ValidationError when Type is not SubscriptionWebhookTypeBilling or
// when Data does not decode.
func (w *SubscriptionWebhook) Billing() (*SubscriptionHistory, error) {
	if w.Type != SubscriptionWebhookTypeBilling {
		return nil, qberrors.NewValidationError(fmt.Sprintf("subscription webhook is %q, not %q",
			w.Type, SubscriptionWebhookTypeBilling))
	}

	var billing SubscriptionHistory
	if err := decodeData(w.Data, &billing); err != nil {
		return nil, qberrors.NewValidationError(fmt.Sprintf("decoding subscription billing record: %v", err))
	}

	return &billing, nil
}

// decodeData decodes a webhook's Data; an absent Data decodes to the zero value, like an
// explicit null.
func decodeData(data json.RawMessage, target any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}
