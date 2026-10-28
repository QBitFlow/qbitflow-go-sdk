// Command refunds lists the pending refunds (customers' requests) and reads one.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/refunds
//
// Refunds are read only in the SDK: they are answered, or sent, from the dashboard.
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
		// pending (a customer's request), approved or rejected.
		fmt.Println(refund.UUID, refund.Status, refund.RefundPercent+"%", refund.Amount, refund.Currency.Symbol)
		// docs:end refunds-get
	}
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
