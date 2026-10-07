# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.5.0] - 2026-09-23

Aligns the SDK with docs revision `5e7d5a5` and with the behaviour contract shared by the Go,
JavaScript, Python and PHP SDKs: one error taxonomy, one retry policy, one set of
client-side validation rules, one response-typing policy and one webhook-verification
contract.

Response types now follow the Go types the API serializes, field for field: a value field
is always usable (an omitted or null value decodes to its zero value), and only fields that
are genuinely nullable on the server are pointers. Several fields changed type accordingly,
so read **Changed (breaking)** before upgrading.

> **⚠️ Breaking changes in a minor release.** They are listed under **Removed** and
> **Changed (breaking)** below. Semver-aware resolvers treat `2.5.0` as a safe upgrade
> from any `2.x`, so `^2` / `~2.1` constraints will pick it up automatically — review
> before updating, or pin.

> The module path stays `github.com/QBitFlow/qbitflow-go-sdk/v2`; no import changes needed.

### Removed

-   **WebSocket status URL helper removed** (`TransactionStatusService.GetTransactionStatusWSURL`)
    — `/transaction/status/ws` is an internal endpoint for the QBitFlow checkout page (it
    rejects non-frontend origins). Use webhooks, or poll `GetTransactionStatus`.
-   **Pay-as-you-go** — `PayAsYouGoService`, the `PayAsYouGo` client field,
    `SubscriptionTypePAYG`, and the PAYG-only `SessionCheckout.FreeCredits` field. The API
    has no PAYG routes (session creation is disabled pending x402), so every method on the
    service could only fail. The PAYG members of `TransactionType` and `TransactionShortType`
    are kept, because a transaction response can still carry them.
-   **`RefundStatusRejected`** (`"rejected"`, a value the API never sends). Use
    `RefundStatusRefused` (`"refused"`); comparisons against the old constant were always
    false, and now fail to compile.
-   **`SubscriptionStatusTransitionWebhook`** — replaced by the real envelope
    `SubscriptionWebhook`, see **Changed (breaking)**.
-   **`qbmodels.StatusResponse`** — no REST route returns it.

### Changed (breaking)

-   **Response field types follow the server's Go types** (field → 2.1.0 type → 2.5.0 type):
    -   `Payment.ProductID`: `*uint64` → `uint64` (0 when the payment has no product).
    -   `Payment.Metadata`, `SubscriptionHistory.Metadata`: `*PaymentMetadata` →
        `PaymentMetadata` (always sent). `CombinedPayment.Metadata` stays a pointer.
    -   `CustomerUUID` on `Payment`, `Subscription`, `SubscriptionHistory` and
        `SessionCheckout`: `string` → `*string` (nil when no customer is attached).
        `CombinedPayment.CustomerUUID` stays a `string` (always sent).
    -   `Subscription.LastBillingDate`: `*time.Time` → `time.Time` (the zero time until the
        first billing — test with `IsZero()`).
    -   `Customer.PhoneNumber`, `Customer.Address`, `Customer.Reference`: `*string` →
        `string` ("" when unset).
    -   `TransactionStatus.Message`: `*string` → `string` ("" when the API gives no message).
    -   `SessionWebhookResponse.Status`: `TransactionStatus` → `*TransactionStatus` (the API
        may send null).
    -   `Currency` stays a value field on `Payment`, `Subscription` and `SubscriptionHistory`
        and is added to `CombinedPayment`: the API sends the expanded currency on all four.
    -   `RefundEntry`, `OrganizationFee` and `ReferralFee` keep their 2.1.0 value types:
        `RefundEntry.TxHash` and `MerchantMessage` are `""` until the refund is handled,
        `RespondedAt` and `Metadata` are nil until then.
    -   JSON tags mirror the server's, so re-encoding a record keeps the fields the API sent
        (for example `userId: 0` on a product).
-   **Lists are never nil.** Every method returning a slice (including `CursorData.Items`
    and `SessionCheckout.AvailableCurrencies` from `GetSession`) returns an empty, non-nil
    slice when the API sends an empty or null list.
