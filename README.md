# QBitFlow Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MPL-2.0](https://upload.wikimedia.org/wikipedia/commons/thumb/9/95/Mozilla_Logo_2024.svg/1920px-Mozilla_Logo_2024.svg.png)](https://opensource.org/licenses/MPL-2.0)

Official Go SDK for [QBitFlow](https://qbitflow.app) - a comprehensive cryptocurrency payment processing platform that enables seamless integration of crypto payments, recurring subscriptions, and pay-as-you-go models into your applications.

## Features

-   🚀 **Easy to Use**: Simple, intuitive API design
-   🔄 **Automatic Retries**: Built-in retry logic for failed requests
-   🧪 **Well Tested**: Comprehensive test coverage
-   📚 **Great Documentation**: Detailed docs with examples
-   🔌 **Webhook Support**: Handle payment notifications easily
-   💳 **One-Time Payments**: Accept cryptocurrency payments with ease
-   🔄 **Recurring Subscriptions**: Automated recurring billing in cryptocurrency
-   📊 **Pay-as-You-Go**: Usage-based billing with cryptocurrency
-   👥 **Customer Management**: Create and manage customer profiles
-   🛍️ **Product Management**: Organize your products and pricing
-   📈 **Transaction Tracking**: Real-time transaction status updates
-   🔒 **Secure Authentication**: API key-based authentication
-   ⚡ **Production Ready**: Error handling and best practices built-in

## Table of Contents

-   [Features](#features)
-   [Installation](#installation)
-   [Quick Start](#quick-start)
    -   [1. Get Your API Key](#1-get-your-api-key)
    -   [2. Initialize the Client](#2-initialize-the-client)
    -   [3. Create a One-Time Payment](#3-create-a-one-time-payment)
    -   [4. Create a Recurring Subscription](#4-create-a-recurring-subscription)
    -   [5. Check Transaction Status](#5-check-transaction-status)
-   [Configuration](#configuration)
-   [One-Time Payments](#one-time-payments)
    -   [Create a Payment Session](#create-a-payment-session)
    -   [With Redirect URLs](#with-redirect-urls)
    -   [Get Payment Session](#get-payment-session)
    -   [Get Completed Payment](#get-completed-payment)
    -   [List All Payments](#list-all-payments)
    -   [List Combined Payments](#list-combined-payments)
-   [Subscriptions](#subscriptions)
    -   [Create a Subscription](#create-a-subscription)
    -   [Frequency Units](#frequency-units)
    -   [Get Subscription](#get-subscription)
    -   [Get Payment History](#get-payment-history)
    -   [Execute Test Billing Cycle](#execute-test-billing-cycle)
-   [Pay-As-You-Go Subscriptions](#pay-as-you-go-subscriptions)
    -   [Create PAYG Subscription](#create-payg-subscription)
    -   [Get PAYG Subscription](#get-payg-subscription)
    -   [Get Payment History](#get-payment-history-1)
    -   [Execute Test Billing Cycle](#execute-test-billing-cycle-1)
    -   [Increase Units Current Period](#increase-units-current-period)
-   [Transaction Status](#transaction-status)
    -   [Check Status](#check-status)
    -   [Transaction Types](#transaction-types)
    -   [Status Values](#status-values)
-   [Customer Management](#customer-management)
    -   [Create a Customer](#create-a-customer)
    -   [Get Customer by UUID](#get-customer-by-uuid)
    -   [Update Customer](#update-customer)
    -   [Delete Customer](#delete-customer)
-   [Product Management](#product-management)
    -   [Create a Product](#create-a-product)
    -   [Update Product](#update-product)
    -   [Delete Product](#delete-product)
-   [Webhook Handling](#webhook-handling)
-   [Error Handling](#error-handling)
-   [Helper Functions](#helper-functions)
-   [Examples](#examples)
-   [Testing](#testing)
-   [Best Practices](#best-practices)
-   [API Reference](#api-reference)
-   [Contributing](#contributing)
-   [License](#license)
-   [Support](#support)
-   [Changelog](#changelog)

## Installation

```bash
go get github.com/QBitFlow/qbitflow-go-sdk
```

## Quick Start

### 1. Get Your API Key

Sign up at [QBitFlow](https://qbitflow.app) and obtain your API key from the dashboard.

### 2. Initialize the Client

```go
package main

import (
	"github.com/QBitFlow/qbitflow-go-sdk"
)

func main() {
	// Initialize the client
	client := qbitflow.New("your-api-key")
}
```

### 3. Create a One-Time Payment

```go
import (
	"fmt"
	"log"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/utils"
)

// Create a one-time payment
payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	ProductID:    new(uint64(1)),
	CustomerUUID: new(string("customer-uuid")),
	WebhookURL:   new(string("https://yourapp.com/webhook")),
	SuccessURL:   new(string("https://yourapp.com/success")),
	CancelURL:    new(string("https://yourapp.com/cancel")),
})

if err != nil {
	log.Fatal(err)
}

fmt.Printf("Payment link: %s\n", payment.Link)
// Send this link to your customer
```

### 4. Create a Recurring Subscription

```go
import (
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
)

subscription, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
	ProductID: 1,
	Frequency: qbmodels.Duration{
		Value: 1,
		Unit:  qbmodels.DurationUnitMonths, // Bill monthly
	},
	TrialPeriod: &qbmodels.Duration{ // 7-day trial (optional)
		Value: 7,
		Unit:  qbmodels.DurationUnitDays,
	},
	WebhookURL:   new(string("https://yourapp.com/webhook")),
	CustomerUUID: new(string("customer-uuid")),
})

if err != nil {
	log.Fatal(err)
}

fmt.Printf("Subscription link: %s\n", subscription.Link)
```

### 5. Check Transaction Status

```go
import (
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

status, err := client.TransactionStatus.GetTransactionStatus(
	"transaction-uuid",
	qbmodels.TransactionTypeOneTimePayment,
)

if err != nil {
	log.Fatal(err)
}

if status.Status == qbmodels.TransactionStatusCompleted {
	fmt.Printf("Payment completed! Transaction hash: %s\n", *status.TxHash)
} else if status.Status == qbmodels.TransactionStatusFailed {
	fmt.Printf("Payment failed: %s\n", *status.Message)
}
```

## Configuration

### Basic Configuration

```go
client := qbitflow.New("your-api-key")
```

### Advanced Configuration

```go
import (
	"time"
	
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
)

client := qbitflow.NewWithConfig(qbf.Config{
	APIKey:     "your-api-key",
	Timeout:    30 * time.Second,           // Optional: request timeout (default: 30s)
	MaxRetries: 3,                          // Optional: max retry attempts (default: 3)
})
```

### Configuration Options

| Option       | Type          | Default                    | Description                                  |
| ------------ | ------------- | -------------------------- | -------------------------------------------- |
| `APIKey`     | string        | (required)                 | Your QBitFlow API key                        |
| `BaseURL`    | string        | `https://api.qbitflow.app` | API base URL                                 |
| `Timeout`    | time.Duration | `30 * time.Second`         | Request timeout                              |
| `MaxRetries` | int           | `3`                        | Number of retry attempts for failed requests |

## One-Time Payments

### Create a Payment Session

Create a payment session for a one-time purchase:

```go
// Using an existing product
payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	ProductID:    new(uint64(1)),
	CustomerUUID: new(string("customer-uuid")), // optional
	WebhookURL:   new(string("https://yourapp.com/webhook")),
})

if err != nil {
	log.Fatal(err)
}

fmt.Printf("Session UUID: %s\n", payment.UUID)
fmt.Printf("Payment link: %s\n", payment.Link)
```

**Or create a custom payment:**

```go
payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	ProductName:  new(string("Custom Product")),
	Description:  new(string("Product description")),
	Price:        new(float64(99.99)), // USD
	CustomerUUID: new(string("customer-uuid")),
	WebhookURL:   new(string("https://yourapp.com/webhook")),
})
```

### With Redirect URLs

You can provide redirect URLs for success and cancellation:

```go
payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	ProductID:    new(uint64(1)),
	SuccessURL:   new(string("https://yourapp.com/success?uuid={{UUID}}&type={{TRANSACTION_TYPE}}")),
	CancelURL:    new(string("https://yourapp.com/cancel")),
	CustomerUUID: new(string("customer-uuid")),
})
```

**Available Placeholders:**

-   `{{UUID}}`: The session UUID
-   `{{TRANSACTION_TYPE}}`: The transaction type (e.g., "payment", "subscription", "payAsYouGo")

### Get Payment Session

Retrieve details of a payment session:

```go
session, err := client.Payments.GetSession("session-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Product: %s, Price: %.2f\n", session.ProductName, session.Price)
```

### Get Completed Payment

Retrieve details of a completed payment:

```go
payment, err := client.Payments.GetPayment("payment-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Transaction Hash: %s\n", payment.TransactionHash)
fmt.Printf("Amount: %.2f\n", payment.Amount)
fmt.Printf("Status: %s\n", payment.Status)
```

### List All Payments

List all one-time payments with pagination:

```go
limit := uint16(10)
result, err := client.Payments.GetAllPayments(&limit, nil)
if err != nil {
	log.Fatal(err)
}

for _, payment := range result.Items {
	fmt.Printf("Payment: %s - Amount: %.2f\n", payment.UUID, payment.Amount)
}

// Check if there are more pages
if result.HasMore() {
	// Fetch next page
	nextPage, err := client.Payments.GetAllPayments(&limit, result.NextCursor)
	if err != nil {
		log.Fatal(err)
	}
	// Process next page...
}
```

### List Combined Payments

Get all payments (one-time and subscription payments combined):

```go
limit := uint16(20)
result, err := client.Payments.GetAllCombinedPayments(&limit, nil)
if err != nil {
	log.Fatal(err)
}

for _, payment := range result.Items {
	fmt.Printf("Payment: %s - Source: %v - Amount: %.2f\n",
		payment.UUID, payment.Source, payment.Amount)
}
```

## Subscriptions

### Create a Subscription

Create a recurring subscription:

```go
minPeriods := uint64(3) // Minimum billing periods (optional)

subscription, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
	ProductID: 1,
	Frequency: qbmodels.Duration{
		Value: 1,
		Unit:  qbmodels.DurationUnitMonths, // Bill monthly
	},
	TrialPeriod: &qbmodels.Duration{ // 7-day trial (optional)
		Value: 7,
		Unit:  qbmodels.DurationUnitDays,
	},
	MinPeriods:   &minPeriods,
	WebhookURL:   new(string("https://yourapp.com/webhook")),
	CustomerUUID: new(string("customer-uuid")),
})

if err != nil {
	log.Fatal(err)
}

fmt.Printf("Subscription link: %s\n", subscription.Link)
```

### Frequency Units

Available units for `Frequency` and `TrialPeriod`:

-   `qbmodels.DurationUnitSeconds`
-   `qbmodels.DurationUnitMinutes`
-   `qbmodels.DurationUnitHours`
-   `qbmodels.DurationUnitDays`
-   `qbmodels.DurationUnitWeeks`
-   `qbmodels.DurationUnitMonths`

### Get Subscription

Retrieve subscription details:

```go
subscription, err := client.Subscriptions.GetSubscription("subscription-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Status: %s\n", subscription.Status)
fmt.Printf("Next Billing Date: %s\n", subscription.NextBillingDate)
```

### Get Payment History

Retrieve payment history for a subscription:

```go
history, err := client.Subscriptions.GetPaymentHistory("subscription-uuid")
if err != nil {
	log.Fatal(err)
}

for _, record := range history {
	fmt.Printf("Payment: %s - Amount: %.2f - Date: %s\n",
		record.UUID, record.Amount, record.CreatedAt)
}
```

### Execute Test Billing Cycle

**Test Mode Only**: Manually trigger a billing cycle for testing.

**For live mode**: Billing cycles are executed automatically based on the subscription frequency.

```go
response, err := client.Subscriptions.ExecuteTestBillingCycle("subscription-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Status Link: %s\n", response.StatusLink)
```

## Pay-As-You-Go Subscriptions

PAYG subscriptions allow customers to pay based on usage with a billing cycle.

### Create PAYG Subscription

```go
freeCredits := 100.0

payg, err := client.PayAsYouGo.CreateSession(&qbf.CreatePAYGSessionOptions{
	ProductID: 1,
	Frequency: qbmodels.Duration{
		Value: 1,
		Unit:  qbmodels.DurationUnitMonths, // Bill monthly
	},
	FreeCredits:  &freeCredits, // Initial free credits (optional)
	WebhookURL:   new(string("https://yourapp.com/webhook")),
	CustomerUUID: new(string("customer-uuid")),
})

if err != nil {
	log.Fatal(err)
}

fmt.Printf("PAYG link: %s\n", payg.Link)
```

### Get PAYG Subscription

```go
payg, err := client.PayAsYouGo.GetSubscription("payg-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Allowance: %.2f\n", payg.Allowance)
fmt.Printf("Units Current Period: %.2f\n", payg.UnitsCurrentPeriod)
```

### Get Payment History

```go
history, err := client.PayAsYouGo.GetPaymentHistory("payg-uuid")
if err != nil {
	log.Fatal(err)
}

for _, record := range history {
	fmt.Printf("Payment: %s - Amount: %.2f - Date: %s\n",
		record.UUID, record.Amount, record.CreatedAt)
}
```

### Execute Test Billing Cycle

**Test Mode Only**: Manually trigger a billing cycle for testing.

**For live mode**: Billing cycles are executed automatically based on the subscription frequency.

```go
response, err := client.PayAsYouGo.ExecuteTestBillingCycle("subscription-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Status Link: %s\n", response.StatusLink)
```

### Increase Units Current Period

Increase the number of units for the current billing period:

```go
// For example, the product is billed per hour of usage,
// and the customer consumed 5 additional hours
response, err := client.PayAsYouGo.IncreaseUnitsCurrentPeriod("payg-uuid", 5.0)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("New Current Units: %.2f\n", response.CurrentUnits)
```

## Transaction Status

### Check Status

Get the current status of a transaction:

```go
status, err := client.TransactionStatus.GetTransactionStatus(
	"transaction-uuid",
	qbmodels.TransactionTypeOneTimePayment,
)

if err != nil {
	log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status) // "created", "pending", "completed", etc.
if status.TxHash != nil {
	fmt.Printf("Transaction Hash: %s\n", *status.TxHash)
}
```

### Transaction Types

```go
const (
	// One-time payment transaction
	TransactionTypeOneTimePayment TransactionType = "payment"

	// Create subscription transaction
	TransactionTypeCreateSubscription TransactionType = "createSubscription"

	// Cancel subscription transaction
	TransactionTypeCancelSubscription TransactionType = "cancelSubscription"

	// Execute subscription payment transaction
	TransactionTypeExecuteSubscriptionPayment TransactionType = "executeSubscription"

	// Create pay-as-you-go subscription transaction
	TransactionTypeCreatePAYGSubscription TransactionType = "createPAYGSubscription"

	// Cancel pay-as-you-go subscription transaction
	TransactionTypeCancelPAYGSubscription TransactionType = "cancelPAYGSubscription"

	// Increase allowance transaction
	TransactionTypeIncreaseAllowance TransactionType = "increaseAllowance"

	// Update max amount transaction
	TransactionTypeUpdateMaxAmount TransactionType = "updateMaxAmount"
)
```

### Status Values

```go
const (
	// Transaction has been created but not yet processed
	TransactionStatusCreated TransactionStatusValue = "created"

	// Waiting for blockchain confirmation
	TransactionStatusWaitingConfirmation TransactionStatusValue = "waitingConfirmation"

	// Transaction is pending processing
	TransactionStatusPending TransactionStatusValue = "pending"

	// Transaction has been successfully completed
	TransactionStatusCompleted TransactionStatusValue = "completed"

	// Transaction has failed
	TransactionStatusFailed TransactionStatusValue = "failed"

	// Transaction has been cancelled
	TransactionStatusCancelled TransactionStatusValue = "cancelled"

	// Transaction has expired
	TransactionStatusExpired TransactionStatusValue = "expired"
)
```

## Customer Management

### Create a Customer

```go
customerData := &qbf.CreateCustomerOptions{
	Name:        "John",
	LastName:    "Doe",
	Email:       "john@example.com",
	PhoneNumber: new(string("+1234567890")),
	Reference:   new(string("CRM-12345")),
}

customer, err := client.Customers.Create(customerData)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Customer created: %s\n", customer.UUID)
```

### Get Customer by UUID

```go
customer, err := client.Customers.Get("customer-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Printf("%s %s - %s\n", customer.Name, customer.LastName, customer.Email)
```

### Update Customer

```go
updateData := &qbf.UpdateCustomerOptions{
	UUID:        "customer-uuid",
	Name:        "John",
	LastName:    "Doe",
	Email:       "john.doe@example.com",
	PhoneNumber: new(string("+9876543210")),
}

updatedCustomer, err := client.Customers.Update("customer-uuid", updateData)
if err != nil {
	log.Fatal(err)
}
```

### Delete Customer

```go
response, err := client.Customers.Delete("customer-uuid")
if err != nil {
	log.Fatal(err)
}

fmt.Println(response.Message)
```

## Product Management

### Create a Product

```go
productData := &qbf.CreateProductOptions{
	Name:        "Premium Subscription",
	Description: "Access to all premium features",
	Price:       29.99,
	Reference:   new(string("PROD-PREMIUM")),
}

product, err := client.Products.Create(productData)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Product created: ID %d\n", product.ID)
```

### Update Product

```go
updateData := &qbf.UpdateProductOptions{
	Name:        "Premium Plus",
	Description: "Enhanced premium features",
	Price:       39.99,
}

updatedProduct, err := client.Products.Update(1, updateData)
if err != nil {
	log.Fatal(err)
}
```

### Delete Product

```go
response, err := client.Products.Delete(1)
if err != nil {
	log.Fatal(err)
}

fmt.Println(response.Message)
```

## Webhook Handling

Handle webhook notifications from QBitFlow:

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var webhook qbmodels.SessionWebhookResponse

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("Webhook received: %s\n", webhook.UUID)
	fmt.Printf("Status: %s\n", webhook.Status.Status)

	// Process webhook based on status
	switch webhook.Status.Status {
	case qbmodels.TransactionStatusCompleted:
		fmt.Println("Payment completed!")
		// Handle successful payment
		// Update your database, fulfill order, etc.

	case qbmodels.TransactionStatusFailed:
		fmt.Println("Payment failed")
		// Handle payment failure

	case qbmodels.TransactionStatusCancelled:
		fmt.Println("Payment cancelled")
		// Handle cancellation
	}

	// Always respond with 200 OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)

	fmt.Println("Webhook server running on :8080")
	http.ListenAndServe(":8080", nil)
}
```

### Webhook Payload

```go
type SessionWebhookResponse struct {
	// Session UUID
	UUID string `json:"uuid"`

	// Current transaction status
	Status TransactionStatus `json:"status"`

	// Complete session information
	Session Session `json:"session"`
}
```

The webhook payload includes:

-   `UUID`: Session UUID
-   `Status`: Current transaction status with type, status value, and optional transaction hash
-   `Session`: Complete session details including product info, price, customer UUID, etc.

## Error Handling

The SDK provides specific error types for different scenarios:

### QBitFlowError

General API errors with status codes.

```go
import qberrors "github.com/QBitFlow/qbitflow-go-sdk/pkg/errors"

session, err := client.Payments.CreateSession(opts)
if err != nil {
	if qbErr, ok := err.(*qberrors.QBitFlowError); ok {
		fmt.Printf("Status Code: %d\n", qbErr.StatusCode)
		fmt.Printf("Message: %s\n", qbErr.Message)
	}
}
```

### NotFoundError

Specific error for 404 not found responses.

```go
payment, err := client.Payments.GetPayment("invalid-uuid")
if err != nil {
	if _, ok := err.(*qberrors.NotFoundError); ok {
		fmt.Println("Payment not found")
	}
}
```

### UnauthorizedError

Error for 401 unauthorized responses (invalid API key).

```go
if _, ok := err.(*qberrors.UnauthorizedError); ok {
	fmt.Println("Invalid API key")
}
```

### ValidationError

Client-side validation errors.

```go
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	// Missing required fields
})
if err != nil {
	if valErr, ok := err.(*qberrors.ValidationError); ok {
		fmt.Printf("Validation Error: %s\n", valErr.Message)
	}
}
```

### NetworkError

Network-related errors.

```go
if netErr, ok := err.(*qberrors.NetworkError); ok {
	fmt.Printf("Network error: %s\n", netErr.Message)
}
```

### Complete Error Handling Example

```go
import qberrors "github.com/QBitFlow/qbitflow-go-sdk/pkg/errors"

payment, err := client.Payments.GetPayment("payment-uuid")
if err != nil {
	switch e := err.(type) {
	case *qberrors.NotFoundError:
		fmt.Println("Payment not found")
	case *qberrors.UnauthorizedError:
		fmt.Println("Invalid API key")
	case *qberrors.ValidationError:
		fmt.Printf("Invalid request: %s\n", e.Message)
	case *qberrors.NetworkError:
		fmt.Printf("Network error: %s\n", e.Message)
	case *qberrors.QBitFlowError:
		fmt.Printf("QBitFlow error: %s\n", e.Message)
	default:
		fmt.Printf("Unknown error: %v\n", err)
	}
	return
}
```

## Examples

See the [examples](examples/) directory for complete working examples:

-   [basic_payment.go](examples/basic_example/basic_payment.go) - Basic one-time payment
-   [subscription.go](examples/subscription/subscription.go) - Creating subscriptions
-   [payg.go](examples/payg/payg.go) - Pay-as-you-go subscriptions
-   [webhook_handler.go](examples/webhook_handler/webhook_handler.go) - Handling webhooks
-   [complete_flow.go](examples/complete_flow/complete_flow.go) - Complete payment flow
-   [customer_management.go](examples/customer_management/customer_management.go) - Customer operations
-   [product_management.go](examples/product_management/product_management.go) - Product operations

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests verbosely
go test -v ./...

# Run integration tests (requires API key)
export QBITFLOW_API_KEY="your-api-key"
go test -v ./tests/...

# Run tests in watch mode (requires gotestsum)
gotestsum --watch
```

## Best Practices

1. **Always handle errors**: Check and handle all errors returned by SDK methods
2. **Use webhooks**: For production, use webhooks to get real-time updates on payment status
3. **Store session UUIDs**: Always store session UUIDs to track payments later
4. **Validate before creating**: Ensure product IDs or product details are valid before creating sessions
5. **Test mode first**: Always test your integration in test mode before going live
6. **Handle status transitions**: Check transaction status after redirects to confirm completion
7. **Use pointer helpers**: Utilize the `utils` package helper functions for optional fields
8. **Context management**: Use appropriate context timeouts for long-running operations
9. **Graceful degradation**: Handle API errors gracefully and provide fallback mechanisms
10. **Secure API keys**: Store API keys securely using environment variables or secret management systems

## API Reference

### QBitFlow Client

Main client struct providing access to all API endpoints.

#### Constructor

```go
// Create client with API key only
client := qbitflow.New("your-api-key")

// Create client with custom configuration
client := qbitflow.NewWithConfig(qbf.Config{
	APIKey:     "your-api-key",
	Timeout:    30 * time.Second,
	MaxRetries: 3,
})
```

#### Properties

-   `Customers` - Customer management operations
-   `Products` - Product management operations
-   `Users` - User management operations
-   `ApiKeys` - API key management operations
-   `Payments` - One-time payment operations
-   `Subscriptions` - Subscription operations
-   `PayAsYouGo` - Pay-as-you-go operations
-   `TransactionStatus` - Transaction status operations

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This SDK is released under the MPL-2.0 License. See [LICENSE](LICENSE) file for details.

## Support

-   📖 [Documentation](https://qbitflow.app/docs)
-   📧 [Email Support](mailto:support@qbitflow.app)
-   🐛 [Issue Tracker](https://github.com/QBitFlow/qbitflow-go-sdk/issues)

## Changelog

### v1.0.0 (Initial Release)

-   Complete API coverage for all QBitFlow endpoints
-   Support for one-time payments
-   Support for subscriptions
-   Support for pay-as-you-go subscriptions
-   Customer management
-   Product management
-   Transaction status checking
-   Comprehensive error handling
-   Full documentation and examples
-   Automatic retry logic
-   Type-safe structs and constants

## Security

For security issues, please email security@qbitflow.app instead of using the issue tracker.

---

Made with ❤️ by the QBitFlow team
