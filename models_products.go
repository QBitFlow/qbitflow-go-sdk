package qbitflow

import (
	"net/url"
	"time"
)

// Product is a product: a one-time product, or a subscription product with its terms.
type Product struct {
	// UUID is the product's id.
	UUID string `json:"uuid"`
	// Name is its name.
	Name string `json:"name"`
	// Description is its description ("" when none).
	Description string `json:"description"`
	// Price is its price in USD (per period for a subscription product).
	Price float64 `json:"price"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// IsActive is false for a hidden product (out of Products.List unless IncludeHidden).
	IsActive bool `json:"isActive"`
	// Reference is the merchant's reference (defaults to the uuid).
	Reference string `json:"reference"`
	// Subscription is a subscription product's terms; nil for a one-time product.
	Subscription *SubscriptionTerms `json:"subscription,omitempty"`
	// PaymentLink is the product's payment link.
	PaymentLink string `json:"paymentLink,omitempty"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// SubscriptionTerms are a subscription product's terms.
type SubscriptionTerms struct {
	// Frequency is how often it bills (e.g. {1, months}: 30 days).
	Frequency Duration `json:"frequency"`
	// TrialPeriod is a free trial before the first bill; nil without one.
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before cancelling; nil without a minimum.
	MinPeriods *uint32 `json:"minPeriods,omitempty"`
}

// SubscriptionTermsParams sets a subscription product's terms (each optional on an update).
type SubscriptionTermsParams struct {
	// Frequency is how often it bills: at least 1 unit, at most 1 year (the API also requires at
	// least 1 hour in live mode, 5 minutes in test mode).
	Frequency *Duration `json:"frequency,omitempty"`
	// TrialPeriod is a free trial before the first bill ({Value: 0}: none, removing one).
	TrialPeriod *Duration `json:"trialPeriod,omitempty"`
	// MinPeriods is the periods the customer commits to before cancelling, at most 1000 (0: none).
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
	// Price is required, in USD, above 0 (at most 5 in test mode).
	Price float64 `json:"price"`
	// Reference is your reference, unique per space; generated when "".
	Reference string `json:"reference,omitempty"`
	// Subscription makes it a subscription product (Frequency required).
	Subscription *SubscriptionTermsParams `json:"subscription,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *CreateProductParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("name", p.Name) {
		v.name("name", p.Name, 2, 100)
	}
	v.text("description", p.Description, 2, 500)
	v.price("price", p.Price)
	v.reference("reference", p.Reference)
	if p.Subscription != nil {
		p.Subscription.validate(&v, "subscription.", true)
	}
	return v.err()
}

// UpdateProductParams updates a product (Products.Update): unset fields are unchanged.
type UpdateProductParams struct {
	// Name is the new name ("" = unchanged).
	Name string `json:"name,omitempty"`
	// Description is the new description; a pointer to "" clears it, nil leaves it unchanged.
	Description *string `json:"description,omitempty"`
	// Price is the new price in USD, above 0: for new checkouts and subscribers only.
	Price *float64 `json:"price,omitempty"`
	// IsActive false hides the product from Products.List, true lists it again.
	IsActive *bool `json:"isActive,omitempty"`
	// Subscription sets (or changes) the subscription terms; a one-time product needs Frequency.
	Subscription *SubscriptionTermsParams `json:"subscription,omitempty"`
	// RemoveSubscription makes it a one-time product (not with Subscription).
	RemoveSubscription bool `json:"removeSubscription,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *UpdateProductParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.name("name", p.Name, 2, 100)
	if p.Description != nil {
		v.text("description", *p.Description, 2, 500)
	}
	if p.Price != nil {
		v.price("price", *p.Price)
	}
	if p.Subscription != nil {
		p.Subscription.validate(&v, "subscription.", false)
	}
	v.exclusive("subscription", p.Subscription != nil, "removeSubscription", p.RemoveSubscription)
	return v.err()
}

// ProductListParams filters Products.List.
type ProductListParams struct {
	// IncludeHidden lists the hidden products too (IsActive false).
	IncludeHidden bool
	// Subscription keeps only the subscription products (true) or the one-time ones (false).
	Subscription *bool
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *ProductListParams) Validate() error { return nil }

func (p *ProductListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setBool(q, "includeHidden", p.IncludeHidden)
	setBoolPtr(q, "subscription", p.Subscription)
	return q
}
