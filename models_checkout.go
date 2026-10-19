package qbitflow

import (
	"fmt"
	"time"
)

// CreatedCheckoutSession is a checkout session just opened: the page to send the customer to,
// its id, and how long it can be paid.
type CreatedCheckoutSession struct {
	// Link is the checkout page for the customer: <the merchant's checkout base URL>/checkout?uuid=<uuid>.
	Link string `json:"link"`
	// UUID is the session's id (pay@…, sub@…), also the id of the payment or subscription it creates.
	UUID string `json:"uuid"`
	// ExpiresAt is when the checkout can no longer be paid (checkout.expired is sent then, if it
	// wasn't paid).
	ExpiresAt time.Time `json:"expiresAt"`
}

// CheckoutSessionStatus is where a checkout session stands, for the merchant that created it
// (CheckoutSessions.Get and Expire).
type CheckoutSessionStatus struct {
	// UUID is the session's id (pay@…, sub@…), also the id of the payment or subscription it creates.
	UUID string `json:"uuid"`
	// Status is created (waiting for the customer; LastAttempt: their last attempt failed, they may
	// try again until the session expires), waitingConfirmation (a transaction was sent), or the
	// final completed and expired.
	Status CheckoutSessionStatusValue `json:"status"`
	// TxHash is the customer's transaction, once sent (not kept for completed subscriptions).
	TxHash string `json:"txHash,omitempty"`
	// Message is an optional detail, e.g. why the session expired.
	Message string `json:"message,omitempty"`
	// LastAttempt is the customer's last failed attempt (its code, reason and time): never final,
	// don't cancel the order on it; wait for completed (payment.completed, subscription.created) or
	// expired (checkout.expired).
	LastAttempt *Attempt `json:"lastAttempt,omitempty"`
	// ExpiresAt is when the checkout can no longer be paid, while it is open (on Expire's answer:
	// when it was expired); nil once completed, or expired at its time.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// PaymentSessionData is a one-time payment's checkout session, as checkout.expired sends it
// (flat; a subscription's is SubscriptionSessionData). No customer is named: match it on UUID,
// Reference or Metadata.
type PaymentSessionData struct {
	// UUID is the session's id (pay@…, sub@…): the id the payment or subscription would have had.
	UUID string `json:"uuid"`
	// Reference is the merchant's reference of the checkout (an order id), set when creating it.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is the product paid for; "" for an inline product of a payment.
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference is the product's reference, when the checkout named it so.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is what is paid for: the product's name as the checkout was opened (or the
	// inline one).
	ProductName string `json:"productName,omitempty"`
	// Description is the product's description as the checkout was opened; "" without one.
	Description string `json:"description,omitempty"`
	// PriceUSD is the product's price in USD, a decimal string.
	PriceUSD string `json:"priceUsd"`
	// Fees is what the checkout adds to the price, line by line, in the order the checkout shows
	// them: the merchant's lines, then the processing fee (when the customer pays it). A
	// subscription's are charged with every bill. Empty without any.
	Fees []FeeLine `json:"fees"`
	// AmountUSD is what the customer pays in USD before the network fee, a decimal string:
	// PriceUSD plus every line of Fees (the payment's AmountUSD, or each bill's of a subscription).
	AmountUSD string `json:"amountUsd"`
	// SuccessURL is the merchant's page for the customer after paying (placeholders filled).
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is the merchant's page for a customer who leaves the checkout (placeholders filled).
	CancelURL string `json:"cancelUrl,omitempty"`
	// RedirectURL is the merchant's success page on the hosted page's copy only: absent from
	// checkout.expired.
	RedirectURL string `json:"redirectUrl,omitempty"`
	// OrganizationName is the merchant's organization, as the checkout shows it.
	OrganizationName string `json:"organizationName"`
	// UserName is the member the checkout pays (a marketplace's seller), by name; "" on the
	// organization's own checkouts.
	UserName string `json:"userName,omitempty"`
	// Test is true for a checkout paid on test networks.
	Test bool `json:"test"`
	// TxType is the session's kind: payment or createSubscription.
	TxType TransactionType `json:"txType"`
	// AvailableCurrencies are the currencies the customer could pay with.
	AvailableCurrencies []Currency `json:"availableCurrencies"`
	// Metadata is the merchant's own data on the checkout; empty without any.
	Metadata map[string]string `json:"metadata"`
	// CreatedAt is when the session was opened.
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// ExpiresAt is when it could no longer be paid (when the merchant expired it, if so).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// SubscriptionSessionData is a subscription's checkout session, as checkout.expired sends it:
// the payment session's fields and its terms.
type SubscriptionSessionData struct {
	PaymentSessionData

	// Frequency is how often it would have billed.
	Frequency Duration `json:"frequency"`
	// TrialPeriod is the free trial; nil without one.
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer would have committed to; nil without a minimum.
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
	// UpgradingFromTrial is true for a checkout confirming a free trial already started (always
	// false in checkout.expired: such a session sends nothing when it expires).
	UpgradingFromTrial bool `json:"upgradingFromTrial"`
}

// CreatePaymentSessionParams creates a one-time payment checkout session
// (CheckoutSessions.CreatePayment). Name the product with exactly one of ProductUUID,
// ProductReference, or an inline product (ProductName + PriceUSD, Description optional).
type CreatePaymentSessionParams struct {
	// Reference is your reference for the payment (an order or invoice id), kept on it: unique per
	// space (open sessions included), 1 to 100 of A-Z a-z 0-9 . _ : @ -.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is one of the space's products (unknown: 404).
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference names the product by its reference instead (unknown: 404).
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is an inline product's name (2 to 100 characters, one line): the payment keeps
	// it as a snapshot, without ProductUUID.
	ProductName string `json:"productName,omitempty"`
	// Description is an inline product's description (2 to 500 characters), optional.
	Description string `json:"description,omitempty"`
	// PriceUSD is an inline product's price in USD, a decimal string ("12.50"): above 0, at most 2
	// decimals, at most 1,000,000. The API caps it at "5" in test mode (details.max "5").
	PriceUSD string `json:"priceUsd,omitempty"`
	// SuccessURL is where the customer goes after paying, through QBitFlow's page for the payment
	// (PlaceholderUUID: the session's id; PlaceholderTransactionType: payment or
	// createSubscription). Default: the space's checkout.defaults setting. https only in live
	// mode. Reaching it is no proof of payment: fulfil on payment.completed.
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is where the customer goes after a cancelled or failed payment (same placeholders).
	CancelURL string `json:"cancelUrl,omitempty"`
	// CustomerUUID is one of the space's customers (unknown: 404); the checkout then asks the
	// customer nothing.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your own reference of the customer, kept on the payment: the space's
	// customer with this reference is linked if there is one (none is created). The checkout then
	// asks the customer nothing.
	CustomerReference string `json:"customerReference,omitempty"`
	// ExpiresInMinutes is how long the checkout can be paid, 10 to 1440 (0 = the default
	// lifetime; never longer than it).
	ExpiresInMinutes uint32 `json:"expiresInMinutes,omitempty"`
	// Metadata is your own data on the checkout (an order id, a CRM key): at most 50 keys of 1 to
	// 40 characters among A-Z a-z 0-9 _ - . :, values of at most 500 characters without control or
	// bidirectional characters ("" values are dropped). Copied onto the payment and into its
	// webhooks (payment.completed, checkout.expired); never shown to the customer. Never put
	// secrets or personal data in it.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Fees adds amounts to the price (a tax, shipping, QBitFlow's processing fee), shown to the
	// customer line by line and paid with it; nil for none. The session's and the payment's
	// AmountUSD is the price plus every line, and QBitFlow's fee is taken on it. The test-mode
	// cap ("5") applies to that amount, fees included (400 validation_failed on fees).
	Fees *CheckoutFees `json:"fees,omitempty"`
}

// CheckoutFees is what a checkout adds to its price: a one-time payment's
// (CreatePaymentSessionParams.Fees), or a subscription's (CreateSubscriptionSessionParams.Fees,
// each of its bills then charging the price and the same lines).
type CheckoutFees struct {
	// ProcessingFee has the customer pay QBitFlow's processing fee on top: a last line
	// {Type: processingFee, Label: "Processing fee"}, computed by QBitFlow on the price and your
	// lines at your platform fee p (never below 0.5 %) as base × p / (1 − p), rounded up to the
	// cent. QBitFlow's fee is also taken on that line, so this gross-up leaves you at least the
	// price and your lines, as if no fee were taken ($100 + $20 VAT at 0.99 % → a $1.20 fee,
	// $121.20 charged, $120.00 kept). A marketplace's organization fee is not passed on.
	// nil: the space's checkout.customerPaysProcessingFee setting decides (off by default; set in
	// the dashboard); false turns it off for this checkout.
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
	// AmountUSD is the line's amount in USD, a decimal string ("4.99"): above 0, at most
	// 1,000,000, with at most 2 decimals.
	AmountUSD string `json:"amountUsd"`
}

// maxFeeItems is the most lines CheckoutFees.Items may have.
const maxFeeItems = 10

// validate checks the fees (behaviour §8): at most 10 lines; each label the name rule, 1 to 40;
// each description the text rule, at most 200; each amount the usd rule.
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
		if v.required(path+".amountUsd", item.AmountUSD) {
			v.usd(path+".amountUsd", item.AmountUSD, maxUSD)
		}
	}
}

// Validate checks the params (behaviour §8): the product choice, names, texts, references,
// URLs, the price, the lifetime, the metadata and the fees.
func (p *CreatePaymentSessionParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	p.validate(&v)
	p.Fees.validate(&v, "fees")
	return v.err()
}

// validate checks every field but the fees (reported last, behaviour §8).
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
	v.metadata("metadata", p.Metadata)

	// Exactly one of: productUuid, productReference, an inline product. Any inline field next
	// to productUuid or productReference counts as mixing (behaviour §10).
	inline := p.ProductName != "" || p.PriceUSD != "" || p.Description != ""
	chosen := 0
	for _, set := range []bool{p.ProductUUID != "", p.ProductReference != "", inline} {
		if set {
			chosen++
		}
	}
	switch {
	case chosen != 1:
		v.add("productUuid", "or productReference, or an inline product (productName and priceUsd), is required: exactly one of them")
	case inline:
		if p.ProductName == "" {
			v.add("productName", "is required for an inline product")
		}
		if p.PriceUSD == "" {
			v.add("priceUsd", "is required for an inline product")
		} else {
			v.usd("priceUsd", p.PriceUSD, maxUSD)
		}
	}
}

