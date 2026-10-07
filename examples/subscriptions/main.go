// Command subscriptions opens a subscription checkout session, lists subscriptions with
// filters, walks a subscription's bills, and cancels one at the end of its paid period.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/subscriptions
package main

import (
	"context"
	"errors"
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

	// 1. A monthly plan with a 14-day trial, as an inline product. The Subscription exists, with
	//    the session's id, once the customer signed (a trial) or paid: act on the
	//    subscription.created webhook.
	session, err := client.CheckoutSessions.CreateSubscription(ctx, &qbitflow.CreateSubscriptionSessionParams{
		ProductName:       "Pro plan",
		Price:             4.99,
		Frequency:         &qbitflow.Duration{Value: 1, Unit: qbitflow.DurationUnitMonths},
		TrialPeriod:       &qbitflow.Duration{Value: 14, Unit: qbitflow.DurationUnitDays},
		CustomerReference: "crm-42",
		SuccessURL:        "https://app.example.com/billing?subscription={{UUID}}",
	})
	if err != nil {
		log.Fatalf("create the checkout: %v", err)
	}
	fmt.Printf("Send the customer to %s (subscription %s once signed)\n", session.Link, session.UUID)

	// 2. The active subscriptions created in the last 90 days. Grant access while
	//    now < CurrentPeriodEnd, whatever the status.
	since := time.Now().AddDate(0, 0, -90)
	page, err := client.Subscriptions.List(ctx, &qbitflow.SubscriptionListParams{
		Status:       qbitflow.SubscriptionStatusActive,
		CreatedAfter: &since,
		Limit:        10,
	})
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	for _, sub := range page.Items {
		access := sub.CurrentPeriodEnd != nil && time.Now().Before(*sub.CurrentPeriodEnd)
		fmt.Printf("%s %s (customer ref %q) access=%v\n", sub.UUID, sub.Status, sub.CustomerReference, access)
	}
	if len(page.Items) == 0 {
		fmt.Println("no active subscription to show bills for")
		return
	}
	sub := page.Items[0]

	// 3. Every bill of the first one, newest first (the iterator fetches the pages lazily).
	for bill, err := range client.Subscriptions.IterateBills(ctx, sub.UUID, &qbitflow.BillListParams{Limit: 50}) {
		if err != nil {
			log.Fatalf("bills: %v", err)
		}
		fmt.Printf("  bill %s: %.2f USD for %s → %s\n", bill.UUID, bill.Amount,
			orNone(bill.PeriodStart, time.DateOnly), orNone(bill.PeriodEnd, time.DateOnly))
	}

	// 4. Stop it at the end of the paid period (Immediate false): never billed again, cancelled
	//    when the period ends. A 202 means the on-chain cancel is still confirming:
	//    subscription.statusChanged tells the end.
	if os.Getenv("CANCEL") != "1" {
		fmt.Println("set CANCEL=1 to stop", sub.UUID, "at the end of its period")
		return
	}
	immediate := false
	res, err := client.Subscriptions.Cancel(ctx, sub.UUID, &qbitflow.CancelSubscriptionParams{Immediate: &immediate})
	var conflict *qbitflow.ConflictError
	switch {
	case errors.As(err, &conflict):
		fmt.Printf("not stopped: %s (%s)\n", conflict.Message, conflict.Code) // e.g. already stopped
	case err != nil:
		log.Fatalf("cancel: %v", err)
	case res.Pending:
		fmt.Println("stopping: still confirming on-chain")
	default:
		fmt.Printf("now %s\n", res.Subscription.Status)
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

// orNone formats an optional time: the API omits it when it has no value.
func orNone(t *time.Time, layout string) string {
	if t == nil {
		return "—"
	}
	return t.Format(layout)
}