-   **Typed errors for every HTTP status.** `pkg/errors` gains `AuthenticationError` (401),
    `ForbiddenError` (403), `ConflictError` (409), `RateLimitError` (429, with
    `RetryAfter *int` from the `Retry-After` header — delta-seconds or an HTTP date),
    `ServerError` (5xx, a 3xx, or a 2xx whose body is empty or not the expected JSON) and
    `NetworkError` (no response received). 400/422 return `ValidationError`, 404
    `NotFoundError`, other 4xx the base `QBitFlowError`; `ValidationError` and
    `NotFoundError` gained `StatusCode` (0 for client-side validation) and `Fields`. Every
    type unwraps to `*QBitFlowError`, so a `*QBitFlowError` target in `errors.As` matches
    any SDK error; code that type-switched on `err.(*qberrors.QBitFlowError)` must use
    `errors.As`. `qberrors.StatusCodeOf` / `FieldsOf` read any SDK error, wrapped or not.
-   **Redirects are never followed.** A 3xx is returned as a `ServerError` instead of being
    replayed — as a GET, with the API key — against the redirect target.
-   **A 2xx with an empty body** where a result is expected (anything but a 204 or a
    `Delete`) is a `ServerError`, not a zero-valued result.
-   **`CreateUser.Role` is `qbmodels.UserRole`.** Untyped string constants (`"user"`) still
    compile; a `string` variable needs a conversion. Only `RoleAdmin` / `RoleUser` are
    accepted (the API binds `oneof=admin user`).
-   **`GetSession` takes a `closeToExpireError *bool`** on payments and subscriptions; pass
    `nil` for the API default (`true`).
-   **`Webhooks.Verify(payload any, signature, timestamp)`** — the payload may be raw JSON
    (`[]byte`, `json.RawMessage`, `string`, forwarded verbatim) or a generic decoded value
    (existing `[]byte` callers compile unchanged). It returns `(false, nil)` only for the
    API's HTTP 400; every other failure — including a 422 — is returned as its typed error. A
    missing or invalid payload is a `ValidationError` raised before any request.
-   **Subscription webhooks use the documented envelope**
    `SubscriptionWebhook{SubscriptionUUID, SubscriptionReference, Type, Data}`, where `Type`
    is `SubscriptionWebhookTypeStatusTransition` or `SubscriptionWebhookTypeBilling`.
    `SubscriptionReference` is a `string` ("" when none). Unwrap `Data` with
    `StatusTransition()` or `Billing()`, which return a `*qberrors.ValidationError` when
    `Type` does not match or `Data` does not decode.
-   **`SessionCheckout.IsPayment()`** is true only for `txType` `"payment"` (or, without a
    `txType`, for a session without a frequency); a session of an unknown type is neither a
    payment nor a subscription.
-   **`HeaderWebhookID` is `"X-Webhook-Id"`** (was `"X-Webhook-ID"`). Header lookups are
    case-insensitive, so behaviour is unchanged; only the constant's value differs.
-   **Client-side validation mirrors the API's rules** (see **Added**) and therefore rejects
    input 2.1.0 let through: an inline session price of 0 (the API answers 400), a
    `CustomerUUID` that is not a bare UUID, a nil options/DTO argument (previously a
    panic), a zero cursor `limit`, and an accounting range whose `To` is before `From`.
-   **Empty optional strings are left out of request bodies.** In session options and in the
    customer/user update DTOs, a pointer to `""` is treated like nil — `new(os.Getenv(...))`
    with an unset variable no longer sends `"customerUUID": ""` (a 400) or `"email": ""`.
-   **A blank API key** makes every request fail with a `ValidationError` before anything is
    sent, and a request body that cannot be encoded as JSON (NaN, ±Inf) is a
    `ValidationError` rather than a base `QBitFlowError`.

### Added

-   **Client-level On-Behalf-Of** — `(*qbitflow.QBitFlow).OnBehalfOf(userID)` returns a
    client whose every service (Currencies included) sends `On-Behalf-Of`, sharing the
    transport and configuration; `0` acts at the organization level. The per-service
    `OnBehalfOf` methods remain, and `qbf.Client.OnBehalfOf` backs both.
-   **Retries for idempotent requests.** GET requests are re-sent after a network/timeout
    error or a 5xx response, with exponential back-off (1s, 2s, 4s) and 3 retries by
    default. POST, PUT and DELETE are never retried, nor are the three GETs that perform an
    action (`ForceCancel`, `ExecuteTestBillingCycle`, `TriggerTestClaimFunds`): a timeout may
    already have performed it. 4xx, 429 and 3xx are never retried. `Config.MaxRetries` sets
    the count (0 keeps the default of 3, a negative value disables retries) and
    `Config.DisableRetries` turns retries off.
-   **`Accounting.ExportCSV`** — the CSV rendering of the export (`format=csv`) as a string;
    JSON error bodies are parsed into the usual typed errors. `Export` remains the JSON form.
