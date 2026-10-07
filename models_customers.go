package qbitflow

import (
	"net/url"
	"time"
)

// Customer is a customer of a space.
type Customer struct {
	// UUID is the customer's id.
	UUID string `json:"uuid"`
	// Name is the first name.
	Name string `json:"name"`
	// LastName is the last name, when given.
	LastName string `json:"lastName,omitempty"`
	// Email is the email address, lowercase. Several customers of a space may share one.
	Email string `json:"email"`
	// Verified is true for a customer the merchant created, false for one created from the
	// email a payer typed at checkout.
	Verified bool `json:"verified"`
	// PhoneNumber is the phone number, when given.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	// Address is the physical address, when given.
	Address string `json:"address,omitempty"`
	// Reference is the merchant's reference, when given.
	Reference string `json:"reference,omitempty"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// UserUUID is the member whose space it is in; "" for the organization's own.
	UserUUID string `json:"userUuid,omitempty"`
}

// CreateCustomerParams creates a customer (Customers.Create).
type CreateCustomerParams struct {
	// Name is required: the first name, 2 to 100 characters, one line.
	Name string `json:"name"`
	// LastName is optional: 1 to 100 characters, one line.
	LastName string `json:"lastName,omitempty"`
	// Email is required (stored lowercase), at most 254 characters.
	Email string `json:"email"`
	// PhoneNumber is optional: at most 32 characters, digits and . - ( ) space, an optional +.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	// Address is optional: at most 500 characters.
	Address string `json:"address,omitempty"`
	// Reference is your reference (e.g. your own customer id), unique per space.
	Reference string `json:"reference,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *CreateCustomerParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("name", p.Name) {
		v.name("name", p.Name, 2, 100)
	}
	v.name("lastName", p.LastName, 1, 100)
	if v.required("email", p.Email) {
		v.email("email", p.Email)
	}
	v.phone("phoneNumber", p.PhoneNumber)
	v.text("address", p.Address, 0, 500)
	v.reference("reference", p.Reference)
	return v.err()
}

// UpdateCustomerParams updates a customer (Customers.Update): unset fields are unchanged.
// The reference cannot be changed.
type UpdateCustomerParams struct {
	// Name is the new first name ("" = unchanged).
	Name string `json:"name,omitempty"`
	// LastName is the new last name ("" = unchanged).
	LastName string `json:"lastName,omitempty"`
	// Email is the new email ("" = unchanged).
	Email string `json:"email,omitempty"`
	// PhoneNumber is the new phone number; a pointer to "" clears it, nil leaves it unchanged.
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	// Address is the new address; a pointer to "" clears it, nil leaves it unchanged.
	Address *string `json:"address,omitempty"`
}

// Validate checks the params (behaviour §8).
func (p *UpdateCustomerParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.name("name", p.Name, 2, 100)
	v.name("lastName", p.LastName, 1, 100)
	v.email("email", p.Email)
	if p.PhoneNumber != nil {
		v.phone("phoneNumber", *p.PhoneNumber)
	}
	if p.Address != nil {
		v.text("address", *p.Address, 0, 500)
	}
	return v.err()
}

// CustomerListParams filters Customers.List.
type CustomerListParams struct {
	// Limit is the page size (server default 10, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
	// Email keeps only the customers with this email, whatever its casing.
	Email string
	// Verified keeps only the customers the merchant created (true) or those created at
	// checkout (false).
	Verified *bool
}

// Validate checks the filters.
func (p *CustomerListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.email("email", p.Email)
	return v.err()
}

func (p *CustomerListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	setString(q, "email", p.Email)
	setBoolPtr(q, "verified", p.Verified)
	return q
}