// CreateSubscriptionSessionParams creates a subscription checkout session
// (CheckoutSessions.CreateSubscription): the payment session's fields, plus the terms, each
// optional over a subscription product's (the merged terms are validated as a product's are;
// without a product's terms, Frequency is required).
type CreateSubscriptionSessionParams struct {
	// Reference is your reference for the subscription (an order or customer id), kept on it:
	// unique per space.
	Reference string `json:"reference,omitempty"`
	// ProductUUID is one of the space's products (a subscription product gives the default terms).
	ProductUUID string `json:"productUuid,omitempty"`
	// ProductReference names the product by its reference instead.
	ProductReference string `json:"productReference,omitempty"`
	// ProductName is an inline product's name (2 to 100 characters): the subscription uses a
	// hidden product, shared by the space's inline products of the same name, description and price.
	ProductName string `json:"productName,omitempty"`
	// Description is an inline product's description (2 to 500 characters), optional.
	Description string `json:"description,omitempty"`
	// PriceUSD is an inline product's price per period in USD, a decimal string: above 0, at most
	// 2 decimals, at most 1,000,000. The API caps it at "5" in test mode and requires more than
	// "0.25" for a live subscription (inline or its product's; details.min).
	PriceUSD string `json:"priceUsd,omitempty"`
	// SuccessURL is where the customer goes after subscribing (placeholders as for CreatePayment).
	SuccessURL string `json:"successUrl,omitempty"`
	// CancelURL is where the customer goes after a cancelled or failed checkout.
	CancelURL string `json:"cancelUrl,omitempty"`
	// CustomerUUID is one of the space's customers.
	CustomerUUID string `json:"customerUuid,omitempty"`
	// CustomerReference is your reference of the customer, kept on the subscription.
	CustomerReference string `json:"customerReference,omitempty"`
	// ExpiresInMinutes is the session's lifetime, 10 to 1440 (0 = the default).
	ExpiresInMinutes uint32 `json:"expiresInMinutes,omitempty"`
	// Metadata is your own data on the checkout, copied onto the subscription (then changed with
	// Subscriptions.Update) and into its webhooks; the rules of CreatePaymentSessionParams.Metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Frequency is how often it bills: at least 1 unit, at most 1 year (the API also requires at
	// least 1 hour in live mode, 5 minutes in test mode). Optional over a subscription product's.
	Frequency *Duration `json:"frequency,omitempty"`
	// TrialPeriod is a free trial before the first bill ({Value: 0}: none, removing a product's).
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before cancelling, at most 1000 (0: none,
	// removing a product's).
	MinPeriods *uint32 `json:"minPeriods,omitempty"`

	// Fees adds amounts to the price (a tax, shipping, QBitFlow's processing fee), shown to the
	// customer line by line and locked with the price: every bill (the first one, at creation,
	// and each next one) charges the price plus every line; nil for none. The subscription keeps
	// them as Subscription.Fees, its PriceUSD staying the price. The maximum per period and the
	// allowance the subscriber signs are sized on that amount, and QBitFlow's fee is taken on it.
	// The test-mode cap ("5") applies to that amount, fees included (400 validation_failed on fees).
	Fees *CheckoutFees `json:"fees,omitempty"`
}

// Validate checks the params: the payment session's rules, the terms, then the fees.
func (p *CreateSubscriptionSessionParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	session := CreatePaymentSessionParams{
		Reference: p.Reference, ProductUUID: p.ProductUUID, ProductReference: p.ProductReference,
		ProductName: p.ProductName, Description: p.Description, PriceUSD: p.PriceUSD,
		SuccessURL: p.SuccessURL, CancelURL: p.CancelURL, CustomerUUID: p.CustomerUUID,
		CustomerReference: p.CustomerReference, ExpiresInMinutes: p.ExpiresInMinutes, Metadata: p.Metadata,
	}
	session.validate(&v)
	terms := SubscriptionTermsParams{Frequency: p.Frequency, TrialPeriod: p.TrialPeriod, MinPeriods: p.MinPeriods}
	terms.validate(&v, "", false)
	p.Fees.validate(&v, "fees")
	return v.err()
}
