package qbitflow

import (
	qbf "github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
)

// QBitFlow is the main SDK client that provides access to all API services
type QBitFlow struct {
	client *qbf.Client

	// Services
	Customers *qbf.CustomerService
	Products  *qbf.ProductService
	Users     *qbf.UserService
	ApiKeys   *qbf.ApiKeyService
	Webhooks  *qbf.WebhookService

	Payments          *qbf.PaymentService
	Subscriptions     *qbf.SubscriptionService
	PayAsYouGo        *qbf.PayAsYouGoService
	TransactionStatus *qbf.TransactionStatusService
}

// New creates a new QBitFlow SDK client with the given API key
func New(apiKey string) *QBitFlow {
	client := qbf.NewClient(apiKey)
	return newQBitFlowWithClient(client)
}

// NewWithConfig creates a new QBitFlow SDK client with custom configuration
func NewWithConfig(config qbf.Config) *QBitFlow {
	client := qbf.NewClientWithConfig(config)
	return newQBitFlowWithClient(client)
}

// newQBitFlowWithClient creates a QBitFlow instance with an existing client
func newQBitFlowWithClient(client *qbf.Client) *QBitFlow {
	return &QBitFlow{
		client: client,

		Customers: qbf.NewCustomerService(client),
		Products:  qbf.NewProductService(client),
		Users:     qbf.NewUserService(client),
		ApiKeys:   qbf.NewApiKeyService(client),
		Webhooks:  qbf.NewWebhookService(client),

		Payments:          qbf.NewPaymentService(client),
		Subscriptions:     qbf.NewSubscriptionService(client),
		PayAsYouGo:        qbf.NewPayAsYouGoService(client),
		TransactionStatus: qbf.NewTransactionStatusService(client),
	}
}

// SetBaseURL sets a custom base URL for the API
func (q *QBitFlow) SetBaseURL(baseURL string) {
	q.client.SetBaseURL(baseURL)
}
