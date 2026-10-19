// Command refunds lists the pending refunds, reads one, and, given a payment, starts a partial
// refund of it.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] [PAYMENT_UUID=pay@…] go run ./examples/refunds
//
// The refund stays pending, and no money moves, until it is signed in the dashboard.
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// docs:start refunds-list
	// The active ones: pending, waiting for an answer (Status empty: every status).
	page, err := client.Refunds.List(ctx, &qbitflow.RefundListParams{
		Status: []qbitflow.RefundStatus{qbitflow.RefundStatusPending},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range page.Items {
		fmt.Println(r.UUID, r.TxUUID, r.InitiatedBy, r.Reason, r.Amount, r.Currency.Symbol)
	}
	// docs:end refunds-list

	if len(page.Items) > 0 {
		refundUUID := page.Items[0].UUID

		// docs:start refunds-get
		refund, err := client.Refunds.Get(ctx, refundUUID) // refund@…
		if err != nil {
			log.Fatal(err)
		}
		// pending, approved, rejected or cancelled (a merchant's draft dropped before it was sent).
		fmt.Println(refund.UUID, refund.Status, refund.RefundPercent+"%", refund.Amount, refund.Currency.Symbol)
		// docs:end refunds-get
	}

	paymentUUID := os.Getenv("PAYMENT_UUID")
	if paymentUUID == "" {
		fmt.Println("set PAYMENT_UUID=pay@… to refund half of a payment")
		return
	}

	// docs:start refunds-create
	refund, err := client.Refunds.Create(ctx, &qbitflow.CreateRefundParams{
		TxUUID:        paymentUUID, // pay@… (or a bill's sub-hist@…)
		RefundPercent: "50",        // a percent string, of what the customer paid; "" = "100"
		Reason:        "Damaged item",
	})
	if err != nil {
		log.Fatal(err) // 409 refund_already_exists: one refund per transaction
	}
	// Pending: no money moves until you sign the transfer in the dashboard (refund.approved).
	fmt.Println(refund.UUID, refund.Status)
	// docs:end refunds-create
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
