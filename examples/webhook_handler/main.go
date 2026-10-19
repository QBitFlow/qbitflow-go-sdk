// Command webhook_handler receives QBitFlow webhooks with a WebhookRouter: it verifies each
// delivery's signature, parses the event, runs the handler of its type with the typed data, and
// answers QBitFlow (200, 400 for a bad delivery, 500 when a handler fails so it is retried).
//
//	QBITFLOW_WEBHOOK_SECRET=whsec_… [QBITFLOW_API_KEY=sk_… QBITFLOW_BASE_URL=…] go run ./examples/webhook_handler
//
// The secret is the one Webhooks.Endpoints.Create returned (shown once). The API key is only
// used for follow-up reads.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v3"
)

func main() {
	if os.Getenv("QBITFLOW_WEBHOOK_SECRET") == "" {
		log.Fatal("set QBITFLOW_WEBHOOK_SECRET")
	}

	// docs:start webhook-handler
	var fulfilled sync.Map // stands for your database: deliveries are at least once
	router := qbitflow.NewWebhookRouter(os.Getenv("QBITFLOW_WEBHOOK_SECRET"),
		qbitflow.WithWebhookErrorHandler(func(event *qbitflow.Event, err error) {
			log.Printf("webhook refused or failed: %v", err) // event is nil when the body could not be parsed
		}),
	).
		OnPaymentCompleted(func(ctx context.Context, payment *qbitflow.PaymentCompleted, event *qbitflow.Event) error {
			if _, done := fulfilled.LoadOrStore(event.ID, true); done {
				return nil // a retry of an event already handled
			}
			log.Printf("fulfil order %q: %s USD (%s, %v)", payment.Reference, payment.AmountUSD, payment.UUID, payment.Metadata)
			return nil // an error answers 500, and QBitFlow delivers the event again
		}).
		OnCheckoutExpired(func(ctx context.Context, expired *qbitflow.CheckoutExpired, event *qbitflow.Event) error {
			if expired.Payment != nil {
				log.Printf("release order %q", expired.Payment.Reference)
			}
			return nil
		})

	// The router is an http.Handler: it verifies the QBitFlow-Signature header on the raw body,
	// parses the event, runs the handlers of its type and answers 200, 400 or 500.
	http.Handle("POST /webhooks/qbitflow", router)
	// docs:end webhook-handler

	// More handlers, registered before serving.
	router.
		OnSubscriptionCreated(func(_ context.Context, sub *qbitflow.Subscription, _ *qbitflow.Event) error {
			log.Printf("subscription %s created (%s), access: %v", sub.UUID, sub.Status, sub.HasAccess(time.Now()))
			return nil
		}).
		OnSubscriptionBilled(func(_ context.Context, billed *qbitflow.SubscriptionBilled, _ *qbitflow.Event) error {
			// {bill, subscription}: the bill paid, and the subscription as it leaves it.
			log.Printf("subscription %s (%s) billed: %s USD (bill %s), paid until %v",
				billed.Subscription.UUID, billed.Subscription.Status, billed.Bill.AmountUSD, billed.Bill.UUID, billed.Bill.PeriodEnd)
			return nil
		}).
		OnSubscriptionStatusChanged(func(_ context.Context, change *qbitflow.SubscriptionStatusChanged, _ *qbitflow.Event) error {
			log.Printf("subscription %s: %s → %s, access: %v", change.UUID, change.PreviousStatus, change.Status, change.HasAccess(time.Now()))
			return nil
		}).
		OnSubscriptionBillingFailed(func(_ context.Context, failed *qbitflow.SubscriptionBillingFailed, _ *qbitflow.Event) error {
			log.Printf("subscription %s: billing failed (%s), %d attempts left", failed.UUID, failed.Reason, failed.RemainingAttempts)
			return nil
		}).
		OnRefundApproved(func(_ context.Context, refund *qbitflow.Refund, _ *qbitflow.Event) error {
			log.Printf("refund %s of %s sent: %s%% (%s %s)", refund.UUID, refund.TxUUID, refund.RefundPercent, refund.Amount, refund.Currency.Symbol)
			return nil
		}).
		OnRefundRejected(func(_ context.Context, refund *qbitflow.Refund, _ *qbitflow.Event) error {
			log.Printf("refund %s of %s rejected: %s", refund.UUID, refund.TxUUID, refund.MerchantMessage)
			return nil
		}).
		OnMemberJoined(func(_ context.Context, joined *qbitflow.MemberJoined, _ *qbitflow.Event) error {
			log.Printf("member %s joined (invitation %s): sell with OnBehalfOf(%q)", joined.Email, joined.InvitationUUID, joined.UserUUID)
			return nil
		}).
		OnWebhookTest(func(_ context.Context, test *qbitflow.WebhookTest, _ *qbitflow.Event) error {
			log.Printf("test event for endpoint %s: %s", test.EndpointUUID, test.Message)
			return nil
		}).
		OnUnknown(func(_ context.Context, e *qbitflow.Event) error {
			log.Printf("ignored %s, a type newer than this SDK", e.Type) // acknowledged with a 200
			return nil
		})

	// A member's payment names them in e.UserUUID: read more in their space with OnBehalfOf.
	// OnAny handlers run after the typed ones, on retries too.
	if client := newClient(); client != nil {
		router.OnAny(func(ctx context.Context, e *qbitflow.Event) error {
			if e.Type != qbitflow.EventTypePaymentCompleted || e.UserUUID == "" {
				return nil
			}
			payment, err := e.PaymentCompleted()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			full, err := client.OnBehalfOf(e.UserUUID).Payments.Get(ctx, payment.UUID, nil)
			if err != nil {
				return err // answers 500: QBitFlow delivers the event again later
			}
			log.Printf("  paid by %s", full.From)
			return nil
		})
	}

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// newClient builds the client for follow-up reads from QBITFLOW_API_KEY (and QBITFLOW_BASE_URL);
// nil without a key: verification needs only the secret.
func newClient() *qbitflow.Client {
	if os.Getenv("QBITFLOW_API_KEY") == "" {
		return nil
	}
	client, err := qbitflow.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return client
}