-   **Client-side validation**, one shared rule set: `alphanumspace` names (Unicode letters,
    decimal digits, spaces and `- _ ' .`, 2–100 code points; whitespace-only accepted, as by
    the API), `producttext` product text, a structural email check (never normalised),
    roles, `OrganizationFeeBps` ≤ 5000, prices finite and > 0, absolute http(s) redirect
    URLs (case-insensitive scheme), bare-UUID `CustomerUUID`, subscription durations
    (`Frequency` 1–4294967295, `TrialPeriod` 0–4294967295, known units; `MinPeriods` 0 =
    none), accounting dates (`YYYY-MM-DD`, real dates, `From` ≤ `To`; the maximum span is left
    to the API), positive identifiers and cursor limits. `CreateCustomer`, `UpdateCustomer`,
    `CreateUser`, `UpdateUser`, `CreateProduct`, `UpdateProduct` and
    `AccountingExportParams` expose `Validate()`.
-   **`User-Agent: qbitflow-go/<Version>`** on every request, and the exported
    `qbf.Version` constant. `Content-Type: application/json` is sent only with a body.
-   **Webhook helpers**: `qbf.MaxWebhookBodySize` (5 MiB) — `VerifyWebhookRequest` reads at
    most that much and refuses a larger body with a `ValidationError`. Thirteen golden
    vectors shared with the other SDKs pin the canonical JSON byte for byte (key order by
    UTF-8 bytes, HTML and U+2028/9 escaping, float64 number formatting, long decimal strings,
    control characters, lone surrogates, `-0`); canonicalisation itself is unchanged.
-   `SubscriptionWebhookType`, `SubscriptionWebhook`, `SubscriptionStatusTransition`,
    `FieldError`, `qberrors.JoinFieldErrors`, `qberrors.StatusCodeOf`, `qberrors.FieldsOf`,
    `SessionCheckout.UpgradingFromTrial`, `CombinedPayment.Currency`, the `RoleHandle` and
    `RoleOwner` roles, and `qbmodels.AccountingEventType*` constants (the subscription-billing
    value is listed under both spellings the API documents, `subscriptionHistory` and
    `subHistory`).

### Fixed

-   **Validation errors are readable**: a 400 reads
    `Validation Error: ProductName: ProductName is too short; Price: Price is too short`,
    with the per-field detail in `Fields` (2.1.0 decoded every field as empty).
-   **User-supplied path segments and query values are escaped.** A value containing `?`,
    `#`, `%` or `&` no longer changes the shape of the request. A `/` is escaped too, but the
    API currently cannot route a reference containing `/` (it answers 404).
-   **Error messages** follow one precedence: `error`, then the joined `errors[]` list, then
    `message`, then a non-JSON body (a proxy's HTML page), then the HTTP status text; the
    `errors[]` list is parsed for any status.
-   **Product text length is counted in code points**, not bytes, so valid accented or CJK
    names are no longer rejected locally; product create also checks the description.
-   **The webhook replay window is overflow-safe**: a signed timestamp centuries away is
    rejected instead of wrapping around into the accepted window.
-   The invalid-email error no longer echoes the caller's address.
-   **`go test ./...` runs without a server**: the offline suite always runs; the
    integration tests skip without `QBITFLOW_API_KEY` and require an explicit
    `QBITFLOW_BASE_URL` when a key is set.
-   Examples no longer use placeholder customer UUIDs the API rejects, keep accounting
    ranges within a month, escape the values they echo into HTML, and handle subscription
    webhooks on their own endpoint.

### Behaviour notes

-   **Unknown enum values are preserved.** Every enum-like field (`SubscriptionStatus`,
    `RefundStatus`, `TransactionStatusValue`, `TransactionType`, `UserRole`, …) is a string
    type: a value this SDK does not know yet decodes verbatim, is never an error, and is never
    coerced to a default.
-   **Webhook test probe.** The examples and README verify the signature *before*
    short-circuiting the dashboard's connectivity probe (`TEST_WEBHOOK_ID`), so the dashboard
    button exercises the real verification path end to end. This assumes the probe is signed
    like a normal delivery.
-   **Configure the client before use.** `SetBaseURL` / `SetTimeout` are not synchronised
    with in-flight requests, and `OnBehalfOf` copies made earlier keep their base URL.
-   `Delete` on customers, products and users returns only an `error` (the idiomatic Go
    shape; the other SDKs return the API's confirmation message).

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




