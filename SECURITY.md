# Security policy

## Supported versions

| Version | Receives security fixes |
|---|---|
| 3.x (the latest release) | yes |
| 2.x and earlier | no: upgrade to 3.x ([MIGRATION-v3.md](MIGRATION-v3.md)) |

Fixes land in a new 3.x release; only the latest 3.x release is supported, so stay on it
(`go get github.com/QBitFlow/qbitflow-go-sdk/v3@latest`).

## Reporting a vulnerability

Email **security@qbitflow.app**. Please do **not** open a public GitHub issue, pull request or
discussion for a suspected vulnerability.

Include what you can:

- the SDK version (`qbitflow.Version`) and your Go version;
- a description of the issue and its impact;
- steps or a minimal program that reproduce it;
- any logs or output, **with API keys, webhook secrets and personal data removed**.

Never send a live API key or webhook secret: if one may have been exposed, revoke or rotate it in
the QBitFlow dashboard first. We will acknowledge your report and keep you informed while we
investigate and fix it.

## Disclosure

Once a fix is released, the vulnerability is published as a
[GitHub Security Advisory](https://github.com/QBitFlow/qbitflow-go-sdk/security/advisories) for this
repository, and the release is described in [CHANGELOG.md](CHANGELOG.md) under **Security**.

## Security properties of the SDK

What the code does today, for your own review:

- **The API key** is sent only in the `X-API-Key` header, to the configured base URL. The client
  keeps it behind a pointer that no `fmt` verb prints: logging a `Client`, a service, a
  `WebhookRouter` or any SDK error never shows the key or the webhook secret, and the
  constructor's errors never quote the key. The key is checked for shape only (`sk_…`, printable
  ASCII without spaces), so it can never inject a header.
- **Transport.** Requests go through Go's `net/http` (TLS with the system's certificate roots) to
  `https://api.qbitflow.app/v2` by default. `WithBaseURL` accepts `http` and `https` URLs only, and
  refuses one carrying credentials, a query or a fragment. Redirects are never followed (a 3xx is
  a `*ServerError`), so the key is never forwarded to another host. Each attempt is bounded by a
  timeout (30 s by default) that covers the whole response body.
- **Webhook signatures** are verified locally with HMAC-SHA256 over the timestamp and the raw body,
  compared in constant time (`hmac.Equal`) against every `v1` signature of the header. A
  timestamp more than 5 minutes from the clock (configurable with `WithTolerance`) is refused, to
  limit replays. `VerifyWebhookRequest` and the router's `ServeHTTP` read at most 1 MiB of body.
- **Input checks.** Every params type is validated before a request is sent (lengths, character
  sets, URLs, amounts as decimal strings, UUIDs, path segments `.` and `..` refused), and the
  `Idempotency-Key`, `X-Request-Id` and `On-Behalf-Of` values are checked so they can't carry line
  breaks. Validation errors name the field, not the rejected value.
- **No telemetry, no logging.** The SDK sends nothing but the requests you make, writes nothing to
  stdout, stderr or files, and stores nothing on disk.
- **No dependencies** outside the Go standard library. Module versions are immutable once
  published and verified by the Go checksum database (keep `GOSUMDB` enabled).
