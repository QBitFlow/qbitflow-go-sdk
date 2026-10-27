package qbitflow

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
)

// WebhookResult is what WebhookRouter.Handle made of a delivery: the HTTP status to answer,
// the event (nil when the body could not be verified or parsed) and the error, if any.
//
//   - 200: verified, parsed and handled (or ignored: an event no handler is registered for).
//   - 400: refused, no handler ran. Err is a *WebhookSignatureError (bad or stale signature) or
//     a *ValidationError (not a v2 event, or data that does not match its type).
//   - 500: a handler returned Err (QBitFlow delivers the event again later), or the router has
//     no secret (a *ValidationError).
type WebhookResult struct {
	// Status is the HTTP status to answer QBitFlow with.
	Status int
	// Event is the parsed event; nil on a 400 for a signature or a body that is not an event.
	Event *Event
	// Err is why the status is not 200; nil on a 200.
	Err error
}

// ResponseBody is the JSON body to answer with: {"received":true} on a 200, else
// {"error":"<short reason>"} (never the error's details). With a framework other than
// net/http, answer Status with it.
func (r WebhookResult) ResponseBody() []byte {
	switch {
	case r.Status >= 200 && r.Status < 300:
		return []byte(`{"received":true}`)
	case r.Status == http.StatusBadRequest:
		if errors.As(r.Err, new(*WebhookSignatureError)) {
			return []byte(`{"error":"invalid signature"}`)
		}
		return []byte(`{"error":"invalid event"}`)
	default:
		return []byte(`{"error":"internal error"}`)
	}
}

// WebhookRouterOption configures NewWebhookRouter: the VerifyOptions (WithTolerance,
// WithClock) and WithWebhookErrorHandler.
type WebhookRouterOption interface {
	applyWebhookRouter(*WebhookRouter)
}

// applyWebhookRouter makes every VerifyOption a WebhookRouterOption: the router verifies with it.
func (o VerifyOption) applyWebhookRouter(r *WebhookRouter) {
	if o != nil {
		r.verify = append(r.verify, o)
	}
}

type webhookErrorHandler func(*Event, error)

func (f webhookErrorHandler) applyWebhookRouter(r *WebhookRouter) { r.onError = f }

// WithWebhookErrorHandler sets a function the router calls with every result's error: a
// refused delivery (400; event nil when the body could not be parsed) and a failed handler or
// a missing secret (500). Errors are reported nowhere by default.
func WithWebhookErrorHandler(f func(event *Event, err error)) WebhookRouterOption {
	return webhookErrorHandler(f)
}

// webhookRoute is one event type's typed handlers and the decoder of its data.
type webhookRoute struct {
	decode   func(*Event) (any, error)
	handlers []func(context.Context, any, *Event) error
}

// WebhookRouter verifies, parses and dispatches webhook deliveries to handlers registered per
// event type, and answers them as QBitFlow expects. It is an http.Handler:
//
//	router := qbitflow.NewWebhookRouter(os.Getenv("QBITFLOW_WEBHOOK_SECRET")).
//		OnPaymentCompleted(func(ctx context.Context, p *qbitflow.PaymentCompleted, e *qbitflow.Event) error {
//			return fulfil(ctx, e.ID, p.Reference) // idempotent: deduplicate on e.ID
//		})
//	http.Handle("POST /webhooks/qbitflow", router)
//
// For each delivery, the handlers of its type run (in registration order, with the typed
// data), then the OnUnknown ones for a type this SDK does not know, then the OnAny ones. The
// first handler to return an error stops the others and the answer is 500: QBitFlow delivers
// the event again later, so handlers must be idempotent. An event no handler is registered
// for is acknowledged (200). Register the handlers before serving; the router is then safe
// for concurrent use.
type WebhookRouter struct {
	// secret is behind a pointer so that no fmt verb ever prints it (see transport.apiKey).
	secret  *string
	verify  []VerifyOption
	onError func(*Event, error)

	mu        sync.RWMutex
	routes    map[EventType]*webhookRoute
	onUnknown []func(context.Context, *Event) error
	onAny     []func(context.Context, *Event) error
}

// NewWebhookRouter creates a router verifying deliveries with the endpoint's whsec_… secret.
// Options: WithTolerance, WithClock, WithWebhookErrorHandler. An empty secret answers every
// delivery 500 (so QBitFlow retries it once the secret is configured).
func NewWebhookRouter(secret string, opts ...WebhookRouterOption) *WebhookRouter {
	r := &WebhookRouter{secret: &secret, routes: map[EventType]*webhookRoute{}}
	for _, opt := range opts {
		if opt != nil {
			opt.applyWebhookRouter(r)
		}
	}
	return r
}

