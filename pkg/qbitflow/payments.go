package qbf

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/utils"
)

// PaymentService handles one-time payment operations
type PaymentService struct {
	client *Client
}

func NewPaymentService(client *Client) *PaymentService {
	return &PaymentService{client: client}
}

// OnBehalfOf returns a PaymentService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *PaymentService) OnBehalfOf(userID uint64) *PaymentService {
	return NewPaymentService(s.client.withOnBehalfOf(userID))
}

// CreateSessionOptions are the options for creating a one-time payment session.
// Provide either ProductID, ProductReference, or the (ProductName + Description + Price) triplet.
type CreateSessionOptions struct {
	ProductID   *uint64 // Use an existing product by ID
	ProductName *string // Or define a product inline (requires Description and Price)
	Description *string
	Price       *float64
	SuccessURL  *string // Redirect URL on successful payment
	CancelURL   *string // Redirect URL on cancelled/failed payment

	// CustomerUUID pre-associates a customer by their QBitFlow UUID.
	CustomerUUID *string
	// Reference is your own reference for the transaction (e.g. an order/invoice ID). It is
	// echoed back on the resulting Payment and in webhooks, and usable with GetPaymentByReference.
	Reference *string
	// ProductReference selects an existing product by your own reference (alternative to ProductID).
	ProductReference *string
	// CustomerReference selects an existing customer by your own reference (alternative to
	// CustomerUUID). A new customer is created during checkout if none matches.
	CustomerReference *string
}

// createPaymentSessionRequest is the JSON body sent to POST /transaction/session-checkout/new/payment
type createPaymentSessionRequest struct {
	ProductID         *uint64  `json:"productId,omitempty"`
	ProductName       *string  `json:"productName,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Price             *float64 `json:"price,omitempty"`
	SuccessURL        *string  `json:"successUrl,omitempty"`
	CancelURL         *string  `json:"cancelUrl,omitempty"`
	CustomerUUID      *string  `json:"customerUUID,omitempty"`
	Reference         *string  `json:"reference,omitempty"`
	ProductReference  *string  `json:"productReference,omitempty"`
	CustomerReference *string  `json:"customerReference,omitempty"`
}

// CreateSession creates a new one-time payment session and returns a checkout link.
func (s *PaymentService) CreateSession(opts *CreateSessionOptions) (*qbmodels.LinkResponse, error) {
	if opts.ProductID == nil && opts.ProductReference == nil && (opts.ProductName == nil || opts.Description == nil || opts.Price == nil) {
		return nil, qberrors.NewValidationError("either ProductID, ProductReference, or ProductName, Description, and Price must be provided")
	}
	// Mirror the API's binding rules for an inline product and the redirect URLs, so
	// invalid input is caught before the round-trip.
	if err := validateInlineProduct(opts.ProductName, opts.Description, opts.Price, opts.SuccessURL, opts.CancelURL); err != nil {
		return nil, err
	}

	req := &createPaymentSessionRequest{
		ProductID:         opts.ProductID,
		ProductName:       opts.ProductName,
		Description:       opts.Description,
		Price:             opts.Price,
		SuccessURL:        opts.SuccessURL,
		CancelURL:         opts.CancelURL,
		CustomerUUID:      opts.CustomerUUID,
		Reference:         opts.Reference,
		ProductReference:  opts.ProductReference,
		CustomerReference: opts.CustomerReference,
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/new/payment", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSession retrieves a payment or subscription session by its UUID.
func (s *PaymentService) GetSession(sessionUUID string) (*qbmodels.SessionCheckout, error) {
	var result qbmodels.SessionCheckout
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s?closeToExpireError=false", sessionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPayment retrieves a completed one-time payment by its UUID.
func (s *PaymentService) GetPayment(paymentUUID string) (*qbmodels.Payment, error) {
	var result qbmodels.Payment
	endpoint := fmt.Sprintf("/transaction/payment/%s", paymentUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPaymentByReference retrieves a completed one-time payment by the reference you assigned
// when creating the session. Lets you resolve a payment from your own order/invoice ID without
// storing QBitFlow's UUID.
func (s *PaymentService) GetPaymentByReference(reference string) (*qbmodels.Payment, error) {
	if reference == "" {
		return nil, qberrors.NewValidationError("payment reference cannot be empty")
	}
	var result qbmodels.Payment
	endpoint := fmt.Sprintf("/transaction/payment/reference/%s", reference)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllPayments retrieves all one-time payments with optional cursor-based pagination.
func (s *PaymentService) GetAllPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Payment], error) {
	endpoint := utils.CursorQueryBuilder("/transaction/payments", limit, cursor)

	var result qbmodels.CursorData[qbmodels.Payment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllCombinedPayments retrieves all payments from both one-time and subscription billing cycles.
func (s *PaymentService) GetAllCombinedPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.CombinedPayment], error) {
	endpoint := utils.CursorQueryBuilder("/transaction/payments/combined", limit, cursor)

	var result qbmodels.CursorData[qbmodels.CombinedPayment]
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCustomerForTransaction retrieves the customer associated with a transaction UUID.
func (s *PaymentService) GetCustomerForTransaction(transactionUUID string) (*qbmodels.Customer, error) {
	var result qbmodels.Customer
	endpoint := fmt.Sprintf("/transaction/customer/%s", transactionUUID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
