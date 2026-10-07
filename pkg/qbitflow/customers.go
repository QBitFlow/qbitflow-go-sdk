package qbf

import (
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// CustomerService manages customer records — the people who pay you. Customers are
// scoped to the (organization, user) pair of the credential; use OnBehalfOf to reach a
// specific user's customers with an organization key.
type CustomerService struct {
	client *Client
}

// NewCustomerService creates the customer service bound to client.
func NewCustomerService(client *Client) *CustomerService {
	return &CustomerService{client: client}
}

// OnBehalfOf returns a CustomerService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *CustomerService) OnBehalfOf(userID uint64) *CustomerService {
	return NewCustomerService(s.client.withOnBehalfOf(userID))
}

// CreateCustomer holds the fields required to create a new customer record.
//
// The optional fields treat "" like nil: the field is left out of the request.
type CreateCustomer struct {
	Name     string `json:"name"`     // 2-100 chars, letters/digits/spaces and - _ ' .
	LastName string `json:"lastName"` // 2-100 chars, same character set
	Email    string `json:"email"`    // required, a valid email, unique per (org, user)

	PhoneNumber *string `json:"phoneNumber,omitempty"` // free-form
	Address     *string `json:"address,omitempty"`     // free-form
	Reference   *string `json:"reference,omitempty"`   // your own identifier, unique per (org, user), immutable
}

// Validate mirrors the API's binding rules for a new customer: name and last name are
// `alphanumspace` (2-100 characters) and the email is required and well-formed. It is
// called by Create, so invalid input is rejected before the round-trip.
func (c *CreateCustomer) Validate() error {
	if err := requireArgument(c == nil, "Customer"); err != nil {
		return err
	}
	if err := validateAlphanumSpace("Name", c.Name, 2, 100); err != nil {
		return err
	}
	if err := validateAlphanumSpace("LastName", c.LastName, 2, 100); err != nil {
		return err
	}
	if err := requireNonEmpty(c.Email, "Email"); err != nil {
		return err
	}
	return validateEmail("Email", c.Email)
}

// UpdateCustomer holds the fields that can be updated on an existing customer.
// The customer UUID is passed as the first argument to Update(), not in this struct.
//
// Every field is optional and the update is PARTIAL: a nil field — or a pointer to "",
// which the API also treats as "not provided" — is left out of the request and the stored
// value is left unchanged. An entirely empty UpdateCustomer is a valid no-op.
//
// Reference is deliberately absent: it is an immutable identifier that the API does not
// accept on update (a reference sent in the body is ignored).
type UpdateCustomer struct {
	Name     *string `json:"name,omitempty"`     // 2-100 chars, alphanumspace
	LastName *string `json:"lastName,omitempty"` // 2-100 chars, alphanumspace
	Email    *string `json:"email,omitempty"`    // must be a valid email, unique per (org, user)

	PhoneNumber *string `json:"phoneNumber,omitempty"`
	Address     *string `json:"address,omitempty"`
}

// Validate checks the fields that are present. Absent (nil) or empty fields are not
// validated: they are not sent, and the stored value is left untouched.
func (c *UpdateCustomer) Validate() error {
	if err := requireArgument(c == nil, "Customer update"); err != nil {
		return err
	}
	if isSet(c.Name) {
		if err := validateAlphanumSpace("Name", *c.Name, 2, 100); err != nil {
			return err
		}
	}
	if isSet(c.LastName) {
		if err := validateAlphanumSpace("LastName", *c.LastName, 2, 100); err != nil {
			return err
		}
	}
	if isSet(c.Email) {
		if err := validateEmail("Email", *c.Email); err != nil {
			return err
		}
	}
	return nil
}

// Create creates a new customer record.
func (s *CustomerService) Create(customer *CreateCustomer) (*qbmodels.Customer, error) {
	if err := customer.Validate(); err != nil {
		return nil, err
	}

	body := &CreateCustomer{
		Name:        customer.Name,
		LastName:    customer.LastName,
		Email:       customer.Email,
		PhoneNumber: omitEmpty(customer.PhoneNumber),
		Address:     omitEmpty(customer.Address),
		Reference:   omitEmpty(customer.Reference),
	}

	var result qbmodels.Customer
	err := s.client.makeRequest("POST", "/customer/", body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a customer by their UUID.
func (s *CustomerService) Get(customerUUID string) (*qbmodels.Customer, error) {
	if err := requireNonEmpty(customerUUID, "Customer UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.Customer
	endpoint := "/customer/uuid/" + escapePath(customerUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByEmail retrieves a customer by their email address.
func (s *CustomerService) GetByEmail(email string) (*qbmodels.Customer, error) {
	if err := requireNonEmpty(email, "Customer email"); err != nil {
		return nil, err
	}
	if err := validateEmail("Customer email", email); err != nil {
		return nil, err
	}

	var result qbmodels.Customer
	endpoint := "/customer/email/" + escapePath(email)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByReference retrieves a customer by the reference you assigned when creating it.
// Lets you resolve a customer from your own identifier without storing QBitFlow's UUID.
//
// The reference is path-escaped correctly, but the API currently cannot route a reference
// containing "/" (it answers 404).
func (s *CustomerService) GetByReference(reference string) (*qbmodels.Customer, error) {
	if err := requireNonEmpty(reference, "Customer reference"); err != nil {
		return nil, err
	}

	var result qbmodels.Customer
	endpoint := "/customer/reference/" + escapePath(reference)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves one page of customers. limit (optional) must be positive; pass the
// previous page's NextCursor as cursor to fetch the next page.
func (s *CustomerService) GetAll(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Customer], error) {
	return getPage[qbmodels.Customer](s.client, "/customer/all", limit, cursor)
}

// Update updates a customer record. The customerUUID identifies which record to update.
func (s *CustomerService) Update(customerUUID string, customer *UpdateCustomer) (*qbmodels.Customer, error) {
	if err := requireNonEmpty(customerUUID, "Customer UUID"); err != nil {
		return nil, err
	}
	if err := customer.Validate(); err != nil {
		return nil, err
	}

	body := &UpdateCustomer{
		Name:        omitEmpty(customer.Name),
		LastName:    omitEmpty(customer.LastName),
		Email:       omitEmpty(customer.Email),
		PhoneNumber: omitEmpty(customer.PhoneNumber),
		Address:     omitEmpty(customer.Address),
	}

	var result qbmodels.Customer
	endpoint := "/customer/" + escapePath(customerUUID)
	err := s.client.makeRequest("PUT", endpoint, body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete soft-deletes a customer record by UUID.
//
// It returns only an error, the idiomatic Go shape; the API's confirmation message carries
// no information beyond success.
func (s *CustomerService) Delete(customerUUID string) error {
	if err := requireNonEmpty(customerUUID, "Customer UUID"); err != nil {
		return err
	}

	endpoint := "/customer/uuid/" + escapePath(customerUUID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
