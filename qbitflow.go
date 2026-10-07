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
		TransactionStatus: qbf.NewTransactionStatusService(client),

		Refunds:    qbf.NewRefundService(client),
		Accounting: qbf.NewAccountingService(client),

		Claims: qbf.NewClaimService(client),
	}
}

// OnBehalfOf returns a client whose every service acts on behalf of the given user of your
// organization: each request carries the On-Behalf-Of header. It requires an
// organization-level (admin/owner) API key.
//
// The returned client shares the transport, connection pool and configuration of q; q
// itself is left untouched, so organization-level and per-user calls can be mixed freely.
// A userID of 0 returns a client that acts at the organization level (no header sent).
//
//	userClient := client.OnBehalfOf(123)
//	products, err := userClient.Products.GetAll()
//	payments, err := userClient.Payments.GetAllPayments(nil, nil)
//
// Every service also has its own OnBehalfOf for a one-off call.
func (q *QBitFlow) OnBehalfOf(userID uint64) *QBitFlow {
	return newQBitFlowWithClient(q.client.OnBehalfOf(userID))
}

// SetBaseURL overrides the API base URL (useful for testing against a local server).
//
// Call it before using the client: it is not synchronised with in-flight requests, and
// clients or services obtained from OnBehalfOf earlier keep the base URL they were
// created with.
func (q *QBitFlow) SetBaseURL(baseURL string) {
	q.client.SetBaseURL(baseURL)
}
