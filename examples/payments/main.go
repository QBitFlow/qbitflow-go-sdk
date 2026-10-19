// Command payments lists a page of last month's payments, walks every payment with the
// iterator, lists the payments-and-bills feed, converts an amount exactly, and reads one payment by its id and by its order
// reference.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] [PAYMENT_UUID=pay@…] go run ./examples/payments
//
// Without PAYMENT_UUID it reads the latest payment. The last step needs a payment for the
// order order-1042 (run examples/checkout and pay it with a test wallet).
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

	// docs:start payments-list
	since := time.Now().AddDate(0, -1, 0)
	page, err := client.Payments.List(ctx, &qbitflow.PaymentListParams{
		CreatedAfter: &since, // the last month
		Limit:        20,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range page.Items {
		// Amounts are decimal strings: the token's (Amount, with Currency) and USD (AmountUSD).
		fmt.Printf("%s %q: %s %s (%s USD)\n", p.UUID, p.Reference, p.Amount, p.Currency.Symbol, p.AmountUSD)
	}
	if page.HasMore {
		// Pass it back, verbatim, as PaymentListParams.Cursor for the next page.
		fmt.Println("next cursor:", page.NextCursor)
	}
	// docs:end payments-list

	// docs:start pagination-iterate
	// Fetches one page at a time, only as the loop consumes it; break stops the fetching.
	// It stops after the page whose HasMore is false.
	count := 0
	for payment, err := range client.Payments.Iterate(ctx, &qbitflow.PaymentListParams{Limit: 50}) {
		if err != nil {
			log.Fatal(err)
		}
		count++
		fmt.Println(payment.UUID, payment.AmountUSD, "USD")
	}
	fmt.Println(count, "payments")
	// docs:end pagination-iterate

	// docs:start transactions-list
	// One feed of one-time payments and subscription bills, newest first; each row reads like
	// its resource. Type filters it to one kind (TransactionKindPayment or TransactionKindBill).
	feed, err := client.Transactions.List(ctx, &qbitflow.TransactionListParams{Limit: 20})
	if err != nil {
		log.Fatal(err)
	}
	for _, tx := range feed.Items {
		fmt.Println(tx.Type, tx.UUID, tx.AmountUSD, "USD")
	}
	// docs:end transactions-list

	amounts()

	paymentUUID := os.Getenv("PAYMENT_UUID")
	if paymentUUID == "" {
		if len(page.Items) == 0 {
			fmt.Println("no payment yet: set PAYMENT_UUID, or pay a checkout first")
			return
		}
		paymentUUID = page.Items[0].UUID
	}

	// docs:start payments-get
	payment, err := client.Payments.Get(ctx, paymentUUID, nil) // pay@…
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(payment.UUID, payment.Reference, payment.AmountUSD, "USD", payment.Metadata)

	// Runs GET /payments?reference= and returns its one payment.
	byReference, err := client.Payments.GetByReference(ctx, "order-1042") // your order id
	if err != nil {
		log.Fatal(err) // a *qbitflow.NotFoundError (404, not_found) when no payment has this reference
	}
	fmt.Println("order-1042 was paid by", byReference.UUID)
	// docs:end payments-get
}

// amounts converts between token amounts in the smallest unit and display strings.
func amounts() {
	// docs:start amounts-display
	// Exact decimal arithmetic on strings, never floats. USDC has 6 decimals.
	display, err := qbitflow.FormatAmount("4990000", 6)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(display, "USDC") // 4.99 USDC

	minUnits, err := qbitflow.ParseAmount("4.99", 6) // more than 6 decimals is a *ValidationError
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(minUnits) // 4990000

	// From a payment: payment.Currency.FormatAmount(payment.AmountMinUnits)
	// docs:end amounts-display
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
