# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.0] - 2026-10-19

The SDK for **QBitFlow API v2**, aligned with the final API v2 contract (api-hub `docs/api` @
`fab875c`) and with the behaviour contract the Go, JavaScript, Python and PHP SDKs 3.0.0 share
(same services, methods, errors, retries, validation and webhook verification). A major release:
the module path, the client, the routes and most names change.
[MIGRATION-v3.md](MIGRATION-v3.md) maps every 2.x method and type to its replacement.

This entry describes the changes since 2.1.0, the last published release (2.5.0 was prepared
but never published; its changes are part of 3.0.0).

### ⚠️ Breaking

-   **Module path `github.com/QBitFlow/qbitflow-go-sdk/v3`**, one root package `qbitflow`: the
    `pkg/qbitflow`, `pkg/models`, `pkg/errors` and `pkg/utils` packages are gone. Requires
    **Go 1.27**.
-   **API v2**: the default base URL is `https://api.qbitflow.app/v2`, with its REST routes:
    plural collections (`/products/:uuid`, `/customers`, `/payments`, `/subscriptions`,
    `/bills/:uuid`, `/refunds`, `/wallets`, `/currencies`, `/checkout-sessions/…`,
    `/payment-failures`, `/transactions`, `/held-funds`), updates as `PATCH`, actions as
    `POST /<resource>/:uuid/<verb>`, lookups by reference as list filters, and every `DELETE`
    answering `204` with no body. `/v1` is gone.
-   **The client.** `qbitflow.NewClient(apiKey, ...ClientOption) (*Client, error)` replaces `New`,
    `NewWithConfig`, `Config`, `SetBaseURL` and `SetTimeout`. Options: `WithBaseURL`,
    `WithTimeout` (now per attempt), `WithMaxRetries` (`0` disables retries), `WithHTTPClient`,
    `WithOnBehalfOf`. The key must be non-blank and start with `sk_` (a `*ValidationError`
    otherwise; nothing is sent). The services are the client's fields; the `New…Service`
    constructors and the per-service `OnBehalfOf` methods are gone.
-   **`context.Context` first and `...RequestOption` last on every method**
    (`WithOnBehalfOf`, `WithIdempotencyKey`, `WithRequestID`).
-   **Ids are UUID strings.** Products are named by `UUID` (was the numeric `ID`), and
    `ProductID` is `ProductUUID` everywhere; people are named by their user UUID (`UserID` →
    `UserUUID`). `On-Behalf-Of` takes a member's user UUID: `client.OnBehalfOf(userUUID string)`;
    `""` means the organization level and the header is then omitted (never sent blank).
    `OrganizationID` is gone from every model. Currencies keep numeric ids. Transaction ids are
    prefixed (`pay@`, `sub@`, `sub-hist@`, `refund@`, `transfer@`, `release@`) and every route
    also takes the bare UUID.
-   **Money is a decimal string, never a float**, in requests and in responses: USD amounts
    (`PriceUSD`, `AmountUSD`, `PaidUSD`, `TotalUSD`, `AttemptedUSD`), token amounts (`Amount`,
    `Paid`, `Owed`, next to their `…MinUnits`) and fee rates in percent (`FeePercent`,
    `OrganizationFeePercent`, `RefundPercent`: `"1.5"` is 1.5 %). `CreateProductParams.Price` →
    `PriceUSD string`, the session params' inline `Price` → `PriceUSD string`,
    `FeeItem.AmountUSD`, `CreateRefundParams.RefundPercent`,
    `CreateInvitationParams.OrganizationFeePercent` and `UpdateMemberParams.OrganizationFeePercent`
    are strings. No `float64` is left in a model or a params type.
-   **Amounts are token amounts.** `Payment.Amount`, `Bill.Amount`, `Transaction.Amount` and
    `HeldFundsReleased.Amount` are now in the currency's units (`"10"` for 10 USDC), next to
    `AmountMinUnits` and the new `AmountUSD` (what `Amount` used to be).
-   **Users and claims become invitations, members and held funds**: `Users.*` →
    `Invitations.*` and `Members.*` (a person exists once they accept an invitation);
    `Claims.*` → `Members.Trust` and the held-funds reads. `ApiKeys.*` is removed (keys are
    managed in the dashboard; `Client.Me` describes the current key).
