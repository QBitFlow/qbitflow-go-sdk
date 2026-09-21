package qbmodels

import "time"

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
	Status  TransactionStatusValue `json:"status"`
	TxHash  string                 `json:"txHash,omitempty"`
	Message *string                `json:"message,omitempty"`

	// SettlementDetails holds the finalized payment metadata for a completed,
	// successful transaction (nil until settled).
	SettlementDetails *PaymentMetadata `json:"settlementDetails,omitempty"`
}

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

// Duration represents a time duration with a value and unit (e.g., {Value: 1, Unit: "months"})
type Duration struct {
	Value int          `json:"value"`
	Unit  DurationUnit `json:"unit"`
}

// SubscriptionType represents the type of subscription (used when resolving a
// subscription by reference).
type SubscriptionType string

const (
	SubscriptionTypeSubscription SubscriptionType = "subscription"
	SubscriptionTypePAYG         SubscriptionType = "payAsYouGo"
)

// Currency represents a cryptocurrency supported by QBitFlow. Tokens reference a
// main (native) currency via MainCurrencyID.
type Currency struct {
	ID             uint64    `json:"id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	Decimals       int       `json:"decimals"`
	Address        string    `json:"address,omitempty"` // contract address for tokens, empty for native currencies
	MainCurrencyID *uint64   `json:"mainCurrencyId,omitempty"`
	MainCurrency   *Currency `json:"mainCurrency,omitempty"`
	Test           bool      `json:"test"`
}

// LinkResponse represents a response containing a payment/subscription checkout link
type LinkResponse struct {
	UUID string `json:"uuid"`
	Link string `json:"link"`
}

// SessionCheckout is the unified response from GET /transaction/session-checkout/:uuid.
// Subscription sessions have non-zero Frequency; payment sessions do not.
//
// Fields marked "authenticated only" (fees, organization/user IDs, customer identifiers)
// are returned only when the session is retrieved with a valid API key; on the public
// checkout page they are omitted.
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

	// Set by the server
	OrganizationName string          `json:"organizationName"`
	TxType           TransactionType `json:"txType"`
	Test             bool            `json:"test"`
	// AvailableCurrencies lists the currency IDs accepted for this payment.
	// Resolve details via the Currencies service.
	AvailableCurrencies []uint64 `json:"availableCurrencies"`

	// Authenticated only — omitted on the public checkout page.
	OrganizationID     uint64 `json:"organizationId,omitempty"`
	FeeBps             uint16 `json:"feeBps,omitempty"`
	OrganizationFeeBps uint16 `json:"organizationFeeBps,omitempty"`
	UserID             uint64 `json:"userId,omitempty"`
	UserName           string `json:"userName,omitempty"`
	CustomerUUID       string `json:"customerUUID,omitempty"`
	// CustomerReference is your own customer reference, if the customer was pre-filled by reference.
	CustomerReference string `json:"customerReference,omitempty"`

	// Subscription-specific fields — zero for payment sessions
	Frequency   uint32 `json:"frequency,omitempty"`
	TrialPeriod uint32 `json:"trialPeriod,omitempty"`
	MinPeriods  uint32 `json:"minPeriods,omitempty"`

	// PAYG-specific — present only on pay-as-you-go sessions (creation currently disabled)
	FreeCredits float64 `json:"freeCredits,omitempty"`
}

// IsSubscription reports whether the session is a subscription (or PAYG) checkout.
func (s *SessionCheckout) IsSubscription() bool {
	return s.Frequency > 0
}

// IsPayment reports whether the session is a one-time payment checkout.
func (s *SessionCheckout) IsPayment() bool {
	return s.Frequency == 0
}

// StatusResponse represents transaction status response data
type StatusResponse struct {
	TransactionUUID string            `json:"transactionUuid"`
	Status          TransactionStatus `json:"status"`
}

// ----- Payment metadata (typed) -----

// PaymentMetadata is the structured metadata attached to a payment or subscription-billing
// record: the fee breakdown, on-chain transaction metadata, and computed amounts.
type PaymentMetadata struct {
	// FeeBps is the QBitFlow platform fee in basis points (1% = 100 bps), deducted from
	// the amount paid; the merchant receives amount - platform fee - organization fee.
	FeeBps          uint16           `json:"feeBps"`
	OrganizationFee *OrganizationFee `json:"organizationFee,omitempty"` // optional additional fee kept by the organization
	ReferralFee     *ReferralFee     `json:"referralFee,omitempty"`
	TxMetadata      TxMetadata       `json:"txMetadata"` // on-chain metadata (populated after confirmation)
	TxAmounts       TxAmountsFull    `json:"txAmounts"`  // computed fee/merchant amounts
}

// OrganizationFee is an additional fee kept by the organization, on top of the platform fee.
type OrganizationFee struct {
	OrganizationID uint64 `json:"organizationId"`
	Organization   string `json:"organization"` // address receiving the fee
	FeeBps         uint16 `json:"feeBps"`
}

// ReferralFee is an optional fee paid to a referrer.
type ReferralFee struct {
	ReferralID uint64    `json:"referralId"`
	Referrer   string    `json:"referrer"` // address receiving the fee
	FeeBps     uint16    `json:"feeBps"`
	Deadline   time.Time `json:"deadline"`
}

// TxMetadata holds on-chain details parsed from the settled transaction.
type TxMetadata struct {
	NetworkFees NetworkFees `json:"networkFees"`
	BlockData   BlockData   `json:"blockData"`
	// MainCurrencyPriceUSD is the native-currency USD price at transaction time, used for
	// accounting on refunds or when the merchant pays the network fees.
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

// TxAmountsUSD holds the per-party amounts in USD.
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
	UUID string `json:"uuid"`
	// Reference is your own reference for the payment, set when the session was created.
	Reference       *string   `json:"reference,omitempty"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`         // USD
	AmountMinUnits  string    `json:"amountMinUnits"` // decimal string, smallest currency units
	ProductID       *uint64   `json:"productId,omitempty"`
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"`
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	CreatedAt       time.Time `json:"createdAt"`

	// Authenticated only.
	OrganizationID uint64           `json:"organizationId,omitempty"`
	UserID         uint64           `json:"userId,omitempty"`
	CustomerUUID   string           `json:"customerUUID,omitempty"`
	Metadata       *PaymentMetadata `json:"metadata,omitempty"`
}

