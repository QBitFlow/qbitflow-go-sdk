// Command refunds lists the active refunds and, given a payment, starts a partial refund of it.
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
	refunds, err := client.Refunds.List(ctx, nil) // the refunds waiting for an answer
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range refunds {
		fmt.Println(r.UUID, r.TxUUID, r.Status, r.Reason, r.AmountUSD)
	}
	// docs:end refunds-list

	paymentUUID := os.Getenv("PAYMENT_UUID")
	if paymentUUID == "" {
		fmt.Println("set PAYMENT_UUID=pay@… to refund half of a payment")
		return
	}

	// docs:start refunds-create
	refund, err := client.Refunds.Initiate(ctx, &qbitflow.InitiateRefundParams{
		TxUUID:        paymentUUID, // pay@… (or a bill's sub-hist@…)
		RefundPercent: new(50.0),   // of what the customer paid; nil = 100
		Reason:        "Damaged item",
	})
	if err != nil {
		log.Fatal(err) // 409 refund_already_exists: one refund per transaction
	}
	// Pending: no money moves until you sign the transfer in the dashboard (refund.completed).
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
