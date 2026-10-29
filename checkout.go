package qbitflow

import (
	"context"
	"net/http"
	"time"
)

// Redirect placeholders for SuccessURL and CancelURL: QBitFlow replaces them literally when it
// redirects the customer (the SDK never URL-encodes them).
//
//	SuccessURL: "https://shop.example.com/thanks?session=" + qbitflow.PlaceholderUUID
const (
	// PlaceholderUUID becomes the session's (and its transaction's) id: pay@… or sub@….
	PlaceholderUUID = "{{UUID}}"
	// PlaceholderTransactionType becomes the session's transaction type (payment or createSubscription).
	PlaceholderTransactionType = "{{TRANSACTION_TYPE}}"
)

// WaitForCompletion's defaults.
const (
	// DefaultWaitTimeout is how long WaitForCompletion polls when WaitOptions.Timeout is 0.
	DefaultWaitTimeout = 10 * time.Minute
	// DefaultWaitInterval is the wait between polls when WaitOptions.Interval is 0.
	DefaultWaitInterval = 3 * time.Second
	// minWaitInterval is the shortest wait between polls.
	minWaitInterval = time.Second
)

// WaitOptions configures CheckoutSessions.WaitForCompletion. nil, or a zero field, keeps the
// default.
type WaitOptions struct {
	// Timeout is how long to poll before giving up (default 10 minutes).
	Timeout time.Duration
	// Interval is the wait between two polls (default 3 s, at least 1 s).
	Interval time.Duration
}

// CreatePayment opens a one-time payment checkout session (POST /checkout-sessions/payment,
// 201) and returns its link, its pay@… id and its expiry. It sends an Idempotency-Key and is
// retried on transient failures.
//
// Send the customer to Link. The Payment exists, with the same id, once the customer's
// transaction is confirmed: fulfil on the payment.completed webhook (or Get "completed"), give
// up on checkout.expired. SuccessURL and CancelURL may carry PlaceholderUUID and
// PlaceholderTransactionType, filled in on redirect. A member's own key may send them while its
// organization's members.checkoutRedirects policy allows it (the default); with the policy off
// they are a 403 policy_disabled and the organization's checkout defaults apply. An organization
// key acting On-Behalf-Of the member may always send them.
//
// Fees (optional) adds lines to the price (a tax, shipping, QBitFlow's processing fee): the
// customer pays the price plus every line (the payment's AmountUSD), and the network fee on top
// of that. Metadata is copied onto the payment and into its webhooks.
//
// Errors: 400 validation_failed (test mode caps the amount, fees included, at "5", details.max;
// live mode needs https URLs); 404 for an unknown product or customer; 409 unique_violation on
// reference; 409 merchant_not_ready (details.reason) when the space's wallets accept no
// currency; 410 merchant_closed for a removed member's or a closed organization's space; 403
// policy_disabled with details.policy "members.checkoutRedirects" and details.field "successUrl"
// or "cancelUrl" (successUrl first) when a member's key sends either while that policy is off.
func (s *CheckoutSessionsService) CreatePayment(ctx context.Context, params *CreatePaymentSessionParams, opts ...RequestOption) (*CreatedCheckoutSession, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[CreatedCheckoutSession](ctx, endpoint{
		method: http.MethodPost, path: "/checkout-sessions/payment", body: params, idempotent: true,
	}, opts)
}

// CreateSubscription opens a subscription checkout session (POST /checkout-sessions/subscription,
// 201) and returns its link, its sub@… id and its expiry. It sends an Idempotency-Key and is
// retried on transient failures.
//
// The Subscription exists, with the same id, once the customer signed (a free trial) or paid:
// act on subscription.created. Terms left unset come from a subscription product.
//
// Fees (optional) adds lines to the price, as for CreatePayment, locked with it: every bill
// charges the price plus every line (the Subscription's PriceUSD plus Fees), and the network fee
// on top of that.
//
// Errors as CreatePayment, plus 400 on frequency / trialPeriod (the frequency minimum is 1 hour
// live, 5 minutes test) and on priceUsd for a live subscription at "0.25" or less (details.min);
// a member's key needs the members.subscriptions policy (403 policy_disabled); 409
// merchant_not_ready with details.reason fundsHeld for a member whose funds the organization
// holds (a held member sells one-time payments only, until Members.Trust).
func (s *CheckoutSessionsService) CreateSubscription(ctx context.Context, params *CreateSubscriptionSessionParams, opts ...RequestOption) (*CreatedCheckoutSession, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[CreatedCheckoutSession](ctx, endpoint{
		method: http.MethodPost, path: "/checkout-sessions/subscription", body: params, idempotent: true,
	}, opts)
}

// Get returns where a checkout session stands (GET /checkout-sessions/:uuid; uuid is the
// session's pay@… or sub@… id, or its bare UUID): created (waiting for its customer),
// waitingConfirmation, completed or expired (the only final ones). It covers both phases: the
// session, then the payment or subscription it created.
//
// A failed attempt is "created" with LastAttempt set: the customer may still pay from the same
// checkout, so never cancel an order on it. An expired session answers "expired" for as long as
// QBitFlow keeps the record, then 404 like an unknown id or another space's session.
func (s *CheckoutSessionsService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*CheckoutSessionStatus, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSessionStatus](ctx, endpoint{
		method: http.MethodGet, path: pathf("/checkout-sessions/%s", uuid),
	}, opts)
}

// Expire ends a checkout session now (POST /checkout-sessions/:uuid/expire) and returns its
// status, expired, with ExpiresAt the time it was expired (checkout.expired follows). Expiring
// an expired session answers expired again, without ExpiresAt (only the first answer has it).
// Not retried.
//
// Errors: 409 tx_already_sent once the customer paid or is paying.
func (s *CheckoutSessionsService) Expire(ctx context.Context, uuid string, opts ...RequestOption) (*CheckoutSessionStatus, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSessionStatus](ctx, endpoint{
		method: http.MethodPost, path: pathf("/checkout-sessions/%s/expire", uuid),
	}, opts)
}

// WaitForCompletion polls Get until the session is final (completed or expired) and returns
// that status. When opts.Timeout (default 10 minutes) elapses first, it returns the last status
// seen, which is not final: check its Status. Get's errors are returned as they are (a 404
// included), and the context's cancellation as its error (errors.Is(err, context.Canceled)
// holds either way). The interval is at least 1 s; the last wait is shortened to the time
// left, then one last poll is made at the deadline.
//
// For scripts, tests and back-office jobs: fulfil orders on the payment.completed webhook,
// never by keeping a request waiting on this.
func (s *CheckoutSessionsService) WaitForCompletion(ctx context.Context, uuid string, opts *WaitOptions, reqOpts ...RequestOption) (*CheckoutSessionStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	timeout, interval := DefaultWaitTimeout, DefaultWaitInterval
	if opts != nil {
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		if opts.Interval != 0 {
			interval = max(opts.Interval, minWaitInterval)
		}
	}

	t := s.client.t
	deadline := t.now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		status, err := s.Get(ctx, uuid, reqOpts...)
		if err != nil {
			return nil, err
		}
		if status.Status == CheckoutSessionStatusCompleted || status.Status == CheckoutSessionStatusExpired {
			return status, nil
		}
		remaining := deadline.Sub(t.now())
		if remaining <= 0 {
			return status, nil
		}
		if err := t.sleep(ctx, min(interval, remaining)); err != nil {
			return nil, err
		}
	}
}
