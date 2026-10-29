# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.1.0] - 2026-10-29

The event log tells which deliveries are still in progress. Nothing you call changes.

### Added

-   **`EndpointDelivery.Pending`, `NextAttemptAt` and `WaitingFor`** (`Webhooks.Events.Get`).
    `Pending` is true while QBitFlow still attempts the event on its own; `NextAttemptAt` is set
    while it is retried after a failed attempt, and `WaitingFor` (an `evt_…`) while it is queued
    behind an earlier event of the same publication. A delivery that is queued, or about to start,
    is now listed with **no `Attempts`** instead of not being listed at all: code that reads
    `Attempts[len(Attempts)-1]` must check the length first. A response without `pending` (an
    older server) decodes as `Pending` false.

### Changed

-   **README: webhook order is per publication, never per subscription.** The events published
    together (a `subscription.created` and its first `subscription.billed`) reach an endpoint in
    order; nothing waits across publications.
-   **README: a checkout's `CustomerReference` without a QBitFlow customer means no receipt** and
    no other customer email. Create the customer with that reference and their email first (even
    after the checkout was created, before the payment is recorded) to have them.

## [3.0.0] - 2026-10-28

The Go SDK for **QBitFlow API v2**. A major release: the module path, the client, the routes and
most names change. The Go, JavaScript, Python and PHP SDKs 3.0.0 share the same services, errors,
retries, validation and webhook verification. This entry lists every change since 2.1.0;
[MIGRATION-v3.md](MIGRATION-v3.md) walks through the upgrade step by step.

### Added

#### Client and configuration

-   `NewClient(apiKey, ...ClientOption) (*Client, error)` with the options `WithBaseURL`,
    `WithTimeout`, `WithMaxRetries`, `WithHTTPClient` and `WithOnBehalfOf`. `NewClient` sends
    nothing. `NewClientFromEnv` reads `QBITFLOW_API_KEY`, `QBITFLOW_BASE_URL` and
    `QBITFLOW_ON_BEHALF_OF`; options passed explicitly override them.
-   `Client.Me`: what the key is (its `Role`, its `Space`: organization or member, test or live,
    and the member it acts for). The recommended start-up check.
-   `Client.OnBehalfOf(userUUID)`: a client acting in a member's space that shares the parent's
    connections. `WithOnBehalfOf` also works per request, where `""` forces the organization level.
-   Request options on every method: `WithOnBehalfOf`, `WithIdempotencyKey`, and `WithRequestID`
    (`X-Request-Id`, 1 to 128 of `A-Z a-z 0-9 - _ . :`).
-   `User-Agent: qbitflow-go/3.0.0` and the `Version` constant.

#### Services

-   `CheckoutSessions`: `CreatePayment`, `CreateSubscription`, `Get`, `Expire`, and
    `WaitForCompletion`, which polls `Get` until the session is completed or expired (default
    timeout 10 minutes, interval 3 s, at least 1 s). The session params take `ExpiresInMinutes`
    (10 to 1440), `Metadata`, and redirect URLs that may carry `PlaceholderUUID` (`{{UUID}}`) and
    `PlaceholderTransactionType` (`{{TRANSACTION_TYPE}}`).
-   Checkout redirects for marketplaces: a member's key may send `SuccessURL` / `CancelURL` while
    its organization's `members.checkoutRedirects` policy allows it (on by default). With the
    policy off, the API answers `403 policy_disabled` with `Details["policy"]` and
    `Details["field"]`. An organization key acting on behalf of the member may always send them.
-   Checkout fees: `Fees *CheckoutFees` on both session params. It takes up to 10 `FeeItem` lines
    of your own (`Label`, `Description`, `AmountUSD`) and `ProcessingFee *bool`, which makes the
    customer pay QBitFlow's processing fee (`nil` keeps the space's dashboard default). The
    customer pays the price plus every line, then the network fee. A subscription's lines are
    locked with its price. The lines come back as `Fees []FeeLine` (with a `FeeLineType`:
    `custom`, `processingFee`) on payments, subscriptions, transactions and session data.
