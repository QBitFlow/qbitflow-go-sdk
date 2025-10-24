# QBitFlow Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://upload.wikimedia.org/wikipedia/commons/thumb/2/2e/MIT_Logo_New.svg/1200px-MIT_Logo_New.svg.png)

The official Go SDK for [QBitFlow](https://qbitflow.app) - a comprehensive cryptocurrency payment processing platform that enables seamless integration of crypto payments, recurring subscriptions, and pay-as-you-go models into your applications.

## Features

-   ✅ **Complete API Coverage**: Support for all QBitFlow API endpoints
-   🔒 **Secure**: Built-in API key authentication
-   🎯 **Type-Safe**: Fully typed structs for all requests and responses
-   🚀 **Easy to Use**: Intuitive interface with comprehensive examples
-   🧪 **Well Tested**: Includes integration tests
-   📚 **Well Documented**: Comprehensive documentation and examples
-   ⚡ **Production Ready**: Error handling and best practices built-in

## Table of Contents

-   [Installation](#installation)
-   [Quick Start](#quick-start)
-   [Configuration](#configuration)
-   [API Reference](#api-reference)
    -   [One-Time Payments](#one-time-payments)
    -   [Subscriptions](#subscriptions)
    -   [Pay-As-You-Go Subscriptions](#pay-as-you-go-subscriptions)
    -   [Transaction Status](#transaction-status)
-   [Examples](#examples)
-   [Error Handling](#error-handling)
-   [Testing](#testing)
-   [Contributing](#contributing)
-   [License](#license)

## Installation

```bash
go get github.com/qbitflow/qbitflow-go-sdk
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
    "github.com/qbitflow/qbitflow-go-sdk/pkg/models"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/utils"
)

func main() {
    // Initialize the client with your API key
    client := qbitflow.New("your-api-key-here")

    // Create a one-time payment session
    session, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
		ProductID:    utils.Uint64Ptr(1),
		SuccessURL:   utils.StringPtr("https://yoursite.com/success?uuid={{UUID}}&type={{TRANSACTION_TYPE}}"),
		CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-123"),
	})

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Payment link: %s\n", session.Link)
    fmt.Printf("Session UUID: %s\n", session.UUID)
}
```

## Configuration

### Basic Configuration

```go
client := qbitflow.New("your-api-key-here")
```

### Advanced Configuration

```go
import "time"

client := qbitflow.NewWithConfig(qbitflow.Config{
    APIKey:  "your-api-key-here",
    Timeout: 30 * time.Second,            // Custom timeout
})
```

## API Reference

### One-Time Payments

#### Create Payment Session

Create a payment session for a one-time payment.

```go
session, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
	ProductID:    utils.Uint64Ptr(1),
	SuccessURL:   utils.StringPtr("https://yoursite.com/success?uuid={{UUID}}&type={{TRANSACTION_TYPE}}"),
	CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
	WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
	CustomerUUID: utils.StringPtr("customer-uuid-123"),
})
```

**URL Placeholders:**

-   `{{UUID}}`: The UUID of the created payment session
-   `{{TRANSACTION_TYPE}}`: The type of transaction (e.g., "payment")

**Using Custom Product (without ProductID):**

```go
client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
	ProductName:  utils.StringPtr("Premium Membership"),
	Description:  utils.StringPtr("One-time payment for premium membership"),
	Price:        utils.Float64Ptr(99.99),
	SuccessURL:   utils.StringPtr("https://yoursite.com/success"),
	CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
	CustomerUUID: utils.StringPtr("customer-uuid-456"),
})
```

#### Get Payment Session

Retrieve details of a payment session.

```go
session, err := client.Payments.GetSession("session-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Session: %+v\n", session)
```

#### Get Payment

Retrieve a completed one-time payment.

```go
payment, err := client.Payments.GetPayment("payment-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Payment Amount: %.2f\n", payment.Amount)
fmt.Printf("Status: %s\n", payment.Status)
```

#### Get All Payments

Retrieve all one-time payments with pagination.

```go
limit := 20
payments, err := client.Payments.GetAllPayments(&limit, nil)
if err != nil {
    log.Fatal(err)
}

for _, payment := range payments.Items {
    fmt.Printf("Payment: %s - Amount: %.2f\n", payment.UUID, payment.Amount)
}

// Paginate through results
if payments.HasMore() {
    nextPage, err := client.Payments.GetAllPayments(&limit, payments.NextCursor)
    // ... handle next page
}
```

#### Get All Combined Payments

Retrieve all payments (one-time + subscription).

```go
limit := 20
payments, err := client.Payments.GetAllCombinedPayments(&limit, nil)
if err != nil {
    log.Fatal(err)
}

for _, payment := range payments.Items {
    fmt.Printf("Payment: %s - Source %v\n",
        payment.UUID, payment.Source)
}
```

### Subscriptions

#### Create Subscription Session

Create a subscription with recurring payments.

```go
session, err := client.Subscriptions.CreateSession(&qbitflow.CreateSubscriptionSessionOptions{
	ProductID: 1,
	Frequency: models.Duration{
		Value: 1,
		Unit:  models.DurationUnitMonths,
	},
	TrialPeriod: &models.Duration{ // Optional trial period
		Value: 7,
		Unit:  models.DurationUnitDays,
	},
	SuccessURL:   utils.StringPtr("https://yoursite.com/subscription/success"),
	CancelURL:    utils.StringPtr("https://yoursite.com/subscription/cancel"),
	WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
	CustomerUUID: utils.StringPtr("customer-uuid-123"),
})
```

**Duration Units:**

-   `models.DurationUnitSeconds`
-   `models.DurationUnitMinutes`
-   `models.DurationUnitHours`
-   `models.DurationUnitDays`
-   `models.DurationUnitWeeks`
-   `models.DurationUnitMonths`

#### Get Subscription

Retrieve subscription details.

```go
subscription, err := client.Subscriptions.GetSubscription("subscription-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Next Billing Date: %s\n", subscription.NextBillingDate)
fmt.Printf("Status: %s\n", subscription.Status)
```

#### Get Subscription Session

```go
session, err := client.Subscriptions.GetSession("session-uuid-here")
```

#### Get Payment History

Retrieve payment history for a subscription.

```go
history, err := client.Subscriptions.GetPaymentHistory("subscription-uuid-here")
if err != nil {
    log.Fatal(err)
}

for _, payment := range history {
    fmt.Printf("Payment Date: %s - TxHash: %s\n",
        payment.CreatedAt, payment.TransactionHash)
}
```

#### Force Cancel Subscription

**⚠️ Use with caution!** Force cancel a subscription immediately, bypassing the customer verification step.

```go
response, err := client.Subscriptions.ForceCancel("subscription-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Message)
```

#### Execute Test Billing Cycle

**Test Mode Only**: Manually trigger a billing cycle for testing.

**For live mode**: Billing cycles are executed automatically based on the subscription frequency.

```go
response, err := client.Subscriptions.ExecuteTestBillingCycle("subscription-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status Link: %s\n", response.StatusLink)
```

### Pay-As-You-Go Subscriptions

Pay-as-you-go subscriptions allow customers to pay based on usage with periodic billing.

#### Create PAYG Session

```go
freeCredits := 10.0

session, err := client.PayAsYouGo.CreateSession(&qbitflow.CreatePAYGSessionOptions{
    ProductID: 1,
    Frequency: models.Duration{
        Value: 1,
        Unit:  models.DurationUnitMonths,
    },
    FreeCredits:  &freeCredits, // Optional free credits to start with
    SuccessURL:   utils.StringPtr("https://yoursite.com/success"),
    CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
    WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
    CustomerUUID: utils.StringPtr("customer-uuid-123"),
})
```

#### Get PAYG Subscription

```go
subscription, err := client.PayAsYouGo.GetSubscription("subscription-uuid-here")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Current Units: %.2f; Max spending USD: %.2f\n",
    subscription.UnitsCurrentPeriod, subscription.MaxSpendingPerPeriod)
```

#### Increase Units for Current Period

Increase the usage allowance for the current billing period.

```go
subscription, err := client.PayAsYouGo.IncreaseUnitsCurrentPeriod(
    "subscription-uuid-here",
    50.0, // Increase by 50 units
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("New Current Units: %.2f\n", subscription.CurrentUnits)
```

#### Get Payment History

```go
history, err := client.PayAsYouGo.GetPaymentHistory("subscription-uuid-here")
```

#### Force Cancel & Execute Test Billing

Same as regular subscriptions:

```go
// Force cancel
response, err := client.PayAsYouGo.ForceCancel("subscription-uuid-here")

// Execute test billing (test mode only)
response, err := client.PayAsYouGo.ExecuteTestBillingCycle("subscription-uuid-here")
```

### Transaction Status

#### Get Transaction Status

Check the status of any transaction.

```go
status, err := client.TransactionStatus.GetTransactionStatus(
    "transaction-uuid-here",
    models.TransactionTypeOneTimePayment,
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status)
if status.TxHash != nil {
    fmt.Printf("Transaction Hash: %s\n", *status.TxHash)
}
```

**Transaction Types:**

-   `models.TransactionTypeOneTimePayment`
-   `models.TransactionTypeCreateSubscription`
-   `models.TransactionTypeCancelSubscription`
-   `models.TransactionTypeExecuteSubscriptionPayment`
-   `models.TransactionTypeCreatePAYGSubscription`
-   `models.TransactionTypeCancelPAYGSubscription`
-   `models.TransactionTypeIncreaseAllowance`
-   `models.TransactionTypeUpdateMaxAmount`

**Transaction Status Values:**

-   `models.TransactionStatusCreated`
-   `models.TransactionStatusWaitingConfirmation`
-   `models.TransactionStatusPending`
-   `models.TransactionStatusCompleted`
-   `models.TransactionStatusFailed`
-   `models.TransactionStatusCancelled`
-   `models.TransactionStatusExpired`

## Examples

See the [examples](examples/) directory for complete working examples:

-   [basic_payment.go](examples/basic_example/basic_payment.go) - Basic one-time payment
-   [subscription.go](examples/subscription/subscription.go) - Creating subscriptions
-   [payg.go](examples/payg/payg.go) - Pay-as-you-go subscriptions
-   [webhook_handler.go](examples/webhook_handler/webhook_handler.go) - Handling webhooks
-   [complete_flow.go](examples/complete_flow/complete_flow.go) - Complete payment flow

## Error Handling

The SDK provides custom error types for better error handling:

### QBitFlowError

General API errors with status codes.

```go
session, err := client.Payments.CreateSession(opts)
if err != nil {
    if qbErr, ok := err.(*errors.QBitFlowError); ok {
        fmt.Printf("Status Code: %d\n", qbErr.StatusCode)
        fmt.Printf("Message: %s\n", qbErr.Message)
    }
}
```

### NotFoundError

Specific error for 404 not found responses.

```go
import qberrors "github.com/qbitflow/qbitflow-go-sdk/pkg/errors"

payment, err := client.Payments.GetPayment("invalid-uuid")
if err != nil {
    if _, ok := err.(*qberrors.NotFoundError); ok {
        fmt.Println("Payment not found")
    }
}
```

### ValidationError

Client-side validation errors.

```go
import qberrors "github.com/qbitflow/qbitflow-go-sdk/pkg/errors"

session, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
    // Missing required fields
})
if err != nil {
    if valErr, ok := err.(*qberrors.ValidationError); ok {
        fmt.Printf("Validation Error: %s\n", valErr.Message)
    }
}
```

## Testing

Run the tests:

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
```

## Helper Functions

The SDK uses pointer values for optional fields. Here are some helpful utility functions:

```go
// Helper functions for pointer conversions
func IntPtr(i int) *int             { return &i }
func Uint64Ptr(u uint64) *uint64    { return &u }
func StringPtr(s string) *string    { return &s }
func Float64Ptr(f float64) *float64 { return &f }
```

## Best Practices

1. **Always handle errors**: Check and handle all errors returned by SDK methods
2. **Use webhooks**: For production, use webhooks to get real-time updates on payment status
3. **Store session UUIDs**: Always store session UUIDs to track payments later
4. **Validate before creating**: Ensure product IDs or product details are valid before creating sessions
5. **Test mode first**: Always test your integration in test mode before going live
6. **Handle status transitions**: Check transaction status after redirects to confirm completion

## Webhook Handling Example

```go
package main

import (
    "encoding/json"
    "net/http"

    "github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    var webhook models.SessionWebhookResponse

    if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
        http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
        return
    }

    // Process webhook
    switch webhook.Status.Status {
    case models.TransactionStatusCompleted:
        // Payment completed successfully
        // Update your database, fulfill order, etc.
    case models.TransactionStatusFailed:
        // Payment failed
        // Handle failure
    case models.TransactionStatusCancelled:
        // Payment cancelled
        // Handle cancellation
    }

	// Always respond with 200 OK
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This SDK is released under the MIT License. See [LICENSE](LICENSE) file for details.

## Support

For support, please contact:

-   Email: support@qbitflow.app
-   Documentation: https://qbitflow.app/docs
-   GitHub Issues: https://github.com/qbitflow/qbitflow-go-sdk/issues

## Changelog

### v1.0.0 (Initial Release)

-   Complete API coverage for all QBitFlow endpoints
-   Support for one-time payments
-   Support for subscriptions
-   Support for pay-as-you-go subscriptions
-   Transaction status checking
-   Comprehensive error handling
-   Full documentation and examples
