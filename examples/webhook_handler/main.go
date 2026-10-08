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
	secret := os.Getenv("QBITFLOW_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("set QBITFLOW_WEBHOOK_SECRET")
	}
	app := &app{client: newClient(), seen: map[string]bool{}}

	router := qbitflow.NewWebhookRouter(secret,
		qbitflow.WithWebhookErrorHandler(func(e *qbitflow.Event, err error) {
			if e == nil {
				log.Printf("refused delivery: %v", err) // bad signature, or not a v2 event
				return
			}
			log.Printf("event %s (%s) failed, QBitFlow will retry it: %v", e.ID, e.Type, err)
		}),
	).
		OnPaymentCompleted(app.paymentCompleted).
		OnSubscriptionCreated(func(_ context.Context, sub *qbitflow.SubscriptionCreated, _ *qbitflow.Event) error {
			log.Printf("subscription %s created (%s), access: %v", sub.UUID, sub.Status, sub.HasAccess(time.Now()))
			return nil
		}).
		OnSubscriptionBilled(func(_ context.Context, bill *qbitflow.SubscriptionBilled, _ *qbitflow.Event) error {
			log.Printf("subscription %s billed: %.2f USD (bill %s)", bill.SubscriptionUUID, bill.Amount, bill.UUID)
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
		OnRefundCompleted(func(_ context.Context, refund *qbitflow.Refund, _ *qbitflow.Event) error {
			log.Printf("refund %s of %s sent", refund.UUID, refund.TxUUID)
			return nil
		}).
		OnMemberJoined(func(_ context.Context, joined *qbitflow.MemberJoined, _ *qbitflow.Event) error {
			log.Printf("member %s joined (invitation %s): sell with OnBehalfOf(%q)", joined.Email, joined.InvitationUUID, joined.UserUUID)
			return nil
		}).
		OnCheckoutExpired(func(_ context.Context, expired *qbitflow.CheckoutExpired, _ *qbitflow.Event) error {
			if expired.Payment != nil {
				log.Printf("payment checkout %s expired: release order %q", expired.Payment.UUID, expired.Payment.Reference)
			} else if expired.Subscription != nil {
				log.Printf("subscription checkout %s expired", expired.Subscription.UUID)
			}
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

	// The router is an http.Handler: 405 for other methods, 413 over 1 MiB.
	http.Handle("/webhooks/qbitflow", router)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type app struct {
	client *qbitflow.Client

	// seen deduplicates on the event's id: deliveries are at least once (a retry carries the
	// same id). Use your database in production.
	mu   sync.Mutex
	seen map[string]bool
}

// paymentCompleted fulfils an order once, whatever the number of deliveries.
func (a *app) paymentCompleted(ctx context.Context, payment *qbitflow.PaymentCompleted, e *qbitflow.Event) error {
	a.mu.Lock()
	duplicate := a.seen[e.ID]
	a.seen[e.ID] = true
	a.mu.Unlock()
	if duplicate {
		return nil
	}

	amount := payment.AmountMinUnits
	if formatted, err := payment.Currency.FormatAmount(payment.AmountMinUnits); err == nil {
		amount = formatted + " " + payment.Currency.Symbol
	}
	log.Printf("payment %s completed: order %q, %s (%.2f USD)", payment.UUID, payment.Reference, amount, payment.Amount)

	// Fulfil the order here. A member's payment names them in e.UserUUID: read more in their
	// space with OnBehalfOf.
	if e.UserUUID != "" && a.client != nil {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		full, err := a.client.OnBehalfOf(e.UserUUID).Payments.Get(ctx, payment.UUID, nil)
		if err != nil {
			// Returning the error answers 500: QBitFlow delivers the event again later.
			a.mu.Lock()
			delete(a.seen, e.ID)
			a.mu.Unlock()
			return err
		}
		log.Printf("  paid by %s", full.From)
	}
	return nil
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