-   **Checkout sessions have their own service.** `Payments.CreateSession` and
    `Subscriptions.CreateSession` → `CheckoutSessions.CreatePayment` / `CreateSubscription`,
    whose params take plain values and return a `*CreatedCheckoutSession` (`Link`, `UUID`,
    `ExpiresAt`, always set). `TransactionStatus.GetTransactionStatus` →
    `CheckoutSessions.Get(ctx, uuid)` (`GET /checkout-sessions/:uuid`), with four statuses
    (`created`, `waitingConfirmation`, `completed`, `expired`) and `ExpiresAt` while open: a
    failed attempt is `created` with `LastAttempt` set, and is never final.
-   **Subscriptions.** `Status` (was `SubscriptionStatus`) takes `trial`, `trialExpired`,
    `active`, `pastDue`, `paused`, `stopped`, `cancelled`; `low_on_funds` and `pending` became
    `ActionRequired` values (`topUpAllowance`, `raiseMaximum`, `confirmTrial`), and the `Stopped`
    bool the `stopped` status. `Frequency` is a `Duration` (was seconds); `LastBillingDate` →
    `LastBilledAt`, `NextBillingDate` → `NextBillingAt`, `MinimumCancellationDate` →
    `CancellableAt` (all `*time.Time`, nil when absent); `Allowance` is an `*Allowance` object
    (`Amount`, `AmountMinUnits`, `AmountUSD`, `RemainingPeriods`, `MaxPerPeriod`,
    `MaxPerPeriodMinUnits`) that absorbs `MaxAmountPerPeriod`; `From`, `ProductUUID` and
    `SubscriptionHash` are absent when unknown; `Currency` is a value; `ManagementPageLink` is on
    the subscription itself. `ForceCancel` → `Cancel(ctx, uuid, *CancelSubscriptionParams)`
    (`POST /subscriptions/:uuid/cancel`; `Immediate` false cancels at the end of the paid period;
    returns `*SubscriptionCancellation` with `Pending` for an HTTP 202);
    `ExecuteTestBillingCycle` → `TestBill` (`POST /subscriptions/:uuid/test-bill`, returns a
    `*TestBill`: `BillUUID` `sub-hist@…`, `Stage`, `Outcome`, `Attempts`, `NextAttemptAt`);
    `GetPaymentHistory` → `ListBills` / `IterateBills`; `Get` returns cancelled subscriptions too.
