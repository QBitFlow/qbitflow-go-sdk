package qbitflow

import (
	"fmt"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/errors"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

// PaymentService handles payment-related operations
type PaymentService struct {
	client *Client
}

// CreateSessionOptions represents options for creating a payment session
type CreateSessionOptions struct {
	ProductID    *uint64  // Either set ProductID or ProductName, Description, and Price must be provided
	ProductName  *string  // Either set ProductID or ProductName, Description, and Price must be provided
	Description  *string  // Either set ProductID or ProductName, Description, and Price must be provided
	Price        *float64 // Either set ProductID or ProductName, Description, and Price must be provided
	SuccessURL   *string  // Optional success URL (your customer is redirected here after payment)
	CancelURL    *string  // Optional cancel URL (your customer is redirected here if the payment failed)
	WebhookURL   *string  // Optional webhook URL for payment status updates
	CustomerUUID *string  // Optional customer UUID to associate with the payment
}

// CreateSession creates a new one-time payment session
func (s *PaymentService) CreateSession(opts *CreateSessionOptions) (*models.LinkResponse, error) {
	// Validate options
	if opts.ProductID == nil && (opts.ProductName == nil || opts.Description == nil || opts.Price == nil) {
		return nil, errors.NewValidationError("either ProductID or ProductName, Description, and Price must be provided")
	}
	if opts.Price != nil && *opts.Price < 0 {
		return nil, errors.NewValidationError("price must be a non-negative value")
	}

	req := &models.CreateSessionRequest{
		ProductID:    opts.ProductID,
		ProductName:  opts.ProductName,
		Description:  opts.Description,
		Price:        opts.Price,
		SuccessURL:   opts.SuccessURL,
		CancelURL:    opts.CancelURL,
		WebhookURL:   opts.WebhookURL,
		CustomerUUID: opts.CustomerUUID,
	}

	var result models.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSession retrieves a payment session by its UUID
func (s *PaymentService) GetSession(sessionUUID string) (*models.Session, error) {
	var result models.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPayment retrieves a one-time payment by its UUID (must be processed already)
func (s *PaymentService) GetPayment(paymentUUID string) (*models.Payment, error) {
	var result models.Payment
	endpoint := fmt.Sprintf("/transaction/payment/%s", paymentUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllPayments retrieves all one-time payments with optional pagination
func (s *PaymentService) GetAllPayments(limit *int, cursor *string) (*models.CursorData[models.Payment], error) {
	endpoint := "/transaction/payments"

	// Build query parameters
	if limit != nil || cursor != nil {
		endpoint += "?"
		params := []string{}
		if limit != nil {
			params = append(params, fmt.Sprintf("limit=%d", *limit))
		}
		if cursor != nil {
			params = append(params, fmt.Sprintf("cursor=%s", *cursor))
		}
		endpoint += joinParams(params)
	}

	var result models.CursorData[models.Payment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllCombinedPayments retrieves all combined payments from one-time and subscription payments
func (s *PaymentService) GetAllCombinedPayments(limit *int, cursor *string) (*models.CursorData[models.CombinedPayment], error) {
	endpoint := "/transaction/payments/combined"

	// Build query parameters
	if limit != nil || cursor != nil {
		endpoint += "?"
		params := []string{}
		if limit != nil {
			params = append(params, fmt.Sprintf("limit=%d", *limit))
		}
		if cursor != nil {
			params = append(params, fmt.Sprintf("cursor=%s", *cursor))
		}
		endpoint += joinParams(params)
	}

	var result models.CursorData[models.CombinedPayment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Helper function to join query parameters
func joinParams(params []string) string {
	result := ""
	for i, param := range params {
		if i > 0 {
			result += "&"
		}
		result += param
	}
	return result
}