-   `Transactions.List` / `Iterate`: payments and bills in one feed. Each item is a `Transaction`
    with its `Type` (`TransactionKind`: `payment`, `bill`). Filters: `Type`, `SubscriptionUUID`,
    `Refunded` and the shared filters.
-   `Failures.List` / `Iterate`: the failed payment attempts (`Failure`: `Kind`, `Category`,
    `Code`, the `Attempted…` amounts). Filters: `Kind`, `Category`, `SubscriptionUUID` and the
    shared filters.
-   `Payments.List` / `Iterate`. Filters: customer, product, creation dates, `Reference`,
    `Refunded`, and `IncludeMembers` / `UserUUID`.
-   `Subscriptions.List` / `Iterate` (status, reference and the shared filters), `Update` (your
    metadata), `ListBills` / `IterateBills`, and `GetBill`. `Subscription.HasAccess(at)` is true
    when `CurrentPeriodEnd` is set and `at` is before it.
-   `Refunds.List` / `Iterate` (filters: repeatable `Status`, `TxUUID`, `Held`, `IncludeMembers`,
    `UserUUID`) and `Refunds.Get`. Refunds are read-only: you answer a customer's request, or send
    your own refund, from the dashboard.
-   `Members` (`List`, `Iterate`, `Get`, `Update`, `Remove`, `Trust`, `ListHeldFunds`,
    `GetHeldFunds`, `GetOwnHeldFunds`) and `Invitations` (`Create`, `List`, `Iterate`, `Revoke`),
    for marketplaces. Setting `CreateInvitationParams.SendEmail` to `new(false)` creates the
    invitation without emailing it, so you can share its `Link` yourself.
-   `Wallets` (`List`, with balances on request; `ListForMember`; `ListAcceptedCurrencies`) and
    `Currencies.Get`.
-   `Accounting.ExportCSV`, plus `Accounting.ExportJSONRange` / `ExportCSVRange`, which export any
    range. They split it into windows of at most 92 days (both ends included) and join the
    results in order, with the CSV header written once.
