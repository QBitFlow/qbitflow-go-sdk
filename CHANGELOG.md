# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.0] - 2026-10-07

The SDK for **QBitFlow API v2**, aligned with the API docs `v2` @ `58460e9` and with the
behaviour contract the Go, JavaScript, Python and PHP SDKs 3.0.0 share (same services, methods,
errors, retries and webhook verification). A major release: the module path, the client and most names change.
[MIGRATION-v3.md](MIGRATION-v3.md) maps every 2.x method and type to its replacement.

This entry describes the changes since 2.1.0, the last published release (2.5.0 was prepared
but never published; its changes are part of 3.0.0).

### ⚠️ Breaking

-   **Module path `github.com/QBitFlow/qbitflow-go-sdk/v3`**, one root package `qbitflow`: the
    `pkg/qbitflow`, `pkg/models`, `pkg/errors` and `pkg/utils` packages are gone. Requires
    **Go 1.27**.
-   **API v2**: the default base URL is `https://api.qbitflow.app/v2`. API v1 runs next to it,
    against the same data, for a transition period.
-   **The client.** `qbitflow.NewClient(apiKey, ...ClientOption) (*Client, error)` replaces `New`,
    `NewWithConfig`, `Config`, `SetBaseURL` and `SetTimeout`. Options: `WithBaseURL`,
    `WithTimeout` (now per attempt), `WithMaxRetries`, `WithHTTPClient`, `WithOnBehalfOf`. The
    key must be non-blank and start with `sk_` (a `*ValidationError` otherwise; nothing is sent).
    The services are the client's fields; the `New…Service` constructors and the per-service
    `OnBehalfOf` methods are gone.
-   **`context.Context` first and `...RequestOption` last on every method**
    (`WithOnBehalfOf`, `WithIdempotencyKey`, `WithRequestID`).
-   **Ids are UUID strings.** Products are named by `UUID` (was the numeric `ID`), and
    `ProductID` is `ProductUUID` everywhere; people are named by their user UUID (`UserID` →
    `UserUUID`). `On-Behalf-Of` takes a member's user UUID: `client.OnBehalfOf(userUUID string)`.
    `OrganizationID` is gone from every model. Currencies keep numeric ids.
-   **Users and claims become invitations, members and the trust layer**: `Users.*` →
    `Invitations.*` and `Members.*` (a person exists once they accept an invitation);
    `Claims.*` → `Members.Trust` and the held-funds reads. `ApiKeys.*` is removed (keys are
    managed in the dashboard; `Client.Me` describes the current key).
-   **Checkout sessions have their own service.** `Payments.CreateSession` and
    `Subscriptions.CreateSession` → `CheckoutSessions.CreatePayment` / `CreateSubscription`,
    whose params take plain values (`ProductUUID`, `Reference`… instead of pointers) and return a
    `*CheckoutSession` (`UUID`, `Link`, `ExpiresAt`). `TransactionStatus.GetTransactionStatus` →
    `CheckoutSessions.GetStatus(ctx, uuid)`, with four statuses (`created`,
    `waitingConfirmation`, `completed`, `expired`): a failed attempt is `created` with
    `LastAttempt` set, and is never final.
