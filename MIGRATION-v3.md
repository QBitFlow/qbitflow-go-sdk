# Migrating the QBitFlow Go SDK from 2.x to 3.0.0

This guide is for anyone using version 2.x of the QBitFlow Go SDK
(`github.com/QBitFlow/qbitflow-go-sdk/v2`). It maps everything 2.1.0, the last 2.x release,
exposed to its 3.0.0 replacement, with before and after code for the common tasks.

3.0.0 is the Go SDK for **QBitFlow API v2**. 2.x talks only to API `/v1`; 3.0.0 talks only to
`/v2`. **`/v1` is gone**: the API v2 launch migrated v1's data (people, products, payments,
subscriptions, customers, wallets, webhook URLs) to v2, so upgrade your whole integration to
3.0.0 at once rather than service by service.

The [README](README.md) documents 3.0.0 in full, and [CHANGELOG.md](CHANGELOG.md) lists every
change.

## Contents

1. [What changed at a glance](#what-changed-at-a-glance)
2. [Upgrade steps](#upgrade-steps)
3. [Breaking changes, before and after](#breaking-changes-before-and-after)
4. [Renamed and removed items](#renamed-and-removed-items)
5. [New in 3.0.0](#new-in-300)
6. [Behaviour changes](#behaviour-changes)
7. [Final checklist](#final-checklist)

## What changed at a glance

- **New module path and one package**: `github.com/QBitFlow/qbitflow-go-sdk/v3`, package
  `qbitflow`. The `pkg/qbitflow`, `pkg/models`, `pkg/errors` and `pkg/utils` packages are gone.
- **Go 1.27 or later** is required.
- **API v2 only**, with new API keys: keys issued before API v2 are refused.
- **A new client**: `qbitflow.NewClient(apiKey, opts...)` returns `(*Client, error)` and is
  configured with options instead of `Config` and setters.
- **`context.Context` first** on every method, and optional request options last.
- **Ids are UUID strings**: products and people are named by UUID; acting for a member takes
  their user UUID.
- **Money is a decimal string**, never a float, in requests and responses; fee rates are percent
  strings (`"1.5"`), not basis points.
- **Checkout sessions have their own service** (`CheckoutSessions`), with four statuses.
- **Users and claims become invitations, members and held funds.**
- **Typed errors**, one per HTTP status, matched with `errors.As`; validation field errors are
  filled in.
- **Automatic retries** for reads and the creates that carry an `Idempotency-Key`.
- **Pagination** returns `Page[T]{Items, HasMore, NextCursor}`, and every paginated list has an
  iterator.
- **A new webhook scheme**: the `QBitFlow-Signature` header, an HMAC over the raw body, typed
  events and a webhook router.

## Upgrade steps

### 1. Install 3.0.0

```bash
go get github.com/QBitFlow/qbitflow-go-sdk/v3@latest   # or @v3.0.0
```

Then remove the 2.x module once nothing imports it any more:

```bash
go mod tidy
```

### 2. Change the imports

| 2.x import | 3.0.0 import |
|---|---|
| `github.com/QBitFlow/qbitflow-go-sdk/v2` (package `qbitflow`) | `github.com/QBitFlow/qbitflow-go-sdk/v3` (package `qbitflow`) |
| `github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow` (usually `qbf`) | gone: use the root package |
| `github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models` (usually `qbmodels`) | gone: use the root package |
| `github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors` (usually `qberrors`) | gone: use the root package |
| `github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/utils` | gone |

So `qbf.CreateSessionOptions`, `qbmodels.Payment` and `qberrors.NotFoundError` all become
`qbitflow.<Name>` (the new names are in [Renamed and removed items](#renamed-and-removed-items)).

### 3. Use Go 1.27

3.0.0 uses generic methods, which need **Go 1.27**: its `go.mod` says `go 1.27.0`. Set your own
`go.mod` to `go 1.27.0` or later and build with a Go 1.27 toolchain. The SDK still has no
dependencies outside the standard library.

### 4. Create new API keys

- API v2 issues keys shaped `sk_<test|live>_<uuid>_<secret>`. **Keys of an earlier shape**
  (v1's `sk_<digits>_…`, or `sk_<uuid>_<test|live>_…`) **are refused by API v2** with a `401`:
  create new keys in the QBitFlow dashboard before you deploy 3.0.0. Treat a key as opaque: the
  SDK never parses it.
- Each key belongs to one space (your organization's, or a member's) and one mode (test or live).
  A removed member's keys answer `401`.
- `NewClient` checks only the key's shape (after trimming, it must start with `sk_` and be
  printable ASCII without spaces) and sends nothing. A bad key is a `*ValidationError` whose
  message never quotes the key. Call `client.Me(ctx)` at start-up to check the key online.

### 5. Check the base URL

The default base URL is `https://api.qbitflow.app/v2`. If you set one with `WithBaseURL`, it must
point at v2, and it may not carry credentials (`user:pass@`), a query or a fragment.

## Breaking changes, before and after

| Area | 2.x | 3.0.0 |
|---|---|---|
| Module | `github.com/QBitFlow/qbitflow-go-sdk/v2`, several packages | `github.com/QBitFlow/qbitflow-go-sdk/v3`, one package `qbitflow` |
| Go version | 1.26 | 1.27 |
| API | `/v1` | `/v2` |
| Client | `qbitflow.New(key)`, `qbitflow.NewWithConfig(qbf.Config{…})` returning `*QBitFlow` | `qbitflow.NewClient(key, opts...)` returning `(*Client, error)` |
| Configuration | `Config{APIKey, BaseURL, Timeout}`, `SetBaseURL`, `SetTimeout` | `WithBaseURL`, `WithTimeout` (per attempt), `WithMaxRetries`, `WithHTTPClient`, `WithOnBehalfOf`, set once at construction |
| Method signature | `Service.Method(args)` | `Service.Method(ctx, args, ...RequestOption)` |
| Acting for a member | `client.<Service>.OnBehalfOf(userID uint64)` | `client.OnBehalfOf(userUUID string)`, or `qbitflow.WithOnBehalfOf(userUUID)` per call |
| Ids | numeric product and user ids | UUID strings |
| Money | `float64` USD, fee rates in basis points | decimal strings (`"4.99"`), fee rates in percent (`"1.5"`) |
| Checkout | `Payments.CreateSession`, `Subscriptions.CreateSession` | `CheckoutSessions.CreatePayment`, `CheckoutSessions.CreateSubscription` |
| Checkout status | `TransactionStatus.GetTransactionStatus(uuid, txType)`, seven statuses | `CheckoutSessions.Get(ctx, uuid)`, four statuses |
| Lists | `GetAll…(limit *uint16, cursor *string)` returning `*CursorData[T]` | `List(ctx, *…ListParams)` returning `*Page[T]`, and `Iterate` |
| Cancel a subscription | `Subscriptions.ForceCancel(uuid)` | `Subscriptions.Cancel(ctx, uuid, *CancelSubscriptionParams)` |
| Users and claims | `Users.*`, `Claims.*`, `ApiKeys.*` | `Invitations.*`, `Members.*` (held funds, trust); API keys are managed in the dashboard |
| Errors | `*qberrors.QBitFlowError`, `*NotFoundError`, `*ValidationError` | one type per status, each embedding `*qbitflow.APIError` |
| Retries | none | reads and the idempotent creates are retried automatically |
| Webhooks | `X-Webhook-Signature-256` over canonical JSON, two payload shapes | `QBitFlow-Signature` over the raw body, typed events, a router |

The examples below assume `ctx := context.Background()` (or your request's context) and
`client` created as shown in the first example. Error handling is shortened in the 2.x
snippets.

### Create the client

```go
// 2.x
import (
	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

client := qbitflow.NewWithConfig(qbf.Config{
	APIKey:  os.Getenv("QBITFLOW_API_KEY"),
	Timeout: 10 * time.Second,
})
```

```go
// 3.0.0
import "github.com/QBitFlow/qbitflow-go-sdk/v3"

client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"),
	qbitflow.WithTimeout(10*time.Second), // per attempt; the base URL defaults to .../v2
)
if err != nil {
	log.Fatal(err) // a *qbitflow.ValidationError: a blank or malformed key
}

me, err := client.Me(ctx) // check the key online at start-up
if err != nil {
	log.Fatal(err)
}
fmt.Println(me.Role, "test mode:", me.Space.Test)
```

`qbitflow.NewClientFromEnv(opts...)` reads `QBITFLOW_API_KEY` (required), and
`QBITFLOW_BASE_URL` and `QBITFLOW_ON_BEHALF_OF` when they are set.

### Open a payment checkout and read its status

```go
// 2.x
session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
	ProductID:  new(uint64(42)),
	Reference:  new("order-1042"),
	SuccessURL: new("https://shop.example.com/thanks"),
})
status, err := client.TransactionStatus.GetTransactionStatus(session.UUID, qbmodels.TransactionTypeOneTimePayment)
if status.Status == qbmodels.TransactionStatusCompleted {
	payment, err := client.Payments.GetPayment(session.UUID)
	fmt.Println(payment.TransactionHash)
}
```

```go
// 3.0.0
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductUUID: "0192f1c2-1111-7c4d-9e5f-6a7b8c9d0e1f",
	Reference:   "order-1042",
	SuccessURL:  "https://shop.example.com/thanks?session=" + qbitflow.PlaceholderUUID,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("Send the customer to", session.Link) // session.UUID, session.ExpiresAt

status, err := client.CheckoutSessions.Get(ctx, session.UUID)
if err != nil {
	log.Fatal(err)
}
if status.Status == qbitflow.CheckoutSessionStatusCompleted {
	payment, err := client.Payments.Get(ctx, session.UUID, nil) // the payment has the session's id
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(payment.TxHash, payment.AmountUSD) // AmountUSD is a decimal string
}
```

Fulfil orders on the `payment.completed` webhook rather than by polling. For scripts and tests,
`CheckoutSessions.WaitForCompletion(ctx, uuid, nil)` polls `Get` until the session is final.

### List payments (for a member)

```go
// 2.x
limit := uint16(50)
page, err := client.Payments.OnBehalfOf(123).GetAllPayments(&limit, nil)
for err == nil {
	for _, p := range page.Items {
		fmt.Println(p.UUID, p.Amount)
	}
	if !page.HasMore() {
		break
	}
	page, err = client.Payments.OnBehalfOf(123).GetAllPayments(&limit, page.NextCursor)
}
```

```go
// 3.0.0
seller := client.OnBehalfOf("0192f1c2-7b3a-7c4d-9e5f-6a7b8c9d0e1f") // the member's Member.UserUUID
for p, err := range seller.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.UUID, p.AmountUSD, "USD")
}
```

### Paginate by hand

```go
// 3.0.0
params := &qbitflow.CustomerListParams{Limit: 100}
for {
	page, err := client.Customers.List(ctx, params)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range page.Items {
		fmt.Println(c.UUID, c.Email)
	}
	if !page.HasMore {
		break
	}
	params.Cursor = page.NextCursor // opaque: pass it back verbatim
}
```

### Cancel a subscription

```go
// 2.x
resp, err := client.Subscriptions.ForceCancel("sub@0192f1c2-3333-7c4d-9e5f-6a7b8c9d0e1f")
fmt.Println(resp.Message)
```

```go
// 3.0.0: cancels immediately by default
res, err := client.Subscriptions.Cancel(ctx, "sub@0192f1c2-3333-7c4d-9e5f-6a7b8c9d0e1f", nil)
if err != nil {
	log.Fatal(err)
}
fmt.Println(res.Subscription.Status, res.Pending) // Pending: still confirming on-chain (HTTP 202)

// Or stop billing now and cancel when the paid period ends:
_, err = client.Subscriptions.Cancel(ctx, "sub@0192f1c2-3333-7c4d-9e5f-6a7b8c9d0e1f",
	&qbitflow.CancelSubscriptionParams{Immediate: new(false)})
```

### Receive and verify a webhook

```go
// 2.x
func handler(w http.ResponseWriter, r *http.Request) {
	body, err := qbf.VerifyWebhookRequest(r, secret, nil)
	if err != nil {
		http.Error(w, "invalid webhook", http.StatusBadRequest)
		return
	}
	var event qbmodels.SessionWebhookResponse
	_ = json.Unmarshal(body, &event)
	if event.Status.Status == qbmodels.TransactionStatusCompleted {
		fulfil(event.Session.Reference)
	}
}
```

```go
// 3.0.0: the router verifies, parses and answers 200 / 400 / 500
router := qbitflow.NewWebhookRouter(os.Getenv("QBITFLOW_WEBHOOK_SECRET")).
	OnPaymentCompleted(func(ctx context.Context, p *qbitflow.PaymentCompleted, e *qbitflow.Event) error {
		return fulfil(ctx, e.ID, p.Reference) // idempotent: deduplicate on e.ID
	})
http.Handle("POST /webhooks/qbitflow", router)
```

Or verify and parse yourself:

```go
// 3.0.0, lower level
func handler(w http.ResponseWriter, r *http.Request) {
	body, err := qbitflow.VerifyWebhookRequest(r, secret) // reads QBitFlow-Signature
	if err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	event, err := qbitflow.ParseEvent(body)
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	if event.Type == qbitflow.EventTypePaymentCompleted {
		p, err := event.PaymentCompleted()
		if err != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		if err := fulfil(r.Context(), event.ID, p.Reference); err != nil {
			http.Error(w, "retry later", http.StatusInternalServerError) // QBitFlow delivers it again
			return
		}
	}
	w.WriteHeader(http.StatusOK) // answer 2xx to the types you ignore too
}
```

### Handle errors

```go
// 2.x
switch e := err.(type) {
case *qberrors.NotFoundError:
	fmt.Println("not found:", e.Message)
case *qberrors.ValidationError:
	fmt.Println("invalid:", e.Message)
case *qberrors.QBitFlowError:
	fmt.Println("error", e.StatusCode)
}
```

```go
// 3.0.0
_, err := client.Products.Get(ctx, "0192f1c2-1111-7c4d-9e5f-6a7b8c9d0e1f")
var (
	notFound   *qbitflow.NotFoundError
	validation *qbitflow.ValidationError
	apiErr     *qbitflow.APIError
)
switch {
case err == nil:
	// found
case errors.As(err, &notFound):
	fmt.Println("not found, request", notFound.RequestID)
case errors.As(err, &validation):
	for _, f := range validation.FieldErrors {
		fmt.Println(f.Field, f.Message)
	}
case errors.As(err, &apiErr): // matches every SDK error
	fmt.Println("error", apiErr.StatusCode, apiErr.Code, "retryable:", qbitflow.IsRetryable(err))
}
```

## Renamed and removed items

In the tables below `ctx` is left out: every 3.0.0 method takes a `context.Context` first and
optional `RequestOption`s last. Types named `…Params` are `*qbitflow.…Params`; a params argument
may be `nil` when all its fields are optional.

### Client and configuration

| 2.x | 3.0.0 |
|---|---|
| `qbitflow.New(apiKey) *QBitFlow` | `qbitflow.NewClient(apiKey, opts...) (*qbitflow.Client, error)` |
| `qbitflow.NewWithConfig(qbf.Config{APIKey, BaseURL, Timeout})` | `qbitflow.NewClient(apiKey, qbitflow.WithBaseURL(u), qbitflow.WithTimeout(d))` |
| `QBitFlow.SetBaseURL(u)`, `qbf.Client.SetBaseURL(u)`, `qbf.Client.SetTimeout(d)` | `WithBaseURL(u)`, `WithTimeout(d)` at construction; a client never changes afterwards |
| `Config.Timeout` (the whole call) | `WithTimeout(d)`: each attempt (default 30 s); bound the whole call with the context |
| `qbf.NewClient`, `qbf.NewClientWithConfig`, `qbf.New<Name>Service(client)` | removed: the services are the client's fields (`client.Products`, `client.Webhooks.Endpoints`, …) |
| `client.<Service>.OnBehalfOf(userID uint64)` | `client.OnBehalfOf(userUUID)` (a client for every service), `qbitflow.WithOnBehalfOf(userUUID)` as a client option or for one call |
| `client.Users.Get()` (the current user) | `client.Me()`: the key's role, its space and its mode |
| none | `WithMaxRetries(n)` (default 3, `0` disables), `WithHTTPClient(c)`, `NewClientFromEnv(opts...)` |
| none | request options `WithIdempotencyKey(key)`, `WithRequestID(id)`, `WithOnBehalfOf(userUUID)` |

### Checkout sessions and status (2.x `Payments`, `Subscriptions`, `TransactionStatus`)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.CreateSession(*qbf.CreateSessionOptions)` | `CheckoutSessions.CreatePayment(*CreatePaymentSessionParams)` | plain values instead of pointers; `ProductID` → `ProductUUID`; `Price` → `PriceUSD` (a string, `"4.99"`); new `ExpiresInMinutes`, `Metadata`, `Fees`; returns `*CreatedCheckoutSession` (`Link`, `UUID`, `ExpiresAt`) |
| `Subscriptions.CreateSession(*qbf.CreateSubscriptionSessionOptions)` | `CheckoutSessions.CreateSubscription(*CreateSubscriptionSessionParams)` | `Frequency` and `TrialPeriod` are `*Duration`, `MinPeriods` is `*uint32`, each optional over a subscription product's own terms |
| `TransactionStatus.GetTransactionStatus(uuid, txType)` | `CheckoutSessions.Get(uuid)` | no type argument; returns `*CheckoutSessionStatus` with four statuses |
| `TransactionStatus.GetTransactionStatusWSURL(…)` | removed | use webhooks, or poll with `CheckoutSessions.WaitForCompletion` |
| `Payments.GetSession(uuid)`, `Subscriptions.GetSession(uuid)` | removed | `CheckoutSessions.Get` for the state; an unpaid session's data comes with the `checkout.expired` event |
| none | `CheckoutSessions.Expire(uuid)` | new: end an unpaid session now |

### Payments

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.GetPayment(uuid)` | `Payments.Get(uuid, *ReadParams)` | `pay@…` or the bare UUID |
| `Payments.GetPaymentByReference(ref)` | `Payments.GetByReference(ref)` | no match is a `*NotFoundError` |
| `Payments.GetAllPayments(limit, cursor)` | `Payments.List(*PaymentListParams)`, `Payments.Iterate` | filters: customer, product, dates, `Reference`, `Refunded`, `IncludeMembers`, `UserUUID` |
| `Payments.GetAllCombinedPayments(limit, cursor)` | `Transactions.List(*TransactionListParams)`, `Transactions.Iterate` | payments and bills in one feed; `CombinedPayment` → `Transaction` |
| `Payments.GetCustomerForTransaction(txUUID)` | removed | `Payment.Customer` (a summary), or `Customers.Get(payment.CustomerUUID)` |
| none | `Failures.List`, `Failures.Iterate` | new: failed payment attempts |

### Subscriptions and pay-as-you-go

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Subscriptions.GetSubscription(uuid)` | `Subscriptions.Get(uuid, *ReadParams)` | returns cancelled subscriptions too |
| `Subscriptions.GetSubscriptionByReference(ref)` | `Subscriptions.GetByReference(ref)` | no match is a `*NotFoundError` |
| `Subscriptions.GetPaymentHistory(uuid)` | `Subscriptions.ListBills(uuid, *BillListParams)`, `Subscriptions.IterateBills` | paginated; `SubscriptionHistory` → `Bill` |
| `Subscriptions.ForceCancel(uuid)` | `Subscriptions.Cancel(uuid, *CancelSubscriptionParams)` | immediate by default; `Immediate: new(false)` cancels at the end of the paid period; returns `*SubscriptionCancellation{Subscription, Pending}`; never retried; one cancelled already is a `*ConflictError` (`409 subscription_already_stopped_or_inactive`), a 404 only another space's |
| `Subscriptions.ExecuteTestBillingCycle(uuid)` | `Subscriptions.TestBill(uuid)` | test mode only; returns a `*TestBill` (`BillUUID`, `Stage`, `Outcome`, …); `409 payment_not_due` before `NextBillingAt`, `400 test_mode_only` on a live subscription |
| none | `Subscriptions.List`, `Subscriptions.Iterate`, `Subscriptions.Update` (metadata), `Subscriptions.GetBill` | new |
| `PayAsYouGo.*` (`GetSession`, `GetSubscription`, `GetSubscriptionByReference`, `GetPaymentHistory`, `ForceCancel`, `ExecuteTestBillingCycle`) | removed | API v2 has no pay-as-you-go subscriptions |

### Refunds

Refunds are read-only in the SDK: a merchant's refund is sent from the dashboard.

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Refunds.GetAll()` (open refunds) | `Refunds.List(&RefundListParams{Status: []RefundStatus{RefundStatusPending}})`, `Refunds.Iterate` | paginated; from the organization's space the members' refunds are included by default (`IncludeMembers: new(false)` leaves them out) |
| `Refunds.GetAllInactive(limit, cursor)` | `Refunds.List(&RefundListParams{Status: []RefundStatus{RefundStatusApproved, RefundStatusRejected}})`, `Refunds.Iterate` | one list, filtered by `Status` |
| `Refunds.GetByTransactionUUID(txUUID)` | `Refunds.List(&RefundListParams{TxUUID: txUUID})` | or read `Payment.Refund` / `Bill.Refund` (a `*RefundSummary`) |
| none | `Refunds.Get(refundUUID)` | new |

### Customers and products

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Customers.Create(*qbf.CreateCustomer)` | `Customers.Create(*CreateCustomerParams)` | `Name` → `FirstName`; optional fields are plain strings (`""` means absent); new `Metadata` |
| `Customers.Get(uuid)`, `Customers.Delete(uuid)` | same names | `Delete` returns only an error |
| `Customers.GetByReference(ref)` | `Customers.GetByReference(ref)` | no match is a `*NotFoundError` |
| `Customers.GetByEmail(email)` | removed | `Customers.List(&CustomerListParams{Email: email})`: an email may match several customers |
| `Customers.Update(uuid, *qbf.UpdateCustomer)` | `Customers.Update(uuid, *UpdateCustomerParams)` | `FirstName`, `Email` are strings (`""` = unchanged); `LastName`, `PhoneNumber`, `Address` are `*string` and `new("")` clears them; `Metadata` is merged |
| `Customers.GetAll(limit, cursor)` | `Customers.List(*CustomerListParams)`, `Customers.Iterate` | filters `Email`, `Reference` |
| `Products.Create(*qbf.CreateProduct)` | `Products.Create(*CreateProductParams)` | `Price` → `PriceUSD` (a string); `Description` optional; `Reference` a string; `Subscription` terms make a subscription product; new `Metadata` |
| `Products.Get(id uint64)` | `Products.Get(uuid string)` | |
| `Products.GetAll()` | `Products.List(*ProductListParams)`, `Products.Iterate` | paginated; filters `IncludeHidden`, `Subscription`, `Reference` |
| `Products.GetByReference(ref)` | `Products.GetByReference(ref)` | hidden products included; no match is a `*NotFoundError` |
| `Products.Update(id, *qbf.UpdateProduct)` | `Products.Update(uuid, *UpdateProductParams)` | `Name`, `PriceUSD` strings (`""` = unchanged); `Description`, `IsActive` pointers; `Subscription`, `RemoveSubscription`; `Metadata` merged |
| `Products.Delete(id)` | `Products.Delete(uuid)` | |
| `CreateProduct.Validate()`, `UpdateProduct.Validate()` | `CreateProductParams.Validate()`, `UpdateProductParams.Validate()` | every params type has `Validate`, and it runs before each request |

### Users, claims and API keys

API v2 creates no accounts for you: a seller exists once they accept an invitation. See
[Members instead of users](#members-instead-of-users) below.

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Users.Create(*qbf.CreateUser)` | `Invitations.Create(*CreateInvitationParams)` | not a rename: the person becomes a member when they accept (`member.joined`) |
| `Users.GetAll()` | `Members.List(*MemberListParams)`, `Members.Iterate` | |
| `Users.GetByID(id)` | `Members.Get(userUUID)` | |
| `Users.GetByEmail(email)` | removed | iterate `Members` and match `Member.Email` |
| `Users.Update(id, *qbf.UpdateUser)` | `Members.Update(userUUID, *UpdateMemberParams)` | only the organization fee (`OrganizationFeePercent`, a required string); names and emails belong to the person |
| `Users.Delete(id)` | `Members.Remove(userUUID)` | ends the membership; `409 held_funds_pending` while you hold their funds |
| `Users.Get()` | `client.Me()` | |
| `Claims.CreateClaimRequest(userID)`, `Claims.GetClaimRequestByUserID(userID)` | `Invitations.Create`, `Invitations.List` | the invitation link replaces the claim link |
| `Claims.GetClaimFunds()` | `Members.ListHeldFunds()`, `Members.GetHeldFunds(userUUID)`, `Members.GetOwnHeldFunds()` | |
| `Claims.TriggerTestClaimFunds(userID)` | removed | release held funds in the dashboard; `Members.Trust(userUUID)` pays the member's new payments directly |
| `ApiKeys.GetAll()`, `ApiKeys.GetForUser(userID)` | removed | keys are managed in the dashboard; `client.Me()` describes the current key |
| none | `Invitations.Revoke`, `Invitations.Iterate`, `Members.Trust` | new |

### Wallets, accounting and currencies

| 2.x | 3.0.0 | Notes |
|---|---|---|
| none | `Wallets.List(*WalletListParams)`, `Wallets.ListForMember(userUUID)`, `Wallets.ListAcceptedCurrencies(*AcceptedCurrenciesParams)` | new; each returns a slice |
| `Accounting.Export(qbf.AccountingExportParams{From, To})` | `Accounting.ExportJSON(from, to)` | dates `YYYY-MM-DD`, both included, at most 92 days |
| none | `Accounting.ExportCSV(from, to)`, `Accounting.ExportJSONRange`, `Accounting.ExportCSVRange` | the CSV export, and any range split into 92-day windows |
| `Currencies.GetAllAvailable(test)` | `Currencies.List(&CurrencyListParams{Test: test})` | returns a slice |
| `Currencies.GetAllMain(test)` | `Currencies.List(&CurrencyListParams{Test: test, Main: true})` | |
| none | `Currencies.Get(id)` | new |

### Webhooks

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `qbf.VerifyWebhookRequest(r, secret, *qbf.VerifyOptions)` | `qbitflow.VerifyWebhookRequest(r, secret, ...VerifyOption)` | reads `QBitFlow-Signature`; at most 1 MiB; puts the body back on `r.Body` |
| `qbf.VerifyWebhookSignature(secret, timestamp, signature, payload, opts)`, `Webhooks.VerifyLocal(…)` | `qbitflow.VerifyWebhook(rawBody, signatureHeader, secret, ...VerifyOption)`, also `client.Webhooks.Verify` | returns `nil` or a `*WebhookSignatureError` |
| `Webhooks.Verify(payload, signature, timestamp) (bool, error)` (checked by the API) | `Webhooks.VerifyRemote(endpointUUID, rawBody, signatureHeader) error` | `nil` means valid; a mismatch is a `*WebhookSignatureError` |
| none | `qbitflow.ConstructEvent`, `qbitflow.ParseEvent`, `qbitflow.NewWebhookRouter` | verify and parse into a typed `*Event`, or dispatch to typed handlers |
| `qbf.ComputeWebhookSignature`, `qbf.CanonicalJSON` | removed | the signature covers the raw body; `qbitflow.SignWebhook` signs a body for your tests |
| `qbf.ExtractWebhookHeaders`, `qbf.WebhookHeaders`, `WebhookHeaders.IsTest()` | removed | `r.Header.Get(qbitflow.SignatureHeader)`; the dashboard's test is a `webhook.test` event (`Event.Test` says test mode) |
| `HeaderSignature`, `HeaderTimestamp`, `HeaderWebhookID`, `Webhooks.Get…Header()` | `qbitflow.SignatureHeader`, `qbitflow.EventIDHeader`, `qbitflow.EventTypeHeader` | the timestamp is part of `QBitFlow-Signature` |
| `TEST_WEBHOOK_ID` | `qbitflow.EventTypeWebhookTest` | |
| `qbf.VerifyOptions{MaxTimestampAge, Now, SkipTimestampCheck}`, `DefaultMaxTimestampAge` | `qbitflow.WithTolerance(d)`, `qbitflow.WithClock(f)`, `qbitflow.DefaultWebhookTolerance` | the timestamp check can no longer be skipped |
| `qbmodels.SessionWebhookResponse`, `qbmodels.SubscriptionStatusTransitionWebhook` | `*qbitflow.Event` and its typed accessors | see [Webhooks](#webhooks-1) |
| none | `Webhooks.Endpoints` (`List`, `Iterate`, `Create`, `Get`, `Update`, `Delete`), `Webhooks.Events` (`List`, `Iterate`, `Get`) | new: manage endpoints and read the event log |

### Errors and helpers

| 2.x | 3.0.0 |
|---|---|
| `qberrors.QBitFlowError` | `*qbitflow.APIError` (and the `qbitflow.QBitFlowError` interface every SDK error implements) |
| `qberrors.NotFoundError` | `*qbitflow.NotFoundError` |
| `qberrors.ValidationError`, `qberrors.ValidationErrors` | `*qbitflow.ValidationError`, with `FieldErrors` |
| `qberrors.NewQBitFlowError`, `NewNotFoundError`, `NewValidationError`, `NewValidationErrorFromList` | removed |
| `utils.CursorQueryBuilder`, `qbf.ValidateProductTextForTest` | removed |
| `qbmodels.SessionCheckout.IsPayment()`, `IsSubscription()` | removed: `CheckoutExpired.Payment` / `CheckoutExpired.Subscription` |

### Models

| 2.x (`qbmodels`) | 3.0.0 (`qbitflow`) | What changed |
|---|---|---|
| `LinkResponse` | `CreatedCheckoutSession` | `Link`, `UUID`, and `ExpiresAt` |
| `TransactionStatus` | `CheckoutSessionStatus` | `Status` is a `CheckoutSessionStatusValue`; `Message *string` → `string`; `SettlementDetails` removed (read the payment's `Settlement`); new `UUID`, `LastAttempt`, `ExpiresAt` |
| `SessionCheckout` | removed | `PaymentSessionData` / `SubscriptionSessionData` in the `checkout.expired` event |
| `Payment` | `Payment` | `Amount` is now the **token amount** (a string) and `AmountUSD` its USD value; `TransactionHash` → `TxHash`; `Name` / `Description` → `ProductName` / `ProductDescription`; `ProductID *uint64` → `ProductUUID string`; `Reference *string` → `string`; `CurrencyID` removed (read `Currency.ID`); `Metadata` (the fee split) → `Settlement`, and `Metadata` is now your checkout's `map[string]string`; `UserID` → `UserUUID`; new `PriceUSD`, `Fees`, `Chain`, `ExplorerURL`, `CustomerReference`, `Customer`, `Refund`, `Refundable`, `Paid`, `PaidMinUnits`, `PaidUSD`, `ConfirmedAt` |
| `CombinedPayment` | `Transaction` | `Source` → `Type` (`TransactionKindPayment`, `TransactionKindBill`); the same renames as `Payment` |
| `Subscription` | `Subscription` | `SubscriptionStatus` → `Status`; `Stopped bool` → the status `stopped`; `Frequency uint32` (seconds) → `Duration`; `ProductID` → `ProductUUID`; `Reference *string` → `string`; `LastBillingDate` → `LastBilledAt`, `NextBillingDate` → `NextBillingAt`, `MinimumCancellationDate` → `CancellableAt` (all `*time.Time`, nil when absent); `Allowance string` → `Allowance *Allowance` (`Amount`, `AmountMinUnits`, `AmountUSD`, `RemainingPeriods`, `MaxPerPeriod`, `MaxPerPeriodMinUnits`); `CurrencyID` removed; new `CurrentPeriodEnd`, `ActionRequired`, `PriceUSD`, `Fees`, `CancelledAt`, `CancellationReason`, `Metadata`, `ManagementPageLink`, `CustomerReference`, `Customer`, `Dunning`, and the `HasAccess` method |
| `SubscriptionHistory` | `Bill` | the same renames as `Payment`; new `Settlement`, `PeriodStart`, `PeriodEnd` |
| `RefundEntry` | `Refund` | `TxID` → `TxUUID`; amounts as strings; `Metadata` → `Settlement` (once approved); new `InitiatedBy`, `RefundPercent`, `Amount`, `AmountUSD`, `Currency`, `Held`, `Chain`, `ExplorerURL`, `Approval`, `Customer` |
| `Customer` | `Customer` | `Name` → `FirstName`; `PhoneNumber`, `Address`, `Reference`: `*string` → `string`; new `Metadata` |
| `Product` | `Product` | `ID uint64` → `UUID string`; `Price float64` → `PriceUSD string`; new `Subscription`, `PaymentLink`, `Metadata` |
| `User` | `Member` (or `Me` for the current key) | `ID` → `UserUUID`; `Name` → `FirstName`; `OrganizationFeeBps` → `OrganizationFeePercent` (a string); `ClaimedAt` → `TrustedAt`; new `JoinedAt`, `AcceptedCurrencies`, `SpaceUUID` |
| `ClaimFund` | `MemberHeldFunds`, `HeldFunds`, `LedgerEntry` | `TotalUSD` as a string, and each held payment's line |
| `UserRole` (`RoleAdmin`, `RoleUser`) | `Role` (`RoleAdmin`, `RoleUser`) | |
| `AccountingEvent` | `AccountingEvent` | `PaymentID` → `TxUUID`; `RelatedPaymentID` → `RelatedTxUUID`; `PaymentReference` / `RelatedPaymentReference` → `Reference` / `RelatedReference`; `ProductID uint64` → `ProductUUID string`; `TokenSymbol` → `Currency`; every amount, USD amount and rate is a decimal string, amounts in the currency's units with a `…MinUnits` twin; `Type` is an `AccountingEventType` and `Chain` a `Chain`; new `UserUUID`, `ReferralFee…`, `CurrencyPriceUSD`, `NetworkFeesPaidBy`, `NetworkFeesCurrency…` |
| `PaymentMetadata` | `Settlement` | `FeeBps` → `FeePercent` (a string); `TxAmounts` → `Split` (`Platform`, `Organization`, `Referral`, `Merchant`, `NetworkFee`, each a `TokenAmount`); the block and network fees → `Transaction` (a `SettlementTransaction`) |
| `OrganizationFee{OrganizationID, Organization, FeeBps}` | `OrganizationFee{Address, FeePercent}` | |
| `ReferralFee{ReferralID, Referrer, FeeBps, Deadline}` | `ReferralFee{Address, FeePercent}` | |
| `Currency` | `Currency` | `Decimals int` → `uint8`; new `Code` (`BASE:USDC`) and `Chain`; `Address` empty for a native coin; no nested `MainCurrency` (`MainCurrencyID` stays) |
| `Duration{Value int, Unit}` | `Duration{Value uint32, Unit}` | |
| `CursorData[T]` | `Page[T]` | `Items`; `HasMore` is a field, not a method; `NextCursor` is a `string`, `""` on the last page |
| `ApiKey`, `Organization`, `CreateClaimRequestResponse`, `StatusResponse`, `SuccessResponse`, `ErrorResponse`, `SubscriptionType`, `TransactionShortType`, `TxMetadata`, `TxAmountsFull` and the other `TxAmounts…` types | removed | |

**Typing rule.** An optional string is `""` when absent; any other optional field is a pointer
(`nil` when absent); a field the API always sends is a value; lists are slices and `Metadata` is a
`map[string]string`. Request params follow the same rule; set a pointer inline with Go's
`new(value)`, for example `Immediate: new(false)`.

### Enums and statuses

Enum values are lowerCamelCase on the wire, and an unknown value received from the API is kept as
is.

**Subscription statuses**

| 2.x | 3.0.0 |
|---|---|
| `SubscriptionStatusActive` (`active`) | `SubscriptionStatusActive` (`active`) |
| `SubscriptionStatusPastDue` (`past_due`) | `SubscriptionStatusPastDue` (`pastDue`) |
| `SubscriptionStatusTrial` (`trial`) | `SubscriptionStatusTrial` (`trial`) |
| `SubscriptionStatusTrialExpired` (`trial_expired`) | `SubscriptionStatusTrialExpired` (`trialExpired`) |
| `SubscriptionStatusCancelled` (`cancelled`) | `SubscriptionStatusCancelled` (`cancelled`) |
| `SubscriptionStatusLowOnFunds` (`low_on_funds`) | no status: `Subscription.ActionRequired == ActionRequiredTopUpAllowance` |
| `SubscriptionStatusPending` (`pending`) | no status: `Subscription.ActionRequired == ActionRequiredRaiseMaximum` |
| `Subscription.Stopped == true` | `SubscriptionStatusStopped` (`stopped`): cancelled when the paid period ends |
| none | `SubscriptionStatusPaused` (`paused`): paused by the customer |

**Grant access while `now` is before `AccessUntil`**, whatever the status, or call
`sub.HasAccess(time.Time{})` (from 3.1.0; `AccessUntil` is `CurrentPeriodEnd`, plus an hour for an
`active` subscription while its renewal is billed). A check like `status == active` cuts off customers who have paid
(`stopped`, `paused`) and keeps serving `pastDue` ones.

**Checkout statuses.** 2.x's seven `TransactionStatusValue`s become four
`CheckoutSessionStatusValue`s:

| 2.x | 3.0.0 |
|---|---|
| `TransactionStatusCreated` | `CheckoutSessionStatusCreated` (waiting for the customer) |
| `TransactionStatusWaitingConfirmation` | `CheckoutSessionStatusWaitingConfirmation` (a transaction was sent) |
| `TransactionStatusCompleted` | `CheckoutSessionStatusCompleted` (final) |
| `TransactionStatusExpired` | `CheckoutSessionStatusExpired` (final: expired unpaid, or ended with `Expire`) |
| `TransactionStatusPending`, `TransactionStatusFailed`, `TransactionStatusCancelled` | no equivalent: a failed attempt leaves the session `created` with `LastAttempt` set |

A failed attempt is never final: the customer may still pay until the session expires, so never
cancel an order on it.

**Other values**

| 2.x | 3.0.0 |
|---|---|
| `TransactionTypeOneTimePayment` (`payment`) | `TransactionTypePayment` (`payment`) |
| `TransactionTypeCreateSubscription` | `TransactionTypeCreateSubscription` |
| every other `TransactionType…` constant | removed: a checkout's `TxType` is `payment` or `createSubscription` |
| `RefundStatusPending`, `RefundStatusApproved`, `RefundStatusRejected` | same names and values |
| `CombinedPayment.Source` (`payment`, `subscription_history`) | `Transaction.Type`: `TransactionKindPayment`, `TransactionKindBill` (`bill`) |
| `AccountingEvent.Type` (a string) | `AccountingEventType`: `AccountingEventTypePayment`, `AccountingEventTypeBill` (a subscription bill), `AccountingEventTypeRefund`, `AccountingEventTypeOrganizationFee`, `AccountingEventTypeReferralFee` |
| none | `ActionRequired`, `CancellationReason`, `LedgerEntryType`, `InvitationStatus`, `FailureKind`, `FailureCategory`, `BillingStage`, `BillingOutcome`, `Chain`, `EventType`, … |

## New in 3.0.0

- **Iterators.** Every paginated list has an `Iterate` method (`IterateBills` for bills)
  returning an `iter.Seq2[T, error]` that fetches one page at a time as you range over it.
- **Webhook router.** `qbitflow.NewWebhookRouter(secret)` is an `http.Handler` that verifies,
  parses and dispatches each delivery to typed handlers (`OnPaymentCompleted`,
  `OnSubscriptionBilled`, `OnRefundRequested`, `OnMemberJoined`, `OnCheckoutExpired`, … plus
  `OnUnknown` and `OnAny`), and answers 200, 400 or 500 as QBitFlow expects. `Handle` serves
  other frameworks.
- **Typed webhook events.** `ParseEvent`, `ConstructEvent`, one accessor per event type
  (`event.PaymentCompleted()`, `event.SubscriptionBilled()`, …) and `event.DecodeData[T]()`.
- **Webhook endpoints and the event log**: `Webhooks.Endpoints` and `Webhooks.Events`.
- **Marketplaces**: `client.OnBehalfOf(userUUID)` / `WithOnBehalfOf`, `Invitations`,
  `Members` (list, update the organization fee, trust, remove) and held funds
  (`Members.ListHeldFunds`, `GetHeldFunds`, `GetOwnHeldFunds`).
- **`client.Me`**: the key's role, space and mode, the recommended start-up check.
- **Checkout sessions**: `Expire`, `WaitForCompletion`, `ExpiresInMinutes`, your own
  `Metadata`, and `Fees` (a processing fee and your own lines such as tax or shipping).
- **Subscriptions**: `List`, `Update` (metadata), `ListBills`, `GetBill`, `Cancel` at the end
  of the period, `TestBill`, and `Subscription.HasAccess`.
- **Refunds read API**: `Refunds.List` (by status, transaction, member), `Refunds.Iterate`,
  `Refunds.Get`.
- **Transactions and failures**: `Transactions.List` (payments and bills in one feed),
  `Failures.List` (failed payment attempts).
- **Wallets**: `Wallets.List`, `Wallets.ListForMember`, `Wallets.ListAcceptedCurrencies`.
- **Accounting export** as JSON or CSV, for any range: `ExportJSON`, `ExportCSV`,
  `ExportJSONRange`, `ExportCSVRange`.
- **Metadata**: your own `map[string]string` on products, customers, checkouts and
  subscriptions.
- **Typed errors** with `errors.As`, field errors, request ids and `qbitflow.IsRetryable`.
- **Retries with an `Idempotency-Key`**, and `WithIdempotencyKey` to retry a create safely
  across processes.
- **Amount helpers**: `qbitflow.FormatAmount`, `qbitflow.ParseAmount` and
  `Currency.FormatAmount` convert between token units and on-chain integers exactly.
- **Client-side validation**: every params type has `Validate`, run before each request.
- **`NewClientFromEnv`** and **`SignWebhook`** (to test your webhook handler).

## Behaviour changes

### Errors

2.x returned `*qberrors.QBitFlowError`, `*qberrors.NotFoundError` or `*qberrors.ValidationError`,
and mapped every 400 and 422 to a validation error. 3.0.0 returns one type per condition, each
embedding `*qbitflow.APIError` (`StatusCode`, `Code`, `Message`, `Details`, `RequestID`,
`FieldErrors`, `RawBody`, `Err`). Match them with `errors.As`, not a type switch: an
`*APIError` target matches every SDK error.

| Condition | 3.0.0 type |
|---|---|
| 400 `validation_failed`, or input the SDK refused before sending (`StatusCode` 0) | `*ValidationError` |
| any other 400 | `*BadRequestError` |
| 401 | `*AuthenticationError` |
| 403 | `*PermissionDeniedError` |
| 404 | `*NotFoundError` |
| 409 | `*ConflictError` |
| 410 | `*GoneError` |
| 422 `idempotency_key_reused` | `*IdempotencyError` |
| 429 | `*RateLimitError` (`RetryAfter`, `Limit`, `PeriodSeconds`) |
| 5xx, an unexpected 3xx (redirects are never followed), an unusable 2xx | `*ServerError` |
| no response (DNS, connection, TLS, timeout, cancelled context) | `*NetworkError` |
| a webhook signature refused | `*WebhookSignatureError` (`Reason`) |
| any other HTTP error (413 `request_too_large`, …) | `*APIError` itself |

- **Field errors are filled in.** 2.x looked for a top-level `errors` list, so validation
  details came back empty. 3.0.0 reads the API's `details.errors`: `FieldErrors` names each
  failing input by the name you sent, dotted when nested (`frequency.unit`).
- **Branch on `Code`, never on `Message`**: for example `unique_violation`
  (`Details["field"]`), `merchant_not_ready`, `tx_already_sent`, `policy_disabled`.
- **403.** `forbidden` (no details) means the key's role or the resource does not allow the call.
  `policy_disabled` means a member's key did something its organization's member policy does not
  allow: `Details["policy"]` names the policy, and `Details["field"]` the request field when the
  policy governs one.
- **`GetByReference`** with no match is a `*NotFoundError` (status 404, code `not_found`), like
  an unknown UUID.
- **`qbitflow.IsRetryable(err)`** reports a transient failure: a network error or timeout, a
  5xx, a 429, or a 409 `idempotency_key_in_use`.
- Every error carries the request's `RequestID`: log it and quote it to support.
  `WithRequestID(id)` lets you set your own.

### Retries and idempotency

2.x never retried. 3.0.0 retries automatically: drop any retry loop you wrapped around reads and
creates, or turn retries off with `WithMaxRetries(0)`.

| | |
|---|---|
| Retried methods | every read (GET), and the 6 idempotent creates: `CheckoutSessions.CreatePayment`, `CheckoutSessions.CreateSubscription`, `Products.Create`, `Customers.Create`, `Webhooks.Endpoints.Create`, `Invitations.Create` |
| Never retried | every other write: updates, deletes, `CheckoutSessions.Expire`, `Subscriptions.Cancel`, `Subscriptions.TestBill`, `Members.Trust`, `Members.Remove`, `Invitations.Revoke`, `Webhooks.VerifyRemote` |
| Retried on | a network error or timeout, a 5xx, a 429, a 409 `idempotency_key_in_use` |
| Not retried on | any other 4xx (422 `idempotency_key_reused` included), a 3xx, an unusable response |
| Attempts | 3 retries by default (`WithMaxRetries(n)`; `0` disables them) |
| Back-off | 1 s, 2 s, 4 s, …; a 429 waits for its `Retry-After` when longer; a wait above 60 s is not made and the `*RateLimitError` is returned at once |

- **Idempotency keys.** Each call of an idempotent create sends a fresh `Idempotency-Key` (a
  UUID v4) and sends **the same key on every retry** of that call, so a create retried after a
  timeout returns the first result instead of creating a second one. To retry across processes
  (a queue re-running a job), pass your own stable key with
  `qbitflow.WithIdempotencyKey("checkout-order-1042")`. Reusing a key with different params is a
  `*IdempotencyError`.
- **Timeouts.** `WithTimeout` bounds each attempt (default 30 s), not the whole call: give the
  context a deadline to bound the whole call. Cancelling the context also stops any wait between
  retries.
- Retry the other writes yourself only after reading the resource's state: a timeout may have done
  the work.

### Pagination

- `GetAll…(limit *uint16, cursor *string)` becomes `List(ctx, &…ListParams{Limit, Cursor,
  filters…})`, returning a `*Page[T]` with `Items`, `HasMore` and `NextCursor`.
- `HasMore` is a field (2.x had a `HasMore()` method), and `NextCursor` is a `string`, `""` on
  the last page.
- Cursors are opaque: pass `page.NextCursor` back as `Cursor` verbatim, never build or parse one.
- `Limit` is an `int`: `0` means the default page size (20), and the maximum is 100.
- `Products.GetAll()` and `Refunds.GetAll()` returned everything at once; `Products.List` and
  `Refunds.List` are paginated.
- Every paginated list has an iterator (`Iterate`, `IterateBills`) that fetches pages lazily and
  stops after the page whose `HasMore` is false, or when you `break`. An error is yielded once
  and ends the loop.
- The short bounded lists return a slice: `Currencies.List`, `Wallets.List`,
  `Wallets.ListForMember`, `Wallets.ListAcceptedCurrencies`, `Members.ListHeldFunds`,
  `Accounting.ExportJSON`.

### Webhooks

The webhook scheme is new: 2.x verification code does not carry over.

| | 2.x | 3.0.0 |
|---|---|---|
| Headers | `X-Webhook-Signature-256: sha256=<hex>`, `X-Webhook-Timestamp`, `X-Webhook-ID` | `QBitFlow-Signature: t=<unix>,v1=<hex>`, `QBitFlow-Event-Id`, `QBitFlow-Event-Type` |
| Signed content | the timestamp, `.`, and the body re-encoded as canonical JSON | `t`, `.`, and the **raw body** exactly as received |
| Algorithm | HMAC-SHA256 | HMAC-SHA256 with the endpoint's `whsec_…` secret, compared in constant time |
| Replay window | 5 minutes, could be skipped | 300 seconds either way (`WithTolerance`), cannot be skipped |
| Secret rotation | one secret | for 24 hours after a rotation the header carries two `v1=` signatures and either secret verifies |
| Body | `SessionWebhookResponse` or `SubscriptionStatusTransitionWebhook` | a typed event: `ID`, `Type`, `CreatedAt`, `Test`, `UserUUID`, `Data` |
| Dashboard test | `X-Webhook-ID: test-webhook-id` | a `webhook.test` event |

- **Read the raw body and never re-serialize it** before verifying. `VerifyWebhookRequest` does
  it for `net/http`; elsewhere call `VerifyWebhook(rawBody, signatureHeader, secret)` or
  `ConstructEvent(rawBody, signatureHeader, secret)`.
- **Event types** replace the two 2.x payloads: the transaction webhook becomes
  `payment.completed` and `subscription.created`; the subscription webhook becomes
  `subscription.billed` and `subscription.statusChanged`. New types:
  `subscription.actionRequiredChanged`, `subscription.billingFailed`,
  `subscription.upcomingBill`, `refund.requested`, `refund.approved`, `refund.rejected`,
  `member.joined`, `member.removed`, `heldFunds.released`, `checkout.expired` and
  `webhook.test`. Each has a typed accessor and a router method.
- Only API v2 events are accepted: a body in another format is a `*ValidationError` from
  `ParseEvent`.
- Money in event data is a decimal string, as in API responses. `subscription.billed` carries
  `{Bill, Subscription}`; `subscription.created` carries the `*Subscription`.
- Deliveries are at least once: deduplicate on `Event.ID`, answer 2xx quickly, also to the
  types you ignore. A non-2xx answer makes QBitFlow deliver the event again.
- Endpoints are created per space and mode, in the dashboard or with
  `Webhooks.Endpoints.Create`, whose answer carries the `whsec_…` secret once: store it.
- `Event.UserUUID` names the member whose space an event happened in: pass it to
  `WithOnBehalfOf` for follow-up reads.

### Test and live mode

- There is one base URL for both modes: **the API key's mode decides**. A test key reads and
  writes test data only, a live key live data only.
- `me.Space.Test` (from `client.Me`) tells you the key's mode: assert it at start-up, for example
  in a job that must only run in test mode.
- Resources, webhook endpoints and members exist per mode. Resources and events carry a `Test`
  field.
- Test mode caps a checkout's total at 5 USD, fees included, and bills subscriptions only when
  you call `Subscriptions.TestBill`.
- `Currencies.List`'s `Test` flag picks the test networks' catalog, whatever the key's mode.

### Ids, references and money

- **Products** are named by `UUID` (was the numeric `ID`), and `ProductID` is `ProductUUID`
  everywhere. **People** are named by their user UUID (`UserID` → `UserUUID`, `""` for the
  organization's own rows). `OrganizationID` is gone from every model;
  `me.Space.OrganizationUUID` names your organization. Currencies keep numeric ids.
- **Transaction ids** keep their prefixes (`pay@…`, `sub@…`, `sub-hist@…`, `refund@…`) as opaque
  strings, and every method also takes the bare UUID. `TxID` → `TxUUID`, `TransactionHash` →
  `TxHash`.
- **Map the numeric ids you stored.** Walk your products
  (`Products.Iterate(ctx, &qbitflow.ProductListParams{IncludeHidden: true})`, in each space you act
  in) and match them on `Reference`, which did not change; walk your members (`Members.Iterate`,
  in each mode) and match them on `Email`. Store the UUIDs next to your old ids.
- **References** (product, customer and checkout references, `ProductReference`,
  `CustomerReference`, the `Reference` list filters and `GetByReference`) are 1 to 100 characters
  among `A-Z a-z 0-9 . _ : @ -`, and never exactly `.` or `..`. Anything else is refused before
  sending with a `*ValidationError`.
- **Money is a decimal string**, never a float: USD amounts (`PriceUSD: "4.99"`, `AmountUSD`,
  `PaidUSD`, `TotalUSD`), token amounts in the token's units (`Amount: "10"` for 10 USDC, next to
  `AmountMinUnits: "10000000"`) and fee rates in **percent** (`FeeBps: 150` →
  `FeePercent: "1.5"`, `OrganizationFeeBps: 250` → `OrganizationFeePercent: "2.5"`). Inputs are
  checked before sending (at most 2 decimals, no sign or exponent). Use a decimal library, or
  `FormatAmount` / `ParseAmount`, to compute with them.
- **Metadata** is new: at most 50 keys of 1 to 40 characters among `A-Z a-z 0-9 _ - . :`, values
  of at most 500 characters. An update merges it, and a `""` value deletes a key.

### Members instead of users

v1 provisioned sellers (`Users.Create`) and could sell for them at once, the seller claiming the
account and its funds later. **API v2 creates no accounts: a seller sells only once they have
accepted an invitation.**

1. `client.Invitations.Create(ctx, &qbitflow.CreateInvitationParams{Email: …, HoldFunds: true,
   OrganizationFeePercent: "2.5"})` returns a `*CreatedInvitation` with its `Link` (also emailed).
2. Wait for the `member.joined` event and store its `UserUUID` (match it on `InvitationUUID`).
3. Sell for them with `client.OnBehalfOf(userUUID)`.

`HoldFunds: true` (the v1 claim model) keeps the seller's payments in your wallet until you trust
them: `Members.GetHeldFunds` shows what you owe, `Members.Trust` makes new payments go to the
seller directly, and releasing what is held is done in the dashboard. Sellers created under v1
are already members; those who never claimed their account are members whose funds you hold
(`TrustedAt == nil`). The [README](README.md#marketplaces) walks through the flow.

### Member keys and checkout redirect URLs

A member's own key may send a checkout's `SuccessURL` and `CancelURL` while its organization's
`members.checkoutRedirects` policy allows it, which is the default. With the policy turned off,
the call fails with a 403 `policy_disabled` (`*PermissionDeniedError`, `Details["policy"]` is
`members.checkoutRedirects` and `Details["field"]` is `successUrl` or `cancelUrl`). An
organization key acting for the member with `OnBehalfOf` may always send them.

## Final checklist

- [ ] `go get github.com/QBitFlow/qbitflow-go-sdk/v3@latest`, then `go mod tidy`.
- [ ] Every import points at `github.com/QBitFlow/qbitflow-go-sdk/v3`; no `pkg/...` import is
      left.
- [ ] `go.mod` says `go 1.27.0` or later, and your build uses Go 1.27.
- [ ] New API v2 keys are created in the dashboard and deployed (test and live).
- [ ] The client is built with `NewClient(key, opts...)` (or `NewClientFromEnv`), its error is
      checked, and `client.Me(ctx)` confirms the key and its mode at start-up.
- [ ] Every call passes a `context.Context`; `OnBehalfOf` takes the member's user UUID.
- [ ] Stored numeric product and user ids are mapped to UUIDs.
- [ ] Amounts and fee rates are handled as decimal strings, never floats.
- [ ] Checkouts use `CheckoutSessions.CreatePayment` / `CreateSubscription`, and status checks
      use the four checkout statuses.
- [ ] Access checks use `AccessUntil` (or `HasAccess`), not `status == active`.
- [ ] Users and claims code is replaced by invitations, members and held funds.
- [ ] Errors are matched with `errors.As`; your own retry loops around reads and creates are
      removed (or `WithMaxRetries(0)` is set).
- [ ] Lists use `List` with `Page{Items, HasMore, NextCursor}`, or `Iterate`.
- [ ] Webhook handlers verify `QBitFlow-Signature` over the raw body (`NewWebhookRouter`,
      `VerifyWebhookRequest` or `ConstructEvent`), handle the typed events, deduplicate on
      `Event.ID` and answer 2xx to ignored types.
- [ ] Webhook endpoints and their `whsec_…` secrets are in place for each space and mode.
- [ ] Your integration is tested end to end with a test-mode key before switching to live.
