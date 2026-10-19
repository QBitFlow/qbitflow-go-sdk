// Command checkout opens a one-time payment checkout session, reads its status, waits for the
// customer for a while, and expires it if they did not pay. It first opens (and expires) a
// checkout that adds fees to the price.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/checkout
//
// In production, fulfil the order on the payment.completed webhook (see
// examples/webhook_handler); polling the status is for scripts and back-office jobs. The order
// reference order-1042 is unique per space: once paid, a second run answers 409 unique_violation.
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	checkoutWithFees(ctx, client)

	// docs:start checkout-create-payment
	// An inline product: no product needed in the catalog. QBitFlow replaces {{UUID}}
	// (qbitflow.PlaceholderUUID) with the session's id on redirect.
	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductName: "T-shirt",
		Description: "Blue, size M",
		PriceUSD:    "4.99",       // a decimal string, never a float
		Reference:   "order-1042", // your order id: unique per space
		SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:   "https://shop.example.com/orders/cancel",
	})
	if err != nil {
		log.Fatal(err)
	}
	// In an HTTP handler: http.Redirect(w, r, session.Link, http.StatusSeeOther)
	fmt.Println("Redirect the customer to", session.Link)
	// docs:end checkout-create-payment

	sessionUUID := session.UUID // pay@…

	// docs:start checkout-status
	status, err := client.CheckoutSessions.Get(ctx, sessionUUID)
	if err != nil {
		log.Fatal(err)
	}
	switch status.Status {
	case qbitflow.CheckoutSessionStatusCompleted:
		fmt.Println("paid, tx", status.TxHash)
	case qbitflow.CheckoutSessionStatusExpired:
		fmt.Println("expired unpaid:", status.Message)
	case qbitflow.CheckoutSessionStatusWaitingConfirmation:
		fmt.Println("sent, waiting for the network")
	default: // created, or a status this SDK does not know
		if status.LastAttempt != nil {
			// Not final: the customer may still pay from the same checkout.
			fmt.Println("last attempt failed:", status.LastAttempt.Code)
		}
	}
	// docs:end checkout-status

	if final(status) {
		return
	}
	waitThenExpire(ctx, client, sessionUUID)
}

// checkoutWithFees opens a checkout that adds a shipping line and the processing fee to the
// price, then expires it: a demo nobody pays.
func checkoutWithFees(ctx context.Context, client *qbitflow.Client) {
	// docs:start checkout-create-payment-fees
	// The customer pays the price plus every line: $3.99 + $0.75 + the processing fee. You
	// keep the price and your lines. Test mode caps the total, fees included, at $5.
	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductName: "T-shirt",
		Description: "Blue, size M",
		PriceUSD:    "3.99",
		Reference:   "order-1044",
		SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:   "https://shop.example.com/orders/cancel",
		Fees: &qbitflow.CheckoutFees{
			Items: []qbitflow.FeeItem{
				{Label: "Shipping", Description: "Standard, 3 to 5 days", AmountUSD: "0.75"},
			},
			// QBitFlow's processing fee, grossed up so you keep the price and your lines.
			// Left nil, your checkout.customerPaysProcessingFee setting decides.
			ProcessingFee: new(true),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Redirect the customer to", session.Link)
	// docs:end checkout-create-payment-fees

	// A demo: nobody pays it.
	if _, err := client.CheckoutSessions.Expire(ctx, session.UUID); err != nil {
		log.Fatal(err)
	}
}

// waitThenExpire waits a little for the customer, and expires the session if they did not pay.
func waitThenExpire(ctx context.Context, client *qbitflow.Client, sessionUUID string) {
	// docs:start wait-for-completion
	// For scripts, tests and back-office jobs: orders are fulfilled on the webhook.
	status, err := client.CheckoutSessions.WaitForCompletion(ctx, sessionUUID, &qbitflow.WaitOptions{
		Timeout:  2 * time.Minute, // default 10 minutes
		Interval: 5 * time.Second, // default 3 s
	})
	if err != nil {
		log.Fatal(err) // Get's errors, or the context's
	}
	if status.Status != qbitflow.CheckoutSessionStatusCompleted {
		fmt.Println("not paid (yet):", status.Status) // expired, or still open at the timeout
	}
	// docs:end wait-for-completion

	if final(status) {
		return
	}

	// docs:start checkout-expire
	// Once the customer paid or is paying, this is a 409 tx_already_sent.
	expired, err := client.CheckoutSessions.Expire(ctx, sessionUUID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(expired.Status) // expired; checkout.expired follows
	// docs:end checkout-expire
}

// final reports whether a session reached a final status.
func final(status *qbitflow.CheckoutSessionStatus) bool {
	return status.Status == qbitflow.CheckoutSessionStatusCompleted || status.Status == qbitflow.CheckoutSessionStatusExpired
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
