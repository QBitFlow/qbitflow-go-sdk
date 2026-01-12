package qbitflow

import (
	"fmt"
	"strings"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/utils"
)

type CustomerService struct {
	client *Client
}

type CreateCustomer struct {
	Name     string `json:"name" binding:"required"`        // Customer's name
	LastName string `json:"lastName" binding:"required"`    // Customer's last name
	Email    string `json:"email" binding:"required,email"` // Customer's email address

	// Optional fields
	PhoneNumber *string `json:"phoneNumber,omitempty"` // Customer's phone number
	Address     *string `json:"address,omitempty"`     // Customer's physical address

	Reference *string `json:"reference,omitempty"` // Optional reference for the customer information (e.g., order ID, user ID, etc.)
}

type UpdateCustomer struct {
	UUID     string `json:"uuid" binding:"required"` // Customer UUID
	Name     string `json:"name"`                    // Customer's name
	LastName string `json:"lastName"`                // Customer's last name
	Email    string `json:"email"`                   // Customer's email address

	// Optional fields
	PhoneNumber *string `json:"phoneNumber,omitempty"` // Customer's phone number
	Address     *string `json:"address,omitempty"`     // Customer's physical address
}

// Create creates a new customer with the provided information
func (s *CustomerService) Create(customer *CreateCustomer) (*qbmodels.Customer, error) {
	var result qbmodels.Customer
	err := s.client.makeRequest("POST", "/customer/", customer, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a customer by their UUID
func (s *CustomerService) Get(customerUUID string) (*qbmodels.Customer, error) {
	var result qbmodels.Customer
	endpoint := "/customer/uuid/" + customerUUID
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

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

func (s *CustomerService) GetAll(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Customer], error) {
	var result qbmodels.CursorData[qbmodels.Customer]

	endpoint := utils.CursorQueryBuilder("/customer/all", limit, cursor)

	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CustomerService) Update(customer *UpdateCustomer) (*qbmodels.Customer, error) {
	if customer.UUID == "" {
		return nil, fmt.Errorf("customer UUID is required for update")
	}

	var result qbmodels.Customer
	err := s.client.makeRequest("PUT", "/customer/", customer, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CustomerService) Delete(customerUUID string) error {
	endpoint := "/customer/uuid/" + customerUUID
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