// onType registers a typed handler for one event type.
func onType[T any](r *WebhookRouter, eventType EventType, decode func(*Event) (*T, error), fn func(context.Context, *T, *Event) error) *WebhookRouter {
	if fn == nil {
		panic("qbitflow: nil webhook handler for " + string(eventType))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	route := r.routes[eventType]
	if route == nil {
		route = &webhookRoute{decode: func(e *Event) (any, error) { return decode(e) }}
		r.routes[eventType] = route
	}
	route.handlers = append(route.handlers, func(ctx context.Context, data any, e *Event) error {
		return fn(ctx, data.(*T), e)
	})
	return r
}

// OnPaymentCompleted handles payment.completed (a payment confirmed: fulfil the order).
func (r *WebhookRouter) OnPaymentCompleted(fn func(ctx context.Context, data *PaymentCompleted, e *Event) error) *WebhookRouter {
	return onType(r, EventTypePaymentCompleted, (*Event).PaymentCompleted, fn)
}

// OnSubscriptionCreated handles subscription.created (status active, or trial): grant access
// until CurrentPeriodEnd.
func (r *WebhookRouter) OnSubscriptionCreated(fn func(ctx context.Context, data *Subscription, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionCreated, (*Event).SubscriptionCreated, fn)
}

// OnSubscriptionBilled handles subscription.billed (data.Bill, data.Subscription).
func (r *WebhookRouter) OnSubscriptionBilled(fn func(ctx context.Context, data *SubscriptionBilled, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionBilled, (*Event).SubscriptionBilled, fn)
}

// OnSubscriptionStatusChanged handles subscription.statusChanged.
func (r *WebhookRouter) OnSubscriptionStatusChanged(fn func(ctx context.Context, data *SubscriptionStatusChanged, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionStatusChanged, (*Event).SubscriptionStatusChanged, fn)
}

// OnSubscriptionActionRequiredChanged handles subscription.actionRequiredChanged.
func (r *WebhookRouter) OnSubscriptionActionRequiredChanged(fn func(ctx context.Context, data *SubscriptionActionRequiredChanged, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionActionRequiredChanged, (*Event).SubscriptionActionRequiredChanged, fn)
}

// OnSubscriptionBillingFailed handles subscription.billingFailed.
func (r *WebhookRouter) OnSubscriptionBillingFailed(fn func(ctx context.Context, data *SubscriptionBillingFailed, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionBillingFailed, (*Event).SubscriptionBillingFailed, fn)
}

// OnSubscriptionUpcomingBill handles subscription.upcomingBill.
func (r *WebhookRouter) OnSubscriptionUpcomingBill(fn func(ctx context.Context, data *SubscriptionUpcomingBill, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeSubscriptionUpcomingBill, (*Event).SubscriptionUpcomingBill, fn)
}

// OnRefundRequested handles refund.requested.
func (r *WebhookRouter) OnRefundRequested(fn func(ctx context.Context, data *Refund, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeRefundRequested, (*Event).RefundRequested, fn)
}

// OnRefundApproved handles refund.approved (was refund.completed): reverse the order, in full
// or in part (RefundPercent below 100).
func (r *WebhookRouter) OnRefundApproved(fn func(ctx context.Context, data *Refund, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeRefundApproved, (*Event).RefundApproved, fn)
}

// OnRefundRejected handles refund.rejected (was refund.denied).
func (r *WebhookRouter) OnRefundRejected(fn func(ctx context.Context, data *Refund, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeRefundRejected, (*Event).RefundRejected, fn)
}

// OnMemberJoined handles member.joined.
func (r *WebhookRouter) OnMemberJoined(fn func(ctx context.Context, data *MemberJoined, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeMemberJoined, (*Event).MemberJoined, fn)
}

// OnMemberRemoved handles member.removed.
func (r *WebhookRouter) OnMemberRemoved(fn func(ctx context.Context, data *Member, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeMemberRemoved, (*Event).MemberRemoved, fn)
}

// OnHeldFundsReleased handles heldFunds.released.
func (r *WebhookRouter) OnHeldFundsReleased(fn func(ctx context.Context, data *HeldFundsReleased, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeHeldFundsReleased, (*Event).HeldFundsReleased, fn)
}

// OnCheckoutExpired handles checkout.expired (data.Payment or data.Subscription is set).
func (r *WebhookRouter) OnCheckoutExpired(fn func(ctx context.Context, data *CheckoutExpired, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeCheckoutExpired, (*Event).CheckoutExpired, fn)
}

// OnWebhookTest handles webhook.test (the dashboard's test delivery).
func (r *WebhookRouter) OnWebhookTest(fn func(ctx context.Context, data *WebhookTest, e *Event) error) *WebhookRouter {
	return onType(r, EventTypeWebhookTest, (*Event).WebhookTest, fn)
}

// OnUnknown handles the events of a type this SDK does not know (newer than it): decode
// their Data with DecodeData. They run before the OnAny handlers.
func (r *WebhookRouter) OnUnknown(fn func(ctx context.Context, e *Event) error) *WebhookRouter {
	if fn == nil {
		panic("qbitflow: nil webhook handler for unknown events")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onUnknown = append(r.onUnknown, fn)
	return r
}

// OnAny handles every event, after its type's handlers (logging, storing the event).
func (r *WebhookRouter) OnAny(fn func(ctx context.Context, e *Event) error) *WebhookRouter {
	if fn == nil {
		panic("qbitflow: nil webhook handler for every event")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onAny = append(r.onAny, fn)
	return r
}

// Handle verifies rawBody (exactly as received) with its QBitFlow-Signature header, parses
// it, runs its handlers, and returns the answer to give QBitFlow (see WebhookResult). With
// net/http, mount the router itself (it is an http.Handler) instead.
func (r *WebhookRouter) Handle(ctx context.Context, rawBody []byte, signatureHeader string) WebhookResult {
	res := r.handle(ctx, rawBody, signatureHeader)
	if res.Err != nil && r.onError != nil {
		r.onError(res.Event, res.Err)
	}
	return res
}

func (r *WebhookRouter) handle(ctx context.Context, rawBody []byte, signatureHeader string) WebhookResult {
	if ctx == nil {
		ctx = context.Background()
	}
	secret := ""
	if r.secret != nil {
		secret = *r.secret
	}
	if err := VerifyWebhook(rawBody, signatureHeader, secret, r.verify...); err != nil {
		if errors.As(err, new(*WebhookSignatureError)) {
			return WebhookResult{Status: http.StatusBadRequest, Err: err}
		}
		return WebhookResult{Status: http.StatusInternalServerError, Err: err} // no secret
	}
	event, err := ParseEvent(rawBody)
	if err != nil {
		return WebhookResult{Status: http.StatusBadRequest, Err: err}
	}
	// Data that does not match its (known) type is refused whole, whether or not a typed
	// handler is registered — identically in every SDK.
	if err := validateEventData(event); err != nil {
		return WebhookResult{Status: http.StatusBadRequest, Event: event, Err: err}
	}

	r.mu.RLock()
	route := r.routes[event.Type]
	var typed []func(context.Context, any, *Event) error
	if route != nil {
		typed = slices.Clone(route.handlers)
	}
	var unknown []func(context.Context, *Event) error
	if !slices.Contains(knownEventTypes, event.Type) {
		unknown = slices.Clone(r.onUnknown)
	}
	anyHandlers := slices.Clone(r.onAny)
	r.mu.RUnlock()

	if len(typed) > 0 {
		// Decode before running anything: data that does not match its type is refused whole.
		data, err := route.decode(event)
		if err != nil {
			return WebhookResult{Status: http.StatusBadRequest, Event: event, Err: err}
		}
		for _, h := range typed {
			if err := h(ctx, data, event); err != nil {
				return WebhookResult{Status: http.StatusInternalServerError, Event: event, Err: err}
			}
		}
	}
	for _, h := range slices.Concat(unknown, anyHandlers) {
		if err := h(ctx, event); err != nil {
			return WebhookResult{Status: http.StatusInternalServerError, Event: event, Err: err}
		}
	}
	return WebhookResult{Status: http.StatusOK, Event: event}
}

// ServeHTTP answers a webhook delivery: 405 for a method other than POST, 413 for a body over
// 1 MiB, else Handle's status with its ResponseBody. The handlers get the request's context.
func (r *WebhookRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeWebhookAnswer(w, http.StatusMethodNotAllowed, []byte(`{"error":"method not allowed"}`))
		return
	}
	if req.ContentLength > maxWebhookBody {
		writeWebhookAnswer(w, http.StatusRequestEntityTooLarge, []byte(`{"error":"body too large"}`))
		return
	}
	var body []byte
	if req.Body != nil {
		read, err := io.ReadAll(io.LimitReader(req.Body, maxWebhookBody+1))
		if len(read) > maxWebhookBody {
			writeWebhookAnswer(w, http.StatusRequestEntityTooLarge, []byte(`{"error":"body too large"}`))
			return
		}
		if err != nil {
			writeWebhookAnswer(w, http.StatusBadRequest, []byte(`{"error":"cannot read the body"}`))
			return
		}
		body = read
	}
	res := r.Handle(req.Context(), body, signatureHeaderOf(req.Header))
	writeWebhookAnswer(w, res.Status, res.ResponseBody())
}

// signatureHeaderOf reads QBitFlow-Signature whatever its case (net/http canonicalizes the
// keys it parses; a hand-built header may not be).
func signatureHeaderOf(h http.Header) string {
	if v := h.Get(SignatureHeader); v != "" {
		return v
	}
	for key, values := range h {
		if strings.EqualFold(key, SignatureHeader) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func writeWebhookAnswer(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
