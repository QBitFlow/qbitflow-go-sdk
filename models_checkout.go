package qbitflow

import (
	"fmt"
	"time"
)

// CheckoutSession is a created checkout session: send the customer to Link.
type CheckoutSession struct {
	// Link is the checkout page for the customer.
	Link string `json:"link"`
	// UUID is the session's id (pay@…, sub@…), also the id of the payment or subscription it creates.
	UUID string `json:"uuid"`
	// ExpiresAt is when the checkout can no longer be paid (checkout.expired is sent then).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// CheckoutSessionStatus is a checkout session's status.
type CheckoutSessionStatus struct {
	// UUID is the session's id (pay@…, sub@…).
	UUID string `json:"uuid"`
	// Status is created (waiting for the customer; LastAttempt: their last attempt failed),
	// waitingConfirmation, completed or expired.
	Status CheckoutSessionStatusValue `json:"status"`
	// TxHash is the customer's transaction, once sent (not kept for completed subscriptions).
	TxHash string `json:"txHash,omitempty"`
	// Message is an optional detail, e.g. why the session expired.
	Message string `json:"message,omitempty"`
	// LastAttempt is the customer's last failed attempt: never final, don't cancel the order on it.
	LastAttempt *Attempt `json:"lastAttempt,omitempty"`
}

// PaymentSessionData is a payment checkout session, as checkout.expired sends it.
type PaymentSessionData struct {
	// UUID is the session's id (pay@…, sub@…).
	UUID string `json:"uuid"`
	// Reference is the merchant's reference, set when creating it.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is the product, when it named one.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference is the product's reference, when it named it so.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is the product's name.
	ProductName string `json:"productName,omitempty"`
	// Description is the product's description.
	Description string `json:"description,omitempty"`
	// Price is the product's price in USD.
	Price *float64 `json:"price,omitempty"`
	// Fees is what a payment checkout adds to the price, line by line, in the order the checkout
	// shows them: the merchant's lines, then the processing fee (when the customer pays it).
	// Empty without any, and always on a subscription session.
	Fees []FeeLine `json:"fees,omitempty"`
	// Amount is what the customer pays in USD before the network fee: Price plus every line of
	// Fees (the payment's Amount). The network fee comes on top. nil on a subscription session.
	Amount *float64 `json:"amount,omitempty"`
	// SuccessURL is where the customer goes after paying (placeholders filled).
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is where a customer who leaves the checkout goes.
	CancelURL string `json:"cancelUrl,omitempty"`
	// RedirectURL is the merchant's success page, on QBitFlow's own success page's copy.
	RedirectURL string `json:"redirectUrl,omitempty"`
	// OrganizationName is the organization's name.
	OrganizationName string `json:"organizationName"`
	// UserName is the member who created it, for a member's session.
	UserName string `json:"userName,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// TxType is payment or createSubscription.
	TxType TransactionType `json:"txType"`
	// AvailableCurrencyIDs are the currencies accepted for the payment.
	AvailableCurrencyIDs []uint64 `json:"availableCurrencyIds"`
	// CreatedAt is when the session was opened.
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// ExpiresAt is when it could no longer be paid.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// SubscriptionSessionData is a subscription checkout session, as checkout.expired sends it.
type SubscriptionSessionData struct {
	PaymentSessionData

	// Frequency is how often it bills.
	Frequency Duration `json:"frequency"`
	// TrialPeriod is the free trial; nil without one.
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to; nil without a minimum.
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
	// UpgradingFromTrial is true when it upgrades a trial.
	UpgradingFromTrial *bool `json:"upgradingFromTrial,omitempty"`
}

// CreatePaymentSessionParams creates a one-time payment checkout session
// (CheckoutSessions.CreatePayment). Name the product with exactly one of ProductUUID,
// ProductReference, or an inline product (ProductName + Price, Description optional).
type CreatePaymentSessionParams struct {
	// Reference is your reference for the payment (an order id): unique per space, 1 to 100 of
	// A-Z a-z 0-9 . _ : @ -.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is one of the space's products.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference names the product by its reference instead.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is an inline product's name (2 to 100 characters).
	ProductName string `json:"productName,omitempty"`
	// Description is an inline product's description (2 to 500 characters), optional.
	Description string `json:"description,omitempty"`
	// Price is an inline product's price in USD, above 0 (at most 5 in test mode).
	Price float64 `json:"price,omitempty"`
	// SuccessURL is where the customer goes after paying ({{UUID}}: the session's id;
	// {{TRANSACTION_TYPE}}: payment or createSubscription).
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is where the customer goes after a cancelled or failed payment.
	CancelURL string `json:"cancelUrl,omitempty"`
	// CustomerUUID is one of the space's customers.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer, kept on the payment.
	CustomerReference string `json:"customerReference,omitempty"`
	// ExpiresInMinutes is the session's lifetime, 10 to 1440 (0 = the default).
	ExpiresInMinutes uint32 `json:"expiresInMinutes,omitempty"`

	// Fees adds amounts to the price (a tax, shipping, QBitFlow's processing fee), shown to the
	// customer line by line and paid with it; nil for none. The session's and the payment's
	// Amount is the price plus every line, and QBitFlow's fee is taken on it. The test-mode
	// cap ($5) applies to that amount, fees included (400 validation_failed on fees).
	Fees *CheckoutFees `json:"fees,omitempty"`
}

// CheckoutFees is what a one-time payment's checkout adds to its price
// (CreatePaymentSessionParams.Fees). Subscription checkouts take no fees.
type CheckoutFees struct {
	// ProcessingFee has the customer pay QBitFlow's processing fee on top: a last line
	// {Type: processingFee, Label: "Processing fee"}, computed by QBitFlow on the price and your
	// lines at your platform fee p (never below 0.5 %) as base × p / (1 − p), rounded up to the
	// cent. QBitFlow's fee is also taken on that line, so this gross-up leaves you at least the
	// price and your lines, as if no fee were taken ($100 + $20 VAT at 1.5 % → a $1.83 fee,
	// $121.83 charged, $120.00 kept). A marketplace's organization fee is not passed on.
	// nil: the space's checkout.customerPaysProcessingFee setting decides (off by default; set
	// in the dashboard); false turns it off for this checkout.
	ProcessingFee *bool `json:"processingFee,omitempty"`
	// Items are up to 10 lines of your own, shown in this order before the processing fee.
	Items []FeeItem `json:"items,omitempty"`
}

// FeeItem is a line of the merchant's own on a checkout (CheckoutFees.Items).
type FeeItem struct {
	// Label is the line's name, as the checkout shows it: one line, 1 to 40 characters
	// ("VAT (20%)", "Shipping").
	Label string `json:"label"`
	// Description is more about the line, at most 200 characters; optional.
	Description string `json:"description,omitempty"`
	// AmountUSD is the line's amount in USD: above 0, at most 1,000,000, with at most 2 decimals
	// (sent as a JSON number).
	AmountUSD float64 `json:"amountUsd"`
}

// maxFeeItems is the most lines CheckoutFees.Items may have.
const maxFeeItems = 10

// maxFeeItemUSD is the largest FeeItem.AmountUSD.
const maxFeeItemUSD = 1_000_000

// validate checks the fees (behaviour §8): at most 10 lines; each label the name rule, 1 to 40;
// each description the text rule, at most 200; each amount in USD above 0, at most 1,000,000,
// with at most 2 decimals.
func (f *CheckoutFees) validate(v *validator, field string) {
	if f == nil {
		return
	}
	if len(f.Items) > maxFeeItems {
		v.add(field+".items", "must have at most %d lines", maxFeeItems)
	}
	for i, item := range f.Items {
		path := fmt.Sprintf("%s.items[%d]", field, i)
		if v.required(path+".label", item.Label) {
			v.name(path+".label", item.Label, 1, 40)
		}
		v.text(path+".description", item.Description, 0, 200)
		v.usd(path+".amountUsd", item.AmountUSD, maxFeeItemUSD)
	}
}

// Validate checks the params (behaviour §8): the product choice, names, texts, references,
// URLs, the price, the lifetime and the fees.
func (p *CreatePaymentSessionParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	p.validate(&v)
	return v.err()
}

func (p *CreatePaymentSessionParams) validate(v *validator) {
	v.reference("reference", p.Reference)
	v.uuid("productUuid", p.ProductUUID)
	v.reference("productReference", p.ProductReference)
	v.name("productName", p.ProductName, 2, 100)
	v.text("description", p.Description, 2, 500)
	v.url("successUrl", p.SuccessURL)
	v.url("cancelUrl", p.CancelURL)
	v.uuid("customerUuid", p.CustomerUUID)
	v.reference("customerReference", p.CustomerReference)
	if p.ExpiresInMinutes != 0 {
		v.intRange("expiresInMinutes", int64(p.ExpiresInMinutes), 10, 1440)
	}

	// Exactly one of: productUuid, productReference, an inline product.
	inline := p.ProductName != "" || p.Price != 0 || p.Description != ""
	chosen := 0
	for _, set := range []bool{p.ProductUUID != "", p.ProductReference != "", inline} {
		if set {
			chosen++
		}
	}
	switch {
	case chosen != 1:
		v.add("productUuid", "or productReference, or an inline product (productName and price), is required: exactly one of them")
	case inline:
		if p.ProductName == "" {
			v.add("productName", "is required for an inline product")
		}
		if p.Price == 0 {
			v.add("price", "is required for an inline product")
		} else {
			v.price("price", p.Price)
		}
	}
	p.Fees.validate(v, "fees")
}

// CreateSubscriptionSessionParams creates a subscription checkout session
// (CheckoutSessions.CreateSubscription): the payment session's fields, plus the terms, each
// optional over a subscription product's.
type CreateSubscriptionSessionParams struct {
	// Reference is your reference for the subscription: unique per space.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is one of the space's products.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference names the product by its reference instead.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is an inline product's name (2 to 100 characters).
	ProductName string `json:"productName,omitempty"`
	// Description is an inline product's description (2 to 500 characters), optional.
	Description string `json:"description,omitempty"`
	// Price is an inline product's price per period in USD, above 0.
	Price float64 `json:"price,omitempty"`
	// SuccessURL is where the customer goes after subscribing.
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is where the customer goes after a cancelled or failed checkout.
	CancelURL string `json:"cancelUrl,omitempty"`
	// CustomerUUID is one of the space's customers.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer.
	CustomerReference string `json:"customerReference,omitempty"`
	// ExpiresInMinutes is the session's lifetime, 10 to 1440 (0 = the default).
	ExpiresInMinutes uint32 `json:"expiresInMinutes,omitempty"`

	// Frequency is how often it bills: at least 1 unit, at most 1 year (the API also requires at
	// least 1 hour in live mode, 5 minutes in test mode). Optional over a subscription product's.
	Frequency *Duration `json:"frequency,omitempty"`
	// TrialPeriod is a free trial before the first bill ({Value: 0}: none, removing a product's).
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before cancelling, at most 1000 (0: none).
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
}

// Validate checks the params: the payment session's rules plus the terms.
func (p *CreateSubscriptionSessionParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	session := CreatePaymentSessionParams{
		Reference: p.Reference, ProductUUID: p.ProductUUID, ProductReference: p.ProductReference,
		ProductName: p.ProductName, Description: p.Description, Price: p.Price,
		SuccessURL: p.SuccessURL, CancelURL: p.CancelURL, CustomerUUID: p.CustomerUUID,
		CustomerReference: p.CustomerReference, ExpiresInMinutes: p.ExpiresInMinutes,
	}
	session.validate(&v)
	terms := SubscriptionTermsParams{Frequency: p.Frequency, TrialPeriod: p.TrialPeriod, MinPeriods: p.MinPeriods}
	terms.validate(&v, "", false)
	return v.err()
}
