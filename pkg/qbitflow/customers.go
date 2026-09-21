package qbf

import (
	"fmt"
	"strings"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/utils"
)

type CustomerService struct {
	client *Client
}

func NewCustomerService(client *Client) *CustomerService {
	return &CustomerService{client: client}
}

// OnBehalfOf returns a CustomerService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *CustomerService) OnBehalfOf(userID uint64) *CustomerService {
	return NewCustomerService(s.client.withOnBehalfOf(userID))
}

// CreateCustomer holds the fields required to create a new customer record.
type CreateCustomer struct {
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Email    string `json:"email"`

	PhoneNumber *string `json:"phoneNumber,omitempty"`
	Address     *string `json:"address,omitempty"`
	Reference   *string `json:"reference,omitempty"`
}

// UpdateCustomer holds the fields that can be updated on an existing customer.
// The customer UUID is passed as the first argument to Update(), not in this struct.
//
// Every field is optional and the update is PARTIAL: a nil field is omitted from the
// request and the stored value is left unchanged. An entirely empty UpdateCustomer is a
// valid no-op.
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

// Create creates a new customer record.
func (s *CustomerService) Create(customer *CreateCustomer) (*qbmodels.Customer, error) {
	var result qbmodels.Customer
	err := s.client.makeRequest("POST", "/customer/", customer, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a customer by their UUID.
func (s *CustomerService) Get(customerUUID string) (*qbmodels.Customer, error) {
	var result qbmodels.Customer
	endpoint := "/customer/uuid/" + customerUUID
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByEmail retrieves a customer by their email address.
func (s *CustomerService) GetByEmail(email string) (*qbmodels.Customer, error) {
	if !strings.Contains(email, "@") {
		return nil, fmt.Errorf("invalid email address: %s", email)
	}

	var result qbmodels.Customer
	endpoint := "/customer/email/" + email
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByReference retrieves a customer by the reference you assigned when creating it.
// Lets you resolve a customer from your own identifier without storing QBitFlow's UUID.
func (s *CustomerService) GetByReference(reference string) (*qbmodels.Customer, error) {
	if reference == "" {
		return nil, fmt.Errorf("customer reference cannot be empty")
	}

	var result qbmodels.Customer
	endpoint := "/customer/reference/" + reference
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves all customers with optional cursor-based pagination.
func (s *CustomerService) GetAll(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Customer], error) {
	var result qbmodels.CursorData[qbmodels.Customer]
	endpoint := utils.CursorQueryBuilder("/customer/all", limit, cursor)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates a customer record. The customerUUID identifies which record to update.
func (s *CustomerService) Update(customerUUID string, customer *UpdateCustomer) (*qbmodels.Customer, error) {
	if customerUUID == "" {
		return nil, fmt.Errorf("customer UUID is required for update")
	}

	var result qbmodels.Customer
	endpoint := "/customer/" + customerUUID
	err := s.client.makeRequest("PUT", endpoint, customer, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a customer record by UUID.
func (s *CustomerService) Delete(customerUUID string) error {
	endpoint := "/customer/uuid/" + customerUUID
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