-   **Subscriptions.** `Status` (was `SubscriptionStatus`) takes `trial`, `trialExpired`,
    `active`, `pastDue`, `paused`, `stopped`, `cancelled`; `low_on_funds` and `pending` became
    `ActionRequired` values (`topUpAllowance`, `raiseMaximum`, `confirmTrial`), and the `Stopped`
    bool the `stopped` status. `Frequency` is a `Duration` (was seconds), `NextBillingDate` a
    `*time.Time` (nil once cancelled), `LastBillingDate` a `time.Time`. `ForceCancel` →
    `Cancel(ctx, uuid, *CancelSubscriptionParams)` (a POST, `Immediate` false cancels at the end
    of the paid period, returns `*SubscriptionCancellation` with `Pending` for an HTTP 202);
    `ExecuteTestBillingCycle` → `ExecuteTestBilling` (returns the bill's `*BillingState`);
    `GetPaymentHistory` → `ListBills` / `IterateBills`, or `GetPublicHistory`; `Get` returns
    cancelled subscriptions too.
-   **Fee rates are percents**: `FeeBps` → `FeePercent`, `OrganizationFeeBps` →
    `OrganizationFeePercent` (`150` bps is `1.5`).
-   **Models renamed and retyped**: `LinkResponse` → `CheckoutSession`, `TransactionStatus` →
    `CheckoutSessionStatus`, `SubscriptionHistory` → `Bill`, `RefundEntry` → `Refund` (`TxID` →
    `TxUUID`), `User` → `Member`, `UserRole` → `Role`, `CursorData` → `Page`, `TxAmountsFull` →
    `TxAmounts`; `TransactionHash` → `TxHash`; `Currency` on payments, bills and subscriptions is
    a `*Currency`; `Payment.Metadata` a value. Optional strings are plain `string`s (`""` when
    absent: `Customer.PhoneNumber`, `Payment.Reference`…); other optional fields are pointers.
-   **Enum values** follow the API's lowerCamelCase (`pastDue`, `trialExpired`,
    `subscriptionHistory`); `TransactionTypeOneTimePayment` → `TransactionTypePayment`,
    `TransactionTypeExecuteSubscriptionPayment` → `TransactionTypeExecuteSubscription`, the PAYG
    values are `createPaygSubscription` / `cancelPaygSubscription`; `RefundStatusFailed` is
    removed.
-   **Errors.** Every error embeds an `*APIError` (`StatusCode`, `Code`, `Message`, `Details`,
    `RequestID`, `FieldErrors`, `RawBody`, `Err`) and has one type per condition, matched with
    `errors.As`: `ValidationError` (400 `validation_failed` and client-side checks),
    `BadRequestError` (other 400s, which 2.x reported as validation errors),
    `AuthenticationError`, `PermissionDeniedError`, `NotFoundError`, `ConflictError`,
    `GoneError`, `IdempotencyError` (422, which 2.x reported as a validation error),
    `RateLimitError`, `ServerError`, `NetworkError`, `WebhookSignatureError`. The `qberrors`
    package and its constructors are gone.
-   **Webhooks** use API v2's scheme: the `QBitFlow-Signature: t=…,v1=…` header, an HMAC-SHA256 of
    `t + "." + rawBody` over the raw body (no canonical JSON), and the event envelope
    `{id, type, version, createdAt, test, userUuid, data}`. Endpoints must be on payload version
    v2: v1 bodies are not parsed. `Webhooks.Verify(payload, signature, timestamp) (bool, error)`
    → `Webhooks.VerifyRemote(ctx, endpointUUID, rawBody, signatureHeader) error`.
-   **Pagination**: `GetAll…(limit, cursor)` → `List(ctx, *…ListParams)` returning a `*Page[T]`,
    with filters.

### Added

-   `Client.Me`: what the key is (role, space, mode), the recommended start-up check;
    `Client.OnBehalfOf`: a client acting in a member's space, sharing the connections.
-   `CheckoutSessions`: `CreatePayment`, `CreateSubscription` (with `ExpiresInMinutes` and the
    `{{UUID}}` / `{{TRANSACTION_TYPE}}` redirect placeholders), `GetStatus`, `Expire`.
