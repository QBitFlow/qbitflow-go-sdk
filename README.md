# QBitFlow Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

Official Go SDK for [QBitFlow](https://qbitflow.app) — a cryptocurrency payment processing platform supporting one-time payments, recurring subscriptions, refunds, and accounting exports.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Acting on Behalf of a User](#acting-on-behalf-of-a-user)
- [One-Time Payments](#one-time-payments)
- [Subscriptions](#subscriptions)
- [Transaction Status](#transaction-status)
- [Refunds](#refunds)
- [Accounting Export](#accounting-export)
- [Account Claims](#account-claims)
- [Customer Management](#customer-management)
- [Product Management](#product-management)
- [User Management](#user-management)
- [API Key Management](#api-key-management)
- [Currencies](#currencies)
- [Webhook Handling](#webhook-handling)
- [Error Handling](#error-handling)
- [Testing](#testing)

## Installation

```bash
go get github.com/QBitFlow/qbitflow-go-sdk/v2
```

**Requires Go 1.26 or later.** The partial-update DTOs use pointer fields, and the examples
below set them with `new(value)` — Go 1.26 extended `new` to accept a value expression, not
just a type.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/QBitFlow/qbitflow-go-sdk/v2"
    qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
    client := qbitflow.New("your-api-key")

    // Create a one-time payment session
    session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
        ProductName:  new("Premium Access"),
        Description:  new("One-time purchase"),
        Price:        new(49.99),
        SuccessURL:   new("https://yourapp.com/success"),
        CancelURL:    new("https://yourapp.com/cancel"),
        CustomerUUID: new("customer-uuid"),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Payment link: %s\n", session.Link)

    // Check transaction status
    status, err := client.TransactionStatus.GetTransactionStatus(
        session.UUID,
        qbmodels.TransactionTypeOneTimePayment,
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Status: %s\n", status.Status)
}
```

## Configuration

```go
// Minimal — API key only
client := qbitflow.New("your-api-key")

// Custom configuration
import "time"

client := qbitflow.NewWithConfig(qbf.Config{
    APIKey:  "your-api-key",
    BaseURL: "https://api.qbitflow.app/v1", // optional, this is the default
    Timeout: 30 * time.Second,              // optional, default 30s
})

// Override base URL after creation (useful for testing)
client.SetBaseURL("http://localhost:3001")
```

## Acting on Behalf of a User

If you hold an **organization (admin) API key**, you can perform any request as one of the users in your organization, without needing that user's own API key. This is useful for admin-level tooling, dashboards, and back-office automation where your server acts for a specific user (e.g. listing *their* products, creating a payment session *for them*, or reading *their* subscriptions).

Every service exposes an `OnBehalfOf(userID)` method. It returns a scoped copy of that service which adds an `On-Behalf-Of` header to each request; the original `client` is left untouched, so you can freely mix org-level and per-user calls.

```go
const userID uint64 = 123

// List the products belonging to user 123
products, err := client.Products.OnBehalfOf(userID).GetAll()

// Read that user's payments
payments, err := client.Payments.OnBehalfOf(userID).GetAllPayments(nil, nil)

// The base client is unaffected — this call still runs at the organization level
allOrgProducts, err := client.Products.GetAll()
```

> **Note:** `OnBehalfOf` requires an admin-level API key. Using it with a regular user key results in a `403` (`*qberrors.QBitFlowError`).

## One-Time Payments

### Create a Payment Session

Provide an existing `ProductID`, a `ProductReference`, **or** the (`ProductName` + `Description` + `Price`) triplet.

```go
// Using an existing product
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
    ProductID:    new(uint64(42)),
    SuccessURL:   new("https://yourapp.com/success"),
    CancelURL:    new("https://yourapp.com/cancel"),
    CustomerUUID: new("customer-uuid"),
})

// Using an inline product definition
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
    ProductName: new("Custom Item"),
    Description: new("Limited edition widget"),
    Price:       new(19.99),
    SuccessURL:  new("https://yourapp.com/success"),
    CancelURL:   new("https://yourapp.com/cancel"),
})
```

#### Using your own references

Instead of storing QBitFlow's internal UUIDs, pass your own identifiers. Set `Reference` to your
order/invoice ID, and use `ProductReference` / `CustomerReference` to select an existing product or
customer by your own reference:

```go
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
    Reference:         new("order-1234"),      // your own transaction reference
    ProductReference:  new("PROD-PREMIUM"),    // use a product by your reference (instead of ProductID)
    CustomerReference: new("user-42"),         // use a customer by your reference (instead of CustomerUUID)
})
```

The `Reference` is echoed back on the resulting `Payment` and in webhook payloads, and you can look
the payment up later with [`GetPaymentByReference`](#get-a-completed-payment). If no customer matches
`CustomerReference`, one is created during checkout.

### Retrieve a Session

```go
session, err := client.Payments.GetSession("session-uuid")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Product: %s  Price: $%.2f\n", session.ProductName, session.Price)
fmt.Printf("Is payment: %v\n", session.IsPayment())
```

### Get a Completed Payment

```go
payment, err := client.Payments.GetPayment("payment-uuid")
fmt.Printf("Amount: $%.2f  Hash: %s\n", payment.Amount, payment.TransactionHash)
```

You can also look a payment up by the `Reference` you assigned when creating the session — no need
to store QBitFlow's UUID:

```go
payment, err := client.Payments.GetPaymentByReference("order-1234")
fmt.Printf("UUID: %s  Amount: $%.2f\n", payment.UUID, payment.Amount)
```

### List Payments

```go
limit := uint16(20)

// One-time payments only
result, err := client.Payments.GetAllPayments(&limit, nil)

// Combined (one-time + subscription billing cycles)
combined, err := client.Payments.GetAllCombinedPayments(&limit, nil)

// Paginate
if result.HasMore() {
    nextPage, err := client.Payments.GetAllPayments(&limit, result.NextCursor)
    _ = nextPage
}
```

### Get Customer for a Transaction

```go
customer, err := client.Payments.GetCustomerForTransaction("transaction-uuid")
fmt.Printf("%s %s — %s\n", customer.Name, customer.LastName, customer.Email)
```

## Subscriptions

### Create a Subscription Session

```go
minPeriods := uint32(3)

session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
    ProductID:   new(uint64(1)),
    Frequency:   qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
    TrialPeriod: &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays}, // optional
    MinPeriods:  &minPeriods,           // optional: minimum periods before cancellation
    SuccessURL:  new("https://yourapp.com/success"),
    CancelURL:   new("https://yourapp.com/cancel"),
    CustomerUUID: new("customer-uuid"),
})
fmt.Printf("Subscription link: %s\n", session.Link)
```

Like one-time payments, subscription sessions accept your own `Reference`, `ProductReference`, and
`CustomerReference` instead of QBitFlow's internal IDs:

```go
session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
    Reference:         new("sub-1234"),   // your own subscription reference
    ProductReference:  new("PLAN-PRO"),   // select a product by your reference
    CustomerReference: new("user-42"),    // select a customer by your reference
    Frequency:         qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
})
```

**Available frequency units:** `DurationUnitSeconds`, `DurationUnitMinutes`, `DurationUnitHours`, `DurationUnitDays`, `DurationUnitWeeks`, `DurationUnitMonths`.

> **Webhook URLs are configured in the dashboard, not per session.** Session creation no longer
> accepts a `webhookUrl`. Set the **Transaction webhook** URL under settings in the
> [QBitFlow dashboard](https://qbitflow.app) — it applies consistently to every transaction.

### Get Subscription Details

```go
sub, err := client.Subscriptions.GetSubscription("subscription-uuid")
fmt.Printf("Status: %s  Next billing: %s\n",
    sub.SubscriptionStatus, sub.NextBillingDate.Format("2006-01-02"))
```

Or look a subscription up by the `Reference` you assigned when creating the session:

```go
sub, err := client.Subscriptions.GetSubscriptionByReference("sub-1234")
fmt.Printf("UUID: %s  Status: %s\n", sub.UUID, sub.SubscriptionStatus)
```

### Subscription Status Changes

A subscription's status changes over its lifetime (`trial`, `active`, `past_due`, `low_on_funds`,
`pending`, `cancelled`, `trial_expired`).

Previously you had to run a cron job that periodically fetched each subscription and diffed its
status to react to these transitions. **This is no longer necessary.** Configure the
**Subscription status webhook** URL under settings in the [QBitFlow dashboard](https://qbitflow.app)
and QBitFlow will POST a `SubscriptionStatusTransitionWebhook` to your endpoint whenever a
subscription changes status:

```go
type SubscriptionStatusTransitionWebhook struct {
    SubscriptionUUID string             // the subscription that changed
    PreviousStatus   SubscriptionStatus // status before the transition
    CurrentStatus    SubscriptionStatus // status after the transition
    UpdatedAt        time.Time          // when the transition occurred
}
```

See [Webhook Handling](#webhook-handling) for a full handler example.

**Subscription statuses:** `SubscriptionStatusTrial`, `SubscriptionStatusActive`, `SubscriptionStatusPastDue`, `SubscriptionStatusLowOnFunds`, `SubscriptionStatusPending`, `SubscriptionStatusCancelled`, `SubscriptionStatusTrialExpired`.

### Payment History

```go
history, err := client.Subscriptions.GetPaymentHistory("subscription-uuid")
for _, record := range history {
    fmt.Printf("%s — $%.2f on %s\n", record.UUID, record.Amount,
        record.CreatedAt.Format("2006-01-02"))
}
```

### Execute Test Billing Cycle (test mode only)

```go
resp, err := client.Subscriptions.ExecuteTestBillingCycle("subscription-uuid")
fmt.Println(resp.Message)
```

### Force Cancel a Subscription

```go
resp, err := client.Subscriptions.ForceCancel("subscription-uuid")
fmt.Println(resp.Message)
```

## Transaction Status

### Poll Status

```go
status, err := client.TransactionStatus.GetTransactionStatus(
    "transaction-uuid",
    qbmodels.TransactionTypeOneTimePayment,
)
if err != nil {
    log.Fatal(err)
}

switch status.Status {
case qbmodels.TransactionStatusCompleted:
    fmt.Printf("Done! tx hash: %s\n", status.TxHash)
case qbmodels.TransactionStatusFailed:
    fmt.Printf("Failed: %s\n", *status.Message)
}
```

### Real-Time Status via WebSocket

The SDK returns the WebSocket URL; connect to it with your preferred WebSocket library.

```go
wsURL := client.TransactionStatus.GetTransactionStatusWSURL(
    "transaction-uuid",
    qbmodels.TransactionTypeOneTimePayment,
)
fmt.Println("Connect to:", wsURL)
// e.g. github.com/gorilla/websocket
```

### Transaction Types

| Constant | Value |
|---|---|
| `TransactionTypeOneTimePayment` | `"payment"` |
| `TransactionTypeCreateSubscription` | `"createSubscription"` |
| `TransactionTypeCancelSubscription` | `"cancelSubscription"` |
| `TransactionTypeExecuteSubscriptionPayment` | `"executeSubscription"` |
| `TransactionTypeIncreaseAllowance` | `"increaseAllowance"` |

## Refunds

```go
// Get the refund for a specific transaction (public)
refund, err := client.Refunds.GetByTransactionUUID("transaction-uuid")
fmt.Printf("Status: %s  Reason: %s\n", refund.Status, refund.Reason)

// List all active refunds
refunds, err := client.Refunds.GetAll()
fmt.Printf("%d active refunds\n", len(refunds))

// List inactive (processed/rejected) refunds with pagination
limit := uint16(20)
result, err := client.Refunds.GetAllInactive(&limit, nil)
for _, r := range result.Items {
    fmt.Printf("%s — %s\n", r.UUID, r.Status)
}
```

**Refund statuses:** `RefundStatusPending`, `RefundStatusApproved`, `RefundStatusRejected`, `RefundStatusFailed`.

## Accounting Export

Export accounting events for a date range as JSON or CSV. Dates must be in `YYYY-MM-DD` format.

```go
// JSON export — returns []AccountingEvent
events, err := client.Accounting.Export(qbf.AccountingExportParams{
    From:   "2025-01-01",
    To:     "2025-12-31",
    Format: "json",
})
if err != nil {
    log.Fatal(err)
}

for _, e := range events {
    fmt.Printf("[%s] %s — gross $%.2f net $%.2f\n",
        e.TxTimeUTC.Format("2006-01-02"), e.PaymentID,
        e.GrossAmountUSD, e.NetAmountUSD)
}

// CSV export — returns raw CSV string
csv, err := client.Accounting.Export(qbf.AccountingExportParams{
    From:   "2025-01-01",
    To:     "2025-12-31",
    Format: "csv",
})
if err != nil {
    log.Fatal(err)
}
os.WriteFile("accounting_2025.csv", []byte(csv), 0644)
```

Each `AccountingEvent` contains full transaction details: product, customer, blockchain info, fees (platform, organization, network), and gross/net amounts.

## Account Claims

Organizations can create users before those users have claimed their accounts. During the unclaimed period all payments go to the organization and a ledger is kept. When the organization creates a claim request, the user receives a link to set up their password and wallet, then the owed funds are transferred.

```go
// Create a claim request for a user (admin only) — returns the claim link
resp, err := client.Claims.CreateClaimRequest(userID)
fmt.Printf("Share this link with the user: %s\n", resp.Link)

// Get the existing claim request for a user (admin only) — same response as CreateClaimRequest
resp, err = client.Claims.GetClaimRequestByUserID(userID)
fmt.Printf("Existing claim link: %s\n", resp.Link)

// List funds the org owes to recently claimed users
funds, err := client.Claims.GetClaimFunds()
for _, f := range funds {
    fmt.Printf("User %d owes $%.2f\n", f.UserID, f.TotalAmountOwed)
}

// Trigger test claim fund computation for a user (test mode only)
resp2, err := client.Claims.TriggerTestClaimFunds(userID)
fmt.Println(resp2.Message)
```

## Customer Management

```go
// Create
customer, err := client.Customers.Create(&qbf.CreateCustomer{
    Name:     "Jane",
    LastName: "Doe",
    Email:    "jane@example.com",
    Reference: new("CRM-42"),
})

// Get by UUID, email, or your own reference
customer, err = client.Customers.Get("customer-uuid")
customer, err = client.Customers.GetByEmail("jane@example.com")
customer, err = client.Customers.GetByReference("CRM-42")

// List (paginated)
limit := uint16(50)
page, err := client.Customers.GetAll(&limit, nil)
if page.HasMore() {
    next, err := client.Customers.GetAll(&limit, page.NextCursor)
    _ = next
}

// Update — partial: only the fields you set are changed, everything else is left
// untouched. The UUID is the first argument, not a field on the struct.
//
// Optional fields are pointers, so nil means "leave unchanged". Go 1.26's new() takes a
// value expression, so you can set one inline: new("Jane"), new(39.99), new(uint16(250)).
updated, err := client.Customers.Update("customer-uuid", &qbf.UpdateCustomer{
    Name: new("Jane"), // last name, email, phone, address all unchanged
})

// Set several fields at once
updated, err = client.Customers.Update("customer-uuid", &qbf.UpdateCustomer{
    Name:     new("Jane"),
    LastName: new("Smith"),
    Email:    new("jane.smith@example.com"),
})

// Note: reference is immutable and cannot be updated.

// Delete
err = client.Customers.Delete("customer-uuid")
```

## Product Management

```go
ref := "PROD-001"

// Create
product, err := client.Products.Create(&qbf.CreateProduct{
    Name:        "Premium Plan",
    Description: "Access to all features",
    Price:       29.99,
    Reference:   &ref, // optional, auto-generated if omitted
})

// Get
product, err = client.Products.Get(product.ID)
product, err = client.Products.GetByReference("PROD-001")

// List all
products, err := client.Products.GetAll()

// Update — partial: omitted fields keep their current value
updated, err := client.Products.Update(product.ID, &qbf.UpdateProduct{
    Price: new(39.99), // name and description unchanged
})

updated, err = client.Products.Update(product.ID, &qbf.UpdateProduct{
    Name:        new("Premium Plus"),
    Description: new("Enhanced features"),
    Price:       new(39.99),
})

// Delete
err = client.Products.Delete(product.ID)
```

## User Management

```go
// Create (admin only)
user, err := client.Users.Create(&qbf.CreateUser{
    Name:               "Alice",
    LastName:           "Smith",
    Email:              "alice@example.com",
    Role:               "user", // "user" or "admin"
    OrganizationFeeBps: 100,   // 1% fee, optional
})

// Get current user (from API key)
me, err := client.Users.Get()

// Get by ID (admin only)
user, err = client.Users.GetByID(42)

// Get by email (admin only for other users in the organization)
user, err = client.Users.GetByEmail("alice@example.com")

// List all (admin only)
users, err := client.Users.GetAll()

// Update — partial: omitted fields keep their current value
updated, err := client.Users.Update(user.ID, &qbf.UpdateUser{
    Name: new("Alicia"),
})

// organizationFeeBps requires admin authority (an admin/owner key, or an
// organization-level key via OnBehalfOf). A non-admin caller sending it gets a 403.
updated, err = client.Users.Update(user.ID, &qbf.UpdateUser{
    OrganizationFeeBps: new(uint16(250)), // 2.5%
})

// Note: passwords cannot be changed through this SDK. It is a JWT-only, self-service
// operation on the API, so the field is intentionally absent from UpdateUser.

// Delete (admin only)
err = client.Users.Delete(user.ID)
```

## API Key Management

The SDK exposes **read** access to API keys. Creating and deleting keys is a
JWT-authenticated operation (performed from the [QBitFlow dashboard](https://qbitflow.app)),
so it is intentionally not available through the API-key–authenticated SDK. The secret key
value is never returned on a key record.

```go
// List for current user
keys, err := client.ApiKeys.GetAll()

// List for a specific user (admin only)
keys, err = client.ApiKeys.GetForUser(userID)
```

## Currencies

Resolve the currency IDs returned in `SessionCheckout.AvailableCurrencies` and on payment
records. These endpoints are public.

```go
// All supported currencies (native + tokens); pass true for test-network currencies
currencies, err := client.Currencies.GetAllAvailable(false)

// Only the main (native/blockchain) currencies
mains, err := client.Currencies.GetAllMain(false)
```

## Webhook Handling

### Verifying a webhook signature

Every webhook QBitFlow sends carries three headers:

| Header | Meaning |
|---|---|
| `X-Webhook-Signature-256` | HMAC signature, formatted `sha256=<hex>` |
| `X-Webhook-Timestamp` | Send time, in unix seconds |
| `X-Webhook-Id` | Transaction id, e.g. `pay@<uuid>` |

There are two ways to check a webhook is genuine, and you can use either:

| | Needs the secret | Network call | Use when |
|---|---|---|---|
| **Local** | yes | none | Default. Faster, and keeps working if the API is unreachable. |
| **Remote** | no | one per webhook | You would rather not hold the secret at all. |

Local verification performs the same three checks the server does: the timestamp is within
a replay window (5 minutes by default), the HMAC matches, and the comparison is
constant-time so a timing side channel cannot be used to guess the signature.

**Why the signature covers a canonical rendering, not the raw bytes.** JSON object key
order is not significant, and proxies, frameworks and logging layers routinely re-serialize
a body and reorder keys. Signing raw bytes would reject a payload that is in fact
untouched. So both sides sign `<timestamp>.<canonical-json>`, where canonical means keys
sorted at every level and no insignificant whitespace. You do not have to do anything for
this — pass the body you received and the SDK handles it.

Get your webhook secret from the QBitFlow dashboard. Treat it like a password: keep it in
your environment or secret manager, never in source control.

```go
import (
    "net/http"

    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    secret := os.Getenv("QBITFLOW_WEBHOOK_SECRET")

    // Reads the body, checks the headers, verifies the signature, and hands back the
    // raw body so you decode it exactly once.
    body, err := qbf.VerifyWebhookRequest(r, secret, nil)
    if err != nil {
        http.Error(w, "invalid webhook", http.StatusBadRequest)
        return
    }

    var event qbmodels.SessionWebhookResponse
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }

    // Act on the event, then acknowledge.
    w.WriteHeader(http.StatusOK)
}
```

If you already have the pieces, verify them directly:

```go
err := qbf.VerifyWebhookSignature(secret, timestamp, signature, body, nil)
```

To widen or narrow the replay window (it must match the server's setting):

```go
opts := &qbf.VerifyOptions{MaxTimestampAge: 10 * time.Minute}
err := qbf.VerifyWebhookSignature(secret, timestamp, signature, body, opts)
```

Pull the headers out yourself when you need the transaction id, or want to short-circuit
the dashboard's connectivity test:

```go
h := qbf.ExtractWebhookHeaders(r.Header)
if h.IsTest() {
    w.WriteHeader(http.StatusOK) // connectivity check, nothing to process
    return
}
```

To verify through the API instead, with no secret in your process:

```go
ok, err := client.Webhooks.Verify(body, signature, timestamp)
```

Webhook destination URLs are configured under settings in the [QBitFlow dashboard](https://qbitflow.app),
**not** in code. There are two independent webhooks you can enable:

| Dashboard setting | Payload type | Fired when |
|---|---|---|
| **Transaction webhook** | `SessionWebhookResponse` | a payment or subscription transaction changes status |
| **Subscription status webhook** | `SubscriptionStatusTransitionWebhook` | a subscription transitions between statuses (e.g. `trial` → `active`, `active` → `past_due`) |

Every webhook is signed with HMAC-SHA256. Verify it with `client.Webhooks.Verify` before trusting
the payload. Responding with a status `>= 400` tells QBitFlow to retry delivery.

Webhook headers: `X-Webhook-Signature-256`, `X-Webhook-Timestamp`, `X-Webhook-ID`.

### Test Webhook Reachability

The dashboard's **"Test webhook"** action sends a fake payload to your configured URL to confirm the
endpoint is reachable. This payload may not match a real webhook's shape, so your handler should
short-circuit it: when the incoming `X-Webhook-ID` header equals `qbf.TEST_WEBHOOK_ID`, return
HTTP `200` immediately and skip normal processing.

### Transaction Webhook Handler

```go
package main

import (
    "encoding/json"
    "io"
    "net/http"

    "github.com/QBitFlow/qbitflow-go-sdk/v2"
    qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

var sdkClient = qbitflow.New("your-api-key")

func transactionWebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    signature := r.Header.Get(sdkClient.Webhooks.GetSignatureHeader())
    timestamp := r.Header.Get(sdkClient.Webhooks.GetTimestampHeader())
    webhookID := r.Header.Get(sdkClient.Webhooks.GetWebhookIDHeader())

    // Verify authenticity — a >= 400 response causes QBitFlow to retry.
    valid, err := sdkClient.Webhooks.Verify(body, signature, timestamp)
    if err != nil || !valid {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Reachability check from the dashboard "Test webhook" action.
    if webhookID == qbf.TEST_WEBHOOK_ID {
        w.WriteHeader(http.StatusOK)
        return
    }

    var event qbmodels.SessionWebhookResponse
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }

    switch event.Status.Status {
    case qbmodels.TransactionStatusCompleted:
        // event.Session.Reference echoes back the reference you set when creating the
        // session, so you can match the transaction to your own order/invoice without
        // storing our UUID.
        // Fulfill order, update DB, etc.
    case qbmodels.TransactionStatusFailed:
        // Notify customer
    }

    w.WriteHeader(http.StatusOK)
}
```

### Subscription Status Webhook Handler

Configure the **Subscription status webhook** in the dashboard to receive lifecycle transitions.
This replaces polling each subscription on a cron schedule to detect status changes.

```go
func subscriptionStatusWebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    signature := r.Header.Get(sdkClient.Webhooks.GetSignatureHeader())
    timestamp := r.Header.Get(sdkClient.Webhooks.GetTimestampHeader())
    webhookID := r.Header.Get(sdkClient.Webhooks.GetWebhookIDHeader())

    valid, err := sdkClient.Webhooks.Verify(body, signature, timestamp)
    if err != nil || !valid {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Reachability check from the dashboard "Test webhook" action.
    if webhookID == qbf.TEST_WEBHOOK_ID {
        w.WriteHeader(http.StatusOK)
        return
    }

    var event qbmodels.SubscriptionStatusTransitionWebhook
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }

    // event.SubscriptionReference carries your own reference for the subscription, if you
    // set one when creating the session (nil otherwise).

    switch event.CurrentStatus {
    case qbmodels.SubscriptionStatusActive:
        // Trial converted or payment recovered — grant/keep access.
    case qbmodels.SubscriptionStatusPastDue, qbmodels.SubscriptionStatusLowOnFunds:
        // Prompt the customer to top up their allowance.
    case qbmodels.SubscriptionStatusCancelled, qbmodels.SubscriptionStatusTrialExpired:
        // Revoke access.
    }

    w.WriteHeader(http.StatusOK)
}
```

## Error Handling

```go
import qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"

payment, err := client.Payments.GetPayment("uuid")
if err != nil {
    switch e := err.(type) {
    case *qberrors.NotFoundError:
        fmt.Println("not found")
    case *qberrors.ValidationError:
        fmt.Printf("validation: %s\n", e.Message)
    case *qberrors.QBitFlowError:
        fmt.Printf("API error %d: %s\n", e.StatusCode, e.Message)
    default:
        fmt.Printf("unexpected: %v\n", err)
    }
}
```


## License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://qbitflow.app/docs)
- 📧 [Email Support](mailto:support@qbitflow.app)
- 🐛 [Issue Tracker](https://github.com/qbitflow/qbitflow-go-sdk/issues)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Security

For security issues, please email security@qbitflow.app instead of using the issue tracker.