-   **The settlement.** `Payment.Metadata` (the fee split) → `Payment.Settlement`, a
    `Settlement` (`FeePercent`, `OrganizationFee`, `ReferralFee`, `Split`, `Transaction`) on
    payments, bills, transactions and held-funds lines; `TransferSettlement` on refunds and held
    funds releases. `PaymentMetadata`, `TxMetadata`, `BlockData`, `NetworkFees`, `TxAmounts`,
    `TxAmountsMinUnits` and `TxAmountsUSD` are replaced by `Settlement`, `Split` (each part a
    `TokenAmount`; `Split.Platform` no longer includes the referrer's share),
    `SettlementTransaction` (`BlockNumber`, `BlockTime`, `GasUsed`, `Fee`, `L1Fee`,
    `NativePriceUSD`) and `TransferSettlement`. `OrganizationFee.Organization` and
    `ReferralFee.Referrer` → `Address`.
-   **`Metadata` is now your own data**: a `map[string]string` on `Product`, `Customer`,
    `Payment` (copied from its checkout, read-only), `Subscription`, `Transaction` and the
    `checkout.expired` session.
-   **Models renamed and retyped**: `LinkResponse` → `CreatedCheckoutSession`,
    `TransactionStatus` → `CheckoutSessionStatus`, `SubscriptionHistory` → `Bill`, `RefundEntry`
    → `Refund` (`TxID` → `TxUUID`), `User` → `Member`, `UserRole` → `Role`, `CursorData` →
    `Page`; `TransactionHash` → `TxHash`. `Name` → `FirstName` on `Customer`, `CustomerSummary`,
    `Member` and `MeMember` (and the customer params); a payment's and a bill's `Name` /
    `Description` → `ProductName` / `ProductDescription`; `Product.Price` → `PriceUSD`;
    `Me.OnBehalfOf` → `OnBehalfOfUserUUID`; `Member.AcceptedCurrencyIDs` → `AcceptedCurrencies`
    (`[]Currency`); `Wallet.PublicKey` → `Address`; `TokenWallet.TokenID` / `Token` → `Currency`;
    `Balance` is the `Currency` plus a `TokenAmount` (was a code, `Balance` and `BalanceUSD`);
    `HeldFunds.TotalAmount` and `MemberHeldFunds.TotalAmount` → `TotalUSD`;
    `LedgerEntry.Amount` → `AmountUSD`, `LedgerEntry.Metadata` → `Settlement`, plus `Owed`;
    `Invitation.TrustLayer` → `HoldFunds` (and on the params); `InvitationCreated
    {Invitation, Link}` → `CreatedInvitation`, flat (the invitation's fields and `Link`);
    `MemberHeldFundsSummary` → `MemberHeldFunds`; `BillingState` → `TestBill`; the
    `CurrencyID` fields are gone (each model carries its `Currency` object, as a value). Optional
    strings are plain `string`s (`""` when absent); other optional fields are pointers.
-   **The currency object** gains `Code` (`BASE:USDC`) and `Chain`; `Address` is absent on a
    native coin; the nested `MainCurrency` is gone (`MainCurrencyID` stays).
-   **Enum values** follow the API's lowerCamelCase (`pastDue`, `trialExpired`);
    `TransactionTypeOneTimePayment` → `TransactionTypePayment`,
    `TransactionTypeExecuteSubscriptionPayment` → `TransactionTypeExecuteSubscription`, the PAYG
    values are `createPaygSubscription` / `cancelPaygSubscription`. `LedgerEntryType` and
    `AccountingEventType` `subscriptionHistory` → `bill` (`LedgerEntryTypeBill`,
    `AccountingEventTypeBill`). `RefundStatus` adds `cancelled` (a merchant's draft dropped
    before it was sent) and keeps `failed` (declared, never set). `WebhookPayloadVersionV1` is
    removed: v2 is the only payload.
-   **Pagination**: `GetAll…(limit, cursor)` → `List(ctx, *…ListParams)` returning a `*Page[T]`
    `{Items, HasMore, NextCursor}`: `HasMore` is a field (no `HasMore()` method), `NextCursor` an
    opaque `string` (`""` on the last page). Iterators stop when `HasMore` is false. Bounded lists
    (`Currencies.List`, `Wallets.List`, `Wallets.ListForMember`, `Wallets.ListAcceptedCurrencies`,
    `Members.ListHeldFunds`, `Accounting.ExportJSON`) return a slice (and follow a `NextCursor`
    should one ever claim more, so an export is never cut short silently). Iterators also stop on
    an empty page that claims more (a server fault).
-   **Lookups by reference are list filters.** `Products`, `Customers`, `Payments` and
    `Subscriptions` `GetByReference(ctx, reference)` run the list's `?reference=` filter and
    return the item whose `Reference` equals the one asked for (exactly), or a `*NotFoundError`
    (status 404, code `not_found`) when none does — never another resource.
    The reference is validated. `Customers.GetByEmail` is removed: an email may match several
    customers (`Customers.List` with `Email`).
-   **Deletes and revokes return only an error**: `Invitations.Revoke` no longer returns the
    invitation (`204`).
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
    `{id, type, version, createdAt, test, userUuid, data}`; a version other than `v2` is refused.
    `Webhooks.Verify(payload, signature, timestamp) (bool, error)` →
    `Webhooks.VerifyRemote(ctx, endpointUUID, rawBody, signatureHeader) error`.
-   **Webhook events renamed and reshaped**: `refund.completed` → `refund.approved`
    (`EventTypeRefundApproved`, `Event.RefundApproved()`, `WebhookRouter.OnRefundApproved`),
    `refund.denied` → `refund.rejected` (`EventTypeRefundRejected`, `RefundRejected()`,
    `OnRefundRejected`). `subscription.billed`'s data is nested: `SubscriptionBilled{Bill,
    Subscription}` (the flattened `SubscriptionReference` / `SubscriptionStatus` are
    `Subscription.Reference` / `.Status`). `subscription.created`'s data is a plain
    `*Subscription` (the `SubscriptionCreated` type is gone). `subscription.upcomingBill`'s
    `BillingDate` → `BillingAt` and its `AmountUSD` is a string. `heldFunds.released`'s
    `TxMetadata` → `Settlement`. The `checkout.expired` session has `PriceUSD`, `AmountUSD` (was
    `Price` / `Amount` numbers), `AvailableCurrencies` (was `AvailableCurrencyIDs`), `Metadata`,
    and `UpgradingFromTrial` as a bool.
-   **Renamed methods** (inside 3.0.0's preparation, before the API v2 final contract):
    `CheckoutSessions.GetStatus` → `Get`; `Payments.ListCombined` / `IterateCombined` →
    `Transactions.List` / `Iterate` (`CombinedPayment` → `Transaction`, `CombinedPaymentSource`
    → `TransactionKind` with `payment` / `bill`, the `Source` filter → `Type`);
    `Subscriptions.ExecuteTestBilling` → `TestBill`; `Refunds.Initiate` → `Create`
    (`InitiateRefundParams` → `CreateRefundParams`); `Refunds.List` (active) and `ListInactive` /
    `IterateInactive` → one `Refunds.List` / `Iterate` with a `Status` filter;
    `Wallets.ListSupportedCurrencies` → `ListAcceptedCurrencies` (`SupportedCurrenciesParams` →
    `AcceptedCurrenciesParams`); `Currencies.ListAvailable` / `ListMain` → `Currencies.List`
    with `Main`; `Webhooks.Endpoints.List` takes `*EndpointListParams` and returns a page.

### Added

-   `Client.Me`: what the key is (role, space, mode, the member acted for), the recommended
    start-up check; `Client.OnBehalfOf`: a client acting in a member's space, sharing the
    connections. `MeSpace` has `Role`, `TrustedAt`, `CreatedAt` and `OrganizationFeePercent`.
-   `CheckoutSessions`: `CreatePayment`, `CreateSubscription` (with `ExpiresInMinutes`,
    `Metadata` and the `{{UUID}}` / `{{TRANSACTION_TYPE}}` redirect placeholders), `Get`,
    `Expire`.
-   Checkout fees: `CreatePaymentSessionParams.Fees` and `CreateSubscriptionSessionParams.Fees`
    (`*CheckoutFees`: up to 10 `FeeItem` lines of your own, `Label`, `Description`, `AmountUSD`,
    and `ProcessingFee *bool`, which has the customer pay QBitFlow's processing fee, grossed up so
    you keep the price and your lines; `nil` follows the dashboard's
    `checkout.customerPaysProcessingFee` setting). The customer pays the price plus every line,
    then the network fee on top; test mode caps that amount, fees included, at `"5"`. A
    subscription's are locked with its price: every bill charges both. `Payment.PriceUSD` and
    `Fees`, `Subscription.Fees`, `Transaction.PriceUSD` and `Fees`, `FeeLine` and the open enum
    `FeeLineType` (`FeeLineTypeCustom`, `FeeLineTypeProcessingFee`).
-   Merchant metadata (`map[string]string`) on `CreateProductParams` / `UpdateProductParams`,
    `CreateCustomerParams` / `UpdateCustomerParams`, both session creates, and
    `Subscriptions.Update` (`PATCH /subscriptions/:uuid`, `UpdateSubscriptionParams`): merged on
    the updates, a key set to `""` deleting it.
-   `Transactions.List` / `Iterate` (`GET /transactions`): the feed of payments and bills, each a
    `Transaction` with its `Type`, filtered by `Type`, `SubscriptionUUID`, `Refunded` and the
    shared filters.
-   `Payments.List` with filters (customer, product, dates, `Reference`, `Refunded`,
    `IncludeMembers` / `UserUUID`); `Payments.Get` with `ReadParams`. `Failures.List`: the failed
    payment attempts (`Attempted`, `AttemptedUSD`, `Currency`).
-   `Products.List` / `Iterate` (paginated: `Limit`, `Cursor`, `IncludeHidden`, `Subscription`,
    `Reference`) and subscription products; `Customers.List` / `Iterate` (`Email`, `Reference`,
    `Verified`).
-   `Subscriptions.List` (status, reference and the shared filters), `Update`, `ListBills`,
    `GetBill` (`GET /bills/:uuid`); `Subscription.CurrentPeriodEnd` (grant access while now is
    before it), `ActionRequired`, `PriceUSD`, `CancellationReason`, `Dunning`, `Metadata`,
    `ManagementPageLink`; `Allowance`.
-   `Refunds.List` / `Iterate` (`Status` (repeatable), `TxUUID`, `Held`, `IncludeMembers`,
    `UserUUID`), `Refunds.Get` (`GET /refunds/:uuid`) and `Refunds.Create` (`POST /refunds`): a
    merchant's refund of a payment or a bill, signed in the dashboard. A `Refund` carries `Paid`,
    `Amount`, its `Currency` and, once approved, its `Settlement`.
-   `Members` (`List`, `Get`, `Update`, `Remove`, `Trust`, `ListHeldFunds`, `GetHeldFunds`,
    `GetOwnHeldFunds`) and `Invitations` (`Create`, `List`, `Revoke`) for marketplaces.
-   `Wallets` (`List` with balances, `ListForMember`, `ListAcceptedCurrencies`),
    `Accounting.ExportCSV`, `Currencies.Get`.
-   `TokenAmount` (`Amount`, `AmountMinUnits`, `AmountUSD`), the shape of every token amount.
-   `Webhooks.Endpoints` (`List` / `Iterate` paginated, `Create` with the secret shown once,
    `Get`, `Update`, `Delete`) and `Webhooks.Events` (the event log: `List`, `Iterate`, `Get` with
    its deliveries).
-   Webhook verification without a client: `VerifyWebhook`, `VerifyWebhookRequest` (net/http,
    1 MiB, the body put back), `ConstructEvent`, `ParseEvent`, `WithTolerance`, `WithClock`;
    every `v1=` signature is checked, so a secret rotation needs nothing on the receiver's side.
    The header constants `SignatureHeader`, `EventIDHeader`, `EventTypeHeader`,
    `WebhookVersionHeader`.
-   Typed events: an `Event` with an accessor per type (`PaymentCompleted()`,
    `SubscriptionBilled()`, `RefundApproved()`, `MemberJoined()`, `CheckoutExpired()`… 15 types)
    and `DecodeData[T]()` for the others; `EventType` constants.
-   An iterator (`iter.Seq2`) next to every paginated list: `Iterate`, `IterateBills`.
-   Retries: reads and the 7 creates are retried on network errors, timeouts, 5xx, 429 (after
    `Retry-After`, at most 60 s) and `409 idempotency_key_in_use`, 3 times by default with a
    1 s · 2ⁿ back-off; `WithMaxRetries(0)` disables them; `IsRetryable`.
-   An `Idempotency-Key` on each create, generated per call and reused by its retries;
    `WithIdempotencyKey` for retries across processes.
-   `WithRequestID` (`X-Request-Id`) and the request id on every error.
-   Client-side validation mirroring the API v2 validators, through each params type's
    `Validate` method, run before every request (no request is sent on failure): names, texts,
    references (also on the `Reference` filters and `GetByReference`), emails (also the customers'
    `Email` filter), phones, URLs, USD amounts as strings (at most 2 decimals, above 0, at most
    1,000,000), percents as strings (at most 2 decimals and 16 characters; `RefundPercent` above
    0 up to 100, `OrganizationFeePercent` 0 to 50, required on `UpdateMemberParams`), metadata
    (at most 50 keys of 1 to 40 of `A-Z a-z 0-9 _ - . :`, values of at most 500 characters
    without control or bidirectional characters), durations, cursors (at most 200 characters),
    the refunds' `Status` filter (at most 5 known statuses), the transactions' `Type`, the
    `release@` transaction prefix, and a clearable `UpdateCustomerParams.LastName`.
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
    -   `CheckoutSessions.WaitForCompletion(ctx, uuid, *WaitOptions)`: polls `Get` until the
        session is final (default timeout 10 minutes, interval 3 s, at least 1 s) and returns it,
        or the last status seen at the timeout.
    -   `Subscription.HasAccess(at)`: `CurrentPeriodEnd` set and `at` before it.
    -   `FormatAmount` / `ParseAmount` and `Currency.FormatAmount`: exact conversions between a
        token's smallest unit and a decimal amount (string arithmetic, never a float).
    -   `Accounting.ExportJSONRange` / `ExportCSVRange`: an export over any range, in windows of
        at most 92 days, both ends included (`[from, from+91d]`), joined in order (the CSV header
        once).
    -   `NewClientFromEnv`: a client from `QBITFLOW_API_KEY`, `QBITFLOW_BASE_URL` and
        `QBITFLOW_ON_BEHALF_OF`, the options given overriding them.
    -   `PlaceholderUUID` and `PlaceholderTransactionType`: the `{{UUID}}` and
        `{{TRANSACTION_TYPE}}` redirect placeholders.
-   Examples: `checkout`, `subscriptions`, `marketplace`, `webhook_handler` (a `WebhookRouter`),
    `errors_and_retries`; `MIGRATION-v3.md`.

### Changed

-   `AccountingEvent` follows the API's export: amounts in the currency's units as decimal
    strings, each with its `…MinUnits` twin (`GrossAmountMinUnits`, `PlatformFeeMinUnits`,
    `ReferralFeeMinUnits`, `OrganizationFeeMinUnits`, `NetAmountMinUnits`,
    `NetworkFeesMinUnits`); USD amounts and rates as decimal strings; `Currency` (was
    `tokenSymbol`), `CurrencyPriceUSD`, and the network fees apart from the net
    (`NetworkFeesPaidBy`, the open enum `NetworkFeesPayer`: `payer`, `merchant`, `organization`).
    `PaymentReference` / `RelatedPaymentReference` → `Reference` / `RelatedReference`,
    `PaymentUUID` / `RelatedPaymentUUID` → `TxUUID` / `RelatedTxUUID`; a fee that doesn't apply is
    absent (was `"0"`). The always-empty `UserName`, `UserLastName`, `CustomerName` and
    `CustomerLastName` are gone. The export window is at most 92 days, both ends included.
-   Responses decode leniently: a field the API leaves out gets its zero value; a value of the
    wrong JSON type is a `ServerError` (a JSON number where a decimal string is expected
    included). Unknown enum values are kept as is, decimal strings are never parsed. Every answer
    is a JSON object: a `null` body is a `ServerError`.
-   Redirects are never followed: a 3xx is a `ServerError` (2.1.0 replayed the request, API key
    included, against the redirect's target).
-   A 2xx with an empty or non-JSON body where a result is expected is a `ServerError`, not a
    zero-valued result.
-   A request body JSON cannot represent (invalid UTF-8) is a `ValidationError`, and nothing is
    sent.
-   Update params send an explicit `""` for the clearable fields
    (`UpdateCustomerParams.LastName` / `PhoneNumber` / `Address`,
    `UpdateProductParams.Description`, `UpdateWebhookEndpointParams.Description`): `new("")`
    clears them, `nil` leaves them.
-   The event log's cursor is opaque (was an event id); the subscriptions' `Status` and the event
    log's `Type` filters are sent as given (the API checks them).
-   `CustomerSummary.Deleted` is a `bool`; `DeliveryAttempt.Skipped` a `bool`.
-   Error messages read `<message> (status <status>, code <code>, request <requestId>)`,
    followed by each field error; the SDK's own validation messages never echo the rejected
    values.

### Removed

-   `PayAsYouGo` (the API has no pay-as-you-go routes), `Users`, `Claims`, `ApiKeys`,
    `TransactionStatus` (and its WebSocket URL helper), `Payments.GetSession`,
    `Subscriptions.GetSession`, `Payments.GetCustomerForTransaction` (read
    `Payment.CustomerUUID`, then `Customers.Get`), `Refunds.GetByTransactionUUID` (read
    `Payment.Refund` / `Bill.Refund`, or `Refunds.List` with `TxUUID`), `Customers.GetByEmail`
    (`Customers.List` with `Email`), `Subscriptions.GetPaymentHistory` and the public bill
    history (the customer's page; `ListBills` has every bill).
-   The v1 webhook verifier: `CanonicalJSON`, `ComputeWebhookSignature`,
    `VerifyWebhookSignature`, `ExtractWebhookHeaders`, `WebhookHeaders`, `VerifyOptions`
    (including `SkipTimestampCheck`), `DefaultMaxTimestampAge`, `TEST_WEBHOOK_ID`, the
    `X-Webhook-*` header constants, and the v1 payloads `SessionWebhookResponse` and
    `SubscriptionStatusTransitionWebhook`.
-   `SessionCheckout`, `StatusResponse`, `SuccessResponse`, `ErrorResponse`, `Organization`,
    `ApiKey`, `ClaimFund`, `CreateClaimRequestResponse`, `TransactionShortType`,
    `SubscriptionType`, `CursorQueryBuilder`, `ValidateProductTextForTest`; `legacyId` is not
    modelled.

### Fixed

-   **Field validation errors were read from the wrong key** (a top-level `errors` list instead
    of the API's `details.errors`), so every field of a 400 came back empty. `FieldErrors` now
    names each failing input by the name sent, dotted when nested (`frequency.unit`).
-   User-supplied path segments and query values are escaped: an id containing `/`, `?`, `#`,
    `%` or `&` no longer changes the request.
-   Lengths are counted in characters (code points), as the API does, not in bytes: accented
    and CJK names are no longer refused locally.
-   The webhook timestamp check is overflow-safe: a timestamp centuries away is refused instead
    of wrapping into the accepted window.

The website snippet ids are unchanged (none renamed or retired).

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




