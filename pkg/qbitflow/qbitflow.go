package qbitflow

// QBitFlow is the main SDK client that provides access to all API services
type QBitFlow struct {
	client *Client

	// Services
	Customers *CustomerService
	Products  *ProductService
	Users     *UserService
	ApiKeys   *ApiKeyService

	Payments          *PaymentService
	Subscriptions     *SubscriptionService
	PayAsYouGo        *PayAsYouGoService
	TransactionStatus *TransactionStatusService
}

// New creates a new QBitFlow SDK client with the given API key
func New(apiKey string) *QBitFlow {
	client := NewClient(apiKey)
	return newQBitFlowWithClient(client)
}

// NewWithConfig creates a new QBitFlow SDK client with custom configuration
func NewWithConfig(config Config) *QBitFlow {
	client := NewClientWithConfig(config)
	return newQBitFlowWithClient(client)
}

// newQBitFlowWithClient creates a QBitFlow instance with an existing client
func newQBitFlowWithClient(client *Client) *QBitFlow {
	return &QBitFlow{
		client: client,

		Customers: &CustomerService{client: client},
		Products:  &ProductService{client: client},
		Users:     &UserService{client: client},
		ApiKeys:   &ApiKeyService{client: client},

		Payments:          &PaymentService{client: client},
		Subscriptions:     &SubscriptionService{client: client},
		PayAsYouGo:        &PayAsYouGoService{client: client},
		TransactionStatus: &TransactionStatusService{client: client},
	}
}

// SetBaseURL sets a custom base URL for the API
func (q *QBitFlow) SetBaseURL(baseURL string) {
	q.client.SetBaseURL(baseURL)
}