// CombinedPayment represents a single item from the combined payment list
// (may originate from a one-time payment or a subscription billing cycle).
type CombinedPayment struct {
	Source           string           `json:"source"` // "payment" or "subscription_history"
	UUID             string           `json:"uuid"`
	From             string           `json:"from"`
	To               string           `json:"to"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Amount           float64          `json:"amount"`
	AmountMinUnits   string           `json:"amountMinUnits"`
	ProductID        *uint64          `json:"productId,omitempty"`
	CustomerUUID     string           `json:"customerUUID"`
	CurrencyID       uint64           `json:"currencyId"`
	Test             bool             `json:"test"`
	TransactionHash  string           `json:"transactionHash"`
	CreatedAt        time.Time        `json:"createdAt"`
	SubscriptionUUID *string          `json:"subscriptionUUID,omitempty"`
	Metadata         *PaymentMetadata `json:"metadata,omitempty"`
}

// Subscription represents an active or historical subscription
type Subscription struct {
	UUID string `json:"uuid"`
	// Reference is your own reference for the subscription, set when the session was created.
	Reference          *string            `json:"reference,omitempty"`
	From               string             `json:"from"`
	To                 string             `json:"to"`
	ProductID          uint64             `json:"productId"`
	CustomerUUID       string             `json:"customerUUID,omitempty"`
	SubscriptionHash   string             `json:"subscriptionHash"`
	CurrencyID         uint64             `json:"currencyId"`
	Currency           Currency           `json:"currency"`
	Frequency          uint32             `json:"frequency"` // billing interval in seconds
	Allowance          string             `json:"allowance"` // remaining allowance in USD (decimal string)
	SubscriptionStatus SubscriptionStatus `json:"subscriptionStatus"`
	Stopped            bool               `json:"stopped"` // will be cancelled at end of current period

	LastBillingDate         *time.Time `json:"lastBillingDate,omitempty"`
	NextBillingDate         time.Time  `json:"nextBillingDate"`
	MinimumCancellationDate *time.Time `json:"minimumCancellationDate,omitempty"`

	Test      bool      `json:"test"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Authenticated only.
	OrganizationID uint64 `json:"organizationId,omitempty"`
	UserID         uint64 `json:"userId,omitempty"`
}

