package qbitflow

import (
	"net/url"
	"time"
)

// Product is something a merchant sells: a one-time product, or a subscription product with its
// terms. Its payment link opens a checkout of it.
type Product struct {
	// UUID is the product's id (a UUIDv7).
	UUID string `json:"uuid"`
	// Name is its name.
	Name string `json:"name"`
	// Description is its description; "" (absent) without one.
	Description string `json:"description,omitempty"`
	// PriceUSD is its price in USD, a decimal string ("12.5"): what a checkout of the product
	// charges before its fees (per period for a subscription product).
	PriceUSD string `json:"priceUsd"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// IsActive is false for a hidden product: out of Products.List unless IncludeHidden, still
	// readable and editable by uuid. Hidden are the inline products of subscription checkouts,
	// those of a removed member or a closed organization, and those hidden with Update.
	IsActive bool `json:"isActive"`
	// Reference is the merchant's reference, unique per space and mode (the uuid when none was
	// given).
	Reference string `json:"reference"`
	// Metadata is the merchant's own data on the product (never copied onto its checkouts,
	// payments or subscriptions); empty without any.
	Metadata map[string]string `json:"metadata"`
	// Subscription is a subscription product's terms (the defaults of its subscription
	// checkouts); nil for a one-time product.
	Subscription *SubscriptionTerms `json:"subscription,omitempty"`
	// PaymentLink is the product's shareable link: opening it starts a checkout of the product.
	PaymentLink string `json:"paymentLink,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// SubscriptionTerms are a subscription product's terms: billed every Frequency, after an
// optional free trial, for at least an optional number of periods.
type SubscriptionTerms struct {
	// Frequency is how often it bills (e.g. {1, months}: 30 days).
	Frequency Duration `json:"frequency"`
	// TrialPeriod is a free trial before the first bill; nil without one.
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before they can cancel; nil without a
	// minimum.
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
}

// SubscriptionTermsParams sets subscription terms (a product's, or a subscription checkout's
// over its product's): each field given replaces the base's, each field left nil keeps it.
type SubscriptionTermsParams struct {
	// Frequency is how often it bills: at least 1 unit, at most 1 year (the API also requires at
	// least 1 hour in live mode, 5 minutes in test mode).
	Frequency *Duration `json:"frequency,omitempty"`
	// TrialPeriod is a free trial before the first bill ({Value: 0}: none, removing one).
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before cancelling, at most 1000 (0: none,
	// removing one).
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
}

// validate checks the terms; prefix is the wire path ("subscription." on products).
func (p *SubscriptionTermsParams) validate(v *validator, prefix string, frequencyRequired bool) {
	if p.Frequency != nil {
		v.duration(prefix+"frequency", *p.Frequency, true)
	} else if frequencyRequired {
		v.add(prefix+"frequency", "is required")
	}
	if p.TrialPeriod != nil {
		v.duration(prefix+"trialPeriod", *p.TrialPeriod, false)
	}
	if p.MinPeriods != nil {
		v.intRange(prefix+"minPeriods", int64(*p.MinPeriods), 0, 1000)
	}
}

// CreateProductParams creates a product (Products.Create).
type CreateProductParams struct {
	// Name is required: 2 to 100 characters, one line.
	Name string `json:"name"`
	// Description is optional: 2 to 500 characters.
	Description string `json:"description,omitempty"`
	// PriceUSD is required: the price in USD, a decimal string ("12.50"): above 0, at most 2
	// decimals, at most 1,000,000. The API also caps it at "5" in test mode (400 with
	// details.max "5") and requires more than "0.25" for a live subscription product (details.min).
	PriceUSD string `json:"priceUsd"`
	// Reference is your reference, unique per space and mode (1 to 100 of A-Z a-z 0-9 . _ : @ -);
	// the product's uuid when "".
	Reference string `json:"reference,omitempty"`
	// Subscription makes it a subscription product (Frequency required); nil: a one-time product.
	Subscription *SubscriptionTermsParams `json:"subscription,omitempty"`
	// Metadata is your own data on the product (an id in your catalog): at most 50 keys of 1 to 40
	// characters among A-Z a-z 0-9 _ - . :, values of at most 500 characters without control or
	// bidirectional characters. A key set to "" is not stored. Never put secrets or personal data
	// in it.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the params.
func (p *CreateProductParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("name", p.Name) {
		v.name("name", p.Name, 2, 100)
	}
	v.text("description", p.Description, 2, 500)
	if v.required("priceUsd", p.PriceUSD) {
		v.usd("priceUsd", p.PriceUSD, maxUSD)
	}
	v.reference("reference", p.Reference)
	if p.Subscription != nil {
		p.Subscription.validate(&v, "subscription.", true)
	}
	v.metadata("metadata", p.Metadata)
	return v.err()
}

// UpdateProductParams updates a product (Products.Update): unset fields are unchanged. The
// reference cannot change.
type UpdateProductParams struct {
	// Name is the new name, 2 to 100 characters ("" = unchanged).
	Name string `json:"name,omitempty"`
	// Description is the new description (2 to 500 characters); a pointer to "" clears it, nil
	// leaves it unchanged.
	Description *string `json:"description,omitempty"`
	// PriceUSD is the new price in USD, a decimal string as at creation ("" = unchanged): for new
	// checkouts and subscribers only (a subscription keeps the price it was subscribed at).
	PriceUSD string `json:"priceUsd,omitempty"`
	// IsActive false hides the product from Products.List, true lists it again.
	IsActive *bool `json:"isActive,omitempty"`
	// Subscription sets (or changes) the subscription terms, field by field, for new subscribers
	// only; a one-time product needs Frequency.
	Subscription *SubscriptionTermsParams `json:"subscription,omitempty"`
	// RemoveSubscription makes it a one-time product again (not with Subscription).
	RemoveSubscription bool `json:"removeSubscription,omitempty"`
	// Metadata is merged into the product's: a key set to a value sets it, a key set to ""
	// deletes it, keys not named are kept (nil: unchanged). At most 50 keys once merged.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the params.
func (p *UpdateProductParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.name("name", p.Name, 2, 100)
	if p.Description != nil {
		v.text("description", *p.Description, 2, 500)
	}
	if p.PriceUSD != "" {
		v.usd("priceUsd", p.PriceUSD, maxUSD)
	}
	if p.Subscription != nil {
		p.Subscription.validate(&v, "subscription.", false)
	}
	v.exclusive("subscription", p.Subscription != nil, "removeSubscription", p.RemoveSubscription)
	v.metadata("metadata", p.Metadata)
	return v.err()
}

// ProductListParams filters Products.List.
type ProductListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
	// IncludeHidden lists the hidden products too (IsActive false).
	IncludeHidden bool
	// Subscription keeps only the subscription products (true) or the one-time ones (false).
	Subscription *bool
	// Reference keeps only the product with this reference, hidden or not (a page of 0 or 1
	// product; Products.GetByReference reads it).
	Reference string
}

// Validate checks the filters.
func (p *ProductListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	v.reference("reference", p.Reference)
	return v.err()
}

func (p *ProductListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	setBool(q, "includeHidden", p.IncludeHidden)
	setBoolPtr(q, "subscription", p.Subscription)
	setString(q, "reference", p.Reference)
	return q
}
