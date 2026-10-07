# QBitFlow Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

Official Go SDK for [QBitFlow](https://qbitflow.app) — a cryptocurrency payment processing platform supporting one-time payments, recurring subscriptions, refunds, and accounting exports.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Response Types](#response-types)
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
- [Retries & Timeouts](#retries--timeouts)
- [Client-Side Validation](#client-side-validation)
- [Testing](#testing)

## Installation

```bash
go get github.com/QBitFlow/qbitflow-go-sdk/v2
```

**Requires Go 1.27 or later.** The optional request fields are pointers, and the examples
below set them with `new(value)` — Go 1.26 extended `new` to accept a value expression, not
just a type.

## Quick Start

```go
package main

import (
    "errors"
    "fmt"
    "log"

    "github.com/QBitFlow/qbitflow-go-sdk/v2"
    qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
    qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
    client := qbitflow.New("your-api-key")

    // Create a one-time payment session with an inline product
    session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
        ProductName: new("Premium Access"),
        Description: new("One-time purchase"),
        Price:       new(49.99),
        SuccessURL:  new("https://yourapp.com/success"),
        CancelURL:   new("https://yourapp.com/cancel"),
        Reference:   new("order-1234"), // your own order ID
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Payment link: %s\n", session.Link)

    // Check the transaction status. Until the customer pays, the API reports the session
    // as not found.
    status, err := client.TransactionStatus.GetTransactionStatus(
        session.UUID,
        qbmodels.TransactionTypeOneTimePayment,
    )
    var notFound *qberrors.NotFoundError
    switch {
    case errors.As(err, &notFound):
        fmt.Println("Not paid yet")
    case err != nil:
        log.Fatal(err)
    default:
        fmt.Printf("Status: %s\n", status.Status)
    }
}
```

## Configuration

```go
// Minimal — API key only
client := qbitflow.New("your-api-key")

// Custom configuration
import "time"

client := qbitflow.NewWithConfig(qbf.Config{
    APIKey:         "your-api-key",
    BaseURL:        "https://api.qbitflow.app/v1", // optional, this is the default
    Timeout:        30 * time.Second,              // optional, default 30s per attempt
    MaxRetries:     3,                             // optional: 0 = default (3), negative = off
    DisableRetries: false,                         // set true to attempt every request exactly once
})

// Override base URL after creation (useful for testing)
client.SetBaseURL("http://localhost:3001")
```

Every request carries `X-API-Key` and `User-Agent: qbitflow-go/<qbf.Version>`. A blank API
key makes every call fail with a `*qberrors.ValidationError` before anything is sent.

Configure the client **before** using it: `SetBaseURL` and `SetTimeout` are not synchronised
with in-flight requests, and clients or services obtained from `OnBehalfOf` earlier keep the
base URL they were created with (the timeout is shared). See
[Retries & Timeouts](#retries--timeouts) for the retry policy.

## Response Types

Every response type in `pkg/models` mirrors the Go type the API serializes, field for field,
so the compiler tells you what can be missing:

- **Value fields** (`string`, numbers, `bool`, nested structs, `time.Time`) are always usable.
  When the API omits one — it omits `omitempty` fields at their zero value — or sends `null`,
  it decodes to the zero value (`""`, `0`, `false`, a zero-valued struct, the zero time). For
  example `Customer.PhoneNumber` is `""` when unset, `Payment.ProductID` is `0` for a payment
  without a product, `Subscription.LastBillingDate.IsZero()` until the first billing, and a
  pending `RefundEntry` has `TxHash == ""` and `MerchantMessage == ""`.
- **Pointer fields** are genuinely nullable on the server. `nil` means none:
  `Payment.Reference`, `Payment.CustomerUUID`, `Subscription.CustomerUUID`,
  `Subscription.MinimumCancellationDate`, `SessionCheckout.CustomerUUID`,
  `TransactionStatus.SettlementDetails`, `PaymentMetadata.OrganizationFee` / `ReferralFee`,
  `RefundEntry.RespondedAt` / `Metadata`, `User.ClaimedAt`, `ApiKey.ExpiresAt`,
  `Currency.MainCurrency`, `CombinedPayment.ProductID` / `SubscriptionUUID` / `Metadata`,
  `SessionWebhookResponse.Status`.
- **Lists** are never `nil` when returned by a service: an absent or `null` list is an empty
  slice.
- `Currency` is a value on `Payment`, `CombinedPayment`, `Subscription` and
  `SubscriptionHistory`: `payment.Currency.Symbol` needs no nil check.
- **Enum-like fields** are string types: a value this SDK does not know yet is kept verbatim —
  never an error, never coerced. Compare against the constants and treat anything else as
  "unknown".

A field present with the wrong JSON type is a response-shape failure, reported as a
`*qberrors.ServerError` carrying the HTTP status.

## Acting on Behalf of a User

If you hold an **organization (admin) API key**, you can perform any request as one of the users in your organization, without needing that user's own API key. This is useful for admin-level tooling, dashboards, and back-office automation where your server acts for a specific user (e.g. listing *their* products, creating a payment session *for them*, or reading *their* subscriptions).

`client.OnBehalfOf(userID)` returns a client whose **every** service sends the
`On-Behalf-Of` header. It shares the transport, connection pool and configuration of the
original client, which is left untouched — so you can freely mix org-level and per-user
calls:

```go
const userID uint64 = 123

userClient := client.OnBehalfOf(userID)

// List the products belonging to user 123
products, err := userClient.Products.GetAll()

// Read that user's payments
payments, err := userClient.Payments.GetAllPayments(nil, nil)

// The base client is unaffected — this call still runs at the organization level
allOrgProducts, err := client.Products.GetAll()
```

For a one-off call, each service (except `Currencies`, whose routes are public) also has its
own `OnBehalfOf`:

```go
products, err := client.Products.OnBehalfOf(userID).GetAll()
```

> **Note:** `OnBehalfOf` requires an organization-level admin API key; the acted-for user's
> role applies to the request. `OnBehalfOf(0)` acts at the organization level — the header is
> simply omitted. A user outside your organization is answered with `404`
> (`*qberrors.NotFoundError`); existence is never revealed across organizations.

## One-Time Payments

### Create a Payment Session

Provide an existing `ProductID`, a `ProductReference`, **or** the (`ProductName` + `Description` + `Price`) triplet.
Every string option treats `""` like `nil`: the field is left out of the request.

```go
// Using an existing product, for a known customer
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
    ProductID:    new(uint64(42)),
    SuccessURL:   new("https://yourapp.com/success"),
    CancelURL:    new("https://yourapp.com/cancel"),
    CustomerUUID: new(customer.UUID), // a bare UUID, from Customers.Create / Get
})

// Using an inline product definition (price must be greater than 0)
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
    ProductName: new("Custom Item"),
    Description: new("Limited edition widget"),
    Price:       new(19.99),
    SuccessURL:  new("https://yourapp.com/success"),
    CancelURL:   new("https://yourapp.com/cancel"),
})
```

Without a `CustomerUUID` (or `CustomerReference`), the customer enters their details during
checkout.

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
session, err := client.Payments.GetSession("pay@...", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Product: %s  Price: $%.2f\n", session.ProductName, session.Price)
fmt.Printf("Is payment: %v  Currencies: %v\n", session.IsPayment(), session.AvailableCurrencies)
```

The second argument (`closeToExpireError *bool`) is `nil` for the API default (reject a session
that is about to expire).

### Get a Completed Payment

```go
payment, err := client.Payments.GetPayment("pay@...")
fmt.Printf("Amount: $%.2f in %s  Hash: %s\n", payment.Amount, payment.Currency.Symbol, payment.TransactionHash)
if payment.CustomerUUID != nil {
    fmt.Println("Customer:", *payment.CustomerUUID)
}
```

You can also look a payment up by the `Reference` you assigned when creating the session — no need
to store QBitFlow's UUID:

```go
payment, err := client.Payments.GetPaymentByReference("order-1234")
fmt.Printf("UUID: %s  Amount: $%.2f\n", payment.UUID, payment.Amount)
```

> References, emails and identifiers are path-escaped correctly, but the API currently cannot
> route a reference containing `/` (it answers 404) — avoid `/` in references you look up.

### List Payments

```go
limit := uint16(20) // optional; must be positive when given

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
customer, err := client.Payments.GetCustomerForTransaction("pay@...") // or a sub@ subscription UUID
fmt.Printf("%s %s — %s\n", customer.Name, customer.LastName, customer.Email)
```

## Subscriptions

### Create a Subscription Session

```go
session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
    ProductID:   new(uint64(1)),
    Frequency:   qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
    TrialPeriod: &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays}, // optional
    MinPeriods:  new(uint32(3)),        // optional: minimum periods before cancellation (0 = none)
    SuccessURL:  new("https://yourapp.com/success"),
    CancelURL:   new("https://yourapp.com/cancel"),
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

**Available frequency units:** `DurationUnitSeconds`, `DurationUnitMinutes`, `DurationUnitHours`, `DurationUnitDays`, `DurationUnitWeeks`, `DurationUnitMonths`, `DurationUnitYears`. `Frequency.Value` must be within 1–4294967295, `TrialPeriod.Value` (if set) within 0–4294967295, and `MinPeriods` (if set) is sent only when greater than 0 — all checked client-side before the request.

> **Webhook URLs are configured in the dashboard, not per session.** Set the
> **Transaction webhook** URL under settings in the [QBitFlow dashboard](https://qbitflow.app) —
> it applies consistently to every transaction.

### Get Subscription Details

A subscription has its own `sub@` UUID, **different from the checkout session's UUID**. You
receive it in the [subscription webhooks](#subscription-status-webhook-handler)
(`SubscriptionWebhook.SubscriptionUUID`), or you can look the subscription up by your reference.

```go
sub, err := client.Subscriptions.GetSubscription("sub@...")
fmt.Printf("Status: %s  Currency: %s  Next billing: %s\n",
    sub.SubscriptionStatus, sub.Currency.Symbol, sub.NextBillingDate.Format("2006-01-02"))
if sub.LastBillingDate.IsZero() {
    fmt.Println("Not billed yet")
}
```

Or look a subscription up by the `Reference` you assigned when creating the session:

```go
sub, err := client.Subscriptions.GetSubscriptionByReference("sub-1234")
fmt.Printf("UUID: %s  Status: %s\n", sub.UUID, sub.SubscriptionStatus)
```

### Subscription Status Changes

A subscription's status changes over its lifetime (`trial`, `active`, `past_due`, `low_on_funds`,
`pending`, `cancelled`, `trial_expired`).

Configure the **Subscription status webhook** URL under settings in the
[QBitFlow dashboard](https://qbitflow.app) and QBitFlow will POST a `SubscriptionWebhook` to your
endpoint whenever a subscription changes status — no need to poll. The same endpoint also
receives a delivery on each successful renewal, so the envelope carries a `Type`
discriminator and the event-specific payload in `Data`:

```go
type SubscriptionWebhook struct {
    SubscriptionUUID      string                  // the subscription this is about
    SubscriptionReference string                  // your own reference ("" if none was set)
    Type                  SubscriptionWebhookType // "status_transition" or "billing"
    Data                  json.RawMessage         // unwrap with StatusTransition() or Billing()
}

// Type == SubscriptionWebhookTypeStatusTransition
type SubscriptionStatusTransition struct {
    PreviousStatus SubscriptionStatus // status before the transition
    CurrentStatus  SubscriptionStatus // status after the transition
    UpdatedAt      time.Time          // when the transition occurred
}

// Type == SubscriptionWebhookTypeBilling -> Data is a SubscriptionHistory
```

See [Webhook Handling](#webhook-handling) for a full handler example.

**Subscription statuses:** `SubscriptionStatusTrial`, `SubscriptionStatusActive`, `SubscriptionStatusPastDue`, `SubscriptionStatusLowOnFunds`, `SubscriptionStatusPending`, `SubscriptionStatusCancelled`, `SubscriptionStatusTrialExpired`.

### Payment History

```go
history, err := client.Subscriptions.GetPaymentHistory("sub@...")
for _, record := range history {
    fmt.Printf("%s — $%.2f on %s\n", record.UUID, record.Amount,
        record.CreatedAt.Format("2006-01-02"))
}
```

An unknown subscription yields an empty list, not an error.

### Execute Test Billing Cycle (test mode only)

```go
resp, err := client.Subscriptions.ExecuteTestBillingCycle("sub@...")
fmt.Println(resp.Message) // {"message": "..."}
```

A subscription that is not yet due is refused with a `*qberrors.ConflictError` (HTTP 409).
`ExecuteTestBillingCycle` and `ForceCancel` are never retried by the SDK, even though they
are GETs: a timeout may already have queued the action.

### Force Cancel a Subscription

```go
resp, err := client.Subscriptions.ForceCancel("sub@...")
fmt.Println(resp.Message)
```

## Transaction Status

### Poll Status

```go
status, err := client.TransactionStatus.GetTransactionStatus(
    "pay@...",
    qbmodels.TransactionTypeOneTimePayment,
)
var notFound *qberrors.NotFoundError
if errors.As(err, &notFound) {
    return // the session exists but has not been processed yet (or the UUID is unknown)
}
if err != nil {
    log.Fatal(err)
}

switch status.Status {
case qbmodels.TransactionStatusCompleted:
    fmt.Printf("Done! tx hash: %s\n", status.TxHash)
case qbmodels.TransactionStatusFailed:
    fmt.Printf("Failed: %s\n", status.Message) // "" when the API gives no reason
}
```

### Real-Time Updates

To react to transactions as they complete, use [webhooks](#webhook-handling) (recommended):
QBitFlow notifies your endpoint, so nothing has to poll. Use `GetTransactionStatus` for an
on-demand check, for example on your success-redirect page.

### Transaction Types

| Constant | Value |
|---|---|
| `TransactionTypeOneTimePayment` | `"payment"` |
| `TransactionTypeCreateSubscription` | `"createSubscription"` |
| `TransactionTypeCancelSubscription` | `"cancelSubscription"` |
| `TransactionTypeExecuteSubscriptionPayment` | `"executeSubscription"` |
| `TransactionTypeIncreaseAllowance` | `"increaseAllowance"` |
| `TransactionTypeUpdateMaxAmount` | `"updateMaxAmount"` |
| `TransactionTypeRefund` | `"refund"` |
| `TransactionTypeClaimFunds` | `"claimFunds"` |

## Refunds

```go
// Get the refund for a specific transaction
refund, err := client.Refunds.GetByTransactionUUID("pay@...")
fmt.Printf("Status: %s  Reason: %s\n", refund.Status, refund.Reason)

// List the refunds awaiting your response
refunds, err := client.Refunds.GetAll()
fmt.Printf("%d active refunds\n", len(refunds))

// List handled (approved / refused / failed) refunds with pagination
limit := uint16(20)
result, err := client.Refunds.GetAllInactive(&limit, nil)
for _, r := range result.Items {
    fmt.Printf("%s — %s\n", r.UUID, r.Status)
}
```

**Refund statuses:** `RefundStatusPending`, `RefundStatusApproved`, `RefundStatusRefused`, `RefundStatusFailed`.

Until the merchant has handled a refund, `RespondedAt` and `Metadata` are `nil` and `TxHash` /
`MerchantMessage` are `""`.

## Accounting Export

Export accounting events for a date range as JSON (`Export`) or CSV (`ExportCSV`). Dates are
inclusive, formatted `YYYY-MM-DD` and must be real calendar dates, and `From` must not be after
`To` — the SDK checks this before the request (`*qberrors.ValidationError`). The maximum span
is enforced by the API.

```go
// JSON export — returns []AccountingEvent
events, err := client.Accounting.Export(qbf.AccountingExportParams{
    From: "2026-09-01",
    To:   "2026-09-30",
})
if err != nil {
    log.Fatal(err)
}

for _, e := range events {
    fmt.Printf("[%s] %s %s — gross $%.2f net $%.2f\n",
        e.TxTimeUTC.Format("2006-01-02"), e.Type, e.PaymentID,
        e.GrossAmountUSD, e.NetAmountUSD)
}

// CSV export — returns the CSV text (header row + one row per event)
csv, err := client.Accounting.ExportCSV(qbf.AccountingExportParams{
    From: "2026-09-01",
    To:   "2026-09-30",
})
if err != nil {
    log.Fatal(err)
}
os.WriteFile("accounting_2026_09.csv", []byte(csv), 0644)
```

Each `AccountingEvent` contains full transaction details: product, customer, blockchain info, fees (platform, organization, network), and gross/net amounts. Decimal amounts (`GrossAmount`, `PlatformFee`, `NetAmount`, …) are decimal strings; the `*USD` fields are `float64`. `Type` is one of the `qbmodels.AccountingEventType*` values (`payment`, `refund`, `organizationFee`, `referralFee`, and the subscription-billing value, which the API documents both as `subscriptionHistory` and `subHistory`).

## Account Claims

Organizations can create users before those users have claimed their accounts. During the unclaimed period all payments go to the organization and a ledger is kept. When the organization creates a claim request, the user receives a link to set up their password and wallet, then the owed funds are transferred.

```go
// Create a claim request for a user (admin only) — returns the claim link
resp, err := client.Claims.CreateClaimRequest(userID)
fmt.Printf("Share this link with the user: %s\n", resp.Link)

// Get the existing claim request for a user (admin only) — same response as CreateClaimRequest
resp, err = client.Claims.GetClaimRequestByUserID(userID)
fmt.Printf("Existing claim link: %s\n", resp.Link)

// List funds the org owes to claimed users
funds, err := client.Claims.GetClaimFunds()
for _, f := range funds {
    fmt.Printf("User %d is owed $%.2f\n", f.UserID, f.TotalAmountOwed)
}

// Trigger test claim fund computation for a user (test mode only; never retried)
resp2, err := client.Claims.TriggerTestClaimFunds(userID)
fmt.Println(resp2.Message)
```

## Customer Management

```go
// Create
customer, err := client.Customers.Create(&qbf.CreateCustomer{
    Name:      "Jane",
    LastName:  "Doe",
    Email:     "jane@example.com",
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
// Optional fields are pointers: nil (or a pointer to "") means "leave unchanged".
// Go 1.26's new() takes a value expression, so you can set one inline: new("Jane").
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

// Delete (soft delete)
err = client.Customers.Delete("customer-uuid")
```

`Create` and `Update` validate before sending: `Name`/`LastName` are 2–100 characters of
letters, decimal digits, spaces and `- _ ' .`, and `Email` is required and must be well-formed.
`Update` checks only the fields you set.

`Delete` returns only an `error` — the idiomatic Go shape for all three `Delete` methods
(customers, products, users); the API's confirmation message carries nothing beyond success.

## Product Management

```go
// Create
product, err := client.Products.Create(&qbf.CreateProduct{
    Name:        "Premium Plan",
    Description: "Access to all features",
    Price:       29.99,          // greater than 0
    Reference:   new("PROD-001"), // optional, auto-generated if omitted
})

// Get
product, err = client.Products.Get(product.ID)
product, err = client.Products.GetByReference("PROD-001")

// List all active products (hidden products created by inline checkouts are excluded)
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

// Delete (soft delete)
err = client.Products.Delete(product.ID)
```

## User Management

```go
// Create (admin only)
user, err := client.Users.Create(&qbf.CreateUser{
    Name:               "Alice",
    LastName:           "Smith",
    Email:              "alice@example.com",
    Role:               qbmodels.RoleUser, // RoleUser or RoleAdmin ("user" / "admin"); owner and handle are read-only
    OrganizationFeeBps: 100,               // 1% fee, optional, 0-5000
})

// Get current user (from API key, or the acted-for user through OnBehalfOf)
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
    OrganizationFeeBps: new(uint16(250)), // 2.5%; new(uint16(0)) explicitly sets 0
})

// Note: passwords cannot be changed through this SDK. It is a JWT-only, self-service
// operation on the API, so the field is intentionally absent from UpdateUser.

// Delete (admin only)
err = client.Users.Delete(user.ID)
```

`Create` and `Update` validate before sending (same name/email rules as customers, `Role` in
`{admin, user}`, `OrganizationFeeBps` ≤ 5000). `User.Role` on responses can also be
`RoleOwner` or `RoleHandle` — the full hierarchy is `handle < user < admin < owner`.

## API Key Management

The SDK exposes **read** access to API keys. Creating and deleting keys is a
JWT-authenticated operation (performed from the [QBitFlow dashboard](https://qbitflow.app)),
so it is intentionally not available through the API-key–authenticated SDK. The secret key
value is never returned on a key record.

```go
// Keys visible to the caller: a user key sees its own keys, an organization key every key
// in the organization
keys, err := client.ApiKeys.GetAll()

// List for a specific user (admin only)
keys, err = client.ApiKeys.GetForUser(userID)
```

## Currencies

Resolve the currency IDs returned in `SessionCheckout.AvailableCurrencies`. These endpoints
are public.

```go
// All supported currencies (native + tokens); pass true for test-network currencies
currencies, err := client.Currencies.GetAllAvailable(false)

// Only the main (native/blockchain) currencies
mains, err := client.Currencies.GetAllMain(false)
```

A token's `MainCurrency` / `MainCurrencyID` point to its native currency; they are `nil` for a
main currency.

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

Local verification checks that the timestamp is within a replay window (5 minutes by default),
that the HMAC matches, and compares in constant time so a timing side channel cannot be used to
guess the signature.

**Why the signature covers a canonical rendering, not the raw bytes.** JSON object key
order is not significant, and proxies, frameworks and logging layers routinely re-serialize
a body and reorder keys. Signing raw bytes would reject a payload that is in fact
untouched. So both sides sign `<timestamp>.<canonical-json>`, where canonical means keys
sorted at every level and no insignificant whitespace. You do not have to do anything for
this — pass the body you received and the SDK handles it. Pass the **raw body** (or a generic
`map[string]any` decoded from it), never an SDK struct such as `SessionWebhookResponse`: a
struct does not model every field the server sends, so re-encoding it changes the payload
and the signature no longer matches.

Get your webhook secret from the QBitFlow dashboard. Treat it like a password: keep it in
your environment or secret manager, never in source control.

```go
import (
    "encoding/json"
    "net/http"
    "os"

    qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    secret := os.Getenv("QBITFLOW_WEBHOOK_SECRET")

    // Reads the body (at most qbf.MaxWebhookBodySize, 5 MiB), checks the headers, verifies
    // the signature, and hands back the raw body so you decode it exactly once.
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

To widen or narrow the replay window:

```go
opts := &qbf.VerifyOptions{MaxTimestampAge: 10 * time.Minute} // 0 or negative = the 5-minute default
err := qbf.VerifyWebhookSignature(secret, timestamp, signature, body, opts)
```

Pull the headers out yourself when you need the transaction id, or to acknowledge the
dashboard's connectivity test — **after** verifying its signature like any other delivery:

```go
h := qbf.ExtractWebhookHeaders(r.Header)
if err := qbf.VerifyWebhookSignature(secret, h.Timestamp, h.Signature, body, nil); err != nil {
    http.Error(w, "invalid webhook", http.StatusBadRequest)
    return
}
if h.IsTest() {
    w.WriteHeader(http.StatusOK) // connectivity check, verified, nothing to process
    return
}
```

To verify through the API instead, with no secret in your process:

```go
ok, err := client.Webhooks.Verify(body, signature, timestamp)
```

`Verify` takes the raw body (`[]byte`, `json.RawMessage` or `string`, forwarded verbatim) or a
generic decoded value. A missing or invalid payload is a `*qberrors.ValidationError` raised
before any request. It returns `(false, nil)` only when the API answers 400 ("signature
rejected"); any other failure — an outage, an expired key — is returned as its typed error, so
you can answer non-200 and let QBitFlow retry instead of dropping a real event.

Webhook destination URLs are configured under settings in the [QBitFlow dashboard](https://qbitflow.app),
**not** in code. There are two independent webhooks you can enable:

| Dashboard setting | Payload type | Fired when |
|---|---|---|
| **Transaction webhook** | `SessionWebhookResponse` | a checkout completes (a one-time payment, or a subscription's first billed period) |
| **Subscription status webhook** | `SubscriptionWebhook` | a subscription transitions between statuses (e.g. `trial` → `active`), or a renewal period is billed |

Every webhook is signed with HMAC-SHA256. Verify it (locally or with `client.Webhooks.Verify`)
before trusting the payload. Responding with a status `>= 400` makes QBitFlow retry the delivery.

Webhook headers: `X-Webhook-Signature-256`, `X-Webhook-Timestamp`, `X-Webhook-Id`
(`qbf.HeaderSignature`, `qbf.HeaderTimestamp`, `qbf.HeaderWebhookID`).

### Test Webhook Reachability

The dashboard's **"Test webhook"** action sends a probe to your configured URL to confirm the
endpoint is reachable. Its payload may not match a real webhook's shape, so your handler should
not process it: when `X-Webhook-Id` equals `qbf.TEST_WEBHOOK_ID`, return HTTP `200` and skip
the business logic — but **only after the signature check has passed**, so the dashboard button
exercises your real verification path end to end rather than passing with a broken secret.
This assumes the probe is signed like a normal delivery.

### Transaction Webhook Handler

```go
package main

import (
    "encoding/json"
    "net/http"
    "os"

    qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
    qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

var webhookSecret = os.Getenv("QBITFLOW_WEBHOOK_SECRET")

func transactionWebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := qbf.VerifyWebhookRequest(r, webhookSecret, nil)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Reachability check from the dashboard "Test webhook" action — after verification.
    if qbf.ExtractWebhookHeaders(r.Header).IsTest() {
        w.WriteHeader(http.StatusOK)
        return
    }

    var event qbmodels.SessionWebhookResponse
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }

    // Status is nullable: acknowledge a delivery without one rather than rejecting it,
    // which would only make QBitFlow retry it.
    if event.Status != nil && event.Status.Status == qbmodels.TransactionStatusCompleted {
        switch event.TxType {
        case qbmodels.TransactionTypeOneTimePayment:
            // event.Session.Reference echoes the reference you set when creating the
            // session, so you can match the transaction to your own order/invoice.
        case qbmodels.TransactionTypeCreateSubscription:
            // The first period is billed. event.UUID is the *session* UUID; the
            // subscription's own sub@ UUID arrives with the subscription webhooks.
        }
    }

    w.WriteHeader(http.StatusOK)
}
```

### Subscription Status Webhook Handler

Configure the **Subscription status webhook** in the dashboard to receive lifecycle transitions
and renewals.

```go
func subscriptionStatusWebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := qbf.VerifyWebhookRequest(r, webhookSecret, nil)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    if qbf.ExtractWebhookHeaders(r.Header).IsTest() {
        w.WriteHeader(http.StatusOK)
        return
    }

    var event qbmodels.SubscriptionWebhook
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }

    // event.SubscriptionReference carries your own reference for the subscription ("" if
    // none was set).

    switch event.Type {
    case qbmodels.SubscriptionWebhookTypeStatusTransition:
        transition, err := event.StatusTransition()
        if err != nil {
            http.Error(w, "bad payload", http.StatusBadRequest)
            return
        }

        switch transition.CurrentStatus {
        case qbmodels.SubscriptionStatusActive:
            // Trial converted or payment recovered — grant/keep access.
        case qbmodels.SubscriptionStatusPastDue, qbmodels.SubscriptionStatusLowOnFunds:
            // Prompt the customer to top up their allowance.
        case qbmodels.SubscriptionStatusCancelled, qbmodels.SubscriptionStatusTrialExpired:
            // Revoke access.
        }

    case qbmodels.SubscriptionWebhookTypeBilling:
        billing, err := event.Billing()
        if err != nil {
            http.Error(w, "bad payload", http.StatusBadRequest)
            return
        }
        _ = billing // Record the renewal (billing.Amount, billing.Currency…) against event.SubscriptionUUID.
    }

    w.WriteHeader(http.StatusOK)
}
```

`examples/webhook_handler` is a complete server with both endpoints, local or remote
verification, and success/cancel redirect pages.

## Error Handling

Every failure is one of the types in `pkg/errors`, chosen by HTTP status — the same mapping
the JavaScript, Python and PHP SDKs use:

| Condition | Type |
|---|---|
| 400, 422 — and client-side validation before a request (`StatusCode` 0) | `*qberrors.ValidationError` |
| 401 | `*qberrors.AuthenticationError` |
| 403 | `*qberrors.ForbiddenError` |
| 404 (also: a resource you do not own — existence is never revealed) | `*qberrors.NotFoundError` |
| 409 (e.g. billing not yet due) | `*qberrors.ConflictError` |
| 429 | `*qberrors.RateLimitError` (`RetryAfter *int` seconds, from `Retry-After` — seconds or an HTTP date) |
| other 4xx | `*qberrors.QBitFlowError` (the base type, with `StatusCode`) |
| 5xx after retries, a 3xx (redirects are never followed), a 2xx whose body is empty or not the expected JSON | `*qberrors.ServerError` |
| no response (connection refused, DNS, TLS, timeout) after retries | `*qberrors.NetworkError` |

Every type carries `Message`, `StatusCode` and `Fields []qberrors.FieldError` — the API's
per-field `errors[]` list, kept for any status. The message follows one precedence: `error`,
then the joined field list (`Field: message; Field2: message`), then `message`, then a
non-JSON body, then the HTTP status text. Error bodies are parsed the same way for the CSV
export.

Use `errors.As`. Every type unwraps to `*qberrors.QBitFlowError`, so a `*QBitFlowError`
target matches any SDK error — put it last:

```go
import (
    "errors"
    qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

payment, err := client.Payments.GetPayment("pay@...")
if err != nil {
    var (
        validation *qberrors.ValidationError
        notFound   *qberrors.NotFoundError
        rateLimit  *qberrors.RateLimitError
        network    *qberrors.NetworkError
        apiErr     *qberrors.QBitFlowError
    )
    switch {
    case errors.As(err, &validation):
        for _, f := range validation.Fields { // empty for client-side validation
            fmt.Printf("%s: %s\n", f.Field, f.Message)
        }
    case errors.As(err, &notFound):
        fmt.Println("not found")
    case errors.As(err, &rateLimit):
        if rateLimit.RetryAfter != nil {
            time.Sleep(time.Duration(*rateLimit.RetryAfter) * time.Second)
        }
    case errors.As(err, &network):
        fmt.Printf("no response: %v\n", network.Err)
    case errors.As(err, &apiErr): // any other failure (401, 403, 409, 5xx, …)
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
}
```

`qberrors.StatusCodeOf(err)` and `qberrors.FieldsOf(err)` read the status and field list from
any SDK error — also when you have wrapped it with `fmt.Errorf("...: %w", err)`.

## Retries & Timeouts

The SDK retries **idempotent requests only** — HTTP GETs — after a network/timeout error or a
`5xx` response, with exponential back-off of 1s, 2s, 4s (3 retries by default). It never
retries:

- `POST`, `PUT` or `DELETE` (creating a session or customer twice is not idempotent);
- the GETs that perform an action: `ForceCancel`, `ExecuteTestBillingCycle` and
  `TriggerTestClaimFunds`;
- `4xx`, `429` or `3xx` responses — they are returned immediately as their typed error.
  Redirects are never followed: following one would re-send your API key to the redirect
  target.

`Config.Timeout` bounds each attempt (default 30s). `Config.MaxRetries` sets the retry count:
following the Go convention that the zero value is the default, `0` keeps the default of 3,
and a negative value disables retries, as does `Config.DisableRetries: true`. The same policy
applies in the JavaScript, Python and PHP SDKs (where an explicit `0` disables retries).

## Client-Side Validation

The SDK mirrors the API's `binding` rules and rejects invalid input with a
`*qberrors.ValidationError` **before** any request is sent, so the same field rejected locally
and rejected by the API surfaces as one type. Input the API accepts is not rejected locally.

| Where | Rule |
|---|---|
| Customer / user `Name`, `LastName` | 2–100 code points of Unicode letters, decimal digits, spaces, `- _ ' .` (whitespace-only is accepted, as by the API) |
| Customer / user `Email`, `GetByEmail` lookups | required; one `@`, non-empty local part, a domain containing `.` that does not start or end with `.`, no whitespace; never normalised |
| `CreateUser.Role` | `RoleAdmin` or `RoleUser` |
| `OrganizationFeeBps` | 0–5000 |
| Product `Name` / `Description` (and inline session products) | `producttext`: 2–100 / 2–500 code points, not blank, no `<>{}[]\`\|;"~^` or control characters other than `\n \r \t` |
| Product `Price`, inline session `Price` | a finite number > 0 |
| Session `ProductID` / `ProductReference` / inline triplet | one of them is required; `ProductID` > 0 |
| Session `CustomerUUID` | a bare UUID (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`); `""` means not provided |
| Session `SuccessURL` / `CancelURL` | absolute `http(s)://` URL (scheme case-insensitive) with a host; `""` means not provided |
| Subscription `Frequency` / `TrialPeriod` / `MinPeriods` | `Frequency.Value` 1–4294967295, `TrialPeriod.Value` 0–4294967295, unit in seconds…years; `MinPeriods` 0 = none |
| Accounting `From` / `To` | required, `YYYY-MM-DD`, real dates, `From` ≤ `To` |
| Cursor `limit` | positive when given |
| Every identifier argument, every DTO / options argument | non-empty string / positive integer / non-nil |

Update DTOs validate only the fields you set; an empty string for a customer's or user's
name, last name or email means "not provided" and is left out of the request. Length is
counted in characters (code points), as the server does, so accented and CJK text is measured
correctly. A request body that cannot be encoded as JSON (NaN or ±Inf in a float) is also a
`*qberrors.ValidationError`.

## Testing

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./... && go test -race ./...
```

`go test ./...` runs the offline suite (stub servers, no key needed). To also run the
integration tests, export **both** `QBITFLOW_API_KEY` (a live-scope key) and
`QBITFLOW_BASE_URL` — the base URL is never defaulted, and a key without a base URL makes the
integration tests fail. See [tests/README.md](tests/README.md).

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
