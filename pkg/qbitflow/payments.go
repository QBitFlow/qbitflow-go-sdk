package qbitflow

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/utils"
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
func (s *PaymentService) CreateSession(opts *CreateSessionOptions) (*qbmodels.LinkResponse, error) {
	// Validate options
	if opts.ProductID == nil && (opts.ProductName == nil || opts.Description == nil || opts.Price == nil) {
		return nil, qberrors.NewValidationError("either ProductID or ProductName, Description, and Price must be provided")
	}
	if opts.Price != nil && *opts.Price < 0 {
		return nil, qberrors.NewValidationError("price must be a non-negative value")
	}

	req := &qbmodels.CreateSessionRequest{
		ProductID:    opts.ProductID,
		ProductName:  opts.ProductName,
		Description:  opts.Description,
		Price:        opts.Price,
		SuccessURL:   opts.SuccessURL,
		CancelURL:    opts.CancelURL,
		WebhookURL:   opts.WebhookURL,
		CustomerUUID: opts.CustomerUUID,
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSession retrieves a payment session by its UUID
func (s *PaymentService) GetSession(sessionUUID string) (*qbmodels.Session, error) {
	var result qbmodels.Session
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPayment retrieves a one-time payment by its UUID (must be processed already)
func (s *PaymentService) GetPayment(paymentUUID string) (*qbmodels.Payment, error) {
	var result qbmodels.Payment
	endpoint := fmt.Sprintf("/transaction/payment/%s", paymentUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllPayments retrieves all one-time payments with optional pagination
func (s *PaymentService) GetAllPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Payment], error) {
	endpoint := utils.CursorQueryBuilder("/transaction/payments", limit, cursor)

	var result qbmodels.CursorData[qbmodels.Payment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllCombinedPayments retrieves all combined payments from one-time and subscription payments
func (s *PaymentService) GetAllCombinedPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.CombinedPayment], error) {
	endpoint := utils.CursorQueryBuilder("/transaction/payments/combined", limit, cursor)

	var result qbmodels.CursorData[qbmodels.CombinedPayment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
