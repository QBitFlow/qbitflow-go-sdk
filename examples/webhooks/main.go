// Command webhooks lists the latest payment.completed events and, with CREATE_ENDPOINT=1,
// registers a webhook endpoint and prints its signing secret (shown only once).
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] [CREATE_ENDPOINT=1] go run ./examples/webhooks
//
// Receive the deliveries with examples/webhook_handler (or examples/webhook_verify).
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

	// docs:start events-list
	page, err := client.Webhooks.Events.List(ctx, &qbitflow.EventListParams{
		Type:  qbitflow.EventTypePaymentCompleted,
		Limit: 20,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range page.Items { // newest first
		fmt.Println(e.ID, e.Type, e.CreatedAt.Format(time.RFC3339))
	}
	// docs:end events-list

	if os.Getenv("CREATE_ENDPOINT") != "1" {
		fmt.Println("set CREATE_ENDPOINT=1 to register https://shop.example.com/webhooks/qbitflow")
		return
	}

	// docs:start webhook-endpoint-create
	created, err := client.Webhooks.Endpoints.Create(ctx, &qbitflow.CreateWebhookEndpointParams{
		URL: "https://shop.example.com/webhooks/qbitflow",
		Events: []qbitflow.EventType{ // nil: every type, including the ones added later
			qbitflow.EventTypePaymentCompleted,
			qbitflow.EventTypeCheckoutExpired,
			qbitflow.EventTypeSubscriptionStatusChanged,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	// The whsec_… secret is returned only this once: put it in your secret store now, and give
	// it to your webhook handler as QBITFLOW_WEBHOOK_SECRET.
	fmt.Println("endpoint", created.UUID, "secret", created.Secret)
	// docs:end webhook-endpoint-create
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
