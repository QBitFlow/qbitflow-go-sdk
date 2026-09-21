package qbitflow

import (
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// QBitFlow is the main SDK client providing access to all API services.
type QBitFlow struct {
	client *qbf.Client

	// Core services
	Customers  *qbf.CustomerService
	Products   *qbf.ProductService
	Users      *qbf.UserService
	ApiKeys    *qbf.ApiKeyService
	Webhooks   *qbf.WebhookService
	Currencies *qbf.CurrencyService

	// Transaction services
	Payments          *qbf.PaymentService
	Subscriptions     *qbf.SubscriptionService
	PayAsYouGo        *qbf.PayAsYouGoService
	TransactionStatus *qbf.TransactionStatusService

	// Financial services
	Refunds    *qbf.RefundService
	Accounting *qbf.AccountingService

	// Account claim services
	Claims *qbf.ClaimService
}

// New creates a new QBitFlow SDK client with the given API key.
func New(apiKey string) *QBitFlow {
	client := qbf.NewClient(apiKey)
	return newQBitFlowWithClient(client)
}

// NewWithConfig creates a new QBitFlow SDK client with custom configuration.
func NewWithConfig(config qbf.Config) *QBitFlow {
	client := qbf.NewClientWithConfig(config)
	return newQBitFlowWithClient(client)
}

func newQBitFlowWithClient(client *qbf.Client) *QBitFlow {
	return &QBitFlow{
		client: client,

		Customers:  qbf.NewCustomerService(client),
		Products:   qbf.NewProductService(client),
		Users:      qbf.NewUserService(client),
		ApiKeys:    qbf.NewApiKeyService(client),
		Webhooks:   qbf.NewWebhookService(client),
		Currencies: qbf.NewCurrencyService(client),

		Payments:          qbf.NewPaymentService(client),
		Subscriptions:     qbf.NewSubscriptionService(client),
		PayAsYouGo:        qbf.NewPayAsYouGoService(client),
		TransactionStatus: qbf.NewTransactionStatusService(client),

		Refunds:    qbf.NewRefundService(client),
		Accounting: qbf.NewAccountingService(client),

		Claims: qbf.NewClaimService(client),
	}
}

// SetBaseURL overrides the API base URL (useful for testing against a local server).
func (q *QBitFlow) SetBaseURL(baseURL string) {
	q.client.SetBaseURL(baseURL)
}
