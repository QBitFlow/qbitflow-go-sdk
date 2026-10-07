package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// PaymentService handles one-time payment operations
type PaymentService struct {
	client *Client
}

// NewPaymentService creates the payment service bound to client.
func NewPaymentService(client *Client) *PaymentService {
	return &PaymentService{client: client}
}

// OnBehalfOf returns a PaymentService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *PaymentService) OnBehalfOf(userID uint64) *PaymentService {
	return NewPaymentService(s.client.withOnBehalfOf(userID))
}

// CreateSessionOptions are the options for creating a one-time payment session.
// Provide either ProductID, ProductReference, or the (ProductName + Description + Price) triplet.
//
// Every string option treats "" like nil: the field is left out of the request.
type CreateSessionOptions struct {
	ProductID   *uint64  // Use an existing product by ID (must be > 0)
	ProductName *string  // Or define a product inline (requires Description and Price); producttext, 2-100 chars
	Description *string  // producttext, 2-500 chars
	Price       *float64 // USD, a finite number greater than 0
	SuccessURL  *string  // Redirect URL on successful payment (absolute http(s) URL)
	CancelURL   *string  // Redirect URL on cancelled/failed payment (absolute http(s) URL)

	// CustomerUUID pre-associates a customer by their QBitFlow UUID (a bare UUID, no prefix).
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
//
// The options are validated against the API's binding rules before any request is sent
// (product selection, inline product text and price, redirect URLs, customer UUID); a
// violation — including nil options — returns a *qberrors.ValidationError.
func (s *PaymentService) CreateSession(opts *CreateSessionOptions) (*qbmodels.LinkResponse, error) {
	if err := requireArgument(opts == nil, "Session options"); err != nil {
		return nil, err
	}
	if err := validateSessionProduct(opts.ProductID, opts.ProductReference, opts.ProductName,
		opts.Description, opts.Price, opts.SuccessURL, opts.CancelURL, opts.CustomerUUID); err != nil {
		return nil, err
	}

	req := &createPaymentSessionRequest{
		ProductID:         opts.ProductID,
		ProductName:       omitEmpty(opts.ProductName),
		Description:       omitEmpty(opts.Description),
		Price:             opts.Price,
		SuccessURL:        omitEmpty(opts.SuccessURL),
		CancelURL:         omitEmpty(opts.CancelURL),
		CustomerUUID:      omitEmpty(opts.CustomerUUID),
		Reference:         omitEmpty(opts.Reference),
		ProductReference:  omitEmpty(opts.ProductReference),
		CustomerReference: omitEmpty(opts.CustomerReference),
	}

	var result qbmodels.LinkResponse
	err := s.client.makeRequest("POST", "/transaction/session-checkout/new/payment", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSession retrieves a payment or subscription session by its UUID. The result is the
// unified SessionCheckout; use IsPayment / IsSubscription to tell the kinds apart.
//
// closeToExpireError controls whether the API rejects a session that is close to expiry.
// Pass nil to use the API's default (true).
func (s *PaymentService) GetSession(sessionUUID string, closeToExpireError *bool) (*qbmodels.SessionCheckout, error) {
	if err := requireNonEmpty(sessionUUID, "Session UUID"); err != nil {
		return nil, err
	}

	return getSession(s.client, sessionUUID, closeToExpireError)
}

// getSession is the shared implementation of PaymentService.GetSession and
// SubscriptionService.GetSession (both read the same route).
func getSession(client *Client, sessionUUID string, closeToExpireError *bool) (*qbmodels.SessionCheckout, error) {
	var result qbmodels.SessionCheckout
	endpoint := fmt.Sprintf("/transaction/session-checkout/%s", escapePath(sessionUUID))
	if closeToExpireError != nil {
		endpoint += fmt.Sprintf("?closeToExpireError=%t", *closeToExpireError)
	}
	err := client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	result.AvailableCurrencies = nonNil(result.AvailableCurrencies)
	return &result, nil
}

// GetPayment retrieves a completed one-time payment by its UUID.
func (s *PaymentService) GetPayment(paymentUUID string) (*qbmodels.Payment, error) {
	if err := requireNonEmpty(paymentUUID, "Payment UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.Payment
	endpoint := fmt.Sprintf("/transaction/payment/%s", escapePath(paymentUUID))
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
	if err := requireNonEmpty(reference, "Payment reference"); err != nil {
		return nil, err
	}

	var result qbmodels.Payment
	endpoint := fmt.Sprintf("/transaction/payment/reference/%s", escapePath(reference))
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllPayments retrieves one page of one-time payments. limit (optional) must be positive;
// pass the previous page's NextCursor as cursor to fetch the next page.
func (s *PaymentService) GetAllPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.Payment], error) {
	return getPage[qbmodels.Payment](s.client, "/transaction/payments", limit, cursor)
}

// GetAllCombinedPayments retrieves one page of the combined feed: one-time payments and
// subscription billing cycles.
func (s *PaymentService) GetAllCombinedPayments(limit *uint16, cursor *string) (*qbmodels.CursorData[qbmodels.CombinedPayment], error) {
	return getPage[qbmodels.CombinedPayment](s.client, "/transaction/payments/combined", limit, cursor)
}

// GetCustomerForTransaction retrieves the customer associated with a transaction UUID.
func (s *PaymentService) GetCustomerForTransaction(transactionUUID string) (*qbmodels.Customer, error) {
	if err := requireNonEmpty(transactionUUID, "Transaction UUID"); err != nil {
		return nil, err
	}

	var result qbmodels.Customer
	endpoint := fmt.Sprintf("/transaction/customer/%s", escapePath(transactionUUID))
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
