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
	// PlaceholderTransactionType becomes the session's transaction type (payment, createSubscription…).
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

// CreatePayment opens a one-time payment checkout session
// (POST /transaction/session-checkout/new/payment, 201) and returns its link, its pay@… id and
// its expiry. It sends an Idempotency-Key and is retried on transient failures.
//
// Send the customer to Link. The Payment exists, with the same id, once the customer's
// transaction is confirmed: fulfil on the payment.completed webhook (or GetStatus "completed"),
// give up on checkout.expired. SuccessURL and CancelURL may carry {{UUID}} and
// {{TRANSACTION_TYPE}}, filled in on redirect.
//
// Errors: 400 validation_failed (test mode caps the price at $5, details.max; live mode needs
// https URLs); 404 for an unknown product or customer; 409 merchant_not_ready
// (details.reason) when the space's wallets accept no currency.
func (s *CheckoutSessionsService) CreatePayment(ctx context.Context, params *CreatePaymentSessionParams, opts ...RequestOption) (*CheckoutSession, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSession](ctx, endpoint{
		method: http.MethodPost, path: "/transaction/session-checkout/new/payment", body: params, idempotent: true,
	}, opts)
}

// CreateSubscription opens a subscription checkout session
// (POST /transaction/session-checkout/new/subscription, 201) and returns its link, its sub@…
// id and its expiry. It sends an Idempotency-Key and is retried on transient failures.
//
// The Subscription exists, with the same id, once the customer signed (a free trial) or paid:
// act on subscription.created. Terms left unset come from a subscription product. Errors as
// CreatePayment, plus 400 on frequency / trialPeriod (the frequency minimum is 1 hour live, 5
// minutes test); a member's key needs the members.subscriptions policy (403 policy_disabled).
func (s *CheckoutSessionsService) CreateSubscription(ctx context.Context, params *CreateSubscriptionSessionParams, opts ...RequestOption) (*CheckoutSession, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSession](ctx, endpoint{
		method: http.MethodPost, path: "/transaction/session-checkout/new/subscription", body: params, idempotent: true,
	}, opts)
}

// GetStatus returns where a checkout session stands (GET /transaction/session-checkout/:uuid/status;
// uuid is the session's pay@… or sub@… id): created (waiting for its customer),
// waitingConfirmation, completed or expired (the only final ones).
//
// A failed attempt is "created" with LastAttempt set: the customer may still pay from the same
// checkout, so never cancel an order on it. An expired session answers "expired" for some
// days, then 404 like an unknown id or another space's session.
func (s *CheckoutSessionsService) GetStatus(ctx context.Context, uuid string, opts ...RequestOption) (*CheckoutSessionStatus, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSessionStatus](ctx, endpoint{
		method: http.MethodGet, path: pathf("/transaction/session-checkout/%s/status", uuid),
	}, opts)
}

// Expire ends a checkout session now (POST /transaction/session-checkout/:uuid/expire) and
// returns its status (checkout.expired follows). Not retried.
//
// Errors: 409 tx_already_sent once the customer paid or is paying.
func (s *CheckoutSessionsService) Expire(ctx context.Context, uuid string, opts ...RequestOption) (*CheckoutSessionStatus, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[CheckoutSessionStatus](ctx, endpoint{
		method: http.MethodPost, path: pathf("/transaction/session-checkout/%s/expire", uuid),
	}, opts)
}

// WaitForCompletion polls GetStatus until the session is final (completed or expired) and
// returns that status. When opts.Timeout (default 10 minutes) elapses first, it returns the
// last status seen, which is not final: check its Status. GetStatus's errors are returned as
// they are (a 404 included), and the context's cancellation as its error (errors.Is(err,
// context.Canceled) holds either way).
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
		status, err := s.GetStatus(ctx, uuid, reqOpts...)
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
