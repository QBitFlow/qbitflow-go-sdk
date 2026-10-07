// Command checkout opens a one-time payment checkout session, polls its status, and expires it.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/checkout
//
// In production, fulfil the order on the payment.completed webhook (see
// examples/webhook_handler); polling GetStatus is for scripts and dashboards.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v3"
)

func main() {
	client := newClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// An inline product (no product needed in the catalog). {{UUID}} is replaced with the
	// session's id on redirect.
	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductName: "T-shirt",
		Description: "Blue, size M",
		Price:       4.99,
		Reference:   fmt.Sprintf("order-%d", time.Now().Unix()),
		SuccessURL:  "https://shop.example.com/thanks?session={{UUID}}",
		CancelURL:   "https://shop.example.com/cart",
	})
	if err != nil {
		log.Fatalf("create the checkout: %v", err)
	}
	fmt.Printf("Send the customer to %s\n(session %s, expires at %s)\n", session.Link, session.UUID, orNone(session.ExpiresAt, time.RFC3339))

	// Poll a few times. "created" with LastAttempt set is a failed attempt: the customer may
	// still pay from the same checkout, so never cancel the order on it.
	for range 3 {
		status, err := client.CheckoutSessions.GetStatus(ctx, session.UUID)
		if err != nil {
			log.Fatalf("status: %v", err)
		}
		fmt.Printf("status: %s\n", status.Status)
		if status.LastAttempt != nil {
			fmt.Printf("  last attempt failed: %s\n", status.LastAttempt.Code)
		}
		if status.Status == qbitflow.CheckoutSessionStatusCompleted {
			payment, err := client.Payments.Get(ctx, session.UUID, nil) // same id as the session
			if err != nil {
				log.Fatalf("payment: %v", err)
			}
			fmt.Printf("paid: %.2f USD (currency %d), tx %s\n", payment.Amount, payment.CurrencyID, payment.TxHash)
			return
		}
		time.Sleep(2 * time.Second)
	}

	// Give up: expire the session (409 tx_already_sent once the customer is paying).
	status, err := client.CheckoutSessions.Expire(ctx, session.UUID)
	if err != nil {
		log.Fatalf("expire: %v", err)
	}
	fmt.Printf("expired: %s\n", status.Status)
}

func newClient() *qbitflow.Client {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		log.Fatal("set QBITFLOW_API_KEY")
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

// orNone formats an optional time: the API omits it when it has no value.
func orNone(t *time.Time, layout string) string {
	if t == nil {
		return "—"
	}
	return t.Format(layout)
}
