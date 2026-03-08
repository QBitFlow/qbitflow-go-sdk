package qbmodels

import "time"

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeOneTimePayment             TransactionType = "payment"
	TransactionTypeCreateSubscription         TransactionType = "createSubscription"
	TransactionTypeCancelSubscription         TransactionType = "cancelSubscription"
	TransactionTypeExecuteSubscriptionPayment TransactionType = "executeSubscription"
	TransactionTypeCreatePAYGSubscription     TransactionType = "createPAYGSubscription"
	TransactionTypeCancelPAYGSubscription     TransactionType = "cancelPAYGSubscription"
	TransactionTypeIncreaseAllowance          TransactionType = "increaseAllowance"
	TransactionTypeUpdateMaxAmount            TransactionType = "updateMaxAmount"
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

// TransactionStatus represents the status details of a transaction
type TransactionStatus struct {
	Type    TransactionType        `json:"type"`
	Status  TransactionStatusValue `json:"status"`
	TxHash  *string                `json:"txHash,omitempty"`
	Message *string                `json:"message,omitempty"`
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
	// After a grace period, the subscription will be automatically cancelled if not resolved.
	SubscriptionStatusPending SubscriptionStatus = "pending"

	// SubscriptionStatusTrial indicates a subscription in trial period.
	SubscriptionStatusTrial SubscriptionStatus = "trial"

	// SubscriptionStatusTrialExpired indicates a subscription whose trial period has expired.
	// The user has a grace period to upgrade to a paid subscription before it is automatically cancelled.
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
)

// Duration represents a time duration
type Duration struct {
	Value int          `json:"value"`
	Unit  DurationUnit `json:"unit"`
}

