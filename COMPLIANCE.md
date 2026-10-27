# Compliance notes

Facts about what the QBitFlow Go SDK does with data, for your own security and privacy reviews.
The SDK is a client library: it runs inside your application and talks only to the QBitFlow API
you configure.

## Data the SDK handles

- **Your credentials.** The API key you give `NewClient` (sent as `X-API-Key` on every request) and
  the webhook secrets you give the verification functions (used locally to compute HMAC-SHA256
  signatures, never sent anywhere). Neither is ever included in an error message or in the output
  of `fmt` on a client, a router or an error.
- **What you send.** The parameters of the calls you make: product and checkout details,
  customer details you choose to pass (names, email, phone number, address), your references and
  `Metadata`. Don't put secrets or personal data in `Metadata`.
- **What the API returns.** Payments, subscriptions, refunds, customers, members, wallets and
  accounting rows, decoded into Go values and handed to your code.
- **Webhook bodies** you pass to `VerifyWebhook`, `ConstructEvent`, `ParseEvent` or the
  `WebhookRouter`, read in memory to verify and decode them.

## What it doesn't do

- **No storage.** The SDK writes nothing to disk and keeps no cache: every value lives only as long
  as your code holds it.
- **No telemetry.** It sends no analytics, usage data or crash reports. The only network traffic
  is the API requests you make (plus their automatic retries), to the base URL you configure
  (`https://api.qbitflow.app/v2` by default); `User-Agent: qbitflow-go/<version>` identifies the
  SDK.
- **No logging.** It writes nothing to stdout, stderr or any logger. What your application logs
  from SDK results and errors is under your control: an `APIError` keeps the error response's body
  (`RawBody`) for debugging, so treat it like the response itself.
- **No custom cryptography.** TLS comes from Go's `net/http` and `crypto/tls`; webhook signatures
  use `crypto/hmac` and `crypto/sha256` with a constant-time comparison.

## Data location and regulation

Data is processed and stored by QBitFlow's services, not by the SDK; QBitFlow's terms and privacy
policy at [qbitflow.app](https://qbitflow.app) govern it. Payments are non-custodial: customers pay
from their own wallets to yours. Checks such as sanctions screening and transaction monitoring are
performed by QBitFlow's services; the SDK does not implement them.

## Dependencies

None outside the Go standard library (see `go.mod`). The module requires Go 1.27 or later.

## Security

See [SECURITY.md](SECURITY.md) for supported versions, how to report a vulnerability, and the
SDK's security properties.
