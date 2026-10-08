# QBitFlow Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/QBitFlow/qbitflow-go-sdk/v3.svg)](https://pkg.go.dev/github.com/QBitFlow/qbitflow-go-sdk/v3)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

The official Go SDK for [QBitFlow](https://qbitflow.app), non-custodial crypto payments: hosted
checkouts, one-time payments, subscriptions, refunds, marketplaces with commissions and held
funds, accounting exports and signed webhooks. Customers pay from their own wallets, on Ethereum,
Base and Solana, straight to yours.

- **API v2**, every integrator route: 13 services, 60 routes, one `qbitflow` package.
- **No dependencies** beyond the standard library.
- **Typed errors** matched with `errors.As`, each carrying the API's code, request id and field errors.
- **Safe retries**: reads and creates are retried on network errors, 5xx and 429, and every create
  sends an `Idempotency-Key`, so a retry never charges or creates twice.
- **Iterators**: `for p, err := range client.Payments.Iterate(ctx, nil)` walks every page lazily.
- **Webhooks** verified locally (`QBitFlow-Signature`, secret rotation included), parsed into
  typed events and dispatched by a `WebhookRouter` that is an `http.Handler`.
- **Integration helpers**: `WaitForCompletion`, `HasAccess`, exact `FormatAmount` / `ParseAmount`,
  exports over any range, `NewClientFromEnv`.

> Coming from 2.x? Read [MIGRATION-v3.md](MIGRATION-v3.md): 3.0.0 targets API v2 and changes the
> module path, the client and most names.

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Integration recipes](#integration-recipes)
- [Authentication and acting for a member](#authentication-and-acting-for-a-member)
- [Checkout sessions](#checkout-sessions)
- [Products and customers](#products-and-customers)
- [Payments and failures](#payments-and-failures)
- [Subscriptions](#subscriptions)
- [Refunds](#refunds)
- [Marketplaces](#marketplaces)
- [Wallets](#wallets)
- [Accounting export](#accounting-export)
- [Webhooks](#webhooks)
- [Currencies](#currencies)
- [Pagination and iterators](#pagination-and-iterators)
- [Errors](#errors)
- [Retries and idempotency](#retries-and-idempotency)
- [Configuration](#configuration)
- [Migrating from 2.x](#migrating-from-2x)
- [Examples](#examples) · [Testing](#testing) · [License](#license) · [Support](#support) · [Security](#security)

## Installation

```bash
go get github.com/QBitFlow/qbitflow-go-sdk/v3
```

```go
import "github.com/QBitFlow/qbitflow-go-sdk/v3" // package qbitflow
```

**Requires Go 1.27 or later.** The SDK has no dependencies outside the standard library.

## Quick start

The snippets in this README run in a `main` (or a handler) with `ctx` a `context.Context`
(`context.Background()` in a `main`) and the `qbitflow` package imported; each comes from a
runnable program in [`examples/`](#examples). Create a client and check its key:

<!-- docs:snippet client-init -->
```go
// One client per API key, shared by the whole program: it is safe for concurrent use.
client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"))
if err != nil {
	log.Fatal(err) // a *qbitflow.ValidationError: no key, or not an sk_… key
}

// The recommended start-up check: what is this key, and which mode is it in?
me, err := client.Me(ctx)
if err != nil {
	log.Fatal(err) // a *qbitflow.AuthenticationError for an unknown or revoked key
}
if me.Space != nil {
	fmt.Printf("%s, role %s, test mode %v\n", me.Space.OrganizationName, me.Role, me.Space.Test)
}
```
<!-- /docs:snippet -->

Then open a hosted checkout for a one-time payment of 4.99 USD:

<!-- docs:snippet checkout-create-payment -->
```go
// An inline product: no product needed in the catalog. QBitFlow replaces {{UUID}}
// (qbitflow.PlaceholderUUID) with the session's id on redirect.
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductName: "T-shirt",
	Description: "Blue, size M",
	Price:       4.99,
	Reference:   "order-1042", // your order id: unique per space
	SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
	CancelURL:   "https://shop.example.com/orders/cancel",
})
if err != nil {
	log.Fatal(err)
}
// In an HTTP handler: http.Redirect(w, r, session.Link, http.StatusSeeOther)
fmt.Println("Redirect the customer to", session.Link)
```
<!-- /docs:snippet -->

Then fulfil the order when the [`payment.completed` webhook](#webhooks) arrives for
`session.UUID`, never on the customer's redirect to your success page.

### Conventions

- **Every method** takes a `context.Context` first and optional [request options](#configuration)
  last. Optional params are pointers: `nil` means "no filter" or "the defaults".
- **Ids are strings.** Resources are named by UUIDs; transactions by prefixed ids (`pay@…` a
  payment, `sub@…` a subscription, `sub-hist@…` a bill, `refund@…` a refund) that you pass back
  verbatim. Only currencies keep numeric ids (`uint64`).
- **Optional response fields:** an optional string is `""` when absent; every other optional field
  (a number, a bool, a time, a nested object) is a pointer, `nil` when absent. A value the API
  leaves out decodes to its zero value, never to an error.
- **Optional request fields** that are pointers are set inline with `new(value)` (Go 1.26+):
  `Immediate: new(false)`, `RefundPercent: new(50.0)`, `MinPeriods: new(uint32(3))`.
- **Amounts:** USD amounts are `float64` (`Amount`, `AmountUSD`); exact amounts in a token's
  smallest unit, and a few USD prices, are decimal strings (`AmountMinUnits`, `PriceUSD`,
  `Allowance`), never rounded. Fee rates are percents: `FeePercent: 1.5` is 1.5 %.
- **Enums** are named string types with constants (`qbitflow.SubscriptionStatusActive`). A value
  this SDK does not know yet is kept as is: give every `switch` a `default` branch.
- **Times** are `time.Time`, with the offset the API sent.

## Integration recipes

A typical integration, each step in a few lines: create a checkout, receive the webhook, grant
access. Every helper below is plain Go over the services, and behaves the same in the
JavaScript, Python and PHP SDKs.

### A webhook endpoint

`WebhookRouter` verifies each delivery's signature, parses the event, runs the handler of its
type with the typed data, and answers QBitFlow: `200`, `400` for a bad signature or body, `500`
when a handler returns an error (QBitFlow then delivers it again). It is an `http.Handler`, for
`net/http`, chi, gorilla/mux…:

<!-- docs:snippet webhook-handler -->
```go
var fulfilled sync.Map // stands for your database: deliveries are at least once
router := qbitflow.NewWebhookRouter(os.Getenv("QBITFLOW_WEBHOOK_SECRET"),
	qbitflow.WithWebhookErrorHandler(func(event *qbitflow.Event, err error) {
		log.Printf("webhook refused or failed: %v", err) // event is nil when the body could not be parsed
	}),
).
	OnPaymentCompleted(func(ctx context.Context, payment *qbitflow.PaymentCompleted, event *qbitflow.Event) error {
		if _, done := fulfilled.LoadOrStore(event.ID, true); done {
			return nil // a retry of an event already handled
		}
		log.Printf("fulfil order %q: %.2f USD (%s)", payment.Reference, payment.Amount, payment.UUID)
		return nil // an error answers 500, and QBitFlow delivers the event again
	}).
	OnCheckoutExpired(func(ctx context.Context, expired *qbitflow.CheckoutExpired, event *qbitflow.Event) error {
		if expired.Payment != nil {
			log.Printf("release order %q", expired.Payment.Reference)
		}
		return nil
	})

// The router is an http.Handler: it verifies the QBitFlow-Signature header on the raw body,
// parses the event, runs the handlers of its type and answers 200, 400 or 500.
http.Handle("POST /webhooks/qbitflow", router)
```
<!-- /docs:snippet -->

With another framework (gin, echo, fiber…), hand `Handle` the **raw** body and the
`QBitFlow-Signature` header, and answer what it returns:

```go
func answerWebhook(ctx context.Context, rawBody []byte, signature string) (status int, body []byte) {
	res := router.Handle(ctx, rawBody, signature)
	if res.Err != nil {
		log.Printf("webhook %d: %v", res.Status, res.Err)
	}
	return res.Status, res.ResponseBody() // {"received":true} or {"error":"…"}
}
```

### A checkout, its success page, and scripts

Create the checkout as in the [Quick start](#quick-start), and send the customer to its `Link`.
On the success page, `GetStatus(ctx, r.URL.Query().Get("uuid"))` tells the customer where their
payment stands; the order itself is fulfilled by the webhook. In a script, a test or a back-office
job, `WaitForCompletion` polls until the session is `completed` or `expired` (every 3 s, for at
most 10 minutes by default):

<!-- docs:snippet wait-for-completion -->
```go
// For scripts, tests and back-office jobs: orders are fulfilled on the webhook.
status, err := client.CheckoutSessions.WaitForCompletion(ctx, sessionUUID, &qbitflow.WaitOptions{
	Timeout:  2 * time.Minute, // default 10 minutes
	Interval: 5 * time.Second, // default 3 s
})
if err != nil {
	log.Fatal(err) // GetStatus's errors, or the context's
}
if status.Status != qbitflow.CheckoutSessionStatusCompleted {
	fmt.Println("not paid (yet):", status.Status) // expired, or still open at the timeout
}
```
<!-- /docs:snippet -->

### Access control

A subscription grants access until the end of the period paid for, whatever its status:

<!-- docs:snippet has-access -->
```go
// Access lasts until the end of the period paid for, whatever the status.
sub, err := client.Subscriptions.Get(ctx, subscriptionUUID, nil)
if err != nil {
	log.Fatal(err)
}
if sub.HasAccess(time.Now()) { // CurrentPeriodEnd set and now before it
	fmt.Println("access granted until", sub.CurrentPeriodEnd.Format(time.RFC3339))
} else {
	fmt.Println("no access")
}
```
<!-- /docs:snippet -->

### Displaying amounts

Exact token amounts are decimal strings in the token's smallest unit. Convert them exactly (no
float on the way):

<!-- docs:snippet amounts-display -->
```go
// Exact decimal arithmetic on strings, never floats. USDC has 6 decimals.
display, err := qbitflow.FormatAmount("4990000", 6)
if err != nil {
	log.Fatal(err)
}
fmt.Println(display, "USDC") // 4.99 USDC

minUnits, err := qbitflow.ParseAmount("4.99", 6) // more than 6 decimals is a *ValidationError
if err != nil {
	log.Fatal(err)
}
fmt.Println(minUnits) // 4990000

// From a payment: payment.Currency.FormatAmount(payment.AmountMinUnits)
```
<!-- /docs:snippet -->

`payment.Currency.FormatAmount(payment.AmountMinUnits)` does the same with the currency a payment
carries (its `Decimals`).

### A yearly accounting export

<!-- docs:snippet accounting-export -->
```go
// The API exports at most 95 days at a time: the …Range methods split any range into
// windows, request them in order and join the results (the CSV header once).
events, err := client.Accounting.ExportJSONRange(ctx, "2026-01-01", "2026-12-31")
if err != nil {
	log.Fatal(err)
}
for _, e := range events {
	fmt.Println(e.Type, e.PaymentUUID, e.TxTimeUTC.Format(time.DateOnly), e.ProductName)
}

csv, err := client.Accounting.ExportCSVRange(ctx, "2026-01-01", "2026-12-31")
if err != nil {
	log.Fatal(err)
}
if err := os.WriteFile("qbitflow-2026.csv", []byte(csv), 0o600); err != nil {
	log.Fatal(err)
}
```
<!-- /docs:snippet -->

### Configuration from the environment

<!-- docs:snippet config-from-env -->
```go
// QBITFLOW_API_KEY (required), QBITFLOW_BASE_URL and QBITFLOW_ON_BEHALF_OF (when set);
// the options given here override them.
client, err := qbitflow.NewClientFromEnv(qbitflow.WithTimeout(10 * time.Second))
if err != nil {
	log.Fatal(err) // a *qbitflow.ValidationError naming QBITFLOW_API_KEY when it is not set
}
```
<!-- /docs:snippet -->

## Authentication and acting for a member

Every request sends your API key in `X-API-Key`. Keys are created in the QBitFlow dashboard; each
belongs to one **space** (your organization's, or one of its members') and one **mode** (test or
live). `NewClient` only checks the key's shape (non-blank, starting with `sk_`) and sends nothing:
call `Me` to check it online, as the [Quick start](#quick-start) does. `me.Role` says what the
key is (`admin`: an organization key) and `me.Space.Test` its mode: a job that must only run in
test mode checks it.

Keep the key in a secret store or an environment variable, never in code. Keys issued before API v2
(`sk_<digits>_…`) still work; rotate them in the dashboard to the `sk_<uuid>_…` format.

### `On-Behalf-Of`: acting in a member's space

A marketplace's **organization key** can act in any of its members' spaces: create their products
and checkouts, read their payments. Name the member by their **user UUID** (`Member.UserUUID`,
also in the `member.joined` webhook). A non-member, an owner or admin of the team, or a member of
the other mode answers 404.

<!-- docs:snippet client-on-behalf-of -->
```go
// A client acting in the member's space (organization key): every request sends
// On-Behalf-Of. It shares the connections and settings of client.
seller := client.OnBehalfOf(memberUUID)
products, err := seller.Products.List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(products), "products in the seller's space")
```
<!-- /docs:snippet -->

For one request only, pass the request option; it wins over the client's (`seller` is the
client above):

```go
page, err := client.Payments.List(ctx, nil, qbitflow.WithOnBehalfOf(memberUUID))
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(page.Items), "payments of the seller")

// "" forces the organization's own space for one request of the seller's client.
own, err := seller.Products.List(ctx, nil, qbitflow.WithOnBehalfOf(""))
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(own), "products of the organization")
```

`qbitflow.WithOnBehalfOf(uuid)` is also a client option: `qbitflow.NewClient(key,
qbitflow.WithOnBehalfOf(uuid))`. A value that is not a UUID (or the nil UUID) is a
`*ValidationError`, and nothing is sent.

## Checkout sessions

A checkout session is a hosted payment page. Create it, send your customer to its `Link`, and act
on the webhook. The session's id (`pay@…` or `sub@…`) is also the id of the payment or the
subscription it creates once the customer's transaction is confirmed.

Name the product with **exactly one** of `ProductUUID`, `ProductReference`, or an inline product
(`ProductName` + `Price`, `Description` optional). The [Quick start](#quick-start) names an
inline product; a catalog product is named by its reference (or its UUID):

<!-- docs:snippet checkout-create-payment-product -->
```go
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductReference: "tshirt-blue-m", // or ProductUUID
	SuccessURL:       "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
	CancelURL:        "https://shop.example.com/orders/cancel",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("Redirect the customer to", session.Link)
```
<!-- /docs:snippet -->

A subscription checkout takes the same fields plus its terms, each optional over a subscription
product's (`MinPeriods: new(uint32(3))` also commits the customer to 3 periods):

<!-- docs:snippet checkout-create-subscription -->
```go
// The Subscription exists, with the session's id (sub@…), once the customer signed: act on
// the subscription.created webhook.
session, err := client.CheckoutSessions.CreateSubscription(ctx, &qbitflow.CreateSubscriptionSessionParams{
	ProductName: "T-shirt",
	Description: "Blue, size M",
	Price:       4.99, // USD per period
	Frequency:   &qbitflow.Duration{Value: 1, Unit: qbitflow.DurationUnitMonths},
	TrialPeriod: &qbitflow.Duration{Value: 7, Unit: qbitflow.DurationUnitDays},
	Reference:   "order-1043",
	SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
	CancelURL:   "https://shop.example.com/orders/cancel",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("Redirect the customer to", session.Link)
```
<!-- /docs:snippet -->

- **Other fields:** `Reference` is your order id, unique per space; `CustomerUUID` or
  `CustomerReference` (kept on the payment) names your customer; `ExpiresInMinutes` sets the
  session's lifetime (10 to 1440; 0 = the default).

- **Redirect placeholders.** In `SuccessURL` and `CancelURL`, QBitFlow replaces `{{UUID}}`
  (`qbitflow.PlaceholderUUID`) with the session's id and `{{TRANSACTION_TYPE}}`
  (`qbitflow.PlaceholderTransactionType`) with `payment` or `createSubscription`; the SDK sends
  them as they are, never URL-encoded. In live mode both
  URLs must be `https`. A redirect proves nothing (anyone can open the URL): fulfil on the
  webhook, or on `GetStatus`.
- **Errors to expect:** `409 merchant_not_ready` (`Details["reason"]`) when the space's wallets
  accept no currency; `409 unique_violation` when another payment or open session holds the
  `Reference`; `400 validation_failed` above 5 USD in test mode (`Details["max"]`); `404` for an
  unknown product or customer.

### Status

<!-- docs:snippet checkout-status -->
```go
status, err := client.CheckoutSessions.GetStatus(ctx, sessionUUID)
if err != nil {
	log.Fatal(err)
}
switch status.Status {
case qbitflow.CheckoutSessionStatusCompleted:
	fmt.Println("paid, tx", status.TxHash)
case qbitflow.CheckoutSessionStatusExpired:
	fmt.Println("expired unpaid:", status.Message)
case qbitflow.CheckoutSessionStatusWaitingConfirmation:
	fmt.Println("sent, waiting for the network")
default: // created, or a status this SDK does not know
	if status.LastAttempt != nil {
		// Not final: the customer may still pay from the same checkout.
		fmt.Println("last attempt failed:", status.LastAttempt.Code)
	}
}
```
<!-- /docs:snippet -->

| `Status` | Meaning | Final |
|---|---|---|
| `created` | Waiting for the customer. With `LastAttempt` set, their last attempt failed (`LastAttempt.Code` says why) | no |
| `waitingConfirmation` | A transaction was sent; waiting for the network. It may last: the transaction can still land | no |
| `completed` | Confirmed and recorded: the `Payment` or `Subscription` exists, with the session's id | yes |
| `expired` | Expired unpaid (`checkout.expired` was sent). Read some days later, an expired session is a 404 | yes |

**Never cancel an order on `LastAttempt`:** a failed attempt is not final, and the customer can pay
from the same checkout until it expires. Release what the order holds on `checkout.expired`.

`WaitForCompletion(ctx, uuid, &qbitflow.WaitOptions{Timeout, Interval})` polls `GetStatus` until the
session is final and returns that status, or the last one seen when the timeout (default 10
minutes) elapses first; the interval (default 3 s) is at least 1 s. It is meant for scripts, tests
and back-office jobs ([recipe](#a-checkout-its-success-page-and-scripts)): orders are fulfilled on
the webhook.

### Expire

End a session early (an order cancelled on your side). It answers its status, and
`checkout.expired` follows. Once the customer paid or is paying it is a `409 tx_already_sent`.

<!-- docs:snippet checkout-expire -->
```go
// Once the customer paid or is paying, this is a 409 tx_already_sent.
expired, err := client.CheckoutSessions.Expire(ctx, sessionUUID)
if err != nil {
	log.Fatal(err)
}
fmt.Println(expired.Status) // expired; checkout.expired follows
```
<!-- /docs:snippet -->

## Products and customers

Products are optional (a checkout can name an inline product) and give you a reusable catalog
with payment links. The reference is generated when `""`. A subscription product carries its
terms: `Subscription: &qbitflow.SubscriptionTermsParams{Frequency: &qbitflow.Duration{Value: 1,
Unit: qbitflow.DurationUnitMonths}}`.

<!-- docs:snippet products-create -->
```go
product, err := client.Products.Create(ctx, &qbitflow.CreateProductParams{
	Name:        "T-shirt",
	Description: "Blue, size M",
	Price:       4.99,
	Reference:   "tshirt-blue-m", // your own id: unique per space
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("created", product.UUID, "payment link:", product.PaymentLink)
```
<!-- /docs:snippet -->

<!-- docs:snippet products-list -->
```go
products, err := client.Products.List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
for _, p := range products {
	fmt.Printf("%s %q: %.2f USD (%s)\n", p.UUID, p.Name, p.Price, p.Reference)
}
```
<!-- /docs:snippet -->

`&qbitflow.ProductListParams{IncludeHidden: true, Subscription: new(true)}` lists the hidden
products too, and only the subscription ones. `Products.Update` changes only the fields set; a new
price applies to new checkouts and subscribers only:

```go
product, err := client.Products.Update(ctx, productUUID, &qbitflow.UpdateProductParams{
	Price:    new(5.99),
	IsActive: new(false), // hidden from Products.List
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(product.Price)
```

`Products.Get`, `GetByReference` and `Delete` complete the set. Deleting a product does not stop
its subscriptions: cancel them with `Subscriptions.Cancel` if the product is gone for good.

```go
customer, err := client.Customers.Create(ctx, &qbitflow.CreateCustomerParams{
	Name:      "Ada",
	LastName:  "Lovelace",
	Email:     "ada@example.com",
	Reference: "crm-42",
})
if err != nil {
	log.Fatal(err)
}

// A pointer to "" clears PhoneNumber or Address; nil leaves them unchanged.
customer, err = client.Customers.Update(ctx, customer.UUID, &qbitflow.UpdateCustomerParams{PhoneNumber: new("")})
if err != nil {
	log.Fatal(err)
}

byEmail, err := client.Customers.GetByEmail(ctx, "ada@example.com")
if err != nil {
	log.Fatal(err)
}
fmt.Println(customer.UUID == byEmail.UUID)
```

`Customers.Get`, `GetByReference`, `List` / `Iterate` (by `Email` or `Verified`) and `Delete`
complete the set. A checkout given `CustomerUUID` or `CustomerReference` asks the customer
nothing (`CustomerReference` is kept on the payment, and links your customer with that reference
if there is one); without either, the checkout asks what your `checkout.customerDetails` setting
says (the full details by default).

## Payments and failures

A `Payment` exists once its transaction is confirmed, with its checkout session's `pay@…` id.

<!-- docs:snippet payments-list -->
```go
since := time.Now().AddDate(0, -1, 0)
page, err := client.Payments.List(ctx, &qbitflow.PaymentListParams{
	CreatedAfter: &since, // the last month
	Limit:        20,
})
if err != nil {
	log.Fatal(err)
}
for _, p := range page.Items {
	fmt.Printf("%s %q: %.2f USD\n", p.UUID, p.Reference, p.Amount)
}
if page.NextCursor != nil {
	// Pass it back as PaymentListParams.Cursor for the next page.
	fmt.Println("next cursor:", *page.NextCursor)
}
```
<!-- /docs:snippet -->

<!-- docs:snippet payments-get -->
```go
payment, err := client.Payments.Get(ctx, paymentUUID, nil) // pay@…
if err != nil {
	log.Fatal(err)
}
fmt.Println(payment.UUID, payment.Reference, payment.Amount, "USD")

byReference, err := client.Payments.GetByReference(ctx, "order-1042") // your order id
if err != nil {
	log.Fatal(err) // a *qbitflow.NotFoundError when no payment has this reference
}
fmt.Println("order-1042 was paid by", byReference.UUID)
```
<!-- /docs:snippet -->

A payment carries its transaction (`TxHash`, `ExplorerURL`), what each party got
(`Metadata.TxAmounts`), and whether it can be refunded (`Refundable`).

- **Filters** (`PaymentListParams`): `CustomerUUID`, `ProductUUID`, `CreatedAfter` / `CreatedBefore`
  (both excluded), `Refunded`, and, with an organization key acting for itself, `IncludeMembers`
  (every member's rows, each naming its `UserUUID`) or `UserUUID` (one member's). The last two
  exclude each other.
- **Reading a member's row** from the organization's space: `Payments.Get(ctx, id,
  &qbitflow.ReadParams{IncludeMembers: true})`, or `OnBehalfOf` the member.
- **The combined feed** of one-time payments and subscription bills, newest first:
  `Payments.ListCombined` / `IterateCombined`, with the same filters plus `Source`
  (`CombinedPaymentSourcePayment`, `CombinedPaymentSourceSubscriptionHistory`) and
  `SubscriptionUUID`.
- **Failures** are the failed attempts to pay a checkout or a bill. They never moved money and
  are not final: the customer can try again.

```go
for f, err := range client.Failures.Iterate(ctx, &qbitflow.FailureListParams{
	Category: qbitflow.FailureCategoryInsufficientBalance,
}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s attempt %d: %s (%.2f USD)\n", f.TxUUID, f.Attempt, f.Code, f.AttemptedUSD)
}
```

## Subscriptions

A subscription exists once its customer signed its checkout: paid the first period, or started
the free trial. It keeps the checkout's `sub@…` id for life.

| `Status` | Meaning |
|---|---|
| `trial` | In its free trial |
| `trialExpired` | The trial ended without the customer confirming it (`ActionRequired` `confirmTrial`); cancelled 7 days later unless confirmed |
| `active` | Billed on its due dates |
| `pastDue` | A bill failed; retried for up to 7 days (`Dunning` on reads), then cancelled |
| `paused` | Paused by its customer: not billed until resumed |
| `stopped` | Cancelled during its period: never billed again, `cancelled` at `NextBillingDate` |
| `cancelled` | Over (`CancellationReason` says why). Final |

**Access rule: grant access while `now < CurrentPeriodEnd`, whatever the status.** A `stopped` or
`paused` subscription has paid for its period; a `pastDue` one's period has ended.
`sub.HasAccess(time.Now())` applies it ([recipe](#access-control)).

`ActionRequired` says what the customer must do (`topUpAllowance`, `raiseMaximum`,
`confirmTrial`; `""`: nothing): point them to the `ManagementPageLink` the subscription webhooks
carry.

<!-- docs:snippet subscriptions-get -->
```go
sub, err := client.Subscriptions.Get(ctx, subscriptionUUID, nil) // sub@…, cancelled ones too
if err != nil {
	log.Fatal(err)
}
fmt.Println(sub.UUID, sub.Status) // trial, active, pastDue, stopped, cancelled…
if sub.CurrentPeriodEnd != nil {
	fmt.Println("paid until", sub.CurrentPeriodEnd.Format(time.RFC3339))
}
```
<!-- /docs:snippet -->

```go
page, err := client.Subscriptions.List(ctx, &qbitflow.SubscriptionListParams{
	Status: qbitflow.SubscriptionStatusPastDue,
})
if err != nil {
	log.Fatal(err)
}
for _, sub := range page.Items {
	if sub.Dunning != nil {
		fmt.Println(sub.UUID, sub.Dunning.RemainingAttempts, "attempts left")
	}
}

// Every bill, newest first: the iterator fetches the pages lazily.
for bill, err := range client.Subscriptions.IterateBills(ctx, subscriptionUUID, nil) {
	if err != nil {
		log.Fatal(err)
	}
	if bill.PeriodEnd != nil {
		fmt.Printf("%s: %.2f USD, paid until %s\n", bill.UUID, bill.Amount, bill.PeriodEnd.Format(time.DateOnly))
	}
}
```

- `Subscriptions.GetByReference` reads one by your checkout's `Reference`; `GetBill` one bill
  (`sub-hist@…`).
- `Subscriptions.GetPublicHistory` returns the 10 latest bills as the customer's page shows them.
  It is a public route: the fields only the merchant sees (`Metadata`, `CustomerUUID`,
  `CustomerReference`, `UserUUID`, `PaidMinUnits`, `Refund`, …) are empty there. Use `ListBills`
  for full bills.
- Before its checkout completes, `Subscriptions.Get` is a 404: read the checkout session instead.

### Cancel

`Cancel` cancels without the customer signing. By default it is immediate (`cancelled`, reason
`merchant`); `Immediate: new(false)` stops it now and cancels it at the end of the period paid for.

<!-- docs:snippet subscriptions-cancel -->
```go
// Immediate false: never billed again, cancelled when the paid period ends.
res, err := client.Subscriptions.Cancel(ctx, subscriptionUUID, &qbitflow.CancelSubscriptionParams{
	Immediate: new(false),
})
if err != nil {
	log.Fatal(err) // 409 subscription_already_stopped_or_inactive
}
if res.Pending {
	// HTTP 202: the on-chain cancellation is still confirming; subscription.statusChanged
	// tells the end.
	fmt.Println("cancellation confirming")
} else {
	fmt.Println("now", res.Subscription.Status) // stopped
}
```
<!-- /docs:snippet -->

`Cancel` answers 200 when done and 202 (`Pending`) while confirming on-chain. It is never retried
automatically.

### Test billing

In test mode a subscription is billed only when you ask, with live's statuses and webhooks:

<!-- docs:snippet subscriptions-test-bill -->
```go
// Test mode only: runs the next billing now, with live's statuses and webhooks.
state, err := client.Subscriptions.ExecuteTestBilling(ctx, subscriptionUUID)
if err != nil {
	log.Fatal(err) // 409 payment_not_due before NextBillingDate; 400 for a live subscription
}
fmt.Println(state.Stage, state.Outcome, state.FailureCode)
```
<!-- /docs:snippet -->

Walk the timeline once in test mode: a 5-minute frequency, pay from a test wallet, trigger the
bill, then empty the wallet and trigger it again to see `subscription.billingFailed` and
`pastDue`.

## Refunds

<!-- docs:snippet refunds-list -->
```go
refunds, err := client.Refunds.List(ctx, nil) // the refunds waiting for an answer
if err != nil {
	log.Fatal(err)
}
for _, r := range refunds {
	fmt.Println(r.UUID, r.TxUUID, r.Status, r.Reason, r.AmountUSD)
}
```
<!-- /docs:snippet -->

From the organization's space the members' refunds are included by default:
`&qbitflow.RefundListParams{IncludeMembers: new(false)}` leaves them out. The answered refunds
(approved or rejected) come page by page:

```go
for r, err := range client.Refunds.IterateInactive(ctx, nil) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r.UUID, r.Status, r.ExplorerURL)
}
```

`Refunds.Initiate` starts a refund of a payment (`pay@…`) or a bill (`sub-hist@…`) of the space.
**It creates a pending refund: no money moves until you sign the transfer in the dashboard**,
from the wallet that was paid. `refund.completed` tells you when it is sent.

<!-- docs:snippet refunds-create -->
```go
refund, err := client.Refunds.Initiate(ctx, &qbitflow.InitiateRefundParams{
	TxUUID:        paymentUUID, // pay@… (or a bill's sub-hist@…)
	RefundPercent: new(50.0),   // of what the customer paid; nil = 100
	Reason:        "Damaged item",
})
if err != nil {
	log.Fatal(err) // 409 refund_already_exists: one refund per transaction
}
// Pending: no money moves until you sign the transfer in the dashboard (refund.completed).
fmt.Println(refund.UUID, refund.Status)
```
<!-- /docs:snippet -->

`RefundPercent` is a percent of everything the customer paid, network fee included;
`MerchantMessage` is shown to the customer. There is one refund per transaction: a second one is a
`409 refund_already_exists`, naming the first in `Details["refundUuid"]`.

A refund is `pending`, `approved` or `rejected`; `InitiatedBy` is `customer` (a request you answer
in the dashboard) or `merchant`. A held seller's payment already released to them can no longer be
refunded (`409 held_funds_released`).

## Marketplaces

A marketplace is an organization whose sellers are its **members**. You invite them, sell for
them with your organization key and `On-Behalf-Of`, take a commission on their payments, and may
hold their funds until you trust them. QBitFlow stays non-custodial: money goes from the customer's
wallet to the seller's (and your commission to yours) in one transaction.

**1. Invite the seller.** The SDK always invites members (`role: user`); the team is invited from
the dashboard.

<!-- docs:snippet members-invite -->
```go
created, err := client.Invitations.Create(ctx, &qbitflow.CreateInvitationParams{
	Email:                  "seller@example.com",
	TrustLayer:             true,      // hold their payments until Members.Trust
	OrganizationFeePercent: new(10.0), // your commission on their payments: 0 to 50 %
	RedirectURL:            "https://shop.example.com/sellers/welcome",
})
if err != nil {
	log.Fatal(err) // 409 already_joined for a member
}
// The link is also emailed. The seller exists once they accepted: wait for member.joined.
fmt.Println("invitation", created.Invitation.UUID, "link:", created.Link)
```
<!-- /docs:snippet -->

The fee has at most 2 decimals; an organization sends at most 50 invitations an hour (`429`).

**2. Wait for `member.joined`.** The seller exists once they accepted: store the event's
`UserUUID` and match `InvitationUUID` to your invitation. Never trust the redirect's
`?invitationUuid=` (anyone can open it).

```go
joined, err := event.MemberJoined()
if err != nil {
	return err
}
fmt.Println("invitation", joined.InvitationUUID, "accepted by", joined.UserUUID)
```

**3. Sell for them.** The seller adds their receiving wallet in their QBitFlow dashboard (only
they can, not needed while you hold their funds). Then act in their space:

```go
currencies, err := client.Wallets.ListSupportedCurrencies(ctx, &qbitflow.SupportedCurrenciesParams{UserUUID: memberUUID})
if err != nil {
	log.Fatal(err)
}
if len(currencies) == 0 {
	log.Fatal("the seller cannot be paid yet: their checkouts would answer 409 merchant_not_ready")
}

seller := client.OnBehalfOf(memberUUID)
session, err := seller.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductName: "Handmade mug",
	Price:       4.5,
	SuccessURL:  "https://market.example.com/orders/{{UUID}}",
})
if err != nil {
	log.Fatal(err) // 403 policy_disabled when your policies don't let members do this
}
fmt.Println(session.Link)
```

**4. Hear of every sale.** An organization webhook endpoint receives every seller's events by
default; the event's `UserUUID` names the seller. Use it as `OnBehalfOf` for follow-up reads.

**5. Commission and held funds.** Each payment records your fee in `Metadata.OrganizationFee` and
`Metadata.TxAmounts`. While you hold a seller's funds (`TrustLayer` true, `Member.TrustedAt`
nil), their payments go to your wallet and the net is owed to them:

<!-- docs:snippet members-held-funds -->
```go
// What the organization holds for its members (their payments while untrusted).
summaries, err := client.Members.ListHeldFunds(ctx)
if err != nil {
	log.Fatal(err)
}
for _, s := range summaries {
	fmt.Printf("%s: %.2f USD over %d lines\n", s.UserUUID, s.TotalAmount, s.Count)
}

held, err := client.Members.GetHeldFunds(ctx, memberUUID)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("owed to %s: %.2f USD over %d lines\n", memberUUID, held.TotalAmount, len(held.Ledgers))
```
<!-- /docs:snippet -->

Trust a seller to pay them directly:

<!-- docs:snippet members-trust -->
```go
// Their new payments go to their own wallets from now on. What is held stays held until
// you release it in the dashboard (heldFunds.released tells you).
member, err := client.Members.Trust(ctx, memberUUID)
if err != nil {
	log.Fatal(err)
}
if member.TrustedAt != nil {
	fmt.Println("trusted since", member.TrustedAt.Format(time.RFC3339))
}
```
<!-- /docs:snippet -->

Change the commission:

<!-- docs:snippet members-update -->
```go
// The organization's commission on their payments, in percent (a checkout already
// created keeps its fee).
member, err := client.Members.Update(ctx, memberUUID, &qbitflow.UpdateMemberParams{
	OrganizationFeePercent: 10,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("fee now", member.OrganizationFeePercent, "%")
```
<!-- /docs:snippet -->

- `Members.List` / `Iterate` / `Get`, `Members.ListHeldFunds` (every member owed), and
  `seller.Members.GetOwnHeldFunds` (the seller's side) complete the reads.
- `Members.Remove` ends a seller's membership in the key's mode: their keys stop working and their
  checkouts close. It is a `409 held_funds_pending` while you hold their live funds: release them
  first.
- `Invitations.List` / `Iterate` (by `Status`) and `Invitations.Revoke` manage the invitations.
- Today a seller whose account already belongs to another organization cannot accept (no
  `member.joined` comes), and test-mode sellers are real accounts that accept from a real inbox.

## Wallets

Wallets are added and removed in the dashboard, by their owner only. The SDK reads them.

```go
wallets, err := client.Wallets.List(ctx, &qbitflow.WalletListParams{WithBalances: true})
if err != nil {
	log.Fatal(err)
}
for _, w := range wallets {
	fmt.Println(w.Currency.Symbol, w.PublicKey)
	for _, tw := range w.TokenWallets {
		if tw.Balance != nil {
			fmt.Printf("  %s: %s (%.2f USD)\n", tw.Token.Symbol, tw.Balance.Balance, tw.Balance.BalanceUSD)
		}
	}
}
```

`Wallets.ListForMember(ctx, userUUID)` reads a member's wallets (organization key), and
`Wallets.ListSupportedCurrencies` the currencies a space's checkouts accept: none means its
checkouts answer `409 merchant_not_ready`.

## Accounting export

Every payment, bill, refund and fee between two dates (`YYYY-MM-DD`, both included), as JSON rows
(`ExportJSON(ctx, from, to)`, `[]AccountingEvent`) or as CSV text (`ExportCSV`); the
[recipe](#a-yearly-accounting-export) exports a year.

The SDK checks the dates and `from <= to` before sending. **The API allows at most 95 days per
export** and answers 400 beyond. `ExportJSONRange` and `ExportCSVRange` take any range: they split
it into consecutive windows of at most 95 days (`[from, from+95d]`, the next one starting the day
after, on calendar dates), request them in order and return the events joined (for CSV, the first
window's header line once, then every window's rows). A range of 95 days or less is one request. Rows are typed `payment`,
`subscriptionHistory`, `refund`, `organizationFee` or `referralFee`; amounts in a token's smallest
unit are decimal strings, and the empty fields of a row are left out.

## Webhooks

QBitFlow posts an **event** to your endpoints when something happens: a payment confirmed, a
subscription billed, a member joined.

### 1. Create an endpoint and store its secret

<!-- docs:snippet webhook-endpoint-create -->
```go
created, err := client.Webhooks.Endpoints.Create(ctx, &qbitflow.CreateWebhookEndpointParams{
	URL: "https://shop.example.com/webhooks/qbitflow",
	Events: []qbitflow.EventType{ // nil: every type, including the ones added later
		qbitflow.EventTypePaymentCompleted,
		qbitflow.EventTypeCheckoutExpired,
		qbitflow.EventTypeSubscriptionStatusChanged,
	},
})
if err != nil {
	log.Fatal(err)
}
// The whsec_… secret is returned only this once: put it in your secret store now, and give
// it to your webhook handler as QBITFLOW_WEBHOOK_SECRET.
fmt.Println("endpoint", created.UUID, "secret", created.Secret)
```
<!-- /docs:snippet -->

`Description` labels the endpoint. Up to 10 endpoints per space and mode; live endpoints need `https` and a public host. An
organization endpoint also receives its members' events unless created with
`IncludeMembers: new(false)`. `Endpoints.List`, `Get`, `Update` (`Enabled: new(false)` pauses it,
`new(true)` enables it again) and `Delete` manage them. An endpoint's secret is shown and rotated in
the dashboard only.

### 2. Handle the deliveries with the router

A `WebhookRouter` holds the endpoint's secret and one handler per event type (the
[recipe](#a-webhook-endpoint) mounts one). For each delivery it verifies the `QBitFlow-Signature`
header over the **raw body**, parses the event, runs the handlers of its type with the typed data,
then the `OnAny` ones, and answers:

| Delivery | Answer | Handlers |
|---|---|---|
| Verified, of a type with handlers | `200 {"received":true}` | its type's (in registration order), then `OnAny` |
| Verified, of a type without handlers | `200` | `OnAny` (and `OnUnknown` first for a type this SDK does not know) |
| Bad, missing or stale signature | `400 {"error":"invalid signature"}` (`Err`: `*WebhookSignatureError`) | none |
| Not a v2 event, or data that does not fit its type | `400 {"error":"invalid event"}` (`Err`: `*ValidationError`) | none |
| A handler returned an error | `500 {"error":"internal error"}`: QBitFlow delivers it again | the ones after it are skipped |
| Not a `POST` / body over 1 MiB (`ServeHTTP`) | `405` / `413 {"error":"body too large"}` | none |

Handlers get the request's context. Return an error only for what a retry can fix, and keep the
handlers idempotent: a retried event runs its handlers again. Register them before serving; the
router is then safe for concurrent use. `OnUnknown` handlers get the types newer than this SDK
(`e.DecodeData[T]()` reads their data). `NewWebhookRouter` takes `WithTolerance` and `WithClock`
too.

**Test your handlers** with `SignWebhook`, which signs a body as QBitFlow does:

```go
func TestWebhook(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"webhook.test","version":"v2","createdAt":"2026-10-01T12:00:00Z","test":true,"data":{"endpointUuid":"e","message":"hi"}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/qbitflow", bytes.NewReader(body))
	req.Header.Set(qbitflow.SignatureHeader, qbitflow.SignWebhook(body, "whsec_test", time.Now()))
	rec := httptest.NewRecorder()
	router := qbitflow.NewWebhookRouter("whsec_test") // with your handlers registered
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}
```

### Lower level: verify and parse yourself

The router is built on these functions, usable on their own. `ConstructEvent` verifies and parses
a raw body:

<!-- docs:snippet webhook-verify -->
```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// The raw body, exactly as received: the signature covers its bytes.
	rawBody, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	event, err := qbitflow.ConstructEvent(rawBody, r.Header.Get(qbitflow.SignatureHeader), os.Getenv("QBITFLOW_WEBHOOK_SECRET"))
	var sigErr *qbitflow.WebhookSignatureError
	if errors.As(err, &sigErr) {
		log.Printf("rejected: %s", sigErr.Reason) // e.g. noMatchingSignature, timestampOutsideTolerance
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest) // not a v2 event
		return
	}

	if event.Type == qbitflow.EventTypePaymentCompleted {
		payment, err := event.PaymentCompleted()
		if err != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		// Deliveries are at least once: fulfil the order once per event.ID.
		log.Printf("event %s: order %q paid", event.ID, payment.Reference)
	}
	w.WriteHeader(http.StatusOK) // to the types you ignore too
}
```
<!-- /docs:snippet -->

`VerifyWebhook` only verifies; `ParseEvent` parses a body already verified. In a `net/http`
handler, `body, err := qbitflow.VerifyWebhookRequest(r, secret)` reads the request's body (at most
1 MiB) and checks its signature in one call, then `qbitflow.ParseEvent(body)` parses it.

- **At least once.** The same event can arrive more than once: **deduplicate on `Event.ID`** (also
  in the `QBitFlow-Event-Id` header) and make the handler idempotent.
- **Answer 2xx fast**, within 30 seconds, **including to the types you ignore**: anything else is
  retried (for 3 days in live mode), and an endpoint failing for 3 days is disabled. Store the
  event, answer, then process it in the background.
- **Secret rotation** needs nothing on your side: for 24 hours after a rotation the header carries
  two `v1=` signatures, the new secret's and the previous one's, and either secret verifies. Switch
  your secret within the day.
- **Timestamps** more than 5 minutes from your clock are refused (replays):
  `qbitflow.WithTolerance(d)` changes it, `qbitflow.WithClock(f)` sets the clock in tests.
- **Members' events** carry the member in `Event.UserUUID`: read their resources with
  `client.OnBehalfOf(event.UserUUID)`.
- **No fixed source IPs**: verify the signature, never allow-list addresses.
- **v2 only.** An endpoint migrated from API v1 receives v1 bodies, which `ParseEvent` refuses:
  move it to v2 in the dashboard, or with `Endpoints.Update(ctx, uuid,
  &qbitflow.UpdateWebhookEndpointParams{PayloadVersion: qbitflow.WebhookPayloadVersionV2})`.
- Not holding the secret? `client.Webhooks.VerifyRemote(ctx, endpointUUID, rawBody, header)` has the
  API check it, then `ParseEvent` parses the body.

| `Event.Type` | Accessor | Data |
|---|---|---|
| `payment.completed` | `PaymentCompleted()` | the `Payment` + `ManagementPageLink` |
| `subscription.created` | `SubscriptionCreated()` | the `Subscription` (`active` or `trial`) + `ManagementPageLink` |
| `subscription.billed` | `SubscriptionBilled()` | the `Bill` (paid until `PeriodEnd`) + `SubscriptionReference`, `SubscriptionStatus` |
| `subscription.statusChanged` | `SubscriptionStatusChanged()` | the `Subscription` + `PreviousStatus` |
| `subscription.actionRequiredChanged` | `SubscriptionActionRequiredChanged()` | the `Subscription` + `PreviousActionRequired` |
| `subscription.billingFailed` | `SubscriptionBillingFailed()` | the `Subscription` + `Reason`, `BillUUID`, `AmountUSD` (a string), `Attempt`, `RemainingAttempts`, `NextAttemptAt` |
| `subscription.upcomingBill` | `SubscriptionUpcomingBill()` | the `Subscription` + `BillingDate`, `AmountUSD` (a number), `TrialEnding`, `BalanceSufficient`, `AllowanceSufficient` |
| `refund.requested` / `.completed` / `.denied` | `RefundRequested()` / `RefundCompleted()` / `RefundDenied()` | the `Refund` |
| `member.joined` | `MemberJoined()` | the `Member` + `InvitationUUID` |
| `member.removed` | `MemberRemoved()` | the `Member` |
| `heldFunds.released` | `HeldFundsReleased()` | the transfer paid to the member + the `Ledgers` it settled |
| `checkout.expired` | `CheckoutExpired()` | the session: `.Payment` or `.Subscription` |
| `webhook.test` | `WebhookTest()` | `EndpointUUID`, `Message` (the dashboard's test) |
| any other | `DecodeData[T]()` | `Data`, raw |

Each accessor has its router method, `On` + its name (`OnPaymentCompleted`, `OnRefundDenied`…;
`OnUnknown` for the other types), whose handler gets the same data. An accessor called on another
type is a `*ValidationError`. Webhook data never carries what only
API reads return (`Customer`, `Refund`/`Refundable`, `Dunning`, `Approval`, `ProductName`).

### The event log

Every event of the space, newest first (`Webhooks.Events.Iterate` walks every page):

<!-- docs:snippet events-list -->
```go
page, err := client.Webhooks.Events.List(ctx, &qbitflow.EventListParams{
	Type:  qbitflow.EventTypePaymentCompleted,
	Limit: 20,
})
if err != nil {
	log.Fatal(err)
}
for _, e := range page.Items { // newest first
	fmt.Println(e.ID, e.Type, e.CreatedAt.Format(time.RFC3339))
}
```
<!-- /docs:snippet -->

And one event with its deliveries:

```go
detail, err := client.Webhooks.Events.Get(ctx, eventID) // evt_…
if err != nil {
	log.Fatal(err)
}
for _, d := range detail.Deliveries {
	fmt.Println(d.URL, d.Delivered, len(d.Attempts))
}
```

## Currencies

<!-- docs:snippet currencies-list -->
```go
// A public route limited to 60 requests a minute per IP: cache the list at start-up.
currencies, err := client.Currencies.ListAvailable(ctx, nil)
if err != nil {
	log.Fatal(err)
}
for _, c := range currencies {
	fmt.Printf("%d: %s (%s), %d decimals\n", c.ID, c.Symbol, c.Name, c.Decimals)
}
```
<!-- /docs:snippet -->

`ListAvailable` lists every currency checkouts can take (`&qbitflow.CurrencyListParams{Test: true}`:
the testnet ones), `ListMain` the chains' native coins, and
`Get` one by id, to resolve the `CurrencyID`, `AvailableCurrencyIDs` and `AcceptedCurrencyIDs`
fields. These routes are public and **limited to 60 requests a minute per IP: cache the list** at
start-up instead of reading it per request. Payments, bills and subscriptions already carry their
`Currency`.

## Pagination and iterators

Paginated lists return a `*Page[T]`: `Items`, and `NextCursor` (nil on the last page, else the
value to pass back as the params' `Cursor`, verbatim).

<!-- docs:snippet customers-list -->
```go
page, err := client.Customers.List(ctx, &qbitflow.CustomerListParams{Limit: 20})
if err != nil {
	log.Fatal(err)
}
for _, c := range page.Items {
	fmt.Println(c.UUID, c.Email)
}
if page.NextCursor != nil {
	// Pass it back as CustomerListParams.Cursor for the next page.
	fmt.Println("next cursor:", *page.NextCursor)
}
```
<!-- /docs:snippet -->

To walk the pages yourself, set `params.Cursor = *page.NextCursor` and list again while
`page.HasMore()`. Each paginated list also has an `Iterate` twin returning an
`iter.Seq2[T, error]`: it fetches one page at a time, only as you consume it, keeps your filters
and page size, and stops when you `break`. An error is yielded once, as `(zero, err)`, and ends
the sequence.

<!-- docs:snippet pagination-iterate -->
```go
// Fetches one page at a time, only as the loop consumes it; break stops the fetching.
var total float64
for payment, err := range client.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
	if err != nil {
		log.Fatal(err)
	}
	total += payment.Amount
}
fmt.Printf("%.2f USD received in total\n", total)
```
<!-- /docs:snippet -->

| Method | Iterator | Page size: default / max |
|---|---|---|
| `Customers.List` | `Iterate` | 10 / 100 |
| `Payments.List`, `Payments.ListCombined` | `Iterate`, `IterateCombined` | 10 / 50 |
| `Failures.List` | `Iterate` | 10 / 50 |
| `Subscriptions.List`, `Subscriptions.ListBills` | `Iterate`, `IterateBills` | 20 / 100 |
| `Refunds.ListInactive` | `IterateInactive` | 10 / 50 |
| `Members.List`, `Invitations.List` | `Iterate` | 20 / 100 |
| `Webhooks.Events.List` (cursor `evt_…`) | `Iterate` | 20 / 100 |

The other lists are short and return a plain slice: `Products.List`, `Refunds.List`,
`Wallets.*`, `Currencies.*`, `Webhooks.Endpoints.List` (at most 10), `Members.ListHeldFunds` and
`Subscriptions.GetPublicHistory` (the 10 latest bills).

## Errors

Every error the SDK returns is one of the types below, matched with `errors.As`. Each embeds an
`*APIError`, so an `*APIError` target matches any of them:

| Type | When | Codes and fields worth knowing |
|---|---|---|
| `*ValidationError` | 400 `validation_failed`, or input the SDK refused **before sending** (`StatusCode` 0) | `FieldErrors`: each failing input by its wire name, dotted when nested (`frequency.unit`) |
| `*BadRequestError` | any other 400 | `bad_request`, `foreign_key_violation` (`Details["field"]`) |
| `*AuthenticationError` | 401 | a missing, unknown, expired or revoked key (a removed member's keys too) |
| `*PermissionDeniedError` | 403 | `forbidden`, `policy_disabled` (`Details["policy"]`), `plan_required` |
| `*NotFoundError` | 404 | unknown, deleted, or outside the request's space |
| `*ConflictError` | 409 | `unique_violation` (`Details["field"]`), `tx_already_sent`, `merchant_not_ready` (`Details["reason"]`), `refund_already_exists` (`Details["refundUuid"]`), `held_funds_released`, `held_funds_pending`, `already_joined`, `payment_not_due`, `idempotency_key_in_use`, `conflict` |
| `*GoneError` | 410 | `merchant_closed`: a removed member's or a closed organization's space |
| `*IdempotencyError` | 422 `idempotency_key_reused` | an `Idempotency-Key` reused for another request: a bug, never retried |
| `*RateLimitError` | 429 | `RetryAfter` (a `time.Duration`), `Limit`, `PeriodSeconds` |
| `*ServerError` | 5xx (503 `network_unavailable`, 504 `timeout`), an unexpected 3xx (redirects are never followed), a 2xx whose body is not the expected JSON | |
| `*NetworkError` | no response: DNS, connection, TLS, timeout, cancelled context | `Err`: `errors.Is(err, context.DeadlineExceeded)` works through it |
| `*WebhookSignatureError` | a webhook signature refused, locally or by `VerifyRemote` | `Reason` |
| `*APIError` | any other HTTP error (e.g. 413 `request_too_large`) | |

`*APIError` carries `StatusCode`, `Code` (branch on it, never on `Message`), `Message`, `Details`
(never nil), `RequestID`, `FieldErrors`, `RawBody` and `Err`. Its message reads
`"<message> (status <status>, code <code>, request <requestId>)"`, followed by
`"; <field>: <message>"` for each field error.

<!-- docs:snippet errors-handling -->
```go
payment, err := client.Payments.GetByReference(ctx, "order-1042")
var (
	validation *qbitflow.ValidationError
	notFound   *qbitflow.NotFoundError
	apiErr     *qbitflow.APIError
)
switch {
case err == nil:
	fmt.Println("order-1042 paid by", payment.UUID)
case errors.As(err, &validation):
	// StatusCode 0: refused before sending; 400: refused by the API.
	for _, f := range validation.FieldErrors {
		fmt.Printf("%s: %s\n", f.Field, f.Message)
	}
case errors.As(err, &notFound):
	fmt.Println("no payment for order-1042")
case errors.As(err, &apiErr):
	// Every SDK error embeds an *APIError: branch on Code, quote RequestID to support.
	fmt.Printf("QBitFlow error %d %s (request %s), retryable: %v\n",
		apiErr.StatusCode, apiErr.Code, apiErr.RequestID, qbitflow.IsRetryable(err))
default:
	log.Fatal(err)
}
```
<!-- /docs:snippet -->

Match the other types the same way: `errors.As(err, &conflict) && conflict.Code ==
"unique_violation"` (`conflict.Details["field"]` names the field taken), `rateLimit.RetryAfter`
for a `*RateLimitError`.

**Client-side validation.** Each params type has a `Validate` method, run before every request:
names and texts (lengths in characters, no markup characters), references
(`A-Z a-z 0-9 . _ : @ -`, 1 to 100), emails, phone numbers, absolute `http(s)` URLs, prices above
0, percents (at most 2 decimals), durations (a frequency between 1 unit and 1 year), UUIDs and
transaction ids, dates, the checkout's product choice, and exclusive filters. A failure is a
`*ValidationError` with `StatusCode` 0 and nothing is sent. What depends on the key's mode or on
stored data is left to the API: the 5 USD test-mode cap, `https`-only live URLs, the frequency
minimum (1 hour live, 5 minutes test), the 95-day export window, uniqueness.

## Retries and idempotency

| | |
|---|---|
| Retried methods | every read (GET), and the 7 creates: `CheckoutSessions.CreatePayment`, `CheckoutSessions.CreateSubscription`, `Products.Create`, `Customers.Create`, `Webhooks.Endpoints.Create`, `Invitations.Create`, `Refunds.Initiate` |
| Never retried | every other write: updates, deletes, `CheckoutSessions.Expire`, `Subscriptions.Cancel`, `Subscriptions.ExecuteTestBilling`, `Members.Trust`, `Members.Remove`, `Invitations.Revoke`, `Webhooks.VerifyRemote` |
| Retried on | a network error or timeout, a 5xx, a 429, a 409 `idempotency_key_in_use` |
| Not retried on | any other 4xx (422 `idempotency_key_reused` included), a 3xx, an unusable response |
| Attempts | 3 retries by default (`WithMaxRetries(n)`; `0` disables them) |
| Back-off | 1 s, 2 s, 4 s…; a 429 waits for its `Retry-After` if longer. A wait above 60 s is not made: the `*RateLimitError` is returned at once |

`qbitflow.IsRetryable(err)` says whether an error is of a transient kind. Cancelling the context
stops the waiting (a `*NetworkError`). `WithTimeout` bounds each attempt; give the context a
deadline to bound the whole call.

**Idempotency keys.** Each call of a create sends a fresh `Idempotency-Key` (a UUID v4) and reuses it
on every retry of that call: a create retried after a timeout answers the first result instead of
opening a second checkout. To retry **across processes** (a queue re-running a job after a crash),
pass your own stable key:

<!-- docs:snippet retries-idempotency -->
```go
// Reads and creates are retried on network errors, 5xx and 429 (1 s, 2 s, 4 s…);
// WithMaxRetries(0) turns that off. The client reads QBITFLOW_API_KEY.
client, err := qbitflow.NewClientFromEnv(qbitflow.WithMaxRetries(5))
if err != nil {
	log.Fatal(err)
}

// Each create sends an Idempotency-Key, the same on its retries. Derive your own from the
// order to retry across processes: the same key returns the same session, never a second one.
orderID := "order-1042"
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductName: "T-shirt",
	Description: "Blue, size M",
	Price:       4.99,
	Reference:   orderID,
	SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
	CancelURL:   "https://shop.example.com/orders/cancel",
}, qbitflow.WithIdempotencyKey("checkout-"+orderID))
var reused *qbitflow.IdempotencyError
if errors.As(err, &reused) {
	log.Fatal("this key was already used with other params") // 422 idempotency_key_reused
}
if err != nil {
	log.Fatal(err)
}
fmt.Println("Redirect the customer to", session.Link)
```
<!-- /docs:snippet -->

`qbitflow.WithRequestID("job-7781")` also sends an `X-Request-Id`, echoed in errors.

A key is 1 to 255 printable ASCII characters without spaces, and only successful answers are kept
(24 hours): after a 4xx, the same key runs the request again. A `409 idempotency_key_in_use` (the
first request still running) is retried automatically. Other methods ignore the option. Retry
the other writes yourself only after reading the resource's state (a 504 may have done the work).

## Configuration

```go
client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"),
	qbitflow.WithTimeout(10*time.Second),
	qbitflow.WithMaxRetries(5),
	qbitflow.WithHTTPClient(&http.Client{Transport: http.DefaultTransport}), // proxies, tracing…
)
if err != nil {
	log.Fatal(err)
}
fmt.Println(client != nil)
```

| Client option | Default | |
|---|---|---|
| `WithBaseURL(url)` | `https://api.qbitflow.app/v2` (`qbitflow.DefaultBaseURL`) | an absolute `http(s)` URL; a trailing `/` is stripped |
| `WithTimeout(d)` | 30 s (`DefaultTimeout`) | per attempt; positive |
| `WithMaxRetries(n)` | 3 (`DefaultMaxRetries`) | `0` disables retries; negative is refused |
| `WithHTTPClient(c)` | a plain `*http.Client` | the SDK uses a copy that never follows redirects |
| `WithOnBehalfOf(userUUID)` | none | every request acts in that member's space |

| Request option (any method, last arguments) | |
|---|---|
| `WithOnBehalfOf(userUUID)` | acts in that member's space for this call; `""` forces the organization's |
| `WithIdempotencyKey(key)` | the 7 creates only: your own `Idempotency-Key` |
| `WithRequestID(id)` | sends `X-Request-Id` (1 to 128 of `A-Z a-z 0-9 - _ . :`) |

| Webhook option (`VerifyWebhook`, `VerifyWebhookRequest`, `ConstructEvent`, `NewWebhookRouter`) | Default |
|---|---|
| `WithTolerance(d)` | 300 s (`DefaultWebhookTolerance`) |
| `WithClock(func() time.Time)` | `time.Now` |
| `WithWebhookErrorHandler(func(*Event, error))` (`NewWebhookRouter` only) | errors reported nowhere |

`qbitflow.NewClientFromEnv(opts...)` reads the key from `QBITFLOW_API_KEY` (required), and
`QBITFLOW_BASE_URL` and `QBITFLOW_ON_BEHALF_OF` when they are set; the options given override them.

A bad option makes `NewClient` return a `*ValidationError`. A client's configuration never changes
after `NewClient`; `client.OnBehalfOf` derives clients that share it. Every request sends
`User-Agent: qbitflow-go/3.0.0` (`qbitflow.Version`).

## Migrating from 2.x

3.0.0 is a rewrite for API v2: module `github.com/QBitFlow/qbitflow-go-sdk/v3`, base URL `/v2`,
one package, `context.Context` on every call, UUID ids, members and invitations instead of users
and claims, a new webhook signature, typed errors. **[MIGRATION-v3.md](MIGRATION-v3.md)** maps
every 2.x method and type to its replacement, with before/after code for the common tasks.

## Examples

Runnable programs in [`examples/`](examples) (`QBITFLOW_API_KEY=sk_… go run ./examples/<name>`):

| Example | Shows | Environment (besides `QBITFLOW_API_KEY`, `QBITFLOW_BASE_URL`) |
|---|---|---|
| [`client`](examples/client) | a client from a key and `Me`, a client from the environment | `QBITFLOW_ON_BEHALF_OF` |
| [`checkout`](examples/checkout) | a payment checkout, its status, `WaitForCompletion`, expiry | |
| [`catalog`](examples/catalog) | a product, the products, a checkout for a product, a page of customers | |
| [`payments`](examples/payments) | a filtered page of payments, the iterator, exact amounts, a payment by id and by reference | `PAYMENT_UUID` |
| [`refunds`](examples/refunds) | the active refunds, a partial refund | `PAYMENT_UUID` |
| [`subscriptions`](examples/subscriptions) | a subscription checkout with a trial, a subscription, its bills, `HasAccess`, test billing, cancel at period end | `SUBSCRIPTION_UUID`, `TEST_BILL=1`, `CANCEL=1` |
| [`marketplace`](examples/marketplace) | members, invitations, sell `OnBehalfOf`, wallets, held funds, fee, trust, removal | `INVITE=1`, `REVOKE=1`, `MEMBER_UUID`, `FEE=1`, `TRUST=1`, `REMOVE=1` |
| [`accounting`](examples/accounting) | a yearly export, as JSON and CSV | |
| [`currencies`](examples/currencies) | the currencies customers can pay with | |
| [`webhooks`](examples/webhooks) | the event log, a webhook endpoint and its secret | `CREATE_ENDPOINT=1` |
| [`webhook_handler`](examples/webhook_handler) | a `WebhookRouter` receiver: typed handlers, deduplication, retries on failure | `QBITFLOW_WEBHOOK_SECRET` |
| [`webhook_verify`](examples/webhook_verify) | a receiver verifying and parsing with `ConstructEvent` | `QBITFLOW_WEBHOOK_SECRET` |
| [`errors_and_retries`](examples/errors_and_retries) | error types, `IsRetryable`, idempotency keys across processes | |

The code blocks of this README and of [qbitflow.app/docs](https://qbitflow.app/docs) come from
these programs (see [CONTRIBUTING.md](CONTRIBUTING.md)).

## Testing

```bash
go build ./... && go vet ./... && test -z "$($(go env GOROOT)/bin/gofmt -l .)" && go test ./... && go test -race ./...
```

`go test ./...` runs the offline suite against stub servers. The live checks are behind a build
tag and need an API key **and** an explicit base URL:

```bash
QBITFLOW_API_KEY=sk_… QBITFLOW_BASE_URL=https://… go test -tags integration -run Live -v .
```

The read-only checks only read; the write checks also need `QBITFLOW_LIVE_WRITES=1` and a test-mode key (`QBITFLOW_ALLOW_LIVE_MODE_WRITES=1` allows a live-mode key, for a disposable server only).

## License

This project is licensed under the MPL-2.0 License: see the [LICENSE](LICENSE) file. See also
[COMPLIANCE.md](COMPLIANCE.md) and the [trademark policy](TRADEMARKS.md).

## Support

- 📖 [Documentation](https://qbitflow.app/docs)
- 📧 [Email Support](mailto:support@qbitflow.app)
- 🐛 [Issue Tracker](https://github.com/QBitFlow/qbitflow-go-sdk/issues)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Security

For security issues, please email security@qbitflow.app instead of using the issue tracker (see
[SECURITY.md](SECURITY.md)).