-   Checkout fees: `CreatePaymentSessionParams.Fees` (`*CheckoutFees`: up to 10 `FeeItem` lines
    of your own, `Label`, `Description`, `AmountUSD`, and `ProcessingFee *bool`, which has the
    customer pay QBitFlow's processing fee, grossed up so you keep the price and your lines;
    `nil` follows the dashboard's `checkout.customerPaysProcessingFee` setting). The customer
    pays the price plus every line, then the network fee on top; test mode caps that amount,
    fees included, at $5. Validated before sending (at most 10 lines; `Label` the name rule, 1
    to 40; `Description` the text rule, at most 200; `AmountUSD` above 0, at most 1,000,000, at
    most 2 decimals), errors on `fees.items[i].<field>`. `Payment.Price` and `Payment.Fees`
    (`Amount` = price + fees, also on `payment.completed`), `PaymentSessionData.Fees` and
    `Amount` (`checkout.expired`), `FeeLine` (`AmountUSD` a decimal string) and the open enum
    `FeeLineType` (`FeeLineTypeCustom`, `FeeLineTypeProcessingFee`).
-   Checkout fees on subscriptions: `CreateSubscriptionSessionParams.Fees` (the same
    `*CheckoutFees` and rules, errors on `fees.items[i].<field>` after the terms'), locked with the
    price: every bill charges the price plus every line. `Subscription.Fees` (the lines, next to
    `PriceUSD`, which stays the price; also on every `subscription.*` webhook); the session's
    `Fees` and `Amount` on a subscription checkout too.
-   `AccountingEvent` follows the API's export: amounts in the currency's units (not min units),
    `Currency` (was `tokenSymbol`), `CurrencyPriceUSD`, and the network fees apart from the net:
    `NetworkFeesPaidBy` (the open enum `NetworkFeesPayer`: `payer`, `merchant`, `organization`),
    `NetworkFees`, `NetworkFeesCurrency`, `NetworkFeesCurrencyDecimals`,
    `NetworkFeesCurrencyPriceUSD`, `NetworkFeesUSD`. Its fields are in the CSV's column order; the
    always-empty `UserName`, `UserLastName`, `CustomerName` and `CustomerLastName` are gone.
-   `Payments.List` / `ListCombined` with filters (customer, product, dates, `Refunded`,
    `IncludeMembers` / `UserUUID`, `Source`, `SubscriptionUUID`); `Payments.Get` with
    `ReadParams`. `Failures.List`: the failed payment attempts.
-   `Subscriptions.List` (status, reference and the shared filters), `ListBills`, `GetBill`;
    `Subscription.CurrentPeriodEnd` (grant access while now is before it), `ActionRequired`,
    `PriceUSD`, `CancellationReason`, `Dunning`.
-   `Refunds.Initiate`: a merchant's refund of a payment or a bill, signed in the dashboard;
    refund list filters (`IncludeMembers`, `UserUUID`, `Held`).
-   `Members` (`List`, `Get`, `Update`, `Remove`, `Trust`, `ListHeldFunds`, `GetHeldFunds`,
    `GetOwnHeldFunds`) and `Invitations` (`Create`, `List`, `Revoke`) for marketplaces.
-   `Wallets` (`List` with balances, `ListForMember`, `ListSupportedCurrencies`),
    `Accounting.ExportCSV`, `Currencies.Get`, product filters and subscription products,
    customer filters.
-   `Webhooks.Endpoints` (`List`, `Create` with the secret shown once, `Get`, `Update`,
    `Delete`) and `Webhooks.Events` (the event log: `List`, `Get` with its deliveries).
-   Webhook verification without a client: `VerifyWebhook`, `VerifyWebhookRequest` (net/http,
    1 MiB, the body put back), `ConstructEvent`, `ParseEvent`, `WithTolerance`, `WithClock`;
    every `v1=` signature is checked, so a secret rotation needs nothing on the receiver's side.
    The header constants `SignatureHeader`, `EventIDHeader`, `EventTypeHeader`,
    `WebhookVersionHeader`.
-   Typed events: an `Event` with an accessor per type (`PaymentCompleted()`,
    `SubscriptionBilled()`, `MemberJoined()`, `CheckoutExpired()`… 15 types) and
    `DecodeData[T]()` for the others; `EventType` constants.
-   An iterator (`iter.Seq2`) next to every paginated list: `Iterate`, `IterateCombined`,
    `IterateBills`, `IterateInactive`.
-   Retries: reads and the 7 creates are retried on network errors, timeouts, 5xx, 429 (after
    `Retry-After`, at most 60 s) and `409 idempotency_key_in_use`, 3 times by default with a
    1 s · 2ⁿ back-off; `WithMaxRetries(0)` disables them; `IsRetryable`.
-   An `Idempotency-Key` on each create, generated per call and reused by its retries;
    `WithIdempotencyKey` for retries across processes.
-   `WithRequestID` (`X-Request-Id`) and the request id on every error.
-   Client-side validation mirroring the API v2 validators, through each params type's
    `Validate` method, run before every request (no request is sent on failure).
-   `User-Agent: qbitflow-go/3.0.0` and the `Version` constant.
-   Integration helpers, with the same behaviour in the four SDKs:
    -   `WebhookRouter` (`NewWebhookRouter(secret, ...WebhookRouterOption)`): verifies, parses
        and dispatches deliveries to a typed handler per event type (`OnPaymentCompleted`… 15
        methods, `OnUnknown`, `OnAny`, chaining), and answers QBitFlow (`200`, `400` for a bad
        signature or body, `500` when a handler fails). `Handle(ctx, rawBody, header)` returns a
        `WebhookResult` (`Status`, `Event`, `Err`, `ResponseBody()`); the router is an
        `http.Handler` (POST only, 1 MiB, the header read case-insensitively);
        `WithWebhookErrorHandler` reports the errors.
    -   `SignWebhook(rawBody, secret, t)`: the `QBitFlow-Signature` header QBitFlow would send,
        to test webhook handlers.
    -   `CheckoutSessions.WaitForCompletion(ctx, uuid, *WaitOptions)`: polls the status until it
        is final (default timeout 10 minutes, interval 3 s, at least 1 s) and returns it, or the
        last status seen at the timeout.
    -   `Subscription.HasAccess(at)`: `CurrentPeriodEnd` set and `at` before it.
    -   `FormatAmount` / `ParseAmount` and `Currency.FormatAmount`: exact conversions between a
        token's smallest unit and a decimal amount (string arithmetic, never a float).
    -   `Accounting.ExportJSONRange` / `ExportCSVRange`: an export over any range, in windows of
        at most 95 days joined in order (the CSV header once).
    -   `NewClientFromEnv`: a client from `QBITFLOW_API_KEY`, `QBITFLOW_BASE_URL` and
        `QBITFLOW_ON_BEHALF_OF`, the options given overriding them.
    -   `PlaceholderUUID` and `PlaceholderTransactionType`: the `{{UUID}}` and
        `{{TRANSACTION_TYPE}}` redirect placeholders.
-   Examples: `checkout`, `subscriptions`, `marketplace`, `webhook_handler` (a `WebhookRouter`),
    `errors_and_retries`; `MIGRATION-v3.md`.

### Changed

-   Responses decode leniently: a field the API leaves out or sends as `null` gets its zero
    value; a value of the wrong JSON type is a `ServerError`. Unknown enum values are kept as
    is, decimal strings are never parsed, timestamps keep the offset the API sends.
-   Redirects are never followed: a 3xx is a `ServerError` (2.1.0 replayed the request, API key
    included, against the redirect's target).
-   A 2xx with an empty or non-JSON body where a result is expected is a `ServerError`, not a
    zero-valued result.
-   A request body JSON cannot represent (NaN, ±Inf, invalid UTF-8) is a `ValidationError`, and
    nothing is sent.
-   Update params send an explicit `""` for the clearable fields
    (`UpdateCustomerParams.PhoneNumber` / `Address`, `UpdateProductParams.Description`,
    `UpdateWebhookEndpointParams.Description`): `new("")` clears them, `nil` leaves them.
-   Error messages read `<message> (status <status>, code <code>, request <requestId>)`,
    followed by each field error; the SDK's own validation messages never echo the rejected
    values.

### Removed

-   `PayAsYouGo` (the API has no pay-as-you-go routes), `Users`, `Claims`, `ApiKeys`,
    `TransactionStatus` (and its WebSocket URL helper), `Payments.GetSession`,
    `Subscriptions.GetSession`, `Payments.GetCustomerForTransaction` (read
    `Payment.CustomerUUID`, then `Customers.Get`), `Refunds.GetByTransactionUUID` (read
    `Payment.Refund` / `Bill.Refund`).
-   The v1 webhook verifier: `CanonicalJSON`, `ComputeWebhookSignature`,
    `VerifyWebhookSignature`, `ExtractWebhookHeaders`, `WebhookHeaders`, `VerifyOptions`
    (including `SkipTimestampCheck`), `DefaultMaxTimestampAge`, `TEST_WEBHOOK_ID`, the
    `X-Webhook-*` header constants, and the v1 payloads `SessionWebhookResponse` and
    `SubscriptionStatusTransitionWebhook`.
-   `SessionCheckout`, `StatusResponse`, `SuccessResponse`, `ErrorResponse`, `Organization`,
    `ApiKey`, `ClaimFund`, `CreateClaimRequestResponse`, `TransactionShortType`,
    `SubscriptionType`, `CursorQueryBuilder`, `ValidateProductTextForTest`.

### Fixed

-   **Field validation errors were read from the wrong key** (a top-level `errors` list instead
    of the API's `details.errors`), so every field of a 400 came back empty. `FieldErrors` now
    names each failing input by the name sent, dotted when nested (`frequency.unit`).
-   User-supplied path segments and query values are escaped: a reference containing `/`, `?`,
    `#`, `%` or `&` no longer changes the request.
-   Lengths are counted in characters (code points), as the API does, not in bytes: accented
    and CJK names are no longer refused locally.
-   The webhook timestamp check is overflow-safe: a timestamp centuries away is refused instead
    of wrapping into the accepted window.

## [2.1.0] - 2026-09-21

Aligns the SDK with docs revision `c3c8831`. Partial updates are now genuinely partial,
and one transaction-type bug is fixed that silently affected subscription sessions.

> **Note:** this release carries breaking changes (listed below) despite the minor version
> bump. The module path stays `github.com/QBitFlow/qbitflow-go-sdk/v2`.


### Added — local webhook verification

- **Verify webhooks without a network call.** Local verification needs your webhook secret
  (available from the QBitFlow dashboard) but no round-trip, so it is faster and keeps
  working when the API is unreachable. The existing API-side `verify` is unchanged and
  still available for callers who would rather not hold the secret.

  It performs the same three checks the server does: the timestamp is within a replay
  window (5 minutes by default, configurable to match your deployment), the HMAC-SHA256 of
  `<timestamp>.<canonical-json>` matches, and the comparison is constant-time so a timing
  side channel cannot be used to guess the signature.

  The signature covers a **canonical** rendering — object keys sorted at every level, no
  insignificant whitespace — rather than the bytes as they arrived, because proxies and
  frameworks routinely re-serialize a body and reorder keys. Signing raw bytes would reject
  payloads that are in fact untouched.

  New: `VerifyWebhookSignature`, `ComputeWebhookSignature`, `CanonicalJSON`,
  `VerifyWebhookRequest` (reads an `*http.Request` and returns the raw body),
  `ExtractWebhookHeaders`, `WebhookHeaders`, `VerifyOptions`, `DefaultMaxTimestampAge`, and
  `WebhookService.VerifyLocal`.

- **Header extraction helpers.** Reading the QBitFlow headers is framework-dependent, so
  the SDK accepts anything header-shaped and does a case-insensitive lookup, returning the
  signature, timestamp, transaction id, and whether this is the dashboard's connectivity
  test (which you can acknowledge immediately without processing).

### Fixed — validation parity with the API

- **An empty optional field now counts as "not provided"**, matching the API. Session
  checkout's `productName`, `description`, `successUrl` and `cancelUrl` are all
  `binding:"omitempty,..."` on the server, which skips validation for an empty value. The
  SDK previously rejected an explicit empty string, so the common
  `successUrl: <env var> || ""` pattern failed locally on a request the API would have
  accepted. Non-empty values are validated exactly as before.

### Changed — examples

- The `webhook_handler` example now uses `ExtractWebhookHeaders` and verifies locally when `QBITFLOW_WEBHOOK_SECRET` is set, falling back to the API otherwise. It also short-circuits the dashboard's connectivity-test webhook and only decodes the payload *after* verification passes.

### Added — client-side request validation

- Session checkout now mirrors the API's `producttext` rule (markup characters rejected,
  2-100 for an inline product name and 2-500 for its description) and requires redirect
  URLs to be absolute `http(s)`. Invalid input fails immediately instead of after a
  round-trip, and a `javascript:` redirect target is refused outright — the API's own `uri`
  rule is more permissive than this.

### ⚠️ Breaking changes

- **`SessionCheckout.TxType` is now `TransactionType`** (long form) instead of
  `TransactionShortType`. A subscription session reports `createSubscription`, not
  `subscription`. If you compared against `TransactionShortTypeSubscription` that check
  never matched and must be updated to `TransactionTypeCreateSubscription`. The webhook
  payload already used `TransactionType`, so both now agree.
> **Setting a pointer field:** Go 1.26's `new` accepts a value expression, so an optional
> field is set inline with no helper — `Price: new(39.99)`, `Name: new("Premium Plus")`,
> `OrganizationFeeBps: new(uint16(250))`.

- **`UpdateProduct` fields are now pointers** (`*string`, `*string`, `*float64`) and all
  optional. Updates are partial: omitted fields keep their stored value. Previously every
  field was required and client-side validation rejected an empty name or a price of 0,
  which made partial updates impossible.
- **`UpdateCustomer` fields are now pointers** and all optional, with the same partial
  semantics. **`Reference` has been removed** — a customer reference is immutable and the
  API ignores it on update, so sending it silently did nothing.
- **`UpdateUser` fields are now pointers** and all optional.
  **`Password` has been removed**: changing a password is a JWT-only, self-service
  operation that cannot be performed with an API key. The API silently ignores a password
  sent with an API key and still returns `200`, so the field was actively misleading.
  **`OrganizationFeeBps` is now `*uint16`** — `nil` omits it, a pointer to `0` explicitly
  sets zero. Note that a non-admin caller sending this field is now rejected with `403`.

### Added

- Integration coverage for the new semantics: partial update leaves other fields untouched
  (customers and products), an empty update body is a no-op, and client-side validation
  rejects a non-positive price.

### Changed

- Documentation for `TransactionShortType` now states it is the UUID-prefix category
  (`pay@`, `sub@`, `sub-hist@`, `refund@`) and **not** the value reported on sessions or
  webhooks.

## [2.0.0] - 2026-09-13

Major release aligning the SDK with the current QBitFlow API. The API base URL is unchanged
(`/v1`); only the SDK version and Go module path change.

### ⚠️ Breaking changes

- **Go module path is now `github.com/QBitFlow/qbitflow-go-sdk/v2`.** Update your imports (and `go get github.com/QBitFlow/qbitflow-go-sdk/v2`).
- **Typed payment metadata.** `PaymentMetadata` is now a structured type (`FeeBps`, `OrganizationFee`, `ReferralFee`, `TxMetadata`, `TxAmounts`) instead of `map[string]any`. `Payment.Metadata`, `CombinedPayment.Metadata`, and `SubscriptionHistory.Metadata` are now `*PaymentMetadata`; `RefundEntry.Metadata` is now `*TxMetadata`.
- **`SessionCheckout.AvailableCurrencies` is now `[]uint64`** (currency IDs) instead of `[]Currency`. Resolve details via the new `Currencies` service.
- **`Subscription.Allowance` is now a `string`** (decimal) instead of `float64`, to preserve precision.
- **`ApiKey.ExpiresAt` is now `*time.Time`** (nil when the key never expires).
- **Removed `ApiKeys.Create` / `ApiKeys.Delete`** (and `CreateApiKeyDto`, `CreatedKeyResponse`). API-key management is a JWT-only API operation and cannot be performed with an API key; manage keys from the dashboard. Read access (`GetAll`, `GetForUser`) is unchanged.

### Added

- **`Currencies` service** — `GetAllAvailable(test)` and `GetAllMain(test)` (public `/utils/all-*-currencies` endpoints) to resolve currency IDs.
- **Authenticated-only fields** now modeled where the API returns them under authentication: `organizationId`/`userId` on `Customer`, `Payment`, `SubscriptionHistory`; `settlementDetails` on `TransactionStatus`; `userName`/`txType` on `SessionCheckout`.
- **`User.ClaimedAt`** — set once an invited user has claimed their account.
- **`UpdateCustomer.Reference`** — update a customer's reference.
- Expanded enums: `TransactionType` (`transfer`, `tokenTransfer`, `refund`, `faucet`, `claimFunds`), new `TransactionShortType`, and `DurationUnit` `years`.

### Changed

- User `organizationFeeBps` validation range widened to `0–5000` bps, matching the API.

## [1.3.1] - 2026-08-25

### Added

- **Act for user**: every service exposes `OnBehalfOf(userID uint64)` (e.g. `client.Products.OnBehalfOf(123).GetAll()`) to temporarily act as a different user for the duration of a request via the `On-Behalf-Of` header. This is useful for admin-level operations that need to be performed on behalf of another user within the same organization. Requires an admin-level (organization) API key.
- **`client.Users.GetByEmail(email)`** — retrieve a user by their email address (admin-level access required for other users within the organization)

## [1.3.0] - 2026-08-17

### Added

- **Reference-based lookups** — resolve resources by the reference you assigned instead of storing QBitFlow's internal UUIDs:
  - `client.Payments.GetPaymentByReference(reference)` — `GET /transaction/payment/reference/:paymentReference`
  - `client.Subscriptions.GetSubscriptionByReference(reference)` — `GET /transaction/subscription/reference/subscription/:subscriptionReference`
  - `client.PayAsYouGo.GetSubscriptionByReference(reference)` — `GET /transaction/subscription/reference/payAsYouGo/:subscriptionReference`
  - `client.Customers.GetByReference(reference)` — `GET /customer/reference/:reference`
- **Your own references on session creation** — `CreateSessionOptions` and `CreateSubscriptionSessionOptions` gained:
  - `Reference` — your own transaction reference (e.g. order/invoice ID), echoed back on the resulting object and in webhooks
  - `ProductReference` — select a product by your own reference (alternative to `ProductID`)
  - `CustomerReference` — select an existing customer by your own reference (alternative to `CustomerUUID`); a new customer is created during checkout if none matches

### Changed

- `CreateSession` for both payments and subscriptions now accepts `ProductReference` as an alternative to `ProductID` (or the inline `ProductName`/`Description`/`Price` triplet).
- **`Payment.Reference`** and **`Subscription.Reference`** (`*string`) — added; the reference you set when creating the session.
- **`SessionCheckout`** — added `Reference`, `ProductReference`, and `CustomerReference`, so session responses and transaction webhook payloads expose the references you provided.
- **`SubscriptionStatusTransitionWebhook.SubscriptionReference`** (`*string`) — added; the subscription's reference is now included on status-transition webhooks.

## [1.2.1] - 2026-07-18

- Removed `webhookUrl` from `CreateSessionOptions` and `CreateSubscriptionSessionOptions` — the API no longer accepts a webhook URL during session creation. Webhook URLs should be configured in the QBitFlow dashboard instead.

- - Added webhooks for subscription status transitions. The new webhook payload includes `SubscriptionUUID`, `PreviousStatus`, `CurrentStatus`, and `UpdatedAt` fields, allowing clients to track subscription lifecycle events more effectively
- Also added a test webhook ID for webhook endpoint reachability checks from the frontend "Test webhook" action. This test sends a fake payload to the configured URL, which some SDKs may not parse like a real webhook. If the incoming webhook ID matches `TEST_WEBHOOK_ID`, handlers should return HTTP `200` immediately and skip normal payload processing.

## [1.2.0] - 2026-05-04

### Added

-   **New `RefundService`** (`client.Refunds`) — query refunds by transaction UUID, list active refunds, and paginate through inactive refunds
-   **New `AccountingService`** (`client.Accounting`) — export accounting events as structured JSON for a given date range
-   **New `ClaimService`** (`client.Claims`) — manage account-claim flow: create claim requests, retrieve claim info, list pending claim funds, and trigger test fund computation
-   **`PaymentService.GetCustomerForTransaction`** — retrieve the customer associated with any transaction UUID
-   **`TransactionStatusService.GetTransactionStatusWSURL`** — returns the WebSocket URL for real-time status monitoring without requiring a WebSocket library dependency in the SDK
-   **`SessionCheckout.IsSubscription()` / `IsPayment()`** helper methods to distinguish session types from a retrieved checkout
-   New models: `Organization`, `GetClaimRequestResponse`, `CreateClaimRequestResponse`, `ClaimFund`, `RefundEntry`, `RefundStatus`, `AccountingEvent`, `PaymentMetadata`
-   `ptr[T]` generic helper in test utilities

### Changed

-   **`POST /transaction/session-checkout/` → split into two endpoints** (breaking):
    -   `PaymentService.CreateSession` now calls `POST /transaction/session-checkout/new/payment`
    -   `SubscriptionService.CreateSession` now calls `POST /transaction/session-checkout/new/subscription`; the `Options` wrapper is replaced by flat `Frequency`, `TrialPeriod`, and `MinPeriods` fields on `CreateSubscriptionSessionOptions`
-   **Transaction status query parameters renamed** (breaking): `transactionUUID` → `txUUID`, `transactionStatusType` → `txType`
-   **`CustomerService.Update` signature changed** (breaking): UUID is now the first argument (`Update(uuid string, data *UpdateCustomer)`), sent in the URL path (`PUT /customer/:uuid`). The `UUID` field has been removed from `UpdateCustomer`
-   **`CreateSubscriptionSessionOptions`** now accepts either `ProductID` or the `(ProductName + Description + Price)` triplet, matching payment session behaviour
-   **`Product.Reference`** changed from `*string` to `string` — the server always returns a reference (auto-generated if not provided)
-   `Payment.ProductID` changed from `*int` to `*uint64`
-   `Subscription.ProductID` changed from `int` to `uint64`; `Allowance` from `float32` to `float64`
-   `CombinedPayment` updated with `AmountMinUnits`, `SubscriptionUUID`, and `Metadata` fields
-   `SubscriptionHistory` updated with `AmountMinUnits`, `SubscriptionUUID`, and `Metadata` fields
-   `Payment` updated with `AmountMinUnits` and `Metadata` fields
-   `Customer` updated with `Test bool` field
-   `Product` updated with `Test bool`, `OrganizationID`, and `UserID` fields
-   `ApiKey` updated with `OrganizationID` field; `ExpiresAt` changed from `*time.Time` to `time.Time`
-   `LinkResponse` — removed unused `ExpiresAt` field
-   `GET /transaction/session-checkout/:uuid` now returns `*SessionCheckout` (replaces `*Session`); subscription-specific fields (`Frequency`, `TrialPeriod`, `MinPeriods`) are populated for subscription sessions and zero for payment sessions
-   `SubscriptionService.ExecuteTestBillingCycle` and `PayAsYouGoService.ExecuteTestBillingCycle` now return `*SuccessResponse` instead of `*StatusLinkResponse`
-   Removed obsolete models: `Session`, `SubscriptionOptions`, `CreateSubscriptionOptions`, `CreateSessionRequest`, `StatusLinkResponse`

### Deprecated

-   **PAYG session creation is temporarily disabled** on the API. `PayAsYouGoService.CreateSession` has been removed. Existing PAYG subscriptions can still be queried and managed via `GetSubscription`, `GetPaymentHistory`, `ForceCancel`, and `ExecuteTestBillingCycle`.

[1.2.0]: https://github.com/qbitflow/qbitflow-go-sdk/releases/tag/v1.2.0


## [1.1.0] - 2026-03-08

### Added

-   HMAC signature verification for webhook requests
-   New `Verify` function in `WebhookService` to verify webhook authenticity
-   Updated documentation with webhook verification examples

### Security

-   Improved HMAC signature verification process
-   Enhanced input validation for webhook requests


## [1.0.0] - 2025-10-23

### Added

-   Initial release of QBitFlow Go SDK
-   Complete API coverage for QBitFlow Cryptocurrency Payment API
-   One-time payment support
    -   Create payment sessions
    -   Retrieve payment details
    -   Get payment history with pagination
    -   Combined payment history (one-time + subscription)
-   Subscription management
    -   Create subscription sessions
    -   Support for trial periods
    -   Retrieve subscription details
    -   Get payment history
    -   Force cancel subscriptions
    -   Execute test billing cycles (test mode)
-   Pay-as-you-go subscription support
    -   Create PAYG sessions
    -   Free credits support
    -   Minimum periods configuration
    -   Increase usage units
    -   Usage tracking
-   Transaction status tracking
    -   Get transaction status
    -   Support for all transaction types
    -   Real-time status updates
-   Custom error types
    -   QBitFlowError for API errors
    -   NotFoundError for 404 responses
    -   ValidationError for client-side validation
-   Comprehensive documentation
    -   Detailed README with examples
    -   API reference documentation
    -   Code comments and doc strings
-   Example code
    -   Basic payment example
    -   Subscription management example
    -   PAYG subscription example
    -   Webhook handler example
    -   Complete payment flow example
-   Integration tests
    -   Payment session tests
    -   Subscription tests
    -   PAYG tests
    -   Status checking tests
    -   Error handling tests
-   Type-safe models
    -   All request/response models
    -   Duration support
    -   Currency models
    -   Transaction types and statuses
-   Configuration options
    -   Custom base URL
    -   Custom timeout
    -   API key authentication

### Developer Experience

-   Clean, idiomatic Go code
-   Comprehensive examples
-   Well-documented API
-   Easy integration
-   Production-ready error handling

[1.0.0]: https://github.com/qbitflow/qbitflow-go-sdk/releases/tag/v1.0.0




