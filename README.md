# QBitFlow Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/QBitFlow/qbitflow-go-sdk/v3.svg)](https://pkg.go.dev/github.com/QBitFlow/qbitflow-go-sdk/v3)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

The official Go SDK for [QBitFlow](https://qbitflow.app), non-custodial crypto payments: hosted
checkouts, one-time payments, subscriptions, refunds, marketplaces with commissions and held
funds, accounting exports and signed webhooks. Customers pay from their own wallets, on Ethereum,
Base and Solana, straight to yours.

- **API v2**, every integrator route: 14 services, 53 public routes, one `qbitflow` package.
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
- [Services at a glance](#services-at-a-glance)
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
	PriceUSD:    "4.99",       // a decimal string, never a float
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
  `Immediate: new(false)`, `IsActive: new(false)`, `MinPeriods: new(uint32(3))`. The API never
  sends `null`: an optional field without a value is absent.
- **Money is a decimal string, never a float**, in requests and responses alike: USD amounts
  (`PriceUSD: "4.99"`, `AmountUSD`, `TotalUSD`), token amounts (`Amount: "10"` in the token's units,
  next to `AmountMinUnits: "10000000"` on-chain and the `Currency` object) and fee rates in percent
  (`FeePercent: "0.99"` is 0.99 %, `RefundPercent: "50"`). Inputs are checked before sending: at
  most 2 decimals, no sign or exponent (`"12.50"` is fine; a JSON number would be refused by the
  API). Compare or add them with a decimal library, or `FormatAmount` / `ParseAmount`. They come
  back canonical (`"12.50"` reads back `"12.5"`, `"12.00"` as `"12"`): compare amounts as
  decimals, never as strings.
- **Metadata:** products, customers, checkouts and subscriptions carry `Metadata`, your own
  `map[string]string` (an order id, a CRM key): at most 50 keys of 1 to 40 characters among
  `A-Z a-z 0-9 _ - . :`, values of at most 500 characters without control or bidirectional
  characters. A checkout's is copied onto its payment or subscription and into their webhooks. On
  an update it is **merged**: a key set to a value sets it, a key set to `""` deletes it, keys not
  named are kept. Never put secrets or personal data in it.
- **Enums** are named string types with constants (`qbitflow.SubscriptionStatusActive`). A value
  this SDK does not know yet is kept as is: give every `switch` a `default` branch.
- **Times** are `time.Time` (the API sends RFC 3339 in UTC; any offset is accepted).

## Services at a glance

Every method takes `ctx` first and request options last; `it` marks a paginated list with an
`Iterate` twin (`IterateBills` for `ListBills`). The SDK calls **53 public routes**.

| Service | Methods (route) |
|---|---|
| `client` | `Me` (`GET /me`), `OnBehalfOf(userUUID)` (a derived client, no request) |
| `Products` | `List` it (`GET /products`), `Create`, `Get`, `GetByReference` (the list's `?reference=` filter), `Update` (`PATCH`), `Delete` |
| `Customers` | `List` it (`GET /customers`), `Create`, `Get`, `GetByReference`, `Update` (`PATCH`), `Delete` |
| `CheckoutSessions` | `CreatePayment` (`POST /checkout-sessions/payment`), `CreateSubscription` (`POST /checkout-sessions/subscription`), `Get`, `Expire` (`POST …/:uuid/expire`), `WaitForCompletion` (polls `Get`) |
| `Payments` | `List` it (`GET /payments`), `Get`, `GetByReference` |
| `Transactions` | `List` it (`GET /transactions`: payments and bills) |
| `Failures` | `List` it (`GET /payment-failures`) |
| `Subscriptions` | `List` it, `Get`, `GetByReference`, `Update` (metadata), `ListBills` it, `GetBill` (`GET /bills/:uuid`), `Cancel`, `TestBill` (test mode) |
| `Refunds` | `List` it, `Get` (read only) |
| `Members` (organization key) | `List` it, `Get`, `Update`, `Remove`, `Trust`, `ListHeldFunds`, `GetHeldFunds`, `GetOwnHeldFunds` (`GET /held-funds`) |
| `Invitations` (organization key) | `Create`, `List` it, `Revoke` |
| `Wallets` | `List`, `ListForMember`, `ListAcceptedCurrencies` |
| `Accounting` | `ExportJSON`, `ExportCSV` (`GET /accounting/export`), `ExportJSONRange`, `ExportCSVRange` (any range, in 92-day windows) |
| `Webhooks` | `Verify`, `ConstructEvent`, `ParseEvent` (local, no request), `VerifyRemote` (`POST /webhooks/verify`); `Endpoints`: `List` it, `Create`, `Get`, `Update`, `Delete`; `Events`: `List` it, `Get` |
| `Currencies` (public) | `List`, `Get` |

Without a client, the package-level `VerifyWebhook`, `VerifyWebhookRequest`, `ConstructEvent`,
`ParseEvent`, `SignWebhook` (for tests) and `NewWebhookRouter` cover webhook receivers, and
`FormatAmount` / `ParseAmount` convert token amounts.

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
		log.Printf("fulfil order %q: %s USD (%s, %v)", payment.Reference, payment.AmountUSD, payment.UUID, payment.Metadata)
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
On the success page, `CheckoutSessions.Get(ctx, r.URL.Query().Get("uuid"))` tells the customer where their
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
	log.Fatal(err) // Get's errors, or the context's
}
if status.Status != qbitflow.CheckoutSessionStatusCompleted {
	fmt.Println("not paid (yet):", status.Status) // expired, or still open at the timeout
}
```
<!-- /docs:snippet -->

### Access control

A subscription grants access while now < `AccessUntil`, whatever its status: the end of the
period paid for, plus an hour for an `active` one, since its renewal is billed within minutes of
its due date. `HasAccess` reads `AccessUntil`, else `CurrentPeriodEnd` (a subscription you stored
before the API sent `AccessUntil`):

<!-- docs:snippet has-access -->
```go
// Access lasts while now < AccessUntil, whatever the status: the end of the period paid
// for, plus an hour for an active subscription while its renewal is billed.
sub, err := client.Subscriptions.Get(ctx, subscriptionUUID, nil)
if err != nil {
	log.Fatal(err)
}
if sub.HasAccess(time.Now()) { // now before AccessUntil
	fmt.Println("access granted until", sub.AccessUntil.Format(time.RFC3339))
} else {
	fmt.Println("no access")
}
```
<!-- /docs:snippet -->

### Displaying amounts

Token amounts come both ways: `Amount` in the token's units (`"4.99"`) and `AmountMinUnits`, its
on-chain integer (`"4990000"`). Convert between the two exactly (no float on the way):

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
// The API exports at most 92 days at a time (both included): the …Range methods split any
// range into windows, request them in order and join the results (the CSV header once).
events, err := client.Accounting.ExportJSONRange(ctx, "2026-01-01", "2026-12-31")
if err != nil {
	log.Fatal(err)
}
for _, e := range events {
	fmt.Println(e.Type, e.TxUUID, e.TxTimeUTC.Format(time.DateOnly), e.GrossAmountUSD, e.ProductName)
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
live). `NewClient` only checks the key's shape (after trimming: starting with `sk_`, printable
ASCII without spaces; its error never quotes the key) and sends nothing:
call `Me` to check it online, as the [Quick start](#quick-start) does. `me.Role` says what the
key is (`admin`: an organization key) and `me.Space.Test` its mode: a job that must only run in
test mode checks it.

Keep the key in a secret store or an environment variable, never in code. API v2 issues keys shaped
`sk_<test|live>_<uuid>_<secret>`; the SDK never parses them. Keys of an earlier shape
(`sk_<digits>_…`, `sk_<uuid>_<test|live>_…`) pass that check but are refused by the API (`401`):
create a new one in the dashboard.

### `On-Behalf-Of`: acting in a member's space

A marketplace's **organization key** can act in any of its members' spaces: create their products
and checkouts, read their payments. Name the member by their **user UUID** (`Member.UserUUID`,
also in the `member.joined` webhook). A non-member, an owner or admin of the team, or a member of
the other mode answers 404. Acting for a member, the request's role is `user`: the admin methods
(`Members`, `Invitations`, `Wallets.ListForMember`, `ListAcceptedCurrencies` with a `UserUUID`)
answer 403.

<!-- docs:snippet client-on-behalf-of -->
```go
// A client acting in the member's space (organization key): every request sends
// On-Behalf-Of. It shares the connections and configuration of client.
seller := client.OnBehalfOf(memberUUID)
products, err := seller.Products.List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(products.Items), "products on the first page of the seller's space")
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
fmt.Println(len(own.Items), "products of the organization")
```

`qbitflow.WithOnBehalfOf(uuid)` is also a client option: `qbitflow.NewClient(key,
qbitflow.WithOnBehalfOf(uuid))`. A value that is not a UUID (or the nil UUID, or blank) is a
`*ValidationError`, and nothing is sent; `""` means the organization level and the header is then
**omitted** (never sent blank: the API answers 403 to a blank one).

## Checkout sessions

A checkout session is a hosted payment page. Create it, send your customer to its `Link`, and act
on the webhook. The session's id (`pay@…` or `sub@…`) is also the id of the payment or the
subscription it creates once the customer's transaction is confirmed.

Name the product with **exactly one** of `ProductUUID`, `ProductReference`, or an inline product
(`ProductName` + `PriceUSD`, `Description` optional). The [Quick start](#quick-start) names an
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
	PriceUSD:    "4.99", // USD per period, a decimal string
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

- **Products and checkout kinds.** A payment checkout of a subscription product sells it **once**,
  at its price (its terms are ignored, nothing recurs). A subscription checkout of a one-time
  product needs `Frequency`, else a `*ValidationError` on `frequency` naming the product. An
  inline product of a **subscription** checkout creates a hidden product (`IsActive` false),
  reused by the next checkouts with the same name, description and price
  (`Products.List` with `IncludeHidden` shows it); a payment's inline product creates none.
- **Other fields:** `Reference` is your order id, unique per space; `CustomerUUID` or
  `CustomerReference` (kept on the payment) names your customer (**no receipt without a QBitFlow
  customer**: see [Products and customers](#products-and-customers)); `ExpiresInMinutes` sets the
  session's lifetime (10 to 1440; 0 = the default); `Metadata` is copied onto the payment or the
  subscription and into their webhooks (`payment.completed`, `subscription.*`, `checkout.expired`).
- **The answer** is a `CreatedCheckoutSession`: `Link`, `UUID` and `ExpiresAt`, always set.

- **Redirect placeholders.** In `SuccessURL` and `CancelURL`, QBitFlow replaces `{{UUID}}`
  (`qbitflow.PlaceholderUUID`) with the session's id and `{{TRANSACTION_TYPE}}`
  (`qbitflow.PlaceholderTransactionType`) with `payment` or `createSubscription`; the SDK sends
  them as they are, never URL-encoded. In live mode both
  URLs must be `https`. A redirect proves nothing (anyone can open the URL): fulfil on the
  webhook, or on `CheckoutSessions.Get`.
- **Redirect URLs and members.** Left empty, they come from the space's checkout defaults, set in
  the dashboard. A member's own key may send `SuccessURL` and `CancelURL` while its organization's
  `members.checkoutRedirects` policy allows it (the default). With the policy off, it gets a
  `403 policy_disabled` (a `*qbitflow.PermissionDeniedError`) with `Details["policy"]`
  (`members.checkoutRedirects`) and `Details["field"]` (`successUrl` first, else `cancelUrl`), and
  nothing is created. An organization key acting for the member (`OnBehalfOf`) may always send them.
- **Errors to expect:** `409 merchant_not_ready` (`Details["reason"]`) when the space's wallets
  accept no currency, or for a subscription of a member whose funds you hold (`fundsHeld`); `409 unique_violation` when another payment or open session holds the
  `Reference`; `400 validation_failed` above `"5"` USD in test mode, fees included
  (`Details["max"]`, a string), and at `"0.25"` or less for a live subscription (`Details["min"]`);
  `404` for an unknown product or customer; `410 merchant_closed` for a removed member's space.

### Fees

A checkout, a payment's or a subscription's, can add amounts to the price (a tax, shipping, a
service fee) and have the customer pay QBitFlow's processing fee. The customer sees each line on
the checkout page and pays them with the price (a subscription: with every bill):

<!-- docs:snippet checkout-create-payment-fees -->
```go
// The customer pays the price plus every line: $3.99 + $0.75 + the processing fee. You
// keep the price and your lines. Test mode caps the total, fees included, at $5.
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductName: "T-shirt",
	Description: "Blue, size M",
	PriceUSD:    "3.99",
	Reference:   "order-1044",
	SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
	CancelURL:   "https://shop.example.com/orders/cancel",
	Fees: &qbitflow.CheckoutFees{
		Items: []qbitflow.FeeItem{
			{Label: "Shipping", Description: "Standard, 3 to 5 days", AmountUSD: "0.75"},
		},
		// QBitFlow's processing fee, grossed up so you keep the price and your lines.
		// Left nil, your checkout.customerPaysProcessingFee setting decides.
		ProcessingFee: new(true),
	},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("Redirect the customer to", session.Link)
```
<!-- /docs:snippet -->

- **Your lines** (`Fees.Items`, at most 10, shown in this order): `Label` (one line, 1 to 40
  characters), `Description` (optional, at most 200) and `AmountUSD` (a decimal string: above 0,
  at most 1,000,000, at most 2 decimals). The SDK checks them before sending: a bad line fails with a
  `*qbitflow.ValidationError` on its path (`fees.items[0].amountUsd`), 11 lines on `fees.items`.
- **The processing fee** (`ProcessingFee: new(true)`) is a last line, "Processing fee", that
  QBitFlow computes on the price and your lines at your platform fee, grossed up (its own fee is
  taken on it too) and rounded up to the cent, so you keep at least the price and your lines, as
  if no fee were taken: $100 + $20 VAT at 0.99 % adds $1.20, the customer pays $121.20 and you
  keep $120.00. The rate is the checkout's: QBitFlow's fee for your space when the checkout is
  created (a schedule by volume, unless QBitFlow set you a custom rate).
  Left `nil`, the space's `checkout.customerPaysProcessingFee` setting decides (off by default;
  the organization's team sets it in the dashboard, for its own space or as an exception for a
  member's; a member can't change it); `false` turns it off for this checkout, and product
  payment links follow the setting.
- **What is charged** is the price plus every line: the session's and the payment's `AmountUSD`,
  which the contracts split (QBitFlow's fee, and a marketplace's organization fee, are taken on
  it). The network fee is paid on top, by the customer's wallet (`Settlement.Split.NetworkFee`).
  Test mode caps the total, fees included, at `"5"` USD (`400 validation_failed` on `fees`).
- **Afterwards**, the `Payment` has `PriceUSD`, `Fees` (the lines as the customer saw them: `Type`
  `custom` or `processingFee`, `Label`, `Description`, `AmountUSD`) and `AmountUSD`;
  `checkout.expired` sends the session with `PriceUSD`, `Fees` and `AmountUSD`.
- **Subscriptions** take the same `Fees` (`CreateSubscriptionSessionParams.Fees`, the same rules),
  locked with the price: every bill, the first one and each next one, charges the price plus every
  line, whatever the product's price or your fee becomes. The `Subscription` keeps them as `Fees`
  next to `PriceUSD` (which stays the price); a bill's `AmountUSD` and `subscription.upcomingBill`'s
  `AmountUSD` include them, and the maximum and allowance the subscriber signs are sized on that
  amount. A trial's upgrade keeps the trial's lines. `checkout.customerPaysProcessingFee` applies
  to subscription checkouts too.

### Status

<!-- docs:snippet checkout-status -->
```go
status, err := client.CheckoutSessions.Get(ctx, sessionUUID)
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
| `expired` | Expired unpaid (`checkout.expired` was sent). Once QBitFlow no longer keeps the record, a 404 | yes |

**Never cancel an order on `LastAttempt`:** a failed attempt is not final, and the customer can pay
from the same checkout until it expires. Release what the order holds on `checkout.expired`.

`ExpiresAt` is set while the session is open. `WaitForCompletion(ctx, uuid, &qbitflow.WaitOptions{Timeout, Interval})` polls `Get` until the
session is final and returns that status, or the last one seen when the timeout (default 10
minutes) elapses first; the interval (default 3 s) is at least 1 s. It is meant for scripts, tests
and back-office jobs ([recipe](#a-checkout-its-success-page-and-scripts)): orders are fulfilled on
the webhook.

### Expire

End a session early (an order cancelled on your side). It answers its status (`expired`,
`ExpiresAt` the time you expired it), and `checkout.expired` follows. Expiring it again answers
`expired` too, without `ExpiresAt` (only the first answer has it). Once the customer paid or is
paying it is a `409 tx_already_sent`.

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

Products are optional for payments (a payment checkout can name an inline product; a
subscription checkout's inline product becomes a hidden product: [Checkout sessions](#checkout-sessions))
and give you a reusable catalog with payment links. The reference is generated when `""`. A subscription product carries its
terms: `Subscription: &qbitflow.SubscriptionTermsParams{Frequency: &qbitflow.Duration{Value: 1,
Unit: qbitflow.DurationUnitMonths}}`. A member's key gives a product terms (on `Create` or
`Update`) only while the organization's `members.subscriptions` policy allows it (`403
policy_disabled` otherwise); with the policy off, the links of the member's subscription products
open nothing.

<!-- docs:snippet products-create -->
```go
product, err := client.Products.Create(ctx, &qbitflow.CreateProductParams{
	Name:        "T-shirt",
	Description: "Blue, size M",
	PriceUSD:    "4.99",          // a decimal string, never a float
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
// Iterate walks every page (List returns one); hidden products only with IncludeHidden.
for p, err := range client.Products.Iterate(ctx, nil) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %q: %s USD (%s)\n", p.UUID, p.Name, p.PriceUSD, p.Reference)
}
```
<!-- /docs:snippet -->

`&qbitflow.ProductListParams{IncludeHidden: true, Subscription: new(true)}` lists the hidden
products too, and only the subscription ones. `Products.Update` changes only the fields set; a new
price applies to new checkouts and subscribers only:

```go
product, err := client.Products.Update(ctx, productUUID, &qbitflow.UpdateProductParams{
	PriceUSD:    "5.99",
	IsActive:    new(false), // hidden from Products.List
	Description: new(""),    // "" clears it
	Metadata:    map[string]string{"sku": "TS-BLUE-M", "legacy": ""}, // merged: "" deletes a key
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(product.PriceUSD, product.Metadata)
```

`Products.Get` and `Delete` (`204`) complete the set. `Products.GetByReference(ctx, "tshirt-blue-m")`
runs the list's reference filter (`GET /products?reference=`, hidden products included) and returns
its one product, or a `*qbitflow.NotFoundError` (status 404, code `not_found`) when there is none,
exactly like an unknown UUID. Deleting a product does not stop its subscriptions: cancel them with
`Subscriptions.Cancel` if the product is gone for good.

```go
customer, err := client.Customers.Create(ctx, &qbitflow.CreateCustomerParams{
	FirstName: "Ada",
	LastName:  "Lovelace",
	Email:     "ada@example.com",
	Reference: "crm-42",
	Metadata:  map[string]string{"plan": "pro"},
})
if err != nil {
	log.Fatal(err)
}

// A pointer to "" clears LastName, PhoneNumber or Address; nil leaves them unchanged.
customer, err = client.Customers.Update(ctx, customer.UUID, &qbitflow.UpdateCustomerParams{PhoneNumber: new("")})
if err != nil {
	log.Fatal(err)
}

// An email is not unique: several customers of a space may share one.
page, err := client.Customers.List(ctx, &qbitflow.CustomerListParams{Email: "ada@example.com"})
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(page.Items), "customers with this email")

byReference, err := client.Customers.GetByReference(ctx, "crm-42") // *NotFoundError when none
if err != nil {
	log.Fatal(err)
}
fmt.Println(customer.UUID == byReference.UUID)
```

`Customers.Get`, `List` / `Iterate` (by `Email` or `Reference`) and `Delete` (`204`) complete the
set. A checkout given `CustomerUUID` or `CustomerReference` asks the customer
nothing (`CustomerReference` is kept on the payment, and links your customer with that reference
if there is one; none is created); without either, the checkout asks what your
`checkout.customerDetails` setting says (the full details by default).

**Without a QBitFlow customer, the buyer gets no receipt and no other customer email** (a
subscription's billing emails included): QBitFlow has no address for them. To have them, create
the customer with that reference and their email first
(`Customers.Create(ctx, &qbitflow.CreateCustomerParams{Reference: "crm-42", FirstName: "Ada", Email: "ada@example.com"})`).
That works even after the checkout was created, as long as it is before the payment is recorded.

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
	// Amounts are decimal strings: the token's (Amount, with Currency) and USD (AmountUSD).
	fmt.Printf("%s %q: %s %s (%s USD)\n", p.UUID, p.Reference, p.Amount, p.Currency.Symbol, p.AmountUSD)
}
if page.HasMore {
	// Pass it back, verbatim, as PaymentListParams.Cursor for the next page.
	fmt.Println("next cursor:", page.NextCursor)
}
```
<!-- /docs:snippet -->

<!-- docs:snippet payments-get -->
```go
payment, err := client.Payments.Get(ctx, paymentUUID, nil) // pay@…
if err != nil {
	log.Fatal(err)
}
fmt.Println(payment.UUID, payment.Reference, payment.AmountUSD, "USD", payment.Metadata)

// Runs GET /payments?reference= and returns its one payment.
byReference, err := client.Payments.GetByReference(ctx, "order-1042") // your order id
if err != nil {
	log.Fatal(err) // a *qbitflow.NotFoundError (404, not_found) when no payment has this reference
}
fmt.Println("order-1042 was paid by", byReference.UUID)
```
<!-- /docs:snippet -->

A payment carries its transaction (`TxHash`, `ExplorerURL`, `ConfirmedAt`), your `Metadata`
(copied from its checkout, read-only), how it settled (`Settlement`), and whether it can be
refunded (`Refundable`, `Refund`).

- **The amounts:** `Amount` is what the customer paid **in the token's units** (`"10"` for 10 USDC,
  with `AmountMinUnits` and the `Currency` object), `AmountUSD` the same in USD: `PriceUSD` (the
  product's price at the checkout) plus every line of `Fees` (the [checkout's fees](#fees)). It is
  what the contracts split; the network fee the customer paid on top is not in it (`Paid`,
  `PaidMinUnits` and `PaidUSD` include it: a refund's base).
- **The settlement** (`Settlement`): QBitFlow's `FeePercent`, the `OrganizationFee` and
  `ReferralFee` when they apply, the `Split` (`Platform`, `Organization`, `Referral`, `Merchant`,
  `NetworkFee`, each a `TokenAmount`: `Amount`, `AmountMinUnits`, `AmountUSD`) and the
  `Transaction` that paid it (`BlockNumber`, `BlockTime`, `GasUsed`, its `Fee` in the native coin).
- **Filters** (`PaymentListParams`): `CustomerUUID`, `ProductUUID`, `CreatedAfter` / `CreatedBefore`
  (both excluded), `Reference`, `Refunded`, and, with an organization key acting for itself,
  `IncludeMembers` (every member's rows, each naming its `UserUUID`) or `UserUUID` (one member's).
  The last two exclude each other. `Payments.GetByReference` runs the `Reference` filter and
  answers a `*NotFoundError` when no payment has it.
- **Reading a member's row** from the organization's space: `Payments.Get(ctx, id,
  &qbitflow.ReadParams{IncludeMembers: true})`, or `OnBehalfOf` the member.
- **The transactions feed** of one-time payments and subscription bills, newest first:
  `Transactions.List` / `Iterate`, with the same filters plus `Type` (`TransactionKindPayment`,
  `TransactionKindBill`), `SubscriptionUUID` and `Refunded`. Each `Transaction` reads like its
  resource (a `Payment` or a `Bill`); open it with `Payments.Get` or `Subscriptions.GetBill`.

<!-- docs:snippet transactions-list -->
```go
// One feed of one-time payments and subscription bills, newest first; each row reads like
// its resource. Type filters it to one kind (TransactionKindPayment or TransactionKindBill).
feed, err := client.Transactions.List(ctx, &qbitflow.TransactionListParams{Limit: 20})
if err != nil {
	log.Fatal(err)
}
for _, tx := range feed.Items {
	fmt.Println(tx.Type, tx.UUID, tx.AmountUSD, "USD")
}
```
<!-- /docs:snippet -->

- **Failures** are the failed attempts to pay a checkout or a bill. They never moved money and
  are not final: the customer can try again.

```go
for f, err := range client.Failures.Iterate(ctx, &qbitflow.FailureListParams{
	Category: qbitflow.FailureCategoryInsufficientBalance,
}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s attempt %d: %s (%s USD)\n", f.TxUUID, f.Attempt, f.Code, f.AttemptedUSD)
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
| `stopped` | Cancelled during its period: never billed again, `cancelled` at `NextBillingAt` |
| `cancelled` | Over (`CancellationReason` says why). Final |

**Access rule: grant access while `now < AccessUntil`, whatever the status.** `AccessUntil` is
`CurrentPeriodEnd` (the paid-through date), plus 1 hour for an `active` subscription: QBitFlow
bills a renewal within minutes of its due date, so checking `CurrentPeriodEnd` would cut every
subscriber off for those minutes at each renewal (show it as "paid through", don't gate on it). A
`stopped` or `paused` subscription has paid for its period; a `pastDue` one's period has ended.
`sub.HasAccess(time.Now())` applies it, reading `CurrentPeriodEnd` for a subscription stored
before `AccessUntil` existed ([recipe](#access-control)).

`ActionRequired` says what the customer must do (`topUpAllowance`, `raiseMaximum`,
`confirmTrial`; `""`: nothing): point them to the subscription's `ManagementPageLink`.

A subscription's price is locked at signup: `PriceUSD` plus its `Fees` lines, every bill. Its
`Allowance` (absent during a trial) is what the subscriber's wallet still lets QBitFlow bill
(`Amount`, `AmountMinUnits`, `AmountUSD`), how many bills that covers (`RemainingPeriods`) and the
most one bill may charge (`MaxPerPeriod`). `RemainingPeriods` counts bills before each bill's
network fee, so it is usually higher than the checkout's "N payments" (an estimate that kept room
for network fees). `LastBilledAt`, `NextBillingAt` and `CancellableAt` are
`nil` when they don't apply.

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
		fmt.Printf("%s: %s USD, paid until %s\n", bill.UUID, bill.AmountUSD, bill.PeriodEnd.Format(time.DateOnly))
	}
}
```

- `Subscriptions.GetByReference` reads one by your checkout's `Reference` (the list's
  `?reference=` filter; a `*NotFoundError` when none); `GetBill` one bill (`sub-hist@…`). Bills
  carry no metadata: read the subscription's. `Subscriptions.Get` and `GetBill` take
  `&qbitflow.ReadParams{IncludeMembers: true}` to read a member's from the organization's space.
- `Subscriptions.Update` changes its `Metadata` only, merged (a key set to `""` deletes it), and
  answers the subscription:

<!-- docs:snippet subscriptions-update -->
```go
// Metadata is merged into the subscription's: "" deletes a key, keys not named are kept.
// It is the only thing a merchant edits on a subscription; webhooks carry it.
updated, err := client.Subscriptions.Update(ctx, subscriptionUUID, &qbitflow.UpdateSubscriptionParams{
	Metadata: map[string]string{"plan": "pro", "coupon": ""},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(updated.Metadata["plan"])
```
<!-- /docs:snippet -->

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
automatically. A trial is cancelled immediately only (`Immediate` false: `400`). A subscription
cancelled already is a `*ConflictError` (`409 subscription_already_stopped_or_inactive`), not a
`*NotFoundError`: a 404 only means it isn't in your space (or doesn't exist).

### Test billing

In test mode a subscription is billed only when you ask, with live's statuses and webhooks.
`TestBill` answers once the attempt settled: a `TestBill` with the bill's `sub-hist@…` id, its
`Stage` and, once done, its `Outcome` (a live subscription: `400 test_mode_only`; a bill not due:
`409 payment_not_due`; a cancelled subscription: `409 subscription_already_stopped_or_inactive`):

<!-- docs:snippet subscriptions-test-bill -->
```go
// Test mode only: runs the next billing now, with live's statuses and webhooks.
bill, err := client.Subscriptions.TestBill(ctx, subscriptionUUID)
if err != nil {
	log.Fatal(err) // 409 payment_not_due before NextBillingAt; 400 test_mode_only for a live one
}
fmt.Println(bill.BillUUID, bill.Stage, bill.Outcome, bill.FailureCode) // sub-hist@…
```
<!-- /docs:snippet -->

Walk the timeline once in test mode: a 5-minute frequency, pay from a test wallet, trigger the
bill, then empty the wallet and trigger it again to see `subscription.billingFailed` and
`pastDue`.

## Refunds

<!-- docs:snippet refunds-list -->
```go
// The active ones: pending, waiting for an answer (Status empty: every status).
page, err := client.Refunds.List(ctx, &qbitflow.RefundListParams{
	Status: []qbitflow.RefundStatus{qbitflow.RefundStatusPending},
})
if err != nil {
	log.Fatal(err)
}
for _, r := range page.Items {
	fmt.Println(r.UUID, r.TxUUID, r.InitiatedBy, r.Reason, r.Amount, r.Currency.Symbol)
}
```
<!-- /docs:snippet -->

`Status` takes up to 5 statuses (empty: every status); `TxUUID` finds the refund of one
transaction. From the organization's space the members' refunds are included by default:
`&qbitflow.RefundListParams{IncludeMembers: new(false)}` leaves them out. The answered refunds
come page by page:

```go
for r, err := range client.Refunds.Iterate(ctx, &qbitflow.RefundListParams{
	Status: []qbitflow.RefundStatus{qbitflow.RefundStatusApproved, qbitflow.RefundStatusRejected},
}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r.UUID, r.Status, r.ExplorerURL)
}
```

`Refunds.Get` reads one refund by its `refund@…` id:

<!-- docs:snippet refunds-get -->
```go
refund, err := client.Refunds.Get(ctx, refundUUID) // refund@…
if err != nil {
	log.Fatal(err)
}
// pending (a customer's request), approved or rejected.
fmt.Println(refund.UUID, refund.Status, refund.RefundPercent+"%", refund.Amount, refund.Currency.Symbol)
```
<!-- /docs:snippet -->

Refunds are read only here: there is no create. A refund is **requested by the customer** (the
refund page; you approve or reject it in the dashboard) or **sent by you from the dashboard**,
signed from the wallet that was paid. Yours exists only once its transfer confirmed: it is recorded
`approved`, and `refund.approved` (with `InitiatedBy` `merchant`) is the first and only event about
it. `refund.requested` is always a customer's request.

`RefundPercent` is a percent string (`"100"`: all of it) of everything the customer paid (`Paid`,
network fee included); `Amount` / `AmountMinUnits` / `AmountUSD` is what it sends back. There is
one refund per transaction, whatever it became: `Payment.Refundable` / `NotRefundableReason` say
whether one would be accepted now.

A refund is `pending` (a customer's request waiting for your answer), `approved` or `rejected`
(both final). `InitiatedBy` is `customer` or `merchant`.

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
	HoldFunds:              true, // hold their payments until Members.Trust (always sent)
	OrganizationFeePercent: "10", // your commission on their payments: "0" to "50" %
	RedirectURL:            "https://shop.example.com/sellers/welcome",
})
if err != nil {
	log.Fatal(err) // 409 already_joined for a member
}
// QBitFlow also emails the seller a link of their own. The seller exists once they
// accepted: wait for member.joined.
fmt.Println("invitation", created.UUID, "link:", created.Link)
```
<!-- /docs:snippet -->

`HoldFunds` is required (always sent); the fee is a percent string with at most 2 decimals
(`"0"` to `"50"`). The answer is a `CreatedInvitation`: the invitation's fields and its `Link`
(only here). QBitFlow emails the person a link of their own (another token, never returned), so
don't compare it with `Link`. `SendEmail: new(false)` creates the invitation without emailing it:
share `Link` in your own onboarding (signing up through it still emails the person a confirmation). An
organization sends at most 50 invitations an hour, emailed or not (`429`, raised at once).

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
currencies, err := client.Wallets.ListAcceptedCurrencies(ctx, &qbitflow.AcceptedCurrenciesParams{UserUUID: memberUUID})
if err != nil {
	log.Fatal(err)
}
if len(currencies) == 0 {
	log.Fatal("the seller cannot be paid yet: their checkouts would answer 409 merchant_not_ready")
}

seller := client.OnBehalfOf(memberUUID)
session, err := seller.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductName: "Handmade mug",
	PriceUSD:    "4.50",
	SuccessURL:  "https://market.example.com/orders/{{UUID}}",
})
if err != nil {
	log.Fatal(err) // 403 policy_disabled when your policies don't let members do this
}
fmt.Println(session.Link)
```

While you hold a seller's funds (`Member.TrustedAt` nil), they sell **one-time payments only**: a
subscription checkout in their space answers `409 merchant_not_ready` with `Details["reason"]`
`fundsHeld` (`qbitflow.MerchantNotReadyReasonFundsHeld`) until you trust them (`Members.Trust`),
and their subscription products' links open nothing.

**4. Hear of every sale.** An organization webhook endpoint receives every seller's events by
default; the event's `UserUUID` names the seller. Use it as `OnBehalfOf` for follow-up reads.

**5. Commission and held funds.** Your commission is taken on what remains after QBitFlow's fee:
`(amount − QBitFlow's fee) × OrganizationFeePercent / 100`, rounded down, the seller getting the
rest (on $7.50 at 0.5 % and 10 %: QBitFlow 0.0375, you 0.74625, the seller 6.71625). Each payment
records it in `Settlement.OrganizationFee` and `Settlement.Split.Organization`. While you hold a seller's funds (`HoldFunds` true,
`Member.TrustedAt` nil), their payments go to your wallet and the net is owed to them
(`HeldFunds.TotalUSD`, each `LedgerEntry` with what it counts for: `Owed`, `OwedMinUnits`,
`OwedUSD`, negative on a refund line):

<!-- docs:snippet members-held-funds -->
```go
// What the organization holds for its members (their payments while untrusted).
// The longest held first.
all, err := client.Members.ListHeldFunds(ctx)
if err != nil {
	log.Fatal(err)
}
for _, s := range all {
	fmt.Printf("%s: %s USD over %d lines\n", s.UserUUID, s.TotalUSD, s.Count)
}

held, err := client.Members.GetHeldFunds(ctx, memberUUID)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("owed to %s: %s USD over %d lines\n", memberUUID, held.TotalUSD, len(held.Ledgers))
```
<!-- /docs:snippet -->

Trust a seller to pay them directly. Trust is one-way and narrows the seller's currencies to their
own wallets': a seller whose wallets accept none of yours is a `409 member_has_no_wallet` (nothing
changes; trust them once they've added one):

<!-- docs:snippet members-trust -->
```go
// Their new payments go to their own wallets from now on (one-way). What is held stays
// held until you release it in the dashboard (heldFunds.released tells you).
member, err := client.Members.Trust(ctx, memberUUID)
var conflict *qbitflow.ConflictError
switch {
case errors.As(err, &conflict) && conflict.Code == "member_has_no_wallet":
	fmt.Println("not trusted: their own wallets accept none of your currencies yet")
case err != nil:
	log.Fatal(err)
case member.TrustedAt != nil:
	fmt.Println("trusted since", member.TrustedAt.Format(time.RFC3339))
}
```
<!-- /docs:snippet -->

Change the commission:

<!-- docs:snippet members-update -->
```go
// The organization's commission on their payments, a percent string ("0" to "50"; a
// checkout already created keeps its fee).
member, err := client.Members.Update(ctx, memberUUID, &qbitflow.UpdateMemberParams{
	OrganizationFeePercent: "10",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("fee now", member.OrganizationFeePercent, "%")
```
<!-- /docs:snippet -->

- `Members.List` / `Iterate` / `Get` (`FirstName`, `LastName`, `OrganizationFeePercent`, the
  `AcceptedCurrencies` their checkouts take now), `Members.ListHeldFunds` (every member owed, the
  longest held first, a slice), and `seller.Members.GetOwnHeldFunds` (the seller's side) complete
  the reads.
- `Members.Remove` ends a seller's membership in the key's mode: their keys stop working and their
  checkouts close (`204`). It is a `409 held_funds_pending` while you hold their live funds:
  release them from the dashboard first.
- `Invitations.List` / `Iterate` (by `Status`; member invitations only, your team's are managed in
  the dashboard) and `Invitations.Revoke` (`204`, returns only an error) manage the invitations.
- A seller who already uses QBitFlow joins your platform with the same account: accepting with an
  account that belongs to another organization succeeds like any acceptance, keeps its other
  memberships, and sends `member.joined` as for any member. Test-mode sellers are real accounts
  that accept from a real inbox.

## Wallets

Wallets are added and removed in the dashboard, by their owner only. The SDK reads them; every
wallet list is short and returned as a slice.

```go
wallets, err := client.Wallets.List(ctx, &qbitflow.WalletListParams{WithBalances: true})
if err != nil {
	log.Fatal(err)
}
for _, w := range wallets {
	fmt.Println(w.Currency.Chain, w.Address)
	for _, tw := range w.TokenWallets {
		if tw.Balance != nil { // nil when the chain couldn't be read
			fmt.Printf("  %s: %s (%s USD)\n", tw.Currency.Code, tw.Balance.Amount, tw.Balance.AmountUSD)
		}
	}
}
```

`Wallets.ListForMember(ctx, userUUID)` reads a member's wallets (organization key), and
`Wallets.ListAcceptedCurrencies` the currencies a space's checkouts accept now (`UserUUID`: a
member's, organization key): none means its checkouts answer `409 merchant_not_ready`.

## Accounting export

Every payment, bill, refund and fee between two dates (`YYYY-MM-DD`, both included), as JSON rows
(`ExportJSON(ctx, from, to)`, a `[]AccountingEvent`) or as CSV text (`ExportCSV`); the
[recipe](#a-yearly-accounting-export) exports a year.

The SDK checks the dates and `from <= to` before sending. **The API allows at most 92 days per
export, both days included** (`400 validation_failed` on `to` beyond, `Details["max"]` 92).
`ExportJSONRange` and `ExportCSVRange` take any range: they split it into consecutive windows of at
most 92 days (`[from, from+91d]`, the next one starting the day after, on calendar dates in UTC),
request them in order and return the events joined (for CSV, the first window's header line once,
then every window's rows). A range of 92 days or less is one request; `2026-01-01` to `2026-12-31`
is four: `[01-01, 04-02]`, `[04-03, 07-03]`, `[07-04, 10-03]`, `[10-04, 12-31]`. Rows are typed
`payment`, `bill`, `refund`, `organizationFee` or `referralFee`; a value a row doesn't have (a fee
that doesn't apply) is `""`, never `"0"`.

- **Dates** are the transactions' (the day their block was mined, in UTC: `TxTimeUTC`), so
  consecutive exports never miss nor repeat an event. Rows are ordered by `TxTimeUTC`, `Type`,
  then `TxUUID`; a refund names what it refunds in `RelatedTxUUID` and `RelatedReference`.
- **Amounts** are in the currency's units (`"7.5"` USDC, not `7500000`), as decimal strings, next
  to `CurrencyDecimals`, each with its on-chain twin (`GrossAmountMinUnits`, …). USD amounts
  (`GrossAmountUSD`, …) and rates (`PlatformFeePercent`, …) are decimal strings too. `Currency` names what was paid as markets do (`USDC`, or the chain's
  native currency: `ETH`, `SOL`), and `CurrencyPriceUSD` is the USD price the amounts were
  converted at. `GrossAmount = PlatformFee + OrganizationFee + NetAmount` exactly; a refund's net
  is the refunded amount, negative.
- **Network fees are never part of the net.** `NetworkFeesPaidBy` says whose wallet paid them:
  `payer` (a payment or a bill: on top of the amount, in its currency), `merchant` (a refund your
  space sent) or `organization` (a refund of a payment the organization held for its member). Book
  `NetworkFees` (in `NetworkFeesCurrency`, the chain's native currency on a refund) apart for
  `merchant` and `organization`.
- **CSV columns** are `AccountingEvent`'s JSON keys in its field order (what happened, the USD
  amounts, the same amounts in the currency, the network fees, the rates, the on-chain proof, the
  ids): read them by header name, not position. Amounts are plain numbers there, an absent value
  an empty cell, and a cell starting with `=`, `+`, `-`, `@`, a tab or a CR gets a leading `'`.
- A `referralFee` row only has what the referrer earned (its type, time, chain, block, currency
  and the referral fee): every other field is empty.

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
`IncludeMembers: new(false)`. `Endpoints.List` / `Iterate` (paginated), `Get`, `Update` (PATCH:
`Enabled: new(false)` pauses it, `new(true)` enables it again, `Description: new("")` clears it,
`Events: []qbitflow.EventType{}` receives every type) and `Delete` (`204`) manage them. `Delete`
also deletes the endpoint's delivery history: `Events.Get` no longer lists its deliveries. To keep
it, disable the endpoint instead. An
endpoint's secret is shown and rotated in the dashboard only. A replay of the create's
`Idempotency-Key` within 24 hours returns the same endpoint and secret.

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

`VerifyWebhook` only verifies; `ParseEvent` parses a body already verified. A client has the
same three as methods (`client.Webhooks.Verify`, `ConstructEvent`, `ParseEvent`). In a `net/http`
handler, `body, err := qbitflow.VerifyWebhookRequest(r, secret)` reads the request's body (at most
1 MiB) and checks its signature in one call, then `qbitflow.ParseEvent(body)` parses it.

- **At least once.** The same event can arrive more than once: **deduplicate on `Event.ID`** (also
  in the `QBitFlow-Event-Id` header) and make the handler idempotent.
- **Answer 2xx fast**, within 30 seconds, **including to the types you ignore**: anything else is
  retried (for 3 days in live mode), and an endpoint failing for 3 days is disabled. Store the
  event, answer, then process it in the background.
- **Order is per publication, never per subscription.** The events published together (a
  `subscription.created` and its first `subscription.billed`) reach an endpoint in that order,
  each once the previous one is delivered or abandoned; meanwhile the event log shows the waiting
  one `Pending` with `WaitingFor` the earlier event ([The event log](#the-event-log)). Nothing
  waits across publications: a renewal's `subscription.billed` never waits for an earlier event
  your endpoint is failing. Order those by `CreatedAt` and the resources' own timestamps.
- **Secret rotation** needs nothing on your side: for 24 hours after a rotation the header carries
  two `v1=` signatures, the new secret's and the previous one's, and either secret verifies. Switch
  your secret within the day.
- **Timestamps** more than 5 minutes from your clock are refused (replays):
  `qbitflow.WithTolerance(d)` changes it, `qbitflow.WithClock(f)` sets the clock in tests.
- **Members' events** carry the member in `Event.UserUUID`: read their resources with
  `client.OnBehalfOf(event.UserUUID)`.
- **No fixed source IPs**: verify the signature, never allow-list addresses.
- **v2 events only.** A body that is not a v2 event (v1 payloads are retired) is refused by
  `ParseEvent` with a `*ValidationError`.
- **Headers:** `QBitFlow-Signature`, `QBitFlow-Event-Id` and `QBitFlow-Event-Type`
  (`qbitflow.SignatureHeader`, `EventIDHeader`, `EventTypeHeader`).
- Not holding the secret? `client.Webhooks.VerifyRemote(ctx, endpointUUID, rawBody, header)` has the
  API check it, then `ParseEvent` parses the body.

| `Event.Type` | Accessor | Data |
|---|---|---|
| `payment.completed` | `PaymentCompleted()` | the `Payment` + `ManagementPageLink` |
| `subscription.created` | `SubscriptionCreated()` | the `Subscription` (`active` or `trial`, its `ManagementPageLink` included) |
| `subscription.billed` | `SubscriptionBilled()` | `{Bill, Subscription}`: the `Bill` paid (until `Bill.PeriodEnd`) and the `Subscription` as it leaves it (`Subscription.Status`, `.Reference`, `.Metadata`) |
| `subscription.statusChanged` | `SubscriptionStatusChanged()` | the `Subscription` + `PreviousStatus` |
| `subscription.actionRequiredChanged` | `SubscriptionActionRequiredChanged()` | the `Subscription` + `PreviousActionRequired` |
| `subscription.billingFailed` | `SubscriptionBillingFailed()` | the `Subscription` + `Reason`, `BillUUID`, `AmountUSD` (a string), `Attempt`, `RemainingAttempts`, `NextAttemptAt` |
| `subscription.upcomingBill` | `SubscriptionUpcomingBill()` | the `Subscription` + `BillingAt`, `AmountUSD` (a string), `TrialEnding`, `BalanceSufficient`, `AllowanceSufficient`. Sent the dashboard's `subscriptions.upcomingBillDays` before the bill (3 days by default, 0 = never), checked daily; not for a subscription whose period (a trial: the trial) is no longer than that |
| `refund.requested` / `.approved` / `.rejected` | `RefundRequested()` / `RefundApproved()` / `RefundRejected()` | the `Refund` |
| `member.joined` | `MemberJoined()` | the `Member` + `InvitationUUID` |
| `member.removed` | `MemberRemoved()` | the `Member` |
| `heldFunds.released` | `HeldFundsReleased()` | the transfer paid to the member (`Amount`, `AmountUSD`, `Settlement.Transaction`) + the `Ledgers` it settled |
| `checkout.expired` | `CheckoutExpired()` | the session: `.Payment` or `.Subscription` |
| `webhook.test` | `WebhookTest()` | `EndpointUUID`, `Message` (the dashboard's test) |
| any other | `DecodeData[T]()` | `Data`, raw |

Each accessor has its router method, `On` + its name (`OnPaymentCompleted`, `OnRefundApproved`…;
`OnUnknown` for the other types), whose handler gets the same data. An accessor called on another
type is a `*ValidationError`. Webhook data never carries what only API reads return (`Customer`,
`Refund`/`Refundable`/`NotRefundableReason`, a subscription's `Dunning`, a refund's `Approval`,
`CustomerUUID` and `ProductName`); it carries your `Metadata` and the `Settlement`.

### The event log

Every event of the space, newest first, each exactly as it was delivered (kept at least 90 days;
`Webhooks.Events.Iterate` walks every page). `Type` is sent as is, not checked against this SDK's
types (newer ones appear):

<!-- docs:snippet events-list -->
```go
page, err := client.Webhooks.Events.List(ctx, &qbitflow.EventListParams{
	Type:  qbitflow.EventTypePaymentCompleted,
	Limit: 20,
})
if err != nil {
	log.Fatal(err)
}
for _, e := range page.Items { // newest first, each exactly as delivered
	fmt.Println(e.ID, e.Type, e.CreatedAt.Format(time.RFC3339))
}
if page.HasMore {
	fmt.Println("more: pass", page.NextCursor, "as Cursor") // opaque, never an event id
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
	switch {
	case d.Delivered:
		fmt.Println(d.URL, "delivered after", len(d.Attempts), "attempts")
	case d.NextAttemptAt != nil:
		fmt.Println(d.URL, "failed, retried at", d.NextAttemptAt.Format(time.RFC3339))
	case d.WaitingFor != "":
		fmt.Println(d.URL, "queued behind", d.WaitingFor) // no attempt yet
	case d.Pending:
		fmt.Println(d.URL, "about to start") // no attempt yet
	default:
		fmt.Println(d.URL, "not delivered, gave up after", len(d.Attempts), "attempts")
	}
}
```

`Pending` is true while QBitFlow still attempts the event on its own: `NextAttemptAt` is set while
it is retried after a failed attempt, `WaitingFor` (an `evt_…`) while it is queued behind an earlier
event of the same publication, and neither while its delivery is about to start (within a minute).
**A queued or starting delivery has no `Attempts` yet**: check the length before reading the last
one. Once `Pending` is false nothing more happens on its own, and `Delivered` false means it was
abandoned after its retries (3 days in live mode, 3 hours in test mode) or the endpoint was
disabled. **A deleted endpoint's deliveries aren't listed at all**: they are deleted with it, the
successful ones included. To keep an endpoint's history, disable it (`Enabled: new(false)`) rather
than delete it. If QBitFlow can't read a delivery's progress at that moment, `Pending` comes from the
attempts: true while the retry schedule still attempts the event again (a delivery being retried
never reads as abandoned), without `NextAttemptAt`; a delivery queued with no attempt yet isn't
listed until its progress can be read: read the event again a moment later.

A member's deliveries to your organization's endpoints show from the organization: list with
`&qbitflow.EventListParams{IncludeMembers: true}`, then `Get`. Read on behalf of the member, the
event lists only the member's own endpoints (no `Deliveries` without any).

Each `DeliveryAttempt` has `Delivered`, `StatusCode` (`nil` when the endpoint didn't answer) and
`Error`. An attempt that was not posted at all is `Skipped`, with no `StatusCode`: a member's
endpoint while the organization's `members.webhooks` policy is off for them.

## Currencies

<!-- docs:snippet currencies-list -->
```go
// A public route limited to 60 requests a minute per IP: cache the list at start-up.
// One bounded list (a slice): the live currencies; Test: the test networks'; Main: the
// chains' native coins only.
currencies, err := client.Currencies.List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
for _, c := range currencies {
	fmt.Printf("%d: %s (%s on %s), %d decimals\n", c.ID, c.Code, c.Name, c.Chain, c.Decimals)
}
```
<!-- /docs:snippet -->

`Currencies.List` returns the catalog as a slice (`&qbitflow.CurrencyListParams{Test: true}`: the
test networks' currencies; `Main: true`: only the chains' native coins), and `Get` one by id (a
token's `MainCurrencyID`). A `Currency` has its `Code` (what requests name it by: `BASE:USDC`, or
`ETH` for a native coin), `Symbol`, `Name`, `Chain`, `Decimals`, and `Address` (absent on a native
coin). These routes are public and **limited to 60 requests a minute per IP: cache the list** at
start-up instead of reading it per request. Every response that names a currency carries this
object (`Payment.Currency`, `Member.AcceptedCurrencies`, …).

## Pagination and iterators

Paginated lists return a `*Page[T]`: `Items`, `HasMore` (another page follows), and `NextCursor`
(set iff `HasMore`: the value to pass back as the params' `Cursor`, verbatim; it is opaque, never
an id to build or parse, at most 200 characters).

<!-- docs:snippet customers-list -->
```go
page, err := client.Customers.List(ctx, &qbitflow.CustomerListParams{Limit: 20})
if err != nil {
	log.Fatal(err)
}
for _, c := range page.Items {
	fmt.Println(c.UUID, c.Email)
}
if page.HasMore {
	// Pass it back, verbatim, as CustomerListParams.Cursor for the next page.
	fmt.Println("next cursor:", page.NextCursor)
}
```
<!-- /docs:snippet -->

To walk the pages yourself, set `params.Cursor = page.NextCursor` and list again while
`page.HasMore`. Each paginated list also has an `Iterate` twin returning an
`iter.Seq2[T, error]`: it fetches one page at a time, only as you consume it, keeps your filters
and page size, and stops when you `break`, after the page whose `HasMore` is `false` (a full last
page answers `false`: there is never an extra empty request), or defensively on a page that claims
more without a new `NextCursor`. An error is yielded once, as `(zero, err)`, and ends the
sequence.

<!-- docs:snippet pagination-iterate -->
```go
// Fetches one page at a time, only as the loop consumes it; break stops the fetching.
// It stops after the page whose HasMore is false.
count := 0
for payment, err := range client.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
	if err != nil {
		log.Fatal(err)
	}
	count++
	fmt.Println(payment.UUID, payment.AmountUSD, "USD")
}
fmt.Println(count, "payments")
```
<!-- /docs:snippet -->

| Method | Iterator | Page size: default / max |
|---|---|---|
| `Products.List`, `Customers.List` | `Iterate` | 20 / 100 |
| `Payments.List`, `Transactions.List`, `Failures.List` | `Iterate` | 20 / 100 |
| `Subscriptions.List`, `Subscriptions.ListBills` | `Iterate`, `IterateBills` | 20 / 100 |
| `Refunds.List` | `Iterate` | 20 / 100 |
| `Members.List`, `Invitations.List` | `Iterate` | 20 / 100 |
| `Webhooks.Endpoints.List`, `Webhooks.Events.List` | `Iterate` | 20 / 100 |

`limit` is clamped by the API (0 or less: 20; above 100: 100). The bounded lists answer one page
and return its items as a plain slice: `Currencies.List`, `Wallets.List`, `Wallets.ListForMember`,
`Wallets.ListAcceptedCurrencies`, `Members.ListHeldFunds` and `Accounting.ExportJSON`.

## Errors

Every error the SDK returns is one of the types below, matched with `errors.As`. Each embeds an
`*APIError`, so an `*APIError` target matches any of them:

| Type | When | Codes and fields worth knowing |
|---|---|---|
| `*ValidationError` | 400 `validation_failed`, or input the SDK refused **before sending** (`StatusCode` 0) | `FieldErrors`: each failing input by its wire name, dotted when nested (`frequency.unit`) |
| `*BadRequestError` | any other 400 | `foreign_key_violation` (`Details["field"]`), `test_mode_only`, `sender_is_receiver` |
| `*AuthenticationError` | 401 | a missing, unknown, expired or revoked key (a removed member's keys too) |
| `*PermissionDeniedError` | 403 | `forbidden` (no details: the key's role or the resource doesn't allow it; also a bad `On-Behalf-Of`), `policy_disabled`: a member's key doing what its organization's member policy doesn't allow (`Details["policy"]`, e.g. `members.subscriptions`; plus `Details["field"]` when the policy governs a request field: `successUrl` / `cancelUrl` under `members.checkoutRedirects`) |
| `*NotFoundError` | 404 | unknown, deleted, or outside the request's space; also a `GetByReference` that found nothing (code `not_found`, the list's `RequestID`) |
| `*ConflictError` | 409 | `unique_violation` (`Details["field"]`), `tx_already_sent`, `merchant_not_ready` (`Details["reason"]`), `held_funds_pending`, `member_has_no_wallet`, `already_joined`, `payment_not_due`, `subscription_already_stopped_or_inactive`, `idempotency_key_in_use`, `conflict` |
| `*GoneError` | 410 | `merchant_closed`: a removed member's or a closed organization's space |
| `*IdempotencyError` | 422 `idempotency_key_reused` | an `Idempotency-Key` reused for another request: a bug, never retried |
| `*RateLimitError` | 429 | `RetryAfter` (a `time.Duration`), `Limit`, `PeriodSeconds` |
| `*ServerError` | 5xx (503 `network_unavailable`, 504 `timeout`), an unexpected 3xx (redirects are never followed), a 2xx whose body is not the expected JSON | |
| `*NetworkError` | no response: DNS, connection, TLS, timeout, cancelled context | `Err`: `errors.Is(err, context.DeadlineExceeded)` works through it |
| `*WebhookSignatureError` | a webhook signature refused, locally or by `VerifyRemote` | `Reason` |
| `*APIError` | any other HTTP error | 413 `request_too_large`; 405 `method_not_allowed` (a path called with a method it doesn't take, the `Allow` header listing its methods: the SDK never does that, so it points at a proxy or a wrong base URL) |

`*APIError` carries `StatusCode`, `Code` (branch on it, never on `Message`), `Message`, `Details`
(never nil, and raw: money in it is a string, `Details["max"] == "5"`, but the accounting window's
`max` is the integer 92), `RequestID`, `FieldErrors`, `RawBody` and `Err`. Its message reads
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
(`A-Z a-z 0-9 . _ : @ -`, 1 to 100, never exactly `.` or `..`; on creates, `ProductReference`,
`CustomerReference`, the `Reference` filters and `GetByReference`),
emails (the customers' `Email` filter too), phone numbers, absolute `http(s)` URLs, USD amounts
(decimal strings with at most 2 decimals, above 0, at most 1,000,000), percents (decimal strings,
at most 2 decimals: `OrganizationFeePercent` 0 to 50),
metadata (keys, values, at most 50 keys), durations (a frequency between 1 unit and 1 year), UUIDs
and transaction ids, dates, cursors (at most 200 characters), the checkout's product choice, the
enum filters (refund `Status`, transaction `Type`, failure `Kind` and `Category`, invitation
`Status`), and exclusive filters. A failure is a `*ValidationError` with `StatusCode` 0, `Code`
`""` and nothing is sent. What depends on the key's mode or on stored data is left to the API: the
`"5"` USD test-mode cap, `https`-only live URLs, the live subscription minimum (`"0.25"`), the
frequency minimum (1 hour live, 5 minutes test; a development server lowers the live one to 1
minute, production never does), the 92-day export window, the 50-key cap on merged
metadata, uniqueness.

## Retries and idempotency

| | |
|---|---|
| Retried methods | every read (GET), and the 6 creates: `CheckoutSessions.CreatePayment`, `CheckoutSessions.CreateSubscription`, `Products.Create`, `Customers.Create`, `Webhooks.Endpoints.Create`, `Invitations.Create` |
| Never retried | every other write: updates (`Subscriptions.Update` included), deletes, `CheckoutSessions.Expire`, `Subscriptions.Cancel`, `Subscriptions.TestBill`, `Members.Trust`, `Members.Remove`, `Invitations.Revoke`, `Webhooks.VerifyRemote`; a 429 on them is returned, not retried |
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
	PriceUSD:    "4.99",
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
| `WithBaseURL(url)` | `https://api.qbitflow.app/v2` (`qbitflow.DefaultBaseURL`) | an absolute `http(s)` URL with a host, without credentials (`user:pass@`), query or fragment; a trailing `/` is stripped |
| `WithTimeout(d)` | 30 s (`DefaultTimeout`) | per attempt, through the last byte of the answer's body; positive |
| `WithMaxRetries(n)` | 3 (`DefaultMaxRetries`) | `0` disables retries; negative is refused |
| `WithHTTPClient(c)` | a plain `*http.Client` | the SDK uses a copy that never follows redirects |
| `WithOnBehalfOf(userUUID)` | none | every request acts in that member's space |

| Request option (any method, last arguments) | |
|---|---|
| `WithOnBehalfOf(userUUID)` | acts in that member's space for this call; `""` forces the organization's (no header) |
| `WithIdempotencyKey(key)` | the 6 creates only: your own `Idempotency-Key` |
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
`User-Agent: qbitflow-go/3.1.1` (`qbitflow.Version`).

## Migrating from 2.x

3.0.0 is a rewrite for API v2: module `github.com/QBitFlow/qbitflow-go-sdk/v3`, base URL `/v2`,
one package, `context.Context` on every call, UUID ids, REST routes, money as decimal strings,
members and invitations instead of users and claims, a new webhook signature, typed errors. **[MIGRATION-v3.md](MIGRATION-v3.md)** maps
every 2.x method and type to its replacement, with before/after code for the common tasks.

## Examples

Runnable programs in [`examples/`](examples) (`QBITFLOW_API_KEY=sk_… go run ./examples/<name>`):

| Example | Shows | Environment (besides `QBITFLOW_API_KEY`, `QBITFLOW_BASE_URL`) |
|---|---|---|
| [`client`](examples/client) | a client from a key and `Me`, a client from the environment | `QBITFLOW_ON_BEHALF_OF` |
| [`checkout`](examples/checkout) | a payment checkout (and one with fees), its status, `WaitForCompletion`, expiry | |
| [`catalog`](examples/catalog) | a product, the products, a checkout for a product, a page of customers | |
| [`payments`](examples/payments) | a filtered page of payments, the iterator, exact amounts, a payment by id and by reference | `PAYMENT_UUID` |
| [`refunds`](examples/refunds) | the pending refunds, one refund by id | |
| [`subscriptions`](examples/subscriptions) | a subscription checkout with a trial, a subscription, its bills, its metadata, `HasAccess`, a test bill, cancel at period end | `SUBSCRIPTION_UUID`, `TEST_BILL=1`, `CANCEL=1` |
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

The offline gates, as CI runs them (no network, no key):

```bash
go build ./... && go vet ./... && test -z "$($(go env GOROOT)/bin/gofmt -l .)" && go test -race -count=1 ./...
```

`go test ./...` runs the offline suite against in-process stub servers: requests, retries,
decoding, validation, webhook signatures and events. The snippet check (`TestSnippets`) belongs to
the maintainers' conformance suite and skips itself outside their workspace.

The live checks are behind the `integration` build tag and need an API key **and** an explicit
base URL (a key without a base URL fails; neither skips them):

```bash
QBITFLOW_API_KEY=sk_… QBITFLOW_BASE_URL=https://… go test -tags integration -run Live -v .
```

The read-only group only reads. The write group also needs `QBITFLOW_LIVE_WRITES=1` and a
test-mode key (`QBITFLOW_ALLOW_LIVE_MODE_WRITES=1` allows a live-mode key, for a disposable server
only); it deletes what it creates. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

This project is licensed under the MPL-2.0 License: see the [LICENSE](LICENSE) file. See also
[COMPLIANCE.md](COMPLIANCE.md) and the [trademark policy](TRADEMARKS.md).

## Support

- [Documentation](https://qbitflow.app/docs)
- [Email support](mailto:support@qbitflow.app)
- [Issue tracker](https://github.com/QBitFlow/qbitflow-go-sdk/issues) (not for security issues)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Security

For security issues, please email security@qbitflow.app instead of using the issue tracker (see
[SECURITY.md](SECURITY.md)).
