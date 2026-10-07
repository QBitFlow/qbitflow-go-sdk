// Command webhook_handler receives QBitFlow webhooks: it verifies each delivery's signature,
// parses the event, and acts on the types it cares about.
//
//	QBITFLOW_WEBHOOK_SECRET=whsec_… QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/webhook_handler
//
// The secret is the one Webhooks.Endpoints.Create returned (shown once). The API key is only
// used for follow-up reads.
package main

import (
	"context"
	"errors"
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
	h := &handler{secret: secret, client: newClient(), seen: map[string]bool{}}
	http.Handle("POST /webhooks/qbitflow", h)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type handler struct {
	secret string
	client *qbitflow.Client

	// seen deduplicates on the event's id: deliveries are at least once (a retry carries the
	// same id). Use your database in production.
	mu   sync.Mutex
	seen map[string]bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verify the QBitFlow-Signature header over the raw body (at most 1 MiB), then parse.
	body, err := qbitflow.VerifyWebhookRequest(r, h.secret)
	if err != nil {
		var sigErr *qbitflow.WebhookSignatureError
		if errors.As(err, &sigErr) {
			log.Printf("rejected delivery: %s", sigErr.Reason)
		}
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	event, err := qbitflow.ParseEvent(body)
	if err != nil {
		// Not a v2 event (an endpoint still on v1: move it to v2 in the dashboard).
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	duplicate := h.seen[event.ID]
	h.seen[event.ID] = true
	h.mu.Unlock()
	if duplicate {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.handle(r.Context(), event); err != nil {
		// A non-2xx makes QBitFlow deliver it again later.
		log.Printf("event %s (%s): %v", event.ID, event.Type, err)
		h.mu.Lock()
		delete(h.seen, event.ID)
		h.mu.Unlock()
		http.Error(w, "try again", http.StatusInternalServerError)
		return
	}
	// Answer 2xx fast, to the events you ignore too.
	w.WriteHeader(http.StatusOK)
}

func (h *handler) handle(ctx context.Context, event *qbitflow.Event) error {
	switch event.Type {
	case qbitflow.EventTypePaymentCompleted:
		payment, err := event.PaymentCompleted()
		if err != nil {
			return err
		}
		log.Printf("payment %s completed: order %q, %.2f USD", payment.UUID, payment.Reference, payment.Amount)
		// Fulfil the order here. A member's payment names them in event.UserUUID: read more
		// in their space with OnBehalfOf.
		if event.UserUUID != "" && h.client != nil {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if full, err := h.client.OnBehalfOf(event.UserUUID).Payments.Get(ctx, payment.UUID, nil); err == nil {
				log.Printf("  paid by %s", full.From)
			}
		}

	case qbitflow.EventTypeSubscriptionCreated:
		sub, err := event.SubscriptionCreated()
		if err != nil {
			return err
		}
		log.Printf("subscription %s created (%s), access until %v", sub.UUID, sub.Status, sub.CurrentPeriodEnd)

	case qbitflow.EventTypeSubscriptionBilled:
		bill, err := event.SubscriptionBilled()
		if err != nil {
			return err
		}
		log.Printf("subscription %s billed: %.2f USD (bill %s)", bill.SubscriptionUUID, bill.Amount, bill.UUID)

	case qbitflow.EventTypeSubscriptionStatusChanged:
		change, err := event.SubscriptionStatusChanged()
		if err != nil {
			return err
		}
		log.Printf("subscription %s: %s → %s", change.UUID, change.PreviousStatus, change.Status)

	case qbitflow.EventTypeSubscriptionBillingFailed:
		failed, err := event.SubscriptionBillingFailed()
		if err != nil {
			return err
		}
		log.Printf("subscription %s: billing failed (%s), %d attempts left", failed.UUID, failed.Reason, failed.RemainingAttempts)

	case qbitflow.EventTypeRefundCompleted:
		refund, err := event.RefundCompleted()
		if err != nil {
			return err
		}
		log.Printf("refund %s of %s sent", refund.UUID, refund.TxUUID)

	case qbitflow.EventTypeMemberJoined:
		joined, err := event.MemberJoined()
		if err != nil {
			return err
		}
		log.Printf("member %s joined (invitation %s): sell with OnBehalfOf(%q)", joined.Email, joined.InvitationUUID, joined.UserUUID)

	case qbitflow.EventTypeCheckoutExpired:
		expired, err := event.CheckoutExpired()
		if err != nil {
			return err
		}
		if expired.Payment != nil {
			log.Printf("payment checkout %s expired: release the order", expired.Payment.UUID)
		} else if expired.Subscription != nil {
			log.Printf("subscription checkout %s expired", expired.Subscription.UUID)
		}

	case qbitflow.EventTypeWebhookTest:
		test, err := event.WebhookTest()
		if err != nil {
			return err
		}
		log.Printf("test event for endpoint %s: %s", test.EndpointUUID, test.Message)

	default:
		// A type this handler ignores, or a newer one: acknowledge it.
		log.Printf("ignored %s", event.Type)
	}
	return nil
}

func newClient() *qbitflow.Client {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		return nil // verification needs only the secret
	}
	var opts []qbitflow.ClientOption
	if base := os.Getenv("QBITFLOW_BASE_URL"); base != "" {
		opts = append(opts, qbitflow.WithBaseURL(base))
	}
	client, err := qbitflow.NewClient(key, opts...)
	if err != nil {
		log.Fatal(err)
	}
	return client
}
