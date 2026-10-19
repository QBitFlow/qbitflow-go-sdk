package qbitflow

import (
	"net/url"
	"time"
)

// Customer is a merchant's customer: who pays, named on the payments and subscriptions it makes.
type Customer struct {
	// UUID is the customer's id (a UUIDv7).
	UUID string `json:"uuid"`
	// FirstName is the first name.
	FirstName string `json:"firstName"`
	// LastName is the last name; "" (absent) when not given.
	LastName string `json:"lastName,omitempty"`
	// Email is the email address, lowercase (matched case-insensitively). Several customers of a
	// space may share one (see Verified).
	Email string `json:"email"`
	// Verified is true for a customer the merchant created (the API, the dashboard), who vouches
	// for its details; false for one QBitFlow created from the email a payer typed at a checkout
	// that named no customer (nobody proved the payer owns that address). A checkout never links a
	// typed email to a verified customer.
	Verified bool `json:"verified"`
	// PhoneNumber is the phone number; "" (absent) when not given.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	// Address is the physical address; "" (absent) when not given.
	Address string `json:"address,omitempty"`
	// Reference is the merchant's reference (its id in your system), unique per space; ""
	// (absent) without one.
	Reference string `json:"reference,omitempty"`
	// Metadata is the merchant's own data on the customer (never copied onto its payments or
	// subscriptions); empty without any.
	Metadata map[string]string `json:"metadata"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// CreateCustomerParams creates a customer (Customers.Create).
type CreateCustomerParams struct {
	// FirstName is required: 2 to 100 characters, one line.
	FirstName string `json:"firstName"`
	// LastName is optional: 1 to 100 characters, one line.
	LastName string `json:"lastName,omitempty"`
	// Email is required (stored lowercase), at most 254 characters. Not unique: several
	// customers of a space may share one.
	Email string `json:"email"`
	// PhoneNumber is optional: at most 32 characters, digits and . - ( ) space, an optional +.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	// Address is optional: at most 500 characters.
	Address string `json:"address,omitempty"`
	// Reference is your reference (e.g. your own customer id), unique per space and mode: 1 to
	// 100 of A-Z a-z 0-9 . _ : @ -.
	Reference string `json:"reference,omitempty"`
	// Metadata is your own data on the customer (a CRM key): at most 50 keys of 1 to 40
	// characters among A-Z a-z 0-9 _ - . :, values of at most 500 characters without control or
	// bidirectional characters. A key set to "" is not stored.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *CreateCustomerParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("firstName", p.FirstName) {
		v.name("firstName", p.FirstName, 2, 100)
	}
	v.name("lastName", p.LastName, 1, 100)
	if v.required("email", p.Email) {
		v.email("email", p.Email)
	}
	v.phone("phoneNumber", p.PhoneNumber)
	v.text("address", p.Address, 0, 500)
	v.reference("reference", p.Reference)
	v.metadata("metadata", p.Metadata)
	return v.err()
}

// UpdateCustomerParams updates a customer (Customers.Update): unset fields are unchanged.
// FirstName and Email cannot be cleared; LastName, PhoneNumber and Address are cleared with a
// pointer to "". The reference cannot change.
type UpdateCustomerParams struct {
	// FirstName is the new first name, 2 to 100 characters ("" = unchanged).
	FirstName string `json:"firstName,omitempty"`
	// LastName is the new last name (1 to 100 characters); a pointer to "" clears it, nil leaves
	// it unchanged.
	LastName *string `json:"lastName,omitempty"`
	// Email is the new email ("" = unchanged).
	Email string `json:"email,omitempty"`
	// PhoneNumber is the new phone number; a pointer to "" clears it, nil leaves it unchanged.
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	// Address is the new address; a pointer to "" clears it, nil leaves it unchanged.
	Address *string `json:"address,omitempty"`
	// Metadata is merged into the customer's: a key set to a value sets it, a key set to ""
	// deletes it, keys not named are kept (nil: unchanged). At most 50 keys once merged.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *UpdateCustomerParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.name("firstName", p.FirstName, 2, 100)
	if p.LastName != nil {
		v.name("lastName", *p.LastName, 1, 100)
	}
	v.email("email", p.Email)
	if p.PhoneNumber != nil {
		v.phone("phoneNumber", *p.PhoneNumber)
	}
	if p.Address != nil {
		v.text("address", *p.Address, 0, 500)
	}
	v.metadata("metadata", p.Metadata)
	return v.err()
}

// CustomerListParams filters Customers.List.
type CustomerListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
	// Email keeps only the customers with this email, whatever its casing (several may share one).
	Email string
	// Reference keeps only the customer with this reference (a page of 0 or 1 customer;
	// Customers.GetByReference reads it).
	Reference string
	// Verified keeps only the customers the merchant created (true) or those created from an
	// email a payer typed at checkout (false).
	Verified *bool
}

// Validate checks the filters.
func (p *CustomerListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	v.email("email", p.Email)
	v.reference("reference", p.Reference)
	return v.err()
}

func (p *CustomerListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	setString(q, "email", p.Email)
	setString(q, "reference", p.Reference)
	setBoolPtr(q, "verified", p.Verified)
	return q
}
