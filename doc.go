// Package qbitflow is the Go SDK for the QBitFlow API (v2): crypto checkouts, one-time
// payments, subscriptions, refunds, marketplaces (members, invitations, held funds), wallets,
// accounting exports and webhooks.
//
// Create one [Client] per API key and share it; it is safe for concurrent use:
//
//	client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	me, err := client.Me(ctx) // the recommended start-up check
//
// Every call takes a [context.Context] and optional [RequestOption]s
// ([WithOnBehalfOf], [WithIdempotencyKey], [WithRequestID]). Failures are typed errors
// matched with [errors.As] ([*NotFoundError], [*ValidationError], [*RateLimitError], …);
// every one of them carries an [*APIError] with the HTTP status, the API's code, its
// request id and its field errors.
//
// Reads and the idempotent creates are retried on network errors, 5xx and 429 (see
// [WithMaxRetries]); creates reuse one Idempotency-Key across their retries.
//
// Webhooks are verified without a client: [VerifyWebhook], [VerifyWebhookRequest],
// [ConstructEvent] and [ParseEvent].
package qbitflow