// Currency represents a cryptocurrency
type Currency struct {
	ID             uint64    `json:"id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	Decimals       int       `json:"decimals"`
	Address        string    `json:"address,omitempty"`
	MainCurrencyID *uint64   `json:"mainCurrencyId,omitempty"`
	MainCurrency   *Currency `json:"mainCurrency,omitempty"`
	Test           bool      `json:"test"`
}

// SubscriptionOptions represents subscription-specific options
type SubscriptionOptions struct {
	SubscriptionType SubscriptionType `json:"subscriptionType"`
	Frequency        int              `json:"frequency"`
	TrialPeriod      *int             `json:"trialPeriod,omitempty"`
	FreeCredits      *float64         `json:"freeCredits,omitempty"`
	MinPeriods       *int             `json:"minPeriods,omitempty"`
}

// Session represents a payment or subscription session
type Session struct {
	UUID                string               `json:"uuid"`
	ProductID           *int                 `json:"productId,omitempty"`
	ProductName         string               `json:"productName"`
	Description         string               `json:"description"`
	Price               float64              `json:"price"`
	OrganizationName    string               `json:"organizationName"`
	SuccessURL          *string              `json:"successUrl,omitempty"`
	CancelURL           *string              `json:"cancelUrl,omitempty"`
	CustomerUUID        string               `json:"customerUUID"`
	Options             *SubscriptionOptions `json:"options,omitempty"`
	AvailableCurrencies []Currency           `json:"availableCurrencies,omitempty"`
}

// SubscriptionType represents the type of subscription
type SubscriptionType string

const (
	SubscriptionTypeSubscription SubscriptionType = "subscription"
	SubscriptionTypePAYG         SubscriptionType = "payAsYouGo"
)

// CreateSubscriptionOptions represents options for creating a subscription
type CreateSubscriptionOptions struct {
	SubscriptionType SubscriptionType `json:"subscriptionType" binding:"required"`
	Frequency        Duration         `json:"frequency" binding:"required"`
	TrialPeriod      *Duration        `json:"trialPeriod,omitempty"`
	FreeCredits      *float64         `json:"freeCredits,omitempty"`
	MinPeriods       *uint32          `json:"minPeriods,omitempty"`
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	ProductID    *uint64                    `json:"productId,omitempty"`
	ProductName  *string                    `json:"productName,omitempty"`
	Description  *string                    `json:"description,omitempty"`
	Price        *float64                   `json:"price,omitempty"`
	SuccessURL   *string                    `json:"successUrl,omitempty"`
	CancelURL    *string                    `json:"cancelUrl,omitempty"`
	WebhookURL   *string                    `json:"webhookUrl,omitempty"`
	CustomerUUID *string                    `json:"customerUuid,omitempty"`
	Options      *CreateSubscriptionOptions `json:"options,omitempty"`
}

// LinkResponse represents a response containing a payment/subscription link
type LinkResponse struct {
	UUID      string `json:"uuid"`
	Link      string `json:"link"`
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
}

// StatusLinkResponse represents a status response. It contains the websocket link to monitor the transaction status
type StatusLinkResponse struct {
	Message    string `json:"message"`
	StatusLink string `json:"statusLink"`
}

// SessionWebhookResponse represents a webhook response for a session
type SessionWebhookResponse struct {
	UUID    string            `json:"uuid"`
	Status  TransactionStatus `json:"status"`
	Session Session           `json:"session"`
}

// Only to satisfy the WebhookData interface to be passed to the webhook verification method
func (s SessionWebhookResponse) GetWebhookData() any {
	return s
}

// StatusResponse represents transaction status response data
type StatusResponse struct {
	TransactionUUID string            `json:"transactionUuid"`
	Status          TransactionStatus `json:"status"`
}

// Payment represents a one-time payment
type Payment struct {
	UUID            string    `json:"uuid"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	ProductID       *int      `json:"productId,omitempty"`
	CustomerUUID    string    `json:"customerUUID"`
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"`
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CombinedPayment represents a combined payment (from one-time or subscription)
type CombinedPayment struct {
	UUID            string    `json:"uuid"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	ProductID       *int      `json:"productId,omitempty"`
	CustomerUUID    string    `json:"customerUUID"`
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"`
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	CreatedAt       time.Time `json:"createdAt"`

	Source           string  `json:"source"` // "one-time" or "subscription"
	SubscriptionUUID *string `json:"subscriptionUUID,omitempty"`
}

// Subscription represents a subscription
type Subscription struct {
	UUID               string             `json:"uuid"`
	From               string             `json:"from"`
	To                 string             `json:"to"`
	ProductID          int                `json:"productId"`
	CustomerUUID       string             `json:"customerUUID"`
	SubscriptionHash   string             `json:"subscriptionHash"`
	CurrencyID         uint64             `json:"currencyId"`
	Currency           Currency           `json:"currency"`
	Frequency          uint32             `json:"frequency"`
	Allowance          float32            `json:"allowance"` // Remaining allowance for this subscription
	SubscriptionStatus SubscriptionStatus `json:"subscriptionStatus"`
	Stopped            bool               `json:"stopped"` // If stopped, then will be cancelled at the end of the billing period

	LastBillingDate         *time.Time `json:"lastBillingDate,omitempty"`
	NextBillingDate         time.Time  `json:"nextBillingDate"`
	MinimumCancellationDate *time.Time `json:"minimumCancellationDate,omitempty"`

	Test bool `json:"test"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PayAsYouGoSubscription represents a pay-as-you-go subscription
type PayAsYouGoSubscription struct {
	UUID               string             `json:"uuid"`
	From               string             `json:"from"`
	To                 string             `json:"to"`
	ProductID          int                `json:"productId"`
	CustomerUUID       string             `json:"customerUUID"`
	SubscriptionHash   string             `json:"subscriptionHash"`
	CurrencyID         uint64             `json:"currencyId"`
	Currency           Currency           `json:"currency"`
	Frequency          uint32             `json:"frequency"`
	Allowance          float32            `json:"allowance"` // Remaining allowance for this subscription
	SubscriptionStatus SubscriptionStatus `json:"subscriptionStatus"`
	Stopped            bool               `json:"stopped"` // If stopped, then will be cancelled at the end of the billing period

	LastBillingDate         *time.Time `json:"lastBillingDate,omitempty"`
	NextBillingDate         time.Time  `json:"nextBillingDate"`
	MinimumCancellationDate *time.Time `json:"minimumCancellationDate,omitempty"`

	UnitsCurrentPeriod   float64 `json:"unitsCurrentPeriod"`   // Current number of units used for the current period. A unit can be a transaction, a request, etc. To get the price the user has to pay, you compute UnitsCurrentPeriod * Product.Price. The product price is for 1 unit. For example, if the user made 1500 requests and the product price is $0.01, then the user has to pay 1500 * 0.01 = $15.00
	MaxSpendingPerPeriod float64 `json:"maxSpendingPerPeriod"` // Maximum spending allowed for the current period, in USD
	FreeCredits          float64 `json:"freeCredits"`          // Free credits for the subscription, in USD. This is used to provide free credits to the user, for example for a trial period or a promotion. The free credits are deducted from the total amount to pay before billing

	Test bool `json:"test"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SubscriptionHistory represents a payment history for a subscription
type SubscriptionHistory struct {
	UUID            string    `json:"uuid"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	ProductID       uint64    `json:"productId"`
	CustomerUUID    string    `json:"customerUUID"`
	CurrencyID      uint64    `json:"currencyId"`
	Currency        Currency  `json:"currency"`
	Test            bool      `json:"test"`
	TransactionHash string    `json:"transactionHash"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CursorData represents paginated cursor-based data
type CursorData[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

func (c *CursorData[T]) HasMore() bool {
	return c.NextCursor != nil
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// IncreaseUnitsRequest represents a request to increase units for PAYG subscription
type IncreaseUnitsRequest struct {
	SubscriptionUUID string  `json:"subscriptionUUID"`
	IncreaseByAmount float64 `json:"increaseByAmount"`
}

type Customer struct {
	UUID     string `json:"uuid"`     // Customer UUID
	Name     string `json:"name"`     // Customer's name
	LastName string `json:"lastName"` // Customer's last name
	Email    string `json:"email"`    // Customer's email address

	// Optional fields
	PhoneNumber *string `json:"phoneNumber,omitempty"` // Customer's phone number
	Address     *string `json:"address,omitempty"`     // Customer's physical address

	Reference *string `json:"reference,omitempty"` // Optional reference for the customer information (e.g., order ID, user ID, etc.)

	CreatedAt time.Time `json:"createdAt"` // Timestamp for when this customer information was created
}

type Product struct {
	ID          uint64    `json:"id"` // Product ID
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"` // Price of the product in USD
	CreatedAt   time.Time `json:"createdAt"`
	IsActive    bool      `json:"isActive"` // Default value is true
	Reference   *string   `json:"reference,omitempty"`
}

type User struct {
	ID                 uint64    `json:"id"` // User ID
	Name               string    `json:"name"`
	LastName           string    `json:"lastName"`
	Email              string    `json:"email"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	OrganizationID     uint64    `json:"organizationId"`
	Role               UserRole  `json:"role"`               // Default role is user, and it cannot be modified
	OrganizationFeeBps uint16    `json:"organizationFeeBps"` // Default organization fee is 0 bps
}

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type ApiKey struct {
	ID     uint64 `json:"id"`     // API Key ID
	Name   string `json:"name"`   // Name of the API key
	UserID uint64 `json:"userId"` // Foreign key to User, can be null if the API key is not associated with a user

	CreatedAt time.Time  `json:"createdAt"`           // Timestamp when the API key was created
	ExpiresAt *time.Time `json:"expiresAt,omitempty"` // Timestamp when the API key expires
	Role      UserRole   `json:"role"`                // Default role is 'user'
	Test      bool       `json:"test"`                // Whether the API key is a test key or not
}
