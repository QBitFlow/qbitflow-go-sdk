package qbitflow

import (
	"fmt"
	"strings"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
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
func (s *CustomerService) Create(customer *CreateCustomer) (*models.Customer, error) {
	var result models.Customer
	err := s.client.makeRequest("POST", "/customer/", customer, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a customer by their UUID
func (s *CustomerService) Get(customerUUID string) (*models.Customer, error) {
	var result models.Customer
	endpoint := "/customer/uuid/" + customerUUID
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CustomerService) GetByEmail(email string) (*models.Customer, error) {
	if !strings.Contains(email, "@") {
		return nil, fmt.Errorf("invalid email address: %s", email)
	}

	var result models.Customer
	endpoint := "/customer/email/" + email
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CustomerService) GetAll() (*models.CursorData[models.Customer], error) {
	var result models.CursorData[models.Customer]
	err := s.client.makeRequest("GET", "/customer/all", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CustomerService) Update(customer *UpdateCustomer) (*models.Customer, error) {
	if customer.UUID == "" {
		return nil, fmt.Errorf("customer UUID is required for update")
	}

	var result models.Customer
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
