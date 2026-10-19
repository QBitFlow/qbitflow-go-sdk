# Migrating from 2.x to 3.0.0

3.0.0 is the Go SDK for **QBitFlow API v2**. API v2 reorganises the platform around spaces
(an organization's, and one per member), invites people instead of provisioning them, names every
resource by a UUID, uses REST routes, writes money as decimal strings and signs webhooks with a new
scheme. The SDK follows: new module path, one package, a new client, and new names for most
methods.

2.x talks to `/v1` only; 3.0.0 talks to `/v2` only, and **`/v1` is gone**: the v2 launch
migrated v1's data (people, products, payments, subscriptions, customers, wallets, webhook URLs) to
v2's shape, so move your integration to 3.0.0 as a whole.

This guide maps everything 2.1.0 exposed (the last 2.x release) to 3.0.0. The
[README](README.md) documents 3.0.0 in full; [CHANGELOG.md](CHANGELOG.md) lists every change.

## Checklist

1. [Module path and imports](#1-module-path-and-imports): `.../v3`, one package, Go 1.27.
2. [Base URL and API keys](#2-base-url-and-api-keys): `/v2`; pre-v2 keys are refused: create new ones.
3. [The client](#3-the-client): `NewClient(key, opts...)`, options instead of `Config` and setters.
4. [`context.Context` and request options](#4-contextcontext-and-request-options) on every call.
5. [Method map](#5-method-map): every 2.x method and its replacement.
6. [Ids are UUID strings](#6-ids-are-uuid-strings): products, members, `On-Behalf-Of`.
7. [Users and claims become invitations, members and the trust layer](#7-users-and-claims-become-invitations-members-and-the-trust-layer).
8. [Models](#8-models): renamed types and fields, money and fee rates as decimal strings.
9. [Enum values and statuses](#9-enum-values-and-statuses): subscription and checkout statuses.
10. [Errors](#10-errors): one typed error per status, `errors.As`, field errors that work.
11. [Webhooks](#11-webhooks): `QBitFlow-Signature` over the raw body, typed events, v2 endpoints.
12. [Pagination](#12-pagination): params structs, `Page{Items, HasMore, NextCursor}` and iterators.
13. [Before and after](#13-before-and-after-five-common-tasks): five common tasks.

## 1. Module path and imports

```bash
go get github.com/QBitFlow/qbitflow-go-sdk/v3
```

| 2.x import | 3.0.0 |
|---|---|
| `github.com/QBitFlow/qbitflow-go-sdk/v2` (package `qbitflow`) | `github.com/QBitFlow/qbitflow-go-sdk/v3` (package `qbitflow`) |
| `.../v2/pkg/qbitflow` (`qbf`), `.../v2/pkg/models` (`qbmodels`), `.../v2/pkg/errors` (`qberrors`), `.../v2/pkg/utils` | gone: everything is in the root package |

So `qbf.CreateSessionOptions`, `qbmodels.Payment` and `qberrors.NotFoundError` all become
`qbitflow.Something` (names in § 5 and § 8). 3.0.0 requires **Go 1.27** (it uses generic
methods) and still has no dependencies.

## 2. Base URL and API keys

- The default base URL is `https://api.qbitflow.app/v2`. If you set one, it must point at v2.
- API v2 issues keys shaped `sk_<uuid>_<test|live>_<secret>`. **Keys issued before**
  (`sk_<digits>_…`) **are refused by API v2** (`401`): create a new key in the dashboard. Treat a key
  as opaque (the SDK only checks the `sk_` prefix).
- `NewClient` refuses a blank key, or one not starting with `sk_`, with a `*ValidationError`
  (2.x sent it and got a 401). It sends nothing: call `client.Me(ctx)` to check the key online,
  and assert its mode (`me.Space.Test`) at start-up.
- A key belongs to one space and one mode. A removed member's keys answer 401.

## 3. The client

| 2.x | 3.0.0 |
|---|---|
| `qbitflow.New(apiKey) *QBitFlow` | `qbitflow.NewClient(apiKey, opts...) (*qbitflow.Client, error)` |
| `qbitflow.NewWithConfig(qbf.Config{APIKey, BaseURL, Timeout})` | `qbitflow.NewClient(apiKey, qbitflow.WithBaseURL(u), qbitflow.WithTimeout(d))` |
| `client.SetBaseURL(u)`, `client.SetTimeout(d)` | the same options, at construction: a client never changes afterwards |
| `qbf.NewClient`, `qbf.NewClientWithConfig`, `qbf.New<Name>Service(client)` | removed: services are the client's fields (`client.Products`, `client.Webhooks.Endpoints`…) |
| `Config.Timeout` (the whole call) | `WithTimeout(d)`: each attempt (default 30 s); bound the whole call with the context |
| — | `WithMaxRetries(n)` (default 3, `0` disables), `WithHTTPClient(c)`, `WithOnBehalfOf(userUUID)` |
| `client.<Service>.OnBehalfOf(userID uint64)` | `client.OnBehalfOf(userUUID)` (a client for every service), or the `qbitflow.WithOnBehalfOf(userUUID)` request option |

Reads and creates are now retried automatically (§ [Retries](README.md#retries-and-idempotency)):
drop any retry loop you wrapped around them, or set `WithMaxRetries(0)`.

## 4. `context.Context` and request options

Every method takes a `context.Context` first, and optional `RequestOption`s last:

```go
payment, err := client.Payments.Get(ctx, "pay@0192f1c2-2222-7c4d-9e5f-6a7b8c9d0e1f", nil,
	qbitflow.WithOnBehalfOf("0192f1c2-7b3a-7c4d-9e5f-6a7b8c9d0e1f"), // a member's space, this call only
	qbitflow.WithRequestID("support-ticket-981"),                    // X-Request-Id, echoed in errors
)
if err != nil {
	log.Fatal(err)
}
fmt.Println(payment.AmountUSD)
```

`WithIdempotencyKey(key)` sets the `Idempotency-Key` of a create (see the README). Cancelling the
context cancels the request and any wait between retries.

## 5. Method map

`ctx` is omitted below; every 3.0.0 method takes it first. Params named `…Params` are
`*qbitflow.…Params` and may be `nil` when every field is optional.

### Checkout sessions and status (2.x `Payments`, `Subscriptions`, `TransactionStatus`)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.CreateSession(*qbf.CreateSessionOptions)` | `CheckoutSessions.CreatePayment(*CreatePaymentSessionParams)` | `POST /checkout-sessions/payment`; plain values instead of pointers; `ProductID` → `ProductUUID`; `Price` → `PriceUSD` (a string: `"4.99"`); new `ExpiresInMinutes`, `Metadata` and `Fees`; returns `*CreatedCheckoutSession` (`Link`, `UUID`, `ExpiresAt`) |
| `Subscriptions.CreateSession(*qbf.CreateSubscriptionSessionOptions)` | `CheckoutSessions.CreateSubscription(*CreateSubscriptionSessionParams)` | `POST /checkout-sessions/subscription`; `Frequency`/`TrialPeriod` are `*Duration`, `MinPeriods` `*uint32`, each optional over a subscription product's terms |
| `TransactionStatus.GetTransactionStatus(uuid, txType)` | `CheckoutSessions.Get(uuid)` | `GET /checkout-sessions/:uuid`; no type argument; four statuses (§ 9) |
| `TransactionStatus.GetTransactionStatusWSURL(…)` | removed | the WebSocket is the checkout page's: use webhooks, or poll `Get` (`WaitForCompletion`) |
| `Payments.GetSession(uuid)`, `Subscriptions.GetSession(uuid)` | removed | `Get` for the state; the session's data comes with `checkout.expired` |
| — | `CheckoutSessions.Expire(uuid)` | new: end an unpaid session |

### Payments

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.GetPayment(uuid)` | `Payments.Get(uuid, *ReadParams)` | `pay@…` or the bare UUID |
| `Payments.GetPaymentByReference(ref)` | `Payments.GetByReference(ref)` | runs `GET /payments?reference=`; a `*NotFoundError` when none |
| `Payments.GetAllPayments(limit, cursor)` | `Payments.List(*PaymentListParams)`, `Payments.Iterate` | filters: customer, product, dates, `Reference`, `Refunded`, `IncludeMembers` / `UserUUID` |
| `Payments.GetAllCombinedPayments(limit, cursor)` | `Transactions.List(*TransactionListParams)`, `Transactions.Iterate` | `GET /transactions`; `CombinedPayment` → `Transaction`; `Source` → `Type` (`TransactionKindPayment`, `TransactionKindBill`); + `SubscriptionUUID` |
| `Payments.GetCustomerForTransaction(txUUID)` | removed | `Payment.Customer` (a summary), or `Customers.Get(payment.CustomerUUID)` |
| — | `Failures.List`, `Failures.Iterate` | new: the failed payment attempts |

### Subscriptions (and 2.x `PayAsYouGo`)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Subscriptions.GetSubscription(uuid)` | `Subscriptions.Get(uuid, *ReadParams)` | cancelled subscriptions are returned too: a 404 no longer means "cancelled" |
| `Subscriptions.GetSubscriptionByReference(ref)` | `Subscriptions.GetByReference(ref)` | runs `GET /subscriptions?reference=`; a `*NotFoundError` when none |
| `Subscriptions.GetPaymentHistory(uuid)` | `Subscriptions.ListBills(uuid, *BillListParams)` / `IterateBills` | `GET /subscriptions/:uuid/bills`: every full bill, page by page (the customer's public 10-bill history is not part of the SDK) |
| `Subscriptions.ForceCancel(uuid)` | `Subscriptions.Cancel(uuid, *CancelSubscriptionParams)` | `POST /subscriptions/:uuid/cancel`, never retried; returns `*SubscriptionCancellation{Subscription, Pending}` (`Pending`: HTTP 202, still confirming on-chain); `Immediate: new(false)` cancels at the end of the paid period |
| `Subscriptions.ExecuteTestBillingCycle(uuid)` | `Subscriptions.TestBill(uuid)` | `POST /subscriptions/:uuid/test-bill`; returns a `*TestBill` (`BillUUID` `sub-hist@…`, `Stage`, `Outcome`); `409 payment_not_due` before `NextBillingAt`, `400 test_mode_only` for a live one |
| — | `Subscriptions.List`, `Iterate`, `Update` (metadata, merged), `GetBill` (`GET /bills/:uuid`) | new |
| `PayAsYouGo.*` (`GetSession`, `GetSubscription`, `GetSubscriptionByReference`, `GetPaymentHistory`, `ForceCancel`, `ExecuteTestBillingCycle`) | removed | the API has no pay-as-you-go routes |

### Refunds

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Refunds.GetAll()` | `Refunds.List(&RefundListParams{Status: []RefundStatus{RefundStatusPending}})`, `Refunds.Iterate` | `GET /refunds`, paginated; from the organization's space the members' are included by default (`IncludeMembers: new(false)` leaves them out) |
| `Refunds.GetAllInactive(limit, cursor)` | `Refunds.List(&RefundListParams{Status: []RefundStatus{RefundStatusApproved, RefundStatusRejected}})`, `Refunds.Iterate` | one list, filtered by `Status` (repeatable, up to 5) |
| `Refunds.GetByTransactionUUID(txUUID)` | `Refunds.List(&RefundListParams{TxUUID: txUUID})` | or `Payment.Refund` / `Bill.Refund` (a `RefundSummary`) |
| — | `Refunds.Get(refundUUID)` | new: `GET /refunds/:uuid` |

### Customers and products

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Customers.Create(*qbf.CreateCustomer)` | `Customers.Create(*CreateCustomerParams)` | `POST /customers`; `Name` → `FirstName`; optional fields are plain strings (`""` = absent); + `Metadata` |
| `Customers.Get`, `Delete` | same names | `ctx` first; `Delete` answers `204` |
| `Customers.GetByReference(ref)` | `Customers.GetByReference(ref)` | runs `GET /customers?reference=`; a `*NotFoundError` when none |
| `Customers.GetByEmail(email)` | `Customers.List(&CustomerListParams{Email: email})` | removed: an email may match several customers |
| `Customers.Update(uuid, *qbf.UpdateCustomer)` | `Customers.Update(uuid, *UpdateCustomerParams)` | `PATCH`; `FirstName`, `Email` are strings (`""` = unchanged); `LastName`, `PhoneNumber`, `Address` are `*string`, and `new("")` **clears** them; `Metadata` is merged |
| `Customers.GetAll(limit, cursor)` | `Customers.List(*CustomerListParams)`, `Customers.Iterate` | filters `Email`, `Reference`, `Verified` |
| `Products.Create(*qbf.CreateProduct)` | `Products.Create(*CreateProductParams)` | `POST /products`; `Price` → `PriceUSD` (a string); `Description` optional; `Reference` a string; `Subscription` terms make a subscription product; + `Metadata` |
| `Products.Get(id uint64)` | `Products.Get(uuid string)` | `GET /products/:uuid` |
| `Products.GetAll()` | `Products.List(*ProductListParams)`, `Products.Iterate` | paginated now; `IncludeHidden`, `Subscription`, `Reference` |
| `Products.GetByReference(ref)` | `Products.GetByReference(ref)` | runs `GET /products?reference=` (hidden ones included); a `*NotFoundError` when none |
| `Products.Update(id, *qbf.UpdateProduct)` | `Products.Update(uuid, *UpdateProductParams)` | `PATCH`; `Name`, `PriceUSD` strings (`""` = unchanged); `Description`, `IsActive` pointers; `Subscription`, `RemoveSubscription`, `Metadata` (merged) |
| `Products.Delete(id)` | `Products.Delete(uuid)` | `204` |
| `CreateProduct.Validate()`, `UpdateProduct.Validate()` | `CreateProductParams.Validate()`, … | every params type has `Validate`, run before each request |

### Users, claims and API keys (→ § 7)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Users.Create(*qbf.CreateUser)` | `Invitations.Create(*CreateInvitationParams)` | not a rename: the person exists once they accept (`member.joined`) |
| `Users.GetAll()` | `Members.List(*MemberListParams)`, `Members.Iterate` | |
| `Users.GetByID(id)` | `Members.Get(userUUID)` | |
| `Users.GetByEmail(email)` | removed | iterate `Members` and match `Member.Email` |
| `Users.Update(id, *qbf.UpdateUser)` | `Members.Update(userUUID, *UpdateMemberParams)` | `PATCH`; only the organization fee (`OrganizationFeePercent`, a required string): names and emails are the person's own |
| `Users.Delete(id)` | `Members.Remove(userUUID)` | ends the membership (`204`; `409 held_funds_pending` while you hold their funds) |
| `Users.Get()` | `client.Me()` | what the key is: its role, space and mode |
| `Claims.CreateClaimRequest(userID)`, `Claims.GetClaimRequestByUserID(userID)` | `Invitations.Create`, `Invitations.List` | the invitation link replaces the claim link |
| `Claims.GetClaimFunds()` | `Members.ListHeldFunds()` (a `[]MemberHeldFunds`), `Members.GetHeldFunds(userUUID)`, `Members.GetOwnHeldFunds()` (`GET /held-funds`) | |
| `Claims.TriggerTestClaimFunds(userID)` | removed | release held funds in the dashboard; `Members.Trust(userUUID)` pays new payments directly |
| `ApiKeys.GetAll()`, `ApiKeys.GetForUser(userID)` | removed | keys are managed in the dashboard; `client.Me()` describes the current one |
| — | `Invitations.Revoke` (`204`: returns only an error), `Invitations.Iterate`, `Members.Trust` | new |

### Wallets, accounting, currencies

| 2.x | 3.0.0 | Notes |
|---|---|---|
| — | `Wallets.List(*WalletListParams)`, `Wallets.ListForMember(userUUID)` (`GET /members/:userUuid/wallets`), `Wallets.ListAcceptedCurrencies(*AcceptedCurrenciesParams)` (`GET /wallets/accepted-currencies`) | new; each returns a slice |
| `Accounting.Export(qbf.AccountingExportParams{From, To})` | `Accounting.ExportJSON(from, to)` | dates `YYYY-MM-DD`; the API allows 92 days at most, both included (`ExportJSONRange` for more) |
| — | `Accounting.ExportCSV(from, to)`, `ExportJSONRange`, `ExportCSVRange` | the same export as CSV text; any range in 92-day windows |
| `Currencies.GetAllAvailable(test)` | `Currencies.List(&CurrencyListParams{Test: test})` | `GET /currencies`, a slice |
| `Currencies.GetAllMain(test)` | `Currencies.List(&CurrencyListParams{Test: test, Main: true})` | |
| — | `Currencies.Get(id)` | new: `GET /currencies/:id` |

### Webhooks (→ § 11)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `qbf.VerifyWebhookRequest(r, secret, *qbf.VerifyOptions)` | `qbitflow.VerifyWebhookRequest(r, secret, ...VerifyOption)` | reads `QBitFlow-Signature`; at most 1 MiB; puts the body back on `r.Body` |
| `qbf.VerifyWebhookSignature(secret, timestamp, signature, payload, opts)`, `Webhooks.VerifyLocal(…)` | `qbitflow.VerifyWebhook(rawBody, signatureHeader, secret, ...VerifyOption)` (also `client.Webhooks.Verify`) | returns `nil` or a `*WebhookSignatureError` |
| `Webhooks.Verify(payload, signature, timestamp) (bool, error)` | `Webhooks.VerifyRemote(endpointUUID, rawBody, signatureHeader) error` | `nil` = valid; a mismatch is a `*WebhookSignatureError` |
| — | `qbitflow.ConstructEvent`, `qbitflow.ParseEvent` | verify and parse into a typed `*Event` |
| `qbf.ComputeWebhookSignature`, `qbf.CanonicalJSON` | removed | the signature covers the raw body |
| `qbf.ExtractWebhookHeaders`, `qbf.WebhookHeaders`, `WebhookHeaders.IsTest()` | removed | `r.Header.Get(qbitflow.SignatureHeader)`; the dashboard's test is a `webhook.test` event |
| `HeaderSignature`, `HeaderTimestamp`, `HeaderWebhookID`, `Webhooks.Get…Header()` | `qbitflow.SignatureHeader`, `EventIDHeader`, `EventTypeHeader`, `WebhookVersionHeader` | |
| `TEST_WEBHOOK_ID` | `qbitflow.EventTypeWebhookTest` | |
| `qbf.VerifyOptions{MaxTimestampAge, Now, SkipTimestampCheck}`, `DefaultMaxTimestampAge` | `qbitflow.WithTolerance(d)`, `qbitflow.WithClock(f)`, `DefaultWebhookTolerance` | the timestamp check can no longer be skipped |
| `qbmodels.SessionWebhookResponse`, `qbmodels.SubscriptionStatusTransitionWebhook` | `*qbitflow.Event` and its accessors | § 11 |
| — | `Webhooks.Endpoints.*` (`List` / `Iterate` paginated, `Create`, `Get`, `Update` (PATCH), `Delete`), `Webhooks.Events.*` (`List`, `Iterate`, `Get`) | new: endpoints and the event log |

### Errors and helpers (→ § 10)

| 2.x | 3.0.0 |
|---|---|
| `qberrors.QBitFlowError` | `*qbitflow.APIError` (and the `qbitflow.QBitFlowError` interface every error implements) |
| `qberrors.NotFoundError`, `qberrors.ValidationError`, `qberrors.ValidationErrors` | `*qbitflow.NotFoundError`, `*qbitflow.ValidationError` (with `FieldErrors`) |
| `qberrors.NewQBitFlowError`, `NewNotFoundError`, `NewValidationError`, `NewValidationErrorFromList` | removed |
| `qbf.CursorQueryBuilder`, `qbf.ValidateProductTextForTest` | removed |
| `qbmodels.SessionCheckout.IsPayment()` / `IsSubscription()` | removed: `CheckoutExpired.Payment` / `.Subscription` |

## 6. Ids are UUID strings

Every id but a currency's is a UUID string now, and no resource names its organization:

- **Products:** `Product.ID uint64` → `Product.UUID string`; `ProductID` → `ProductUUID` on payments,
  bills, subscriptions, sessions and the accounting export; checkouts take `ProductUUID`.
- **Members and `On-Behalf-Of`:** a member is named by their **user UUID** (`Member.UserUUID`).
  `OnBehalfOf(123)` becomes `OnBehalfOf("0192f1c2-…")`; a numeric value is refused. `UserID` →
  `UserUUID` everywhere (`""` for the organization's own rows).
- **`OrganizationID` is gone** from every model; `Me().Space.OrganizationUUID` names the organization.
- **Transactions** keep their prefixed ids (`pay@…`, `sub@…`, `sub-hist@…`, `refund@…`,
  `transfer@…`, `release@…`), as opaque strings; every route also takes the bare UUID. `TxID` →
  `TxUUID`, `TransactionHash` → `TxHash`.
- **JSON keys** are lowerCamelCase with acronyms as words (`customerUuid`, `subscriptionUuid`,
  `txHash`); only matters if you stored raw responses.

**Map the numeric ids you stored**: walk your products
(`Products.Iterate(ctx, &qbitflow.ProductListParams{IncludeHidden: true})`, in each space you act in)
and match them on `Reference`, which did not change, and your members (`Members.Iterate`, in each
mode) on `Email`; store the UUIDs next to your old ids. The SDK does not expose the API's
temporary `legacyId` field.

## 7. Users and claims become invitations, members and the trust layer

v1 provisioned sellers (`Users.Create`) and could sell for them at once, the seller claiming the
account and its funds later. **v2 creates no accounts: a seller sells only once they accepted an
invitation.**

1. `Invitations.Create(ctx, &qbitflow.CreateInvitationParams{Email, HoldFunds, OrganizationFeePercent, RedirectURL})`
   returns a `*CreatedInvitation`: the invitation's fields and its `Link` (also emailed).
2. Wait for the `member.joined` webhook and store its `UserUUID` (match `InvitationUUID`).
3. Sell with `client.OnBehalfOf(userUUID)`.

`HoldFunds: true` (the v1 "claim" model; always sent) holds the seller's payments in your wallet until you
trust them: `Members.GetHeldFunds` shows what you owe, `Members.Trust` makes new payments go to
the seller directly, and the release of what is held is signed in the dashboard. Sellers v1
provisioned are already members; those who never claimed are members whose funds you hold
(`TrustedAt == nil`). The [README](README.md#marketplaces) walks the flow.

## 8. Models

**Money is a decimal string, never a float**, in requests and in responses: USD amounts
(`PriceUSD: "4.99"`, `AmountUSD`, `PaidUSD`, `TotalUSD`), token amounts in the token's units
(`Amount: "10"` for 10 USDC, next to `AmountMinUnits: "10000000"` and the `Currency` object) and
fee rates in **percent** (`FeeBps: 150` → `FeePercent: "1.5"`, `OrganizationFeeBps: 250` →
`OrganizationFeePercent: "2.5"`, `RefundPercent: "50"`). Inputs are validated before sending (at
most 2 decimals, no sign or exponent); a JSON number would be refused by the API. Use a decimal
library, or `FormatAmount` / `ParseAmount`, to compute with them.

| 2.x (`qbmodels`) | 3.0.0 (`qbitflow`) | What changed |
|---|---|---|
| `LinkResponse` | `CreatedCheckoutSession` | `Link`, `UUID`, `ExpiresAt` (always set) |
| `TransactionStatus` | `CheckoutSessionStatus` | `Status` is a `CheckoutSessionStatusValue`; + `UUID`, `TxHash`, `Message`, `LastAttempt`, `ExpiresAt` |
| `SessionCheckout` | removed | `PaymentSessionData` / `SubscriptionSessionData` in `checkout.expired` (`PriceUSD`, `Fees`, `AmountUSD`, `AvailableCurrencies`, `Metadata`) |
| `Payment` | `Payment` | `Amount` is now the **token amount** (a string), its USD value `AmountUSD`; `TransactionHash` → `TxHash`; `Name` / `Description` → `ProductName` / `ProductDescription`; `ProductID *uint64` → `ProductUUID string`; `Reference *string` → `string`; `CurrencyID` → the `Currency` object (a value); `Metadata` (the fee split) → `Settlement`, and `Metadata` is now your checkout's `map[string]string`; `UserID` → `UserUUID`; + `PriceUSD`, `Fees`, `Chain`, `ExplorerURL`, `CustomerReference`, `Customer`, `Refund`, `Refundable`, `Paid`, `PaidMinUnits`, `PaidUSD`, `ConfirmedAt` |
| `CombinedPayment` | `Transaction` (`Transactions.List`) | `Source` → `Type` (`TransactionKind`: `payment`, `bill`); the `Payment` renames; + `PriceUSD`, `Fees`, `PeriodStart`, `PeriodEnd`, `Settlement`, `Paid*`, `Metadata` |
| `Subscription` | `Subscription` | `SubscriptionStatus` → `Status`; `Stopped bool` → the status `stopped`; `Frequency uint32` (seconds) → `Duration`; `ProductID` → `ProductUUID`; `Reference *string` → `string`; `LastBillingDate` → `LastBilledAt`, `NextBillingDate` → `NextBillingAt`, `MinimumCancellationDate` → `CancellableAt` (all `*time.Time`, nil when absent); `Allowance` (min units) and `MaxAmountPerPeriod` → `Allowance *Allowance` (`Amount`, `AmountMinUnits`, `AmountUSD`, `RemainingPeriods`, `MaxPerPeriod`, `MaxPerPeriodMinUnits`); `CurrencyID` → `Currency`; + `CurrentPeriodEnd`, `ActionRequired`, `PriceUSD`, `Fees`, `CancelledAt`, `CancellationReason`, `Metadata`, `ManagementPageLink`, `CustomerReference`, `Customer`, `Dunning` |
| `SubscriptionHistory` | `Bill` | the `Payment` renames (no `Metadata`, no `PriceUSD`: read the subscription's); + `Settlement`, `PeriodStart`, `PeriodEnd` |
| `RefundEntry` | `Refund` | `TxID` → `TxUUID`; amounts as strings; `Metadata` → `Settlement` (a `TransferSettlement`, once approved); `CurrencyID` → `Currency`; + `InitiatedBy`, `RefundPercent`, `Paid`, `PaidMinUnits`, `PaidUSD`, `Amount`, `AmountMinUnits`, `AmountUSD`, `Held`, `Chain`, `ExplorerURL`, `Approval`, `Customer` |
| `Customer` | `Customer` | `Name` → `FirstName`; `PhoneNumber`, `Address`, `Reference`: `*string` → `string`; + `Verified`, `Metadata` |
| `Product` | `Product` | `ID uint64` → `UUID string`; `Price` (a float) → `PriceUSD` (a string); + `Subscription`, `PaymentLink`, `Metadata` |
| `User` | `Member` (or `Me`) | `ID` → `UserUUID`; `Name` → `FirstName`; `OrganizationFeeBps` → `OrganizationFeePercent` (a string); `ClaimedAt` → `TrustedAt`; + `JoinedAt`, `AcceptedCurrencies` (`[]Currency`), `SpaceUUID` |
| — | `HeldFunds`, `LedgerEntry`, `MemberHeldFunds` | `TotalUSD`; each line's `AmountUSD`, `Settlement`, `Currency`, `Owed`, `OwedMinUnits`, `OwedUSD` |
| — | `Invitation`, `CreatedInvitation` | `HoldFunds`, `OrganizationFeePercent` (a string); `CreatedInvitation` is flat: the invitation's fields and `Link` |
| — | `Wallet`, `TokenWallet`, `Balance` | `Wallet.Address`, `Wallet.Currency`; `TokenWallet.Currency`; `Balance` = the `Currency` + `Amount`, `AmountMinUnits`, `AmountUSD` |
| `UserRole` (`RoleAdmin`, `RoleUser`) | `Role` (+ `RoleOwner`, `RoleHandle`) | |
| `AccountingEvent` | `AccountingEvent` | `PaymentID` → `TxUUID`; `RelatedPaymentID` → `RelatedTxUUID`; `PaymentReference` / `RelatedPaymentReference` → `Reference` / `RelatedReference`; `ProductID uint64` → `ProductUUID string`; `TokenSymbol` → `Currency` (always set, a native currency's too); every amount, USD amount and rate is a decimal string, the amounts in the currency's units (`"7.5"`, not `"7500000"`) with a `…MinUnits` twin; a fee that doesn't apply is `""` (absent), never `"0"`; network fees are never part of `NetAmount`; `Type` and `Chain` are typed; fields in the CSV's column order; + `UserUUID`, `ReferralFee*`, `CurrencyPriceUSD`, `NetworkFeesPaidBy` (`NetworkFeesPayer`), `NetworkFeesCurrency`, `NetworkFeesCurrencyDecimals`, `NetworkFeesCurrencyPriceUSD` |
| `PaymentMetadata` | `Settlement` | `FeeBps` → `FeePercent` (a string); `TxAmounts` → `Split` (`Platform`, `Organization`, `Referral`, `Merchant`, `NetworkFee`, each a `TokenAmount`: `Amount`, `AmountMinUnits`, `AmountUSD`; `Platform` no longer includes the referrer's share); the transaction's block and fees → `Transaction` (`SettlementTransaction`: `BlockNumber`, `BlockTime`, `GasUsed`, `Fee`, `L1Fee`, `NativePriceUSD`) |
| `OrganizationFee{OrganizationID, Organization, FeeBps}` | `OrganizationFee{Address, FeePercent}` | |
| `ReferralFee{ReferralID, Referrer, FeeBps, Deadline}` | `ReferralFee{Address, FeePercent, Deadline}` | |
| `Currency` | `Currency` | `Decimals int` → `uint8`; + `Code` (`BASE:USDC`), `Chain`; `Address` absent on a native coin; no nested `MainCurrency` |
| `Duration{Value int, Unit}` | `Duration{Value uint32, Unit}` | `Value: 0` means none |
| `CursorData[T]` | `Page[T]` | `Items`, `HasMore` (a field), `NextCursor` (a `string`, `""` on the last page) |
| `ApiKey`, `Organization`, `CreateClaimRequestResponse`, `ClaimFund`, `StatusResponse`, `SuccessResponse`, `ErrorResponse`, `SubscriptionType`, `TransactionShortType` | removed | |

**Typing rule.** An optional string is `""` when absent; any other optional field is a pointer
(`nil` when absent); a field the API always sends is a value; lists are slices and `Metadata` a
`map[string]string`. Request params follow the same rule: set a pointer inline with
`new(value)`, e.g. `Immediate: new(false)`.

**Metadata** is new: your own `map[string]string` on products, customers, checkouts (copied onto
the payment or subscription and into the webhooks) and subscriptions (`Subscriptions.Update`). At
most 50 keys of 1 to 40 characters among `A-Z a-z 0-9 _ - . :`, values of at most 500 characters;
an update merges it (`""` deletes a key).

## 9. Enum values and statuses

Enum values are lowerCamelCase on the wire, and every enum keeps unknown values as is.

**Subscription statuses**

| 2.x | 3.0.0 |
|---|---|
| `SubscriptionStatusActive` (`active`) | `SubscriptionStatusActive` |
| `SubscriptionStatusPastDue` (`past_due`) | `SubscriptionStatusPastDue` (`pastDue`) |
| `SubscriptionStatusTrial` (`trial`) | `SubscriptionStatusTrial` |
| `SubscriptionStatusTrialExpired` (`trial_expired`) | `SubscriptionStatusTrialExpired` (`trialExpired`) |
| `SubscriptionStatusCancelled` (`cancelled`) | `SubscriptionStatusCancelled` |
| `SubscriptionStatusLowOnFunds` (`low_on_funds`) | no status: `ActionRequired == ActionRequiredTopUpAllowance` |
| `SubscriptionStatusPending` (`pending`) | no status: `ActionRequired == ActionRequiredRaiseMaximum` |
| `Subscription.Stopped == true` | `SubscriptionStatusStopped` (`stopped`): cancelled at `NextBillingAt` |
| — | `SubscriptionStatusPaused` (`paused`): paused by the customer |

**Grant access while `now < CurrentPeriodEnd`**, whatever the status: a check like
`status == active` cuts off paying customers (`stopped`, `paused`) and keeps serving `pastDue` ones.

**Checkout statuses**: 2.x's seven `TransactionStatusValue`s become four
`CheckoutSessionStatusValue`s: `created`, `waitingConfirmation`, `completed`, `expired`. `pending`,
`failed` and `cancelled` are gone: **a failed attempt is `created` with `LastAttempt` set** (its
`Code` says why). It is never final: the customer may still pay until the session expires, so never
cancel an order on it.

**Other values**

| 2.x | 3.0.0 |
|---|---|
| `TransactionTypeOneTimePayment` | `TransactionTypePayment` (`payment`) |
| `TransactionTypeExecuteSubscriptionPayment` | `TransactionTypeExecuteSubscription` |
| `TransactionTypeCreatePAYGSubscription`, `TransactionTypeCancelPAYGSubscription` (`createPAYGSubscription`…) | `TransactionTypeCreatePaygSubscription`, `TransactionTypeCancelPaygSubscription` (`createPaygSubscription`…) |
| — | `TransactionTypeForceCancelSubscription`, `TransactionTypeReleaseHeldFunds` |
| `RefundStatus` | `pending` (a customer's request), `approved`, `rejected`; `failed` is declared, never set |
| `CombinedPayment.Source` `"subscription_history"` | `Transaction.Type` `TransactionKindBill` (`bill`) |
| `AccountingEvent.Type` (a string) | `AccountingEventType` constants (`subscription_history` → `AccountingEventTypeBill`, `bill`) |
| ledger lines `subscription_history` | `LedgerEntryTypeBill` (`bill`) |

## 10. Errors

2.x returned `*qberrors.QBitFlowError`, `*qberrors.NotFoundError` or `*qberrors.ValidationError`.
3.0.0 returns one type per condition, each embedding `*qbitflow.APIError` (`StatusCode`, `Code`,
`Message`, `Details`, `RequestID`, `FieldErrors`, `RawBody`, `Err`). Match them with `errors.As`,
not a type switch:

| Status | 3.0.0 type |
|---|---|
| 400 `validation_failed`, and client-side checks (`StatusCode` 0) | `*ValidationError` |
| other 400 | `*BadRequestError` (2.x returned a validation error for every 400) |
| 401 | `*AuthenticationError` |
| 403 | `*PermissionDeniedError` |
| 404 | `*NotFoundError` |
| 409 | `*ConflictError` |
| 410 | `*GoneError` |
| 422 `idempotency_key_reused` | `*IdempotencyError` (2.x mapped 422 to a validation error) |
| 429 | `*RateLimitError` (`RetryAfter`, `Limit`, `PeriodSeconds`) |
| 5xx, unexpected 3xx, unusable 2xx | `*ServerError` |
| no response | `*NetworkError` |
| bad webhook signature | `*WebhookSignatureError` (`Reason`) |

- **Field errors work now.** 2.x read a validation error's fields from the wrong key (a
  top-level `errors` list), so every field came back empty. 3.0.0 reads `details.errors`:
  `FieldErrors` names each failing input by the name you sent, dotted when nested
  (`frequency.unit`).
- **Branch on `Code`**, never on `Message`: `unique_violation` (`Details["field"]`),
  `merchant_not_ready`, `tx_already_sent`, `policy_disabled`…
- **`GetByReference`** runs the list's `?reference=` filter: no match is a `*NotFoundError`
  (status 404, code `not_found`), like an unknown UUID.
- Every error carries the request's `RequestID`: log it, and quote it to support.

## 11. Webhooks

The webhook scheme is new; 2.x verification code does not carry over.

| | 2.x | 3.0.0 |
|---|---|---|
| Header | `X-Webhook-Signature-256: sha256=<hex>`, `X-Webhook-Timestamp`, `X-Webhook-ID` | `QBitFlow-Signature: t=<unix>,v1=<hex>[,v1=<hex>]`, `QBitFlow-Event-Id`, `QBitFlow-Event-Type` |
| Signed content | `timestamp.` + the body re-encoded as canonical JSON | `t.` + the **raw body**, exactly as received |
| Rotation | one secret | two `v1=` for 24 hours after a rotation; either secret verifies |
| Body | `SessionWebhookResponse` or `SubscriptionStatusTransitionWebhook` | the event envelope `{id, type, version, createdAt, test, userUuid, data}` |
| Dashboard test | `X-Webhook-ID: test-webhook-id` | a `webhook.test` event |

- **Read the raw body and never re-serialize it** before verifying: `VerifyWebhookRequest` does it
  for `net/http`.
- **v2 only.** Every endpoint receives the v2 envelope (v1 is retired and every endpoint was moved
  to v2); a body of another version is a `*ValidationError` from `ParseEvent`.
- **Event types** replace the two 2.x payloads: v1's transaction webhook becomes
  `payment.completed` / `subscription.created`, its subscription webhook `subscription.billed` /
  `subscription.statusChanged`, plus 11 new types (`checkout.expired`, `refund.requested`,
  `refund.approved`, `refund.rejected`, `member.joined`, …). Decode each with its accessor
  (`event.PaymentCompleted()`, …), or register a typed handler on a `WebhookRouter`.
- `subscription.billed`'s data is `{Bill, Subscription}` (the bill paid and the subscription as it
  leaves it); `subscription.created`'s is the `*Subscription`. Money in event data is decimal
  strings, as in the API's answers.
- Deliveries are at least once: deduplicate on `Event.ID`, and answer 2xx to the types you ignore.
- Endpoints are created per space with `Webhooks.Endpoints.Create`, whose answer carries the
  `whsec_…` secret once.

## 12. Pagination

`GetAll…(limit *uint16, cursor *string)` becomes `List(ctx, &…ListParams{Limit, Cursor, filters…})`,
returning a `*Page[T]` (`Items`, `HasMore`, `NextCursor`). While `page.HasMore`, pass
`page.NextCursor` back as `Cursor`, verbatim (it is opaque). Every paginated list also has an
iterator that walks every page lazily, and stops after the page whose `HasMore` is false. The short
bounded lists (`Currencies.List`, `Wallets.*`, `Members.ListHeldFunds`, `Accounting.ExportJSON`)
return a slice:

```go
for sub, err := range client.Subscriptions.Iterate(ctx, &qbitflow.SubscriptionListParams{Limit: 100}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sub.UUID, sub.Status)
}
```

## 13. Before and after: five common tasks

### Create the client

```go
// 2.x
client := qbitflow.NewWithConfig(qbf.Config{
	APIKey:  os.Getenv("QBITFLOW_API_KEY"),
	BaseURL: "https://api.qbitflow.app/v1",
	Timeout: 10 * time.Second,
})
```

```go
client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"),
	qbitflow.WithTimeout(10*time.Second), // per attempt; the base URL defaults to .../v2
)
if err != nil {
	log.Fatal(err)
}
me, err := client.Me(context.Background()) // check the key at start-up
if err != nil {
	log.Fatal(err)
}
fmt.Println(me.Role)
```

### Open a payment checkout and read its outcome

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
}
```

```go
session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
	ProductUUID: "0192f1c2-1111-7c4d-9e5f-6a7b8c9d0e1f",
	Reference:   "order-1042",
	SuccessURL:  "https://shop.example.com/thanks?session={{UUID}}",
})
if err != nil {
	log.Fatal(err)
}
status, err := client.CheckoutSessions.Get(ctx, session.UUID)
if err != nil {
	log.Fatal(err)
}
if status.Status == qbitflow.CheckoutSessionStatusCompleted {
	payment, err := client.Payments.Get(ctx, session.UUID, nil) // the payment has the session's id
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(payment.TxHash)
}
```

### List a seller's payments

```go
// 2.x
limit := uint16(50)
page, err := client.Payments.OnBehalfOf(123).GetAllPayments(&limit, nil)
for err == nil && page.HasMore() {
	page, err = client.Payments.OnBehalfOf(123).GetAllPayments(&limit, page.NextCursor)
}
```

```go
seller := client.OnBehalfOf("0192f1c2-7b3a-7c4d-9e5f-6a7b8c9d0e1f") // Member.UserUUID
for p, err := range seller.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.UUID, p.AmountUSD, "USD") // a decimal string
}
```

### Cancel a subscription

```go
// 2.x
resp, err := client.Subscriptions.ForceCancel("sub@0192f1c2-3333-7c4d-9e5f-6a7b8c9d0e1f")
fmt.Println(resp.Message)
```

```go
res, err := client.Subscriptions.Cancel(ctx, "sub@0192f1c2-3333-7c4d-9e5f-6a7b8c9d0e1f", nil) // immediate
if err != nil {
	log.Fatal(err)
}
fmt.Println(res.Subscription.Status, res.Pending) // Pending: still confirming on-chain (HTTP 202)
```

### Receive a webhook

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
router := qbitflow.NewWebhookRouter(secret).
	OnPaymentCompleted(func(ctx context.Context, payment *qbitflow.PaymentCompleted, event *qbitflow.Event) error {
		log.Println("fulfil", payment.Reference, "once for", event.ID) // deduplicate on event.ID
		return nil // an error answers 500: QBitFlow delivers it again
	})
http.Handle("POST /webhooks/qbitflow", router) // verifies, parses, answers 200 / 400 / 500
```

Or, lower level, `qbitflow.VerifyWebhookRequest(r, secret)` then `qbitflow.ParseEvent(body)` and
`event.PaymentCompleted()`.

### Handle an error

```go
// 2.x
switch e := err.(type) {
case *qberrors.NotFoundError:
	fmt.Println("not found:", e.Message)
case *qberrors.ValidationError:
	fmt.Println("invalid:", e.Message) // the field details were lost
case *qberrors.QBitFlowError:
	fmt.Println("error", e.StatusCode)
}
```

```go
_, err := client.Products.Get(ctx, "0192f1c2-1111-7c4d-9e5f-6a7b8c9d0e1f")
var (
	notFound   *qbitflow.NotFoundError
	validation *qbitflow.ValidationError
	apiErr     *qbitflow.APIError
)
switch {
case errors.As(err, &notFound):
	fmt.Println("not found, request", notFound.RequestID)
case errors.As(err, &validation):
	for _, f := range validation.FieldErrors {
		fmt.Println(f.Field, f.Message)
	}
case errors.As(err, &apiErr):
	fmt.Println("error", apiErr.StatusCode, apiErr.Code)
}
```

## Names from the unreleased 2.5.0

2.5.0 was prepared but never published; its changes are part of 3.0.0. If you built against that
branch: `QBitFlow.OnBehalfOf(userID)` → `client.OnBehalfOf(userUUID)`; `Config.MaxRetries` /
`DisableRetries` → `WithMaxRetries(n)` (**`0` now disables retries**; it meant "the default"
in 2.5.0); `Accounting.ExportCSV(params)` → `ExportCSV(ctx, from, to)`; `qberrors.ForbiddenError` →
`PermissionDeniedError`; `AuthenticationError`, `ConflictError`, `RateLimitError` (`RetryAfter *int`
→ `time.Duration`), `ServerError`, `NetworkError` → the same names in `qbitflow`;
`qberrors.StatusCodeOf` / `FieldsOf` / `JoinFieldErrors` → `errors.As` with an `*APIError` and its
`StatusCode` / `FieldErrors`; `qbf.Version` → `qbitflow.Version`; `qbf.MaxWebhookBodySize` (5 MiB)
→ a fixed 1 MiB limit; `SubscriptionWebhook` and its `StatusTransition()` / `Billing()` → the
`subscription.*` events; `RefundStatusRefused` → `RefundStatusRejected` (`rejected`, the value the
API sends).

## Names from the 3.0.0 previews

3.0.0 was rebuilt against API v2's final contract before its release; if you tried a preview
build, these names changed: `CheckoutSessions.GetStatus` → `Get`; `Payments.ListCombined` /
`IterateCombined` → `Transactions.List` / `Iterate` (`CombinedPayment` → `Transaction`,
`CombinedPaymentListParams` → `TransactionListParams`, `Source` → `Type`, `CombinedPaymentSource`
→ `TransactionKind`); `Subscriptions.ExecuteTestBilling` → `TestBill` (`BillingState` →
`TestBill`); `Refunds.Initiate` is removed (refunds are read only: a merchant's refund is sent from the dashboard);
`Refunds.ListInactive` / `IterateInactive` → `Refunds.List` / `Iterate` with `Status`;
`Wallets.ListSupportedCurrencies` → `ListAcceptedCurrencies` (`SupportedCurrenciesParams` →
`AcceptedCurrenciesParams`); `Currencies.ListAvailable` / `ListMain` → `Currencies.List` (`Main`);
`CheckoutSession` → `CreatedCheckoutSession`; `InvitationCreated` → `CreatedInvitation` (flat);
`MemberHeldFundsSummary` → `MemberHeldFunds`; `PaymentMetadata` & co. → `Settlement`; `Page.HasMore()`
→ the `HasMore` field; `EventTypeRefundCompleted` / `RefundDenied` → `EventTypeRefundApproved` /
`RefundRejected` (and the accessors and router methods); `Customers.GetByEmail` and
`Subscriptions.GetPublicHistory` are removed; `Invitations.Revoke` returns only an error; money,
percents and fee amounts are strings.
