# Migrating from 2.x to 3.0.0

3.0.0 is the Go SDK for **QBitFlow API v2**. API v2 reorganises the platform around spaces
(an organization's, and one per member), invites people instead of provisioning them, names every
resource by a UUID and signs webhooks with a new scheme. The SDK follows: new module path, one
package, a new client, and new names for most methods.

API v1 keeps running next to v2, **against the same data**, for some weeks after v2 ships: your
2.x integration keeps working meanwhile, so you can move one part at a time. 2.x talks to `/v1`
only; 3.0.0 talks to `/v2` only.

This guide maps everything 2.1.0 exposed (the last 2.x release) to 3.0.0. The
[README](README.md) documents 3.0.0 in full; [CHANGELOG.md](CHANGELOG.md) lists every change.

## Checklist

1. [Module path and imports](#1-module-path-and-imports): `.../v3`, one package, Go 1.27.
2. [Base URL and API keys](#2-base-url-and-api-keys): `/v2`; rotate pre-v2 keys.
3. [The client](#3-the-client): `NewClient(key, opts...)`, options instead of `Config` and setters.
4. [`context.Context` and request options](#4-contextcontext-and-request-options) on every call.
5. [Method map](#5-method-map): every 2.x method and its replacement.
6. [Ids are UUID strings](#6-ids-are-uuid-strings): products, members, `On-Behalf-Of`.
7. [Users and claims become invitations, members and the trust layer](#7-users-and-claims-become-invitations-members-and-the-trust-layer).
8. [Models](#8-models): renamed types and fields, fees as percents.
9. [Enum values and statuses](#9-enum-values-and-statuses): subscription and checkout statuses.
10. [Errors](#10-errors): one typed error per status, `errors.As`, field errors that work.
11. [Webhooks](#11-webhooks): `QBitFlow-Signature` over the raw body, typed events, v2 endpoints.
12. [Pagination](#12-pagination): params structs and iterators.
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
- API v2 issues keys shaped `sk_<uuid>_<test|live>_<secret>`. **Keys issued before** (`sk_<digits>_…`)
  **still authenticate**: rotate them in the dashboard when you move to v2. Treat a key as opaque.
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
fmt.Println(payment.Amount)
```

`WithIdempotencyKey(key)` sets the `Idempotency-Key` of a create (see the README). Cancelling the
context cancels the request and any wait between retries.

## 5. Method map

`ctx` is omitted below; every 3.0.0 method takes it first. Params named `…Params` are
`*qbitflow.…Params` and may be `nil` when every field is optional.

### Checkout sessions and status (2.x `Payments`, `Subscriptions`, `TransactionStatus`)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.CreateSession(*qbf.CreateSessionOptions)` | `CheckoutSessions.CreatePayment(*CreatePaymentSessionParams)` | plain values instead of pointers; `ProductID` → `ProductUUID`; new `ExpiresInMinutes` and `Fees`; returns `*CheckoutSession` (`UUID`, `Link`, `ExpiresAt`) |
| `Subscriptions.CreateSession(*qbf.CreateSubscriptionSessionOptions)` | `CheckoutSessions.CreateSubscription(*CreateSubscriptionSessionParams)` | `Frequency`/`TrialPeriod` are `*Duration`, `MinPeriods` `*uint32`, each optional over a subscription product's terms |
| `TransactionStatus.GetTransactionStatus(uuid, txType)` | `CheckoutSessions.GetStatus(uuid)` | no type argument; four statuses (§ 9) |
| `TransactionStatus.GetTransactionStatusWSURL(…)` | removed | the WebSocket is the checkout page's: use webhooks, or poll `GetStatus` |
| `Payments.GetSession(uuid)`, `Subscriptions.GetSession(uuid)` | removed | `GetStatus` for the state; the session's data comes with `checkout.expired` |
| — | `CheckoutSessions.Expire(uuid)` | new: end an unpaid session |

### Payments

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Payments.GetPayment(uuid)` | `Payments.Get(uuid, *ReadParams)` | `pay@…` or the bare UUID |
| `Payments.GetPaymentByReference(ref)` | `Payments.GetByReference(ref)` | |
| `Payments.GetAllPayments(limit, cursor)` | `Payments.List(*PaymentListParams)`, `Payments.Iterate` | filters: customer, product, dates, `Refunded`, `IncludeMembers` / `UserUUID` |
| `Payments.GetAllCombinedPayments(limit, cursor)` | `Payments.ListCombined(*CombinedPaymentListParams)`, `Payments.IterateCombined` | + `Source`, `SubscriptionUUID` |
| `Payments.GetCustomerForTransaction(txUUID)` | removed | `Payment.Customer` (a summary), or `Customers.Get(payment.CustomerUUID)` |
| — | `Failures.List`, `Failures.Iterate` | new: the failed payment attempts |

### Subscriptions (and 2.x `PayAsYouGo`)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Subscriptions.GetSubscription(uuid)` | `Subscriptions.Get(uuid, *ReadParams)` | cancelled subscriptions are returned too: a 404 no longer means "cancelled" |
| `Subscriptions.GetSubscriptionByReference(ref)` | `Subscriptions.GetByReference(ref)` | |
| `Subscriptions.GetPaymentHistory(uuid)` | `Subscriptions.ListBills(uuid, *BillListParams)` / `IterateBills`, or `GetPublicHistory(uuid)` | `ListBills` pages every full bill; `GetPublicHistory` is the public route (10 latest, merchant-only fields empty) |
| `Subscriptions.ForceCancel(uuid)` | `Subscriptions.Cancel(uuid, *CancelSubscriptionParams)` | a POST now, never retried; returns `*SubscriptionCancellation{Subscription, Pending}` (`Pending`: HTTP 202, still confirming on-chain); `Immediate: new(false)` cancels at the end of the paid period |
| `Subscriptions.ExecuteTestBillingCycle(uuid)` | `Subscriptions.ExecuteTestBilling(uuid)` | a POST; returns the bill's `*BillingState`; `409 payment_not_due` before `NextBillingDate` |
| — | `Subscriptions.List`, `Iterate`, `GetBill` | new |
| `PayAsYouGo.*` (`GetSession`, `GetSubscription`, `GetSubscriptionByReference`, `GetPaymentHistory`, `ForceCancel`, `ExecuteTestBillingCycle`) | removed | the API has no pay-as-you-go routes |

### Refunds

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Refunds.GetAll()` | `Refunds.List(*RefundListParams)` | the refunds awaiting an answer; from the organization's space the members' are included by default (`IncludeMembers: new(false)` leaves them out) |
| `Refunds.GetAllInactive(limit, cursor)` | `Refunds.ListInactive(*RefundListParams)`, `Refunds.IterateInactive` | |
| `Refunds.GetByTransactionUUID(txUUID)` | removed | `Payment.Refund` / `Bill.Refund` (a `RefundSummary`), or the lists |
| — | `Refunds.Initiate(*InitiateRefundParams)` | new: a pending refund you sign in the dashboard |

### Customers and products

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Customers.Create(*qbf.CreateCustomer)` | `Customers.Create(*CreateCustomerParams)` | optional fields are plain strings (`""` = absent) |
| `Customers.Get`, `GetByEmail`, `GetByReference`, `Delete` | same names | `ctx` first |
| `Customers.Update(uuid, *qbf.UpdateCustomer)` | `Customers.Update(uuid, *UpdateCustomerParams)` | `Name`, `LastName`, `Email` are strings (`""` = unchanged); `PhoneNumber`, `Address` stay `*string`, and `new("")` now **clears** them |
| `Customers.GetAll(limit, cursor)` | `Customers.List(*CustomerListParams)`, `Customers.Iterate` | filters `Email`, `Verified` |
| `Products.Create(*qbf.CreateProduct)` | `Products.Create(*CreateProductParams)` | `Description` optional; `Reference` a string; `Subscription` terms make a subscription product |
| `Products.Get(id uint64)` | `Products.Get(uuid string)` | |
| `Products.GetAll()` | `Products.List(*ProductListParams)` | `IncludeHidden`, `Subscription` |
| `Products.GetByReference(ref)` | `Products.GetByReference(ref)` | |
| `Products.Update(id, *qbf.UpdateProduct)` | `Products.Update(uuid, *UpdateProductParams)` | `Name` a string (`""` = unchanged); `Description`, `Price`, `IsActive` pointers; `Subscription`, `RemoveSubscription` |
| `Products.Delete(id)` | `Products.Delete(uuid)` | |
| `CreateProduct.Validate()`, `UpdateProduct.Validate()` | `CreateProductParams.Validate()`, … | every params type has `Validate`, run before each request |

### Users, claims and API keys (→ § 7)

| 2.x | 3.0.0 | Notes |
|---|---|---|
| `Users.Create(*qbf.CreateUser)` | `Invitations.Create(*CreateInvitationParams)` | not a rename: the person exists once they accept (`member.joined`) |
| `Users.GetAll()` | `Members.List(*MemberListParams)`, `Members.Iterate` | |
| `Users.GetByID(id)` | `Members.Get(userUUID)` | |
| `Users.GetByEmail(email)` | removed | iterate `Members` and match `Member.Email` |
| `Users.Update(id, *qbf.UpdateUser)` | `Members.Update(userUUID, *UpdateMemberParams)` | only the organization fee: names and emails are the person's own |
| `Users.Delete(id)` | `Members.Remove(userUUID)` | ends the membership (`409 held_funds_pending` while you hold their funds) |
| `Users.Get()` | `client.Me()` | what the key is: its role, space and mode |
| `Claims.CreateClaimRequest(userID)`, `Claims.GetClaimRequestByUserID(userID)` | `Invitations.Create`, `Invitations.List` | the invitation link replaces the claim link |
| `Claims.GetClaimFunds()` | `Members.ListHeldFunds()`, `Members.GetHeldFunds(userUUID)`, `Members.GetOwnHeldFunds()` | |
| `Claims.TriggerTestClaimFunds(userID)` | removed | release held funds in the dashboard; `Members.Trust(userUUID)` pays new payments directly |
| `ApiKeys.GetAll()`, `ApiKeys.GetForUser(userID)` | removed | keys are managed in the dashboard; `client.Me()` describes the current one |
| — | `Invitations.Revoke`, `Invitations.Iterate`, `Members.Trust` | new |

### Wallets, accounting, currencies

| 2.x | 3.0.0 | Notes |
|---|---|---|
| — | `Wallets.List(*WalletListParams)`, `Wallets.ListForMember(userUUID)`, `Wallets.ListSupportedCurrencies(*SupportedCurrenciesParams)` | new |
| `Accounting.Export(qbf.AccountingExportParams{From, To})` | `Accounting.ExportJSON(from, to)` | dates `YYYY-MM-DD`; the API allows 95 days at most |
| — | `Accounting.ExportCSV(from, to)` | the same export as CSV text |
| `Currencies.GetAllAvailable(test)` | `Currencies.ListAvailable(&CurrencyListParams{Test: test})` | |
| `Currencies.GetAllMain(test)` | `Currencies.ListMain(&CurrencyListParams{Test: test})` | |
| — | `Currencies.Get(id)` | new |

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
| — | `Webhooks.Endpoints.*`, `Webhooks.Events.*` | new: endpoints and the event log |

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
- **Transactions** keep their prefixed ids (`pay@…`, `sub@…`, `sub-hist@…`, `refund@…`), as opaque
  strings. `TxID` → `TxUUID`, `TransactionHash` → `TxHash`.
- **JSON keys** are lowerCamelCase with acronyms as words (`customerUuid`, `subscriptionUuid`,
  `txHash`); only matters if you stored raw responses.

**Map the numeric ids you stored** before v1 is turned off: list your products
(`Products.List(ctx, &qbitflow.ProductListParams{IncludeHidden: true})`, in each space you act in)
and match them on `Reference`, which did not change, and your members (`Members.Iterate`, in each
mode) on `Email`; store the UUIDs next to your old ids. The SDK does not expose the API's
temporary `legacyId` field.

## 7. Users and claims become invitations, members and the trust layer

v1 provisioned sellers (`Users.Create`) and could sell for them at once, the seller claiming the
account and its funds later. **v2 creates no accounts: a seller sells only once they accepted an
invitation.**

1. `Invitations.Create(ctx, &qbitflow.CreateInvitationParams{Email, TrustLayer, OrganizationFeePercent, RedirectURL})`
   returns the invitation and its link (also emailed).
2. Wait for the `member.joined` webhook and store its `UserUUID` (match `InvitationUUID`).
3. Sell with `client.OnBehalfOf(userUUID)`.

`TrustLayer: true` (the v1 "claim" model) holds the seller's payments in your wallet until you
trust them: `Members.GetHeldFunds` shows what you owe, `Members.Trust` makes new payments go to
the seller directly, and the release of what is held is signed in the dashboard. Sellers v1
provisioned are already members; those who never claimed are members whose funds you hold
(`TrustedAt == nil`). The [README](README.md#marketplaces) walks the flow.

## 8. Models

Fee rates are **percents** now, everywhere: `FeeBps: 150` → `FeePercent: 1.5`,
`OrganizationFeeBps: 250` → `OrganizationFeePercent: 2.5`, `ReferralFee.FeeBps` → `FeePercent`.

| 2.x (`qbmodels`) | 3.0.0 (`qbitflow`) | What changed |
|---|---|---|
| `LinkResponse` | `CheckoutSession` | + `ExpiresAt` |
| `TransactionStatus` | `CheckoutSessionStatus` | `Status` is a `CheckoutSessionStatusValue`; + `UUID`, `TxHash`, `Message`, `LastAttempt` |
| `SessionCheckout` | removed | `PaymentSessionData` / `SubscriptionSessionData` in `checkout.expired` |
| `Payment` | `Payment` | `TransactionHash` → `TxHash`; `ProductID *uint64` → `ProductUUID string`; `Reference *string` → `string`; `Currency` → `*Currency`; `Metadata *PaymentMetadata` → `PaymentMetadata`; `UserID` → `UserUUID`; `CustomerUUID` kept; + `Chain`, `ExplorerURL`, `CustomerReference`, `Customer`, `Refund`, `Refundable`, `PaidMinUnits`, `PaidUSD`, `ConfirmedAt`, `Price`, `Fees` |
| `CombinedPayment` | `CombinedPayment` | `Source` is a `CombinedPaymentSource` (`subscription_history` → `subscriptionHistory`); `SubscriptionUUID *string` → `string`; + `Currency`, `Refund`, `Reference`, `SubscriptionReference` |
| `Subscription` | `Subscription` | `SubscriptionStatus` → `Status`; `Stopped bool` → the status `stopped`; `Frequency uint32` (seconds) → `Duration`; `ProductID` → `ProductUUID`; `Reference *string` → `string`; `LastBillingDate *time.Time` → `time.Time`; `NextBillingDate time.Time` → `*time.Time` (nil once cancelled); `Currency` → `*Currency`; + `CurrentPeriodEnd`, `ActionRequired`, `PriceUSD`, `MaxAmountPerPeriod`, `CancelledAt`, `CancellationReason`, `CustomerReference`, `Customer`, `Dunning` |
| `SubscriptionHistory` | `Bill` | the `Payment` renames; + `PeriodStart`, `PeriodEnd` |
| `RefundEntry` | `Refund` | `TxID` → `TxUUID`; + `InitiatedBy`, `RefundPercent`, `PaidMinUnits`, `PaidUSD`, `AmountUSD`, `CurrencyID`, `Held`, `Chain`, `ExplorerURL`, `Customer` |
| `Customer` | `Customer` | `PhoneNumber`, `Address`, `Reference`: `*string` → `string`; + `Verified` |
| `Product` | `Product` | `ID uint64` → `UUID string`; + `Subscription`, `PaymentLink` |
| `User` | `Member` (or `Me`) | `ID` → `UserUUID`; `OrganizationFeeBps` → `OrganizationFeePercent`; `ClaimedAt` → `TrustedAt`; + `JoinedAt`, `AcceptedCurrencyIDs`, `SpaceUUID` |
| `UserRole` (`RoleAdmin`, `RoleUser`) | `Role` (+ `RoleOwner`, `RoleHandle`) | |
| `AccountingEvent` | `AccountingEvent` | `PaymentID` → `PaymentUUID`; `RelatedPaymentID` → `RelatedPaymentUUID`; `ProductID uint64` → `ProductUUID string`; `Type` and `Chain` are typed; optional USD amounts and percents are pointers; + `UserUUID`, member and customer names |
| `PaymentMetadata` | `PaymentMetadata` | `FeeBps` → `FeePercent`; `TxAmounts TxAmountsFull` → `TxAmounts TxAmounts` (`Usd` → `USD`); optional USD amounts are pointers; + network fees |
| `OrganizationFee{OrganizationID, Organization, FeeBps}` | `OrganizationFee{Organization, FeePercent}` | |
| `ReferralFee{ReferralID, Referrer, FeeBps, Deadline}` | `ReferralFee{Referrer, FeePercent, Deadline}` | |
| `Currency` | `Currency` | `Decimals int` → `uint8` |
| `Duration{Value int, Unit}` | `Duration{Value uint32, Unit}` | `Value: 0` means none |
| `CursorData[T]` | `Page[T]` | same `Items`, `NextCursor`, `HasMore()` |
| `ApiKey`, `Organization`, `CreateClaimRequestResponse`, `ClaimFund`, `StatusResponse`, `SuccessResponse`, `ErrorResponse`, `SubscriptionType`, `TransactionShortType` | removed | |

**Typing rule.** An optional string is `""` when absent; any other optional field is a pointer
(`nil` when absent); a field the API always sends is a value. Request params follow the same
rule: set a pointer inline with `new(value)`, e.g. `RefundPercent: new(50.0)`.

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
| `Subscription.Stopped == true` | `SubscriptionStatusStopped` (`stopped`): cancelled at `NextBillingDate` |
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
| `RefundStatusFailed` | removed (`pending`, `approved`, `rejected` remain) |
| `CombinedPayment.Source` `"subscription_history"` | `CombinedPaymentSourceSubscriptionHistory` (`subscriptionHistory`) |
| `AccountingEvent.Type` (a string) | `AccountingEventType` constants |

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
- **Endpoints must be on payload version v2.** v1 webhook URLs were migrated as endpoints with
  `PayloadVersion` `v1`, which still receive v1's bodies; 3.0.0 does not parse them (`ParseEvent`
  returns a `*ValidationError`). Move each endpoint to v2 in the dashboard, or with
  `client.Webhooks.Endpoints.Update(ctx, uuid, &qbitflow.UpdateWebhookEndpointParams{PayloadVersion: qbitflow.WebhookPayloadVersionV2})`,
  when your receiver runs 3.0.0. Moving keeps the endpoint's secret.
- **Event types** replace the two 2.x payloads: v1's transaction webhook becomes
  `payment.completed` / `subscription.created`, its subscription webhook `subscription.billed` /
  `subscription.statusChanged`, plus 11 new types (`checkout.expired`, `refund.*`,
  `member.joined`, …). Decode each with its accessor (`event.PaymentCompleted()`, …).
- Deliveries are at least once: deduplicate on `Event.ID`, and answer 2xx to the types you ignore.
- Endpoints are created per space with `Webhooks.Endpoints.Create`, whose answer carries the
  `whsec_…` secret once.

## 12. Pagination

`GetAll…(limit *uint16, cursor *string)` becomes `List(ctx, &…ListParams{Limit, Cursor, filters…})`,
returning a `*Page[T]` (`Items`, `NextCursor`, `HasMore()`). Pass `*page.NextCursor` back as
`Cursor`, verbatim. Every paginated list also has an iterator that walks every page lazily:

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
status, err := client.CheckoutSessions.GetStatus(ctx, session.UUID)
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
	fmt.Println(p.UUID, p.Amount)
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
func handler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := qbitflow.VerifyWebhookRequest(r, secret)
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
		event, err := qbitflow.ParseEvent(body)
		if err != nil {
			http.Error(w, "not a v2 event", http.StatusBadRequest)
			return
		}
		if event.Type == qbitflow.EventTypePaymentCompleted { // deduplicate on event.ID first
			payment, err := event.PaymentCompleted()
			if err != nil {
				http.Error(w, "bad event", http.StatusBadRequest)
				return
			}
			log.Println("fulfil", payment.Reference)
		}
		w.WriteHeader(http.StatusOK) // to every type, the ignored ones too
	}
}
```

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
