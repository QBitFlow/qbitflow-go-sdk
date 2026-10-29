// Command subscriptions opens a subscription checkout session with a trial, lists active
// subscriptions, reads one and its bills, tags it with metadata, checks access to it, and, on
// request, bills it now (test mode) or cancels it at the end of its paid period.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] [SUBSCRIPTION_UUID=sub@…] [TEST_BILL=1] [CANCEL=1] go run ./examples/subscriptions
//
// Without SUBSCRIPTION_UUID it opens the checkout, then uses the latest active subscription. The
// order reference order-1043 is unique per space: once subscribed, a second checkout answers 409
// unique_violation.
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

	subscriptionUUID := os.Getenv("SUBSCRIPTION_UUID")
	if subscriptionUUID == "" {
		openCheckout(ctx, client)

		page, err := client.Subscriptions.List(ctx, &qbitflow.SubscriptionListParams{
			Status: qbitflow.SubscriptionStatusActive,
			Limit:  1,
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(page.Items) == 0 {
			fmt.Println("no active subscription yet: set SUBSCRIPTION_UUID, or subscribe first")
			return
		}
		subscriptionUUID = page.Items[0].UUID
	}

	// docs:start subscriptions-get
	sub, err := client.Subscriptions.Get(ctx, subscriptionUUID, nil) // sub@…, cancelled ones too
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sub.UUID, sub.Status) // trial, active, pastDue, stopped, cancelled…
	if sub.CurrentPeriodEnd != nil {
		fmt.Println("paid until", sub.CurrentPeriodEnd.Format(time.RFC3339))
	}
	// docs:end subscriptions-get

	// Every bill, newest first: the iterator fetches the pages lazily.
	for bill, err := range client.Subscriptions.IterateBills(ctx, subscriptionUUID, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  bill %s: %s USD, paid until %v\n", bill.UUID, bill.AmountUSD, bill.PeriodEnd)
	}

	// docs:start subscriptions-update
	// Metadata is merged into the subscription's: "" deletes a key, keys not named are kept.
	// It is the only thing a merchant edits on a subscription; webhooks carry it.
	updated, err := client.Subscriptions.Update(ctx, subscriptionUUID, &qbitflow.UpdateSubscriptionParams{
		Metadata: map[string]string{"plan": "pro", "coupon": ""},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Metadata["plan"])
	// docs:end subscriptions-update

	checkAccess(ctx, client, subscriptionUUID)

	if os.Getenv("TEST_BILL") == "1" {
		// docs:start subscriptions-test-bill
		// Test mode only: runs the next billing now, with live's statuses and webhooks.
		bill, err := client.Subscriptions.TestBill(ctx, subscriptionUUID)
		if err != nil {
			log.Fatal(err) // 409 payment_not_due before NextBillingAt; 400 test_mode_only for a live one
		}
		fmt.Println(bill.BillUUID, bill.Stage, bill.Outcome, bill.FailureCode) // sub-hist@…
		// docs:end subscriptions-test-bill
	}

	if os.Getenv("CANCEL") == "1" {
		// docs:start subscriptions-cancel
		// Immediate false: never billed again, cancelled when the paid period ends.
		res, err := client.Subscriptions.Cancel(ctx, subscriptionUUID, &qbitflow.CancelSubscriptionParams{
			Immediate: new(false),
		})
		if err != nil {
			log.Fatal(err) // 409 subscription_already_stopped_or_inactive
		}
		if res.Pending {
			// HTTP 202: the on-chain cancellation is still confirming; subscription.statusChanged
			// tells the end.
			fmt.Println("cancellation confirming")
		} else {
			fmt.Println("now", res.Subscription.Status) // stopped
		}
		// docs:end subscriptions-cancel
	}
}

// openCheckout opens a subscription checkout session.
func openCheckout(ctx context.Context, client *qbitflow.Client) {
	// docs:start checkout-create-subscription
	// The Subscription exists, with the session's id (sub@…), once the customer signed: act on
	// the subscription.created webhook.
	session, err := client.CheckoutSessions.CreateSubscription(ctx, &qbitflow.CreateSubscriptionSessionParams{
		ProductName: "T-shirt",
		Description: "Blue, size M",
		PriceUSD:    "4.99", // USD per period, a decimal string
		Frequency:   &qbitflow.Duration{Value: 1, Unit: qbitflow.DurationUnitMonths},
		TrialPeriod: &qbitflow.Duration{Value: 7, Unit: qbitflow.DurationUnitDays},
		Reference:   "order-1043",
		SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:   "https://shop.example.com/orders/cancel",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Redirect the customer to", session.Link)
	// docs:end checkout-create-subscription
}

// checkAccess tells whether the subscriber has access now.
func checkAccess(ctx context.Context, client *qbitflow.Client, subscriptionUUID string) {
	// docs:start has-access
	// Access lasts while now < AccessUntil, whatever the status: the end of the period paid
	// for, plus an hour for an active subscription while its renewal is billed.
	sub, err := client.Subscriptions.Get(ctx, subscriptionUUID, nil)
	if err != nil {
		log.Fatal(err)
	}
	if sub.HasAccess(time.Now()) { // now before AccessUntil
		fmt.Println("access granted until", sub.AccessUntil.Format(time.RFC3339))
	} else {
		fmt.Println("no access")
	}
	// docs:end has-access
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
