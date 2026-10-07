// Package qbitflow is the Go SDK for the QBitFlow API v2: non-custodial crypto checkouts,
// one-time payments, subscriptions, refunds, marketplaces (members, invitations, held funds),
// wallets, accounting exports and signed webhooks.
//
// # Getting started
//
// Create one [Client] per API key with [NewClient] and share it: it is safe for concurrent use.
// [Client.Me] is the recommended start-up check (it says what the key is, and its mode).
//
//	client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"))
//	if err != nil {
//		log.Fatal(err) // not an sk_… key
//	}
//	ctx := context.Background()
//	if _, err := client.Me(ctx); err != nil {
//		log.Fatal(err)
//	}
//
//	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
//		ProductName: "Premium access",
//		Price:       4.99,
//		Reference:   "order-1042",
//		SuccessURL:  "https://shop.example.com/thanks?session={{UUID}}",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(session.Link) // send the customer there; fulfil on the payment.completed webhook
//
// The client's fields group the methods by area: Products, Customers, CheckoutSessions,
// Payments, Failures, Subscriptions, Refunds, Members, Invitations, Wallets, Accounting,
// Webhooks (with Webhooks.Endpoints and Webhooks.Events) and Currencies.
//
// # Requests
//
// Every method takes a [context.Context] first and optional [RequestOption] values last:
// [WithOnBehalfOf] acts in a member's space (an organization key; [Client.OnBehalfOf] does it
// for a whole client), [WithIdempotencyKey] sets a create's Idempotency-Key, [WithRequestID]
// sends X-Request-Id. Params are pointers to …Params structs, nil when every field is optional.
//
// Ids are strings: UUIDs, and prefixed transaction ids (pay@…, sub@…, sub-hist@…, refund@…)
// passed back verbatim; only currency ids are numbers. In responses an optional string is ""
// when absent and any other optional field is a pointer, nil when absent. Exact token amounts
// are decimal strings, fee rates percents, enums open string types (unknown values are kept).
//
// # Lists
//
// Paginated lists return a [Page]; each has an Iterate twin (an [iter.Seq2]) that fetches the
// pages lazily, keeps the filters, and yields an error once before stopping:
//
//	for p, err := range client.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(p.UUID, p.Amount)
//	}
//
// # Errors
//
// Failures are typed errors matched with [errors.As]: [*ValidationError] (also returned before
// sending for invalid input), [*BadRequestError], [*AuthenticationError],
// [*PermissionDeniedError], [*NotFoundError], [*ConflictError], [*GoneError],
// [*IdempotencyError], [*RateLimitError], [*ServerError], [*NetworkError] and
// [*WebhookSignatureError]. Each embeds an [*APIError] carrying the HTTP status, the API's
// code, its details, the request id and the field errors:
//
//	_, err := client.Customers.Create(ctx, &qbitflow.CreateCustomerParams{Name: "Ada", Email: "ada@example.com"})
//	var conflict *qbitflow.ConflictError
//	if errors.As(err, &conflict) && conflict.Code == "unique_violation" {
//		fmt.Println("already taken:", conflict.Details["field"], "request", conflict.RequestID)
//	}
//
// # Retries
//
// Reads and the seven creates are retried on network errors, timeouts, 5xx and 429, three
// times by default with a 1 s, 2 s, 4 s back-off ([WithMaxRetries]; 0 disables). Each create
// sends one Idempotency-Key, reused by its retries, so a retry never creates twice. Other
// writes are never retried.
//
// # Webhooks
//
// Deliveries are verified without a client: [VerifyWebhookRequest] (net/http), [VerifyWebhook]
// and [ConstructEvent] check the QBitFlow-Signature header over the raw body, accepting either
// signature during a secret rotation; [ParseEvent] parses the [Event], whose typed accessors
// decode its data. Deliveries are at least once: deduplicate on Event.ID, and answer 2xx to
// every event, the ignored ones too.
//
//	func handle(w http.ResponseWriter, r *http.Request) {
//		body, err := qbitflow.VerifyWebhookRequest(r, os.Getenv("QBITFLOW_WEBHOOK_SECRET"))
//		if err != nil {
//			http.Error(w, "invalid signature", http.StatusBadRequest)
//			return
//		}
//		event, err := qbitflow.ParseEvent(body)
//		if err != nil {
//			http.Error(w, "not a v2 event", http.StatusBadRequest)
//			return
//		}
//		if event.Type == qbitflow.EventTypePaymentCompleted {
//			if payment, err := event.PaymentCompleted(); err == nil {
//				log.Println("fulfil", payment.Reference)
//			}
//		}
//		w.WriteHeader(http.StatusOK)
//	}
//
// The README (https://github.com/QBitFlow/qbitflow-go-sdk) covers every service, and
// MIGRATION-v3.md the move from 2.x.
package qbitflow