// SubscriptionHistory represents a single billing event for a subscription
type SubscriptionHistory struct {
	UUID             string    `json:"uuid"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Amount           float64   `json:"amount"`
	AmountMinUnits   string    `json:"amountMinUnits"`
	ProductID        uint64    `json:"productId"`
	SubscriptionUUID string    `json:"subscriptionUUID"`
	CurrencyID       uint64    `json:"currencyId"`
	Currency         Currency  `json:"currency"`
	Test             bool      `json:"test"`
	TransactionHash  string    `json:"transactionHash"`
	CreatedAt        time.Time `json:"createdAt"`

	// Authenticated only.
	OrganizationID uint64           `json:"organizationId,omitempty"`
	UserID         uint64           `json:"userId,omitempty"`
	CustomerUUID   string           `json:"customerUUID,omitempty"`
	Metadata       *PaymentMetadata `json:"metadata,omitempty"`
}

// CursorData represents paginated cursor-based data
type CursorData[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

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
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Email    string `json:"email"`

	PhoneNumber *string `json:"phoneNumber,omitempty"`
	Address     *string `json:"address,omitempty"`
	Reference   *string `json:"reference,omitempty"`

	Test      bool      `json:"test"`
	CreatedAt time.Time `json:"createdAt"`

	// Authenticated only.
	OrganizationID uint64 `json:"organizationId,omitempty"`
	UserID         uint64 `json:"userId,omitempty"` // 0 for organization-level customers
}

// Product represents a product record
type Product struct {
	ID             uint64    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"` // USD
	Reference      string    `json:"reference"`
	IsActive       bool      `json:"isActive"`
	Test           bool      `json:"test"`
	OrganizationID uint64    `json:"organizationId"`
	UserID         uint64    `json:"userId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// User represents a user within an organization
type User struct {
	ID                 uint64     `json:"id"`
	Name               string     `json:"name"`
	LastName           string     `json:"lastName"`
	Email              string     `json:"email"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	OrganizationID     uint64     `json:"organizationId"`
	Role               UserRole   `json:"role"`
	OrganizationFeeBps uint16     `json:"organizationFeeBps"`
	ClaimedAt          *time.Time `json:"claimedAt,omitempty"` // set once the invited user has claimed their account
}

// UserRole represents the role of a user
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

// ApiKey represents an API key record (the secret key value is never returned).
type ApiKey struct {
	ID             uint64     `json:"id"`
	Name           string     `json:"name"`
	OrganizationID uint64     `json:"organizationId"`
	UserID         uint64     `json:"userId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	Role           UserRole   `json:"role"`
	Test           bool       `json:"test"`
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
	RefundStatusRejected RefundStatus = "rejected"
	RefundStatusFailed   RefundStatus = "failed"
)

// RefundEntry represents a refund record
type RefundEntry struct {
	UUID            string       `json:"uuid"`
	TxID            string       `json:"txId"`
	Test            bool         `json:"test"`
	Reason          string       `json:"reason"`
	Status          RefundStatus `json:"status"`
	CreatedAt       time.Time    `json:"createdAt"`
	MerchantMessage string       `json:"merchantMessage,omitempty"`
	RespondedAt     *time.Time   `json:"respondedAt,omitempty"`
	TxHash          string       `json:"txHash,omitempty"`
	AmountMinUnits  string       `json:"amountMinUnits,omitempty"`

	// Authenticated only.
	OrganizationID uint64      `json:"organizationId,omitempty"`
	UserID         uint64      `json:"userId,omitempty"`
	Metadata       *TxMetadata `json:"metadata,omitempty"`
}

// AccountingEvent represents a single line item in an accounting export
type AccountingEvent struct {
	PaymentID               string    `json:"paymentId"`
	PaymentReference        string    `json:"paymentReference"`
	Type                    string    `json:"type"` // "payment", "subscriptionHistory", "refund", "organizationFee", "referralFee"
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

// SessionWebhookResponse represents a webhook payload for a session event
type SessionWebhookResponse struct {
	UUID               string            `json:"uuid"`
	Status             TransactionStatus `json:"status"`
	Session            SessionCheckout   `json:"session"`
	TxType             TransactionType   `json:"txType"`
	ManagementPageLink string            `json:"managementPageLink"` // link to QBitFlow dashboard page for managing the transaction/subscription
}

// SubscriptionStatusTransitionWebhook represents a webhook payload for a subscription status change event
type SubscriptionStatusTransitionWebhook struct {
	SubscriptionUUID string `json:"subscriptionUUID"`
	// SubscriptionReference is your own reference for the subscription, if one was set at creation.
	SubscriptionReference *string            `json:"subscriptionReference,omitempty"`
	PreviousStatus        SubscriptionStatus `json:"previousStatus"`
	CurrentStatus         SubscriptionStatus `json:"currentStatus"`
	UpdatedAt             time.Time          `json:"updatedAt"`
}