-   `Webhooks.Endpoints` (`List`, `Iterate`, `Get`, `Update`, `Delete`, and `Create`, which returns
    the `whsec_…` secret this one time only) and `Webhooks.Events`, the event log (`List`,
    `Iterate`, and `Get`, which includes the event's deliveries).
-   `Products.List` / `Iterate` (paginated; `IncludeHidden`, `Subscription`, `Reference`) and
    subscription products (`CreateProductParams.Subscription`,
    `UpdateProductParams.RemoveSubscription`). `Customers.List` / `Iterate` (`Email`,
    `Reference`).

#### Models

-   Your own `Metadata` (`map[string]string`) on products, customers, checkout sessions (copied
    onto the payment or subscription and into its webhooks) and subscriptions. Updates merge it;
    setting a key to `""` deletes that key.
-   `TokenAmount` (`Amount`, `AmountMinUnits`, `AmountUSD`), used for every token amount.
    `Settlement` (`FeePercent`, `OrganizationFee`, `ReferralFee`, `Split`, `Transaction`),
    `SettlementTransaction` and `TransferSettlement`. Also `Allowance`, `DunningStatus`,
    `RefundSummary`, `CustomerSummary`, `Attempt`, `SubscriptionCancellation` and `TestBill`.
-   New fields. Payments, bills and transactions carry `Customer` (a summary), their refund state
    (`Refund`, `Refundable`, `NotRefundableReason`), the `Paid…` amounts, `Chain` and
    `ExplorerURL`. Subscriptions carry `CurrentPeriodEnd`, `ActionRequired`, `PriceUSD`,
    `CancelledAt`, `CancellationReason`, `Dunning` and `ManagementPageLink`. Products carry a
    `PaymentLink`.
-   New open enums. A value the SDK doesn't know yet is kept as is. The enums: `ActionRequired`,
    `CancellationReason`, `BillingFailureReason`, `BillingStage`, `BillingOutcome`,
    `CheckoutSessionStatusValue`, `RefundInitiator`, `NotRefundableReason`, `TransactionKind`,
    `FailureKind`, `FailureCategory`, `Chain`, `LedgerEntryType`, `AccountingEventType`,
    `NetworkFeesPayer`, `InvitationStatus`, `EventType`, `EndpointDisabledReason`, and
    `MerchantNotReadyReason` (`Details["reason"]` of a `409 merchant_not_ready`: `noWallet`,
    `organizationNoWallet`, `noTokenWallet`, `fundsHeld`).

#### Errors

-   One error type per condition, matched with `errors.As`: `ValidationError`, `BadRequestError`,
    `AuthenticationError`, `PermissionDeniedError`, `NotFoundError`, `ConflictError`, `GoneError`,
    `IdempotencyError`, `RateLimitError` (`RetryAfter`, `Limit`, `PeriodSeconds`), `ServerError`,
    `NetworkError` and `WebhookSignatureError` (`Reason`). Each embeds an `*APIError`
    (`StatusCode`, `Code`, `Message`, `Details`, `RequestID`, `FieldErrors`, `RawBody`, `Err`), and
    every error carries the request id. `IsRetryable` tells whether an error is transient.

#### Retries and idempotency

-   Automatic retries for every GET and the 6 idempotent creates: `CheckoutSessions.CreatePayment`,
    `CheckoutSessions.CreateSubscription`, `Products.Create`, `Customers.Create`,
    `Webhooks.Endpoints.Create` and `Invitations.Create`. Calls are retried on network errors and
    timeouts, 5xx, 429 and `409 idempotency_key_in_use`, 3 times by default, with a 1 s · 2^n
    back-off. A 429 waits at least its `Retry-After`; one asking for more than 60 s is returned at
    once. `WithMaxRetries(0)` disables retries. Other writes are never retried.
-   Each idempotent create sends an `Idempotency-Key` (a UUID v4 per call), the same on every
    retry. To retry a create across processes, set your own key with `WithIdempotencyKey`.

#### Pagination

-   `Page[T]` (`Items`, `HasMore`, `NextCursor`). Every paginated list also has an iterator
    (`iter.Seq2[T, error]`: `Iterate`, `IterateBills`) that fetches pages lazily. It stops after
    the last page or on an error. The bounded lists return slices.

#### Webhooks

-   Local verification of API v2's signature scheme, without a client: `VerifyWebhook`,
    `VerifyWebhookRequest` (`net/http`; at most 1 MiB; puts the body back on `r.Body`),
    `ConstructEvent` and `ParseEvent`, with the options `WithTolerance` / `WithClock` and
    `DefaultWebhookTolerance` (300 s). The client offers the same as `Webhooks.Verify`,
    `Webhooks.ConstructEvent` and `Webhooks.ParseEvent`. Every `v1=` signature in the header is
    checked, so a secret rotation needs no change on your side. Header constants:
    `SignatureHeader`, `EventIDHeader`, `EventTypeHeader`.
-   Typed events: `Event` (`ID`, `Type`, `CreatedAt`, `Test`, `UserUUID`, `Data`) has an accessor
    for each of the 15 types (`PaymentCompleted()`, `SubscriptionBilled()`, `RefundApproved()`,
    `MemberJoined()`, `CheckoutExpired()`, `WebhookTest()`, …). `DecodeData[T]()` decodes any
    other. The `EventType` constants name the types.
-   `WebhookRouter` (`NewWebhookRouter(secret, ...WebhookRouterOption)`) verifies and parses each
    delivery, then dispatches it to the handler for its type (`OnPaymentCompleted`, …, plus
    `OnUnknown` and `OnAny`). It answers `200`, `400` for a bad signature or body, or `500` when
    a handler fails, so the event is delivered again. It is an `http.Handler` (POST only, at most
    1 MiB). For other frameworks, `Handle(ctx, rawBody, signatureHeader)` returns a
    `WebhookResult`. `WithWebhookErrorHandler` reports errors.
-   `SignWebhook(rawBody, secret, t)` builds the `QBitFlow-Signature` header QBitFlow would send,
    for testing your handlers.

#### Validation

-   Every params type has a `Validate` method that runs before each request; nothing is sent on
    failure. It checks names and texts (lengths counted in characters), emails, phone numbers,
    http(s) URLs, and USD amounts as decimal strings (above 0, at most 1,000,000, at most 2
    decimals). It also checks percentages (at most 2 decimals; `OrganizationFeePercent` 0 to 50)
    and metadata (at most 50 keys of 1 to 40 characters among `A-Z a-z 0-9 _ - . :`, values of at
    most 500 characters without control or bidirectional characters). Durations, cursors (at most
    200 characters) and enum filters are checked too. Ids in paths must be UUIDs or transaction
    ids (`pay@…`, `sub@…`, `sub-hist@…`, `refund@…`, `transfer@…`, `release@…`, or the bare
    UUID).
-   References: 1 to 100 characters among `A-Z a-z 0-9 . _ : @ -`, never exactly `.` or `..`
    (refused client-side). The rule applies everywhere a reference is taken: request bodies
    (`Reference`, `ProductReference`, `CustomerReference`), the `?reference=` list filters, and
    `GetByReference`.

#### Helpers and examples

-   `FormatAmount` / `ParseAmount` and `Currency.FormatAmount` convert exactly between a token's
    smallest unit and a decimal amount (string arithmetic, never a float).
-   Examples: `client`, `checkout`, `payments`, `subscriptions`, `refunds`, `catalog`,
    `currencies`, `accounting`, `marketplace`, `webhooks`, `webhook_verify`, `webhook_handler` (a
    `WebhookRouter`) and `errors_and_retries`.

### Changed

#### Client and configuration

-   **Breaking:** the module path is `github.com/QBitFlow/qbitflow-go-sdk/v3`, with one root
    package, `qbitflow`. It replaces `pkg/qbitflow`, `pkg/models`, `pkg/errors` and `pkg/utils`.
    Requires Go 1.27.
-   **Breaking:** API v2 only. The default base URL is `https://api.qbitflow.app/v2` (was `/v1`).
    API v2 uses REST routes: plural collections, `PATCH` for updates,
    `POST /<resource>/:uuid/<verb>` for actions, and `DELETE` answering `204`. The SDK covers the
    53 public routes.
-   **Breaking:** `qbitflow.NewClient(apiKey, ...ClientOption)` returns `(*Client, error)`. It
    replaces `New`, `NewWithConfig`, `Config`, `SetBaseURL`, `SetTimeout` and the `QBitFlow`
    struct. After trimming, the key must start with `sk_` and be printable ASCII without spaces.
    `WithBaseURL` takes an absolute http(s) URL without credentials, query or fragment. The
    services are fields of `*Client`.
-   **Breaking:** `WithTimeout` bounds each attempt (default 30 s), up to the last byte of the
    response body. For an overall deadline, use the context.
-   **Breaking:** every method takes a `context.Context` first and `...RequestOption` last.
-   **Breaking:** `On-Behalf-Of` names a member by user UUID: `client.OnBehalfOf(userUUID string)`
    replaces each service's `OnBehalfOf(userID uint64)`. `""` means the organization level, and
    the header is then left out.

#### Services

The full method map is in [MIGRATION-v3.md](MIGRATION-v3.md).

-   **Breaking:** checkout. `Payments.CreateSession` / `Subscriptions.CreateSession` →
    `CheckoutSessions.CreatePayment` / `CreateSubscription`. The params take plain values
    (`ProductID` → `ProductUUID`, `Price` → `PriceUSD` as a string). Both return a
    `*CreatedCheckoutSession` (`Link`, `UUID`, `ExpiresAt`).
    `TransactionStatus.GetTransactionStatus(uuid, type)` → `CheckoutSessions.Get(uuid)`.
-   **Breaking:** payments. `GetPayment` → `Get`, `GetPaymentByReference` → `GetByReference`,
    `GetAllPayments` → `List` / `Iterate`, and `GetAllCombinedPayments` → `Transactions.List` /
    `Iterate`.
-   **Breaking:** subscriptions. `GetSubscription` → `Get`, which now returns cancelled
    subscriptions too. `GetSubscriptionByReference` → `GetByReference`, and `GetPaymentHistory` →
    `ListBills` / `IterateBills`. `ForceCancel` → `Cancel(ctx, uuid, *CancelSubscriptionParams)`:
    `Immediate: new(false)` cancels at the end of the paid period, and
    `SubscriptionCancellation.Pending` reports an HTTP 202. `ExecuteTestBillingCycle` →
    `TestBill`, which returns a `*TestBill`.
-   **Breaking:** refunds. `Refunds.GetAll` / `GetAllInactive` → one `Refunds.List` / `Iterate`
    with a `Status` filter.
-   **Breaking:** products and customers. `Products.Get` / `Update` / `Delete` take the product's
    UUID (was its numeric id), and `Products.GetAll` → `List` (paginated). `Customers.GetAll` →
    `List`. `CreateProduct`, `UpdateProduct`, `CreateCustomer` and `UpdateCustomer` →
    `CreateProductParams`, `UpdateProductParams`, `CreateCustomerParams` and
    `UpdateCustomerParams` (`Name` → `FirstName`). Update params send only the fields you set.
    `new("")` clears `UpdateCustomerParams.LastName` / `PhoneNumber` / `Address`,
    `UpdateProductParams.Description` and `UpdateWebhookEndpointParams.Description`.
-   **Breaking:** people. `Users.*` → `Invitations.*` and `Members.*`, because a person joins by
    accepting an invitation. `Users.Create` → `Invitations.Create`, `GetAll` → `Members.List`,
    `GetByID` → `Members.Get`, `Update` → `Members.Update` (the organization fee only), `Delete` →
    `Members.Remove`, and `Get` → `Client.Me`. `Claims.*` → invitations, `Members.Trust` and the
    held-funds reads.
-   **Breaking:** accounting and currencies. `Accounting.Export(params)` →
    `ExportJSON(ctx, from, to)`: dates are `YYYY-MM-DD`, at most 92 days apart, both ends
    included. `Currencies.GetAllAvailable` / `GetAllMain` → `Currencies.List` (`Test`, `Main`).
-   Lookups by reference run the list's `?reference=` filter. They return the item whose
    reference matches exactly, or a `*NotFoundError` (status 404, code `not_found`).
-   Deletes and revokes return only an error.

#### Models

-   **Breaking:** ids are UUID strings. `Product.ID` → `UUID`, `ProductID` → `ProductUUID`, and
    `UserID` → `UserUUID` everywhere. `OrganizationID` is gone from every model. Currencies keep
    numeric ids, and transaction ids keep their prefixes.
-   **Breaking:** money is a decimal string, never a float, in requests and in responses. This
    covers USD amounts (`PriceUSD`, `AmountUSD`, `PaidUSD`, `TotalUSD`), token amounts (`Amount`,
    next to `AmountMinUnits`) and fee rates in percent (`FeeBps` → `FeePercent`,
    `OrganizationFeeBps` → `OrganizationFeePercent`; `"1.5"` is 1.5 %).
-   **Breaking:** `Amount` on payments, bills and transactions is now the token amount. Its USD
    value is `AmountUSD`.
-   **Breaking:** renamed types. `LinkResponse` → `CreatedCheckoutSession`. `TransactionStatus` →
    `CheckoutSessionStatus` (`UUID`, `Status`, `TxHash`, `Message`, `LastAttempt`, `ExpiresAt`).
    `SubscriptionHistory` → `Bill`. `CombinedPayment` → `Transaction` (`Source` → `Type`).
    `RefundEntry` → `Refund` (`TxID` → `TxUUID`). `User` → `Member`, and `UserRole` → `Role`.
    `CursorData` → `Page`. `PaymentMetadata` → `Settlement`: `TxAmounts` → `Split`, the block and
    fee data → `SettlementTransaction`, and `Split.Platform` no longer includes the referrer's
    share.
-   **Breaking:** renamed fields. `TransactionHash` → `TxHash`. `Name` → `FirstName` on customers
    and members. On payments and bills, `Name` / `Description` → `ProductName` /
    `ProductDescription`. `Product.Price` → `PriceUSD`, and `User.ClaimedAt` → `Member.TrustedAt`.
    `OrganizationFee` and `ReferralFee` are now `{Address, FeePercent}`. `Payment.Metadata` (the
    fee split) → `Settlement`, and `Metadata` now holds your own data. The `CurrencyID` fields are
    gone: each model carries its `Currency`, as a value.
-   **Breaking:** `Subscription`. `SubscriptionStatus` → `Status`. `Frequency` is a `Duration`
    (was seconds). `LastBillingDate` / `NextBillingDate` / `MinimumCancellationDate` →
    `LastBilledAt` / `NextBillingAt` / `CancellableAt` (`*time.Time`). `Allowance` is an
    `*Allowance`: a `TokenAmount` plus `RemainingPeriods`, `MaxPerPeriod` and
    `MaxPerPeriodMinUnits`. `Stopped` → the `stopped` status.
-   `Currency` gains `Code` (`BASE:USDC`) and `Chain`. `Decimals` is a `uint8`, `Address` is absent
    on a native coin, and the nested `MainCurrency` is gone. `Duration.Value` is a `uint32`.
-   Optional strings are plain `string`s (`""` when absent). Other optional fields are pointers.
-   **Breaking:** enum values follow the API's lowerCamelCase (`pastDue`, `trialExpired`).
    `SubscriptionStatus` adds `paused` and `stopped`. `low_on_funds` and `pending` are no longer
    statuses: they became the `ActionRequired` values `topUpAllowance` and `raiseMaximum`
    (alongside the new `confirmTrial`). A checkout has four statuses (`created`,
    `waitingConfirmation`, `completed`, `expired`). A failed attempt leaves it `created` with
    `LastAttempt` set; that is never final. `TransactionTypeOneTimePayment` →
    `TransactionTypePayment`. `RefundStatus` is `pending`, `approved` or `rejected`. Every enum
    keeps an unknown value as is.
-   `AccountingEvent` follows the API v2 export:
    -   amounts are decimal strings in the currency's units, each with a `…MinUnits` twin; USD
        amounts and rates are decimal strings;
    -   `Type` is an `AccountingEventType` (`subscriptionHistory` → `bill`), and `TokenSymbol` →
        `Currency`;
    -   `PaymentID` / `RelatedPaymentID` → `TxUUID` / `RelatedTxUUID`, `PaymentReference` /
        `RelatedPaymentReference` → `Reference` / `RelatedReference`, and `ProductID` →
        `ProductUUID`;
    -   network fees are reported apart from the net (`NetworkFeesPaidBy`, …), and a fee that
        doesn't apply is absent.
-   Responses decode leniently: a field the API leaves out gets its zero value. A value of the
    wrong JSON type is a `ServerError`, and so is an empty, `null` or non-JSON body where a
    result is expected.

#### Errors

-   **Breaking:** the typed errors above replace the `qberrors` package. `QBitFlowError` is now an
    interface that every SDK error implements. A 400 other than `validation_failed` is a
    `BadRequestError`, and a 422 is an `IdempotencyError`; 2.1.0 reported both as validation
    errors.
-   Error messages read `<message> (status <status>, code <code>, request <requestId>)`, followed
    by each field error. The SDK's own validation messages never echo the rejected value.

#### Pagination

-   **Breaking:** `GetAll…(limit, cursor)` → `List(ctx, *…ListParams)`, which returns a
    `*Page[T]`. `HasMore` is a field (was a method), and `NextCursor` is an opaque `string` (`""`
    on the last page).

#### Webhooks

-   **Breaking:** API v2's signature scheme. The `QBitFlow-Signature: t=<unix>,v1=<hex>` header
    carries an HMAC-SHA256 of `t + "." + rawBody`, computed over the raw body exactly as received.
    2.1.0 signed canonical JSON and used the `X-Webhook-*` headers.
-   **Breaking:** the verification methods. `VerifyWebhookRequest` takes `...VerifyOption`.
    `Webhooks.VerifyLocal` / `VerifyWebhookSignature` → `VerifyWebhook(rawBody, signatureHeader,
    secret)`. `Webhooks.Verify(payload, signature, timestamp) (bool, error)` →
    `Webhooks.VerifyRemote(ctx, endpointUUID, rawBody, signatureHeader) error`, and
    `Webhooks.Verify` now verifies locally.
-   **Breaking:** deliveries are v2 events. The `Event` envelope and its typed data
    (`payment.completed`, `subscription.created`, `subscription.billed`,
    `subscription.statusChanged`, …) replace `SessionWebhookResponse` and
    `SubscriptionStatusTransitionWebhook`. The dashboard's test is a `webhook.test` event.
    `ParseEvent` refuses an envelope whose version is not v2.

### Removed

-   **Breaking:** these services and methods, each with its replacement where there is one:
    -   `PayAsYouGo`: API v2 has no pay-as-you-go routes.
    -   `ApiKeys`: keys are managed in the dashboard, and `Client.Me` describes the current one.
    -   `Users.GetByEmail`.
    -   `Customers.GetByEmail`: an email may match several customers, so use `Customers.List`
        with `Email`.
    -   `Payments.GetCustomerForTransaction`: read `Payment.Customer`, or call `Customers.Get`.
    -   `Refunds.GetByTransactionUUID`: use `Refunds.List` with `TxUUID`, or read
        `Payment.Refund`.
    -   `Payments.GetSession` / `Subscriptions.GetSession`: the session's data comes with
        `checkout.expired`.
    -   `TransactionStatus.GetTransactionStatusWSURL`.
    -   `Claims.TriggerTestClaimFunds`.
-   **Breaking:** the `New…Service` constructors and the per-service `OnBehalfOf` methods.
-   **Breaking:** the v1 webhook verifier: `CanonicalJSON`, `ComputeWebhookSignature`,
    `VerifyWebhookSignature`, `WebhookService.VerifyLocal`, `ExtractWebhookHeaders`,
    `WebhookHeaders`, `DefaultMaxTimestampAge`, `TEST_WEBHOOK_ID`, `HeaderSignature`,
    `HeaderTimestamp`, `HeaderWebhookID` and the `Get…Header` methods. Also `VerifyOptions`,
    including `SkipTimestampCheck`: the timestamp check can no longer be skipped.
-   **Breaking:** models. `SessionCheckout` (with `IsPayment` / `IsSubscription`; an expired
    session's data is in `CheckoutExpired`), `StatusResponse`, `SuccessResponse`,
    `ErrorResponse`, `Organization`, `ApiKey`, `ClaimFund` and `CreateClaimRequestResponse`. Also
    `SessionWebhookResponse`, `SubscriptionStatusTransitionWebhook`, `TransactionShortType`,
    `SubscriptionType`, `TxMetadata`, `BlockData`, `NetworkFees`, `TxAmountsFull`,
    `TxAmountsMinUnits` and `TxAmountsUSD`, plus `ReferralFee.ReferralID` and `Deadline`.
-   **Breaking:** enum values. The `TransactionType` constants other than `payment` and
    `createSubscription`; the checkout statuses `pending`, `failed` and `cancelled`; and the
    fourth refund status (a refund is pending, approved or rejected).
-   **Breaking:** the `qberrors` constructors, `CursorQueryBuilder` and
    `ValidateProductTextForTest`.

### Fixed

-   Field errors are read from the API's `details.errors`. 2.1.0 looked for a top-level `errors`
    list, so the failing fields came back empty. `FieldErrors` names each failing input by the
    name you sent, dotted when nested (`frequency.unit`).
-   The webhook timestamp check is overflow-safe: a timestamp far in the past or future is refused
    instead of wrapping into the accepted window.

### Security

-   Redirects are never followed: a 3xx is a `ServerError`. 2.1.0 used Go's default redirect
    policy, which replays the request, `X-API-Key` header included, at the redirect's target. A
    client given to `WithHTTPClient` is copied with redirects turned off.
-   Ids you pass are checked (UUIDs, transaction ids; `.` and `..` refused) and escaped before
    they go into a path, and query values are encoded. An id containing `/`, `?`, `#` or `%` can
    no longer change the request; 2.1.0 pasted ids into the path as given.
-   Printing a `Client` with `fmt`, with any verb (`%v` and `%+v` included), never shows the API
    key. Printing a `WebhookRouter` never shows the webhook secret. The API key and base URL
    checks never quote the rejected value.
-   Webhook verification compares every `v1=` signature in constant time. It requires the
    timestamp to be within the tolerance (300 s by default, either direction), and that check
    cannot be turned off. Webhook bodies are capped at 1 MiB (`VerifyWebhookRequest`,
    `WebhookRouter`, `VerifyRemote`). The router's error answers never include error details.
-   `WithTimeout` bounds each attempt up to the last byte of the response body, so a server
    trickling its body cannot hold a call past it.
-   The SDK sends only the API calls you make, and only to the configured base URL: no telemetry.

## [2.1.0] - 2026-09-21

Aligns the SDK with the API. Partial updates are now genuinely partial,
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

### Breaking changes

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

## 2.0.0 - 2026-09-13

Major release aligning the SDK with the current QBitFlow API. The API base URL is unchanged
(`/v1`); only the SDK version and Go module path change.

### Breaking changes

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

## 1.3.1 - 2026-08-25

### Added

- **Act for user**: every service exposes `OnBehalfOf(userID uint64)` (e.g. `client.Products.OnBehalfOf(123).GetAll()`) to temporarily act as a different user for the duration of a request via the `On-Behalf-Of` header. This is useful for admin-level operations that need to be performed on behalf of another user within the same organization. Requires an admin-level (organization) API key.
- **`client.Users.GetByEmail(email)`** — retrieve a user by their email address (admin-level access required for other users within the organization)

## 1.3.0 - 2026-08-17

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

## 1.2.1 - 2026-07-18

- Removed `webhookUrl` from `CreateSessionOptions` and `CreateSubscriptionSessionOptions` — the API no longer accepts a webhook URL during session creation. Webhook URLs should be configured in the QBitFlow dashboard instead.

- Added webhooks for subscription status transitions. The new webhook payload includes `SubscriptionUUID`, `PreviousStatus`, `CurrentStatus`, and `UpdatedAt` fields, allowing clients to track subscription lifecycle events more effectively
- Also added a test webhook ID for webhook endpoint reachability checks from the frontend "Test webhook" action. This test sends a fake payload to the configured URL, which some SDKs may not parse like a real webhook. If the incoming webhook ID matches `TEST_WEBHOOK_ID`, handlers should return HTTP `200` immediately and skip normal payload processing.

## 1.2.0 - 2026-05-04

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

## 1.1.0 - 2026-03-08

### Added

-   HMAC signature verification for webhook requests
-   New `Verify` function in `WebhookService` to verify webhook authenticity
-   Updated documentation with webhook verification examples

### Security

-   Improved HMAC signature verification process
-   Enhanced input validation for webhook requests

## 1.0.0 - 2025-10-23

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

[3.1.0]: https://github.com/QBitFlow/qbitflow-go-sdk/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/QBitFlow/qbitflow-go-sdk/releases/tag/v3.0.0
[2.1.0]: https://github.com/QBitFlow/qbitflow-go-sdk/releases/tag/v2.1.0
