# Tests

This directory holds the QBitFlow Go SDK test suite: offline unit tests that run
everywhere, and integration tests that need an API server.

## Running

```bash
# Offline suite (no server, no key) — this is what CI runs
go test ./...
go test -race ./...

# Offline + integration, against a server. Both variables are required: the base URL is
# never defaulted. From the SDK root, with the workspace credentials file:
set -a; source ../.local.env; set +a   # exports QBITFLOW_API_KEY and QBITFLOW_BASE_URL
go test -v ./tests/...

# One test (every integration test creates the records it needs)
go test -v ./tests/... -run TestPaymentSession

# Coverage / benchmarks
go test -coverprofile=coverage.out ./tests/... && go tool cover -html=coverage.out
go test -bench=. ./tests/...
```

| `QBITFLOW_API_KEY` | `QBITFLOW_BASE_URL` | Integration tests |
|---|---|---|
| unset | any | skipped — the offline suite still runs |
| set | unset | **fail**, with a message asking for the base URL |
| set | set | run against `QBITFLOW_BASE_URL` |

## Offline tests (always run)

They use `net/http/httptest` stub servers, so they need neither a key nor network access:

- `error_mapping_test.go` — HTTP status → typed error (400/422 `ValidationError` with every
  field, 401 `AuthenticationError`, 403 `ForbiddenError`, 404 `NotFoundError`, 409
  `ConflictError`, 429 `RateLimitError` + `RetryAfter`, other 4xx `QBitFlowError`, 3xx / 5xx /
  empty or non-JSON 2xx `ServerError`, transport `NetworkError`), message precedence,
  `errors[]` on any status, every type matching `*QBitFlowError`, `StatusCodeOf` / `FieldsOf`
  on wrapped errors, a real redirect (with `Location`) not followed, CSV export error bodies.
- `request_building_test.go` — the method, path and query of every endpoint, path/query
  escaping, identifier / nil-argument / page-limit guards, per-service
  and client-level `OnBehalfOf`, webhook `Verify` semantics (false only on 400) and request
  body, `closeToExpireError`, subscription-webhook envelope, response hydration (every field,
  absent / null → zero values, nullable fields, currency objects, null lists → empty lists,
  wrong JSON type → `ServerError`), session kind helpers, unknown enum values.
- `validation_test.go` — client-side validation parity with the API, including the cases
  verified against the live API (whitespace-only names accepted, `Jo²` rejected, inline
  price 0 rejected, `customerUUID` rules, trial period and `minPeriods` 0 accepted, uint32
  bounds, empty optional strings left out of the body, accounting dates).
- `webhook_verify_test.go` — canonical JSON, signature computation, local verification,
  header extraction, replay-window boundaries and overflow safety, timestamp parsing, the
  request-body size bound, and the thirteen cross-SDK golden vectors.
- `../pkg/qbitflow/retry_test.go` — the retry policy (GET only; never POST/PUT/DELETE, never
  force-cancel / execute-billing / claim-funds test-trigger; network + 5xx only; 1s/2s/4s
  back-off; `MaxRetries` 0 / negative / `DisableRetries`), redirects never followed,
  `Retry-After` (seconds and HTTP date), blank API key, unencodable bodies, empty bodies,
  and the request headers.

## Integration tests

`integration_test.go` exercises the SDK against a real server (see the table above).
Every test creates the customers, products and users it needs and deletes them afterwards,
so any test can run on its own. An unexpected error is a failure, never a skip.

Requirements:

1. A running QBitFlow API (`GET $QBITFLOW_BASE_URL/healthz` → 200).
2. `QBITFLOW_API_KEY` — a **live-scope** organization key bound to a dedicated test user.
   Live keys hit local chains only, and cannot use `test=true`.
3. `QBITFLOW_BASE_URL` — the server to test against.

Never hard-code a key in a test, example or fixture.

`additional_tests/` holds an interactive walkthrough (`go run ./tests/additional_tests`)
that needs you to complete payments in a browser; it also requires both variables.

## Adding tests

- Offline behaviour goes in one of the offline files (stub the server with `httptest`).
- A test that needs the API starts with `client := requireLive(t)` and creates its own
  records (`newCustomer`, `newProduct`, `newUser`).
- Name tests `Test<Feature>`, prefer table-driven cases, and never claim a green run that
  did not happen — if the server was down, the integration tests were skipped, not passed.
