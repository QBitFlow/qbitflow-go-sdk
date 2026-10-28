package qbitflow

// The service types group the API's methods by area; reach them through the Client's fields
// (client.Products, client.Webhooks.Endpoints, …). They are created by NewClient and
// Client.OnBehalfOf, never directly.

// ProductsService manages the products.
type ProductsService struct{ client *Client }

// CustomersService manages the customers.
type CustomersService struct{ client *Client }

// CheckoutSessionsService creates checkout sessions (one-time payments and subscriptions),
// reads their status and expires them.
type CheckoutSessionsService struct{ client *Client }

// PaymentsService reads the one-time payments.
type PaymentsService struct{ client *Client }

// TransactionsService reads the transactions feed: the one-time payments and the subscriptions'
// bills together.
type TransactionsService struct{ client *Client }

// FailuresService reads the failed payment attempts.
type FailuresService struct{ client *Client }

// SubscriptionsService reads subscriptions and their bills, updates their metadata, cancels
// them, and runs a test bill.
type SubscriptionsService struct{ client *Client }

// RefundsService lists and reads refunds (a merchant sends refunds from the dashboard).
type RefundsService struct{ client *Client }

// MembersService manages the organization's members, their trust and their held funds
// (organization key).
type MembersService struct{ client *Client }

// InvitationsService invites members to the organization (organization key).
type InvitationsService struct{ client *Client }

// WalletsService reads the wallets and the currencies a space accepts.
type WalletsService struct{ client *Client }

// AccountingService exports the accounting events.
type AccountingService struct{ client *Client }

// WebhooksService verifies webhook deliveries; Endpoints and Events manage the endpoints and
// the event log.
type WebhooksService struct {
	client *Client

	// Endpoints manages the webhook endpoints (/webhooks/endpoints…).
	Endpoints *WebhookEndpointsService
	// Events reads the event log (/webhooks/events…).
	Events *WebhookEventsService
}

// WebhookEndpointsService manages the webhook endpoints.
type WebhookEndpointsService struct{ client *Client }

// WebhookEventsService reads the webhook event log.
type WebhookEventsService struct{ client *Client }

// CurrenciesService reads the currency catalog (public routes; the key is sent anyway).
type CurrenciesService struct{ client *Client }
