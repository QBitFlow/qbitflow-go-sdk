// Command additional_tests is a manual, interactive end-to-end walkthrough against a real
// server: it creates sessions, waits for you to pay them in the browser, then reads the
// results back.
//
//	QBITFLOW_API_KEY        required
//	QBITFLOW_BASE_URL       required (never defaulted — point it at your test server)
//	QBITFLOW_CUSTOMER_UUID  optional: a customer UUID to pre-associate with the sessions
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
	key := os.Getenv("QBITFLOW_API_KEY")
	baseURL := os.Getenv("QBITFLOW_BASE_URL")
	if key == "" || baseURL == "" {
		panic("QBITFLOW_API_KEY and QBITFLOW_BASE_URL must both be set")
	}

	client := qbitflow.NewWithConfig(qbf.Config{
		APIKey:  key,
		BaseURL: baseURL,
	})
	customerUUID := os.Getenv("QBITFLOW_CUSTOMER_UUID") // "" is left out of the request
	runID := time.Now().Unix()

	//////////////////// One-time Payment \\\\\\\\\\\\\\\\\\\\

	orderRef := fmt.Sprintf("manual-order-%d", runID)
	payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductName:  new("Manual test item"),
		Description:  new("Created by tests/additional_tests"),
		Price:        new(1.5),
		CustomerUUID: new(customerUUID),
		Reference:    new(orderRef),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment session link: %s\n", payment.Link)
	fmt.Println("Press Enter after completing the payment...")
	fmt.Scanln()

	status, err := client.TransactionStatus.GetTransactionStatus(payment.UUID, qbmodels.TransactionTypeOneTimePayment)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment status: %+v\n", status)
	if status.Status != qbmodels.TransactionStatusCompleted {
		fmt.Println("Payment not completed. Exiting.")
		return
	}

	paymentDetails, err := client.Payments.GetPayment(payment.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Payment details: %+v\n", paymentDetails)

	if paymentDetails.CustomerUUID != nil {
		customer, err := client.Payments.GetCustomerForTransaction(payment.UUID)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Customer: %s %s (%s)\n", customer.Name, customer.LastName, customer.Email)
	}

	//////////////////// Subscription \\\\\\\\\\\\\\\\\\\\

	subRef := fmt.Sprintf("manual-sub-%d", runID)
	sub, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductName:  new("Manual test plan"),
		Description:  new("Created by tests/additional_tests"),
		Price:        new(2.5),
		Frequency:    qbmodels.Duration{Unit: qbmodels.DurationUnitMonths, Value: 1},
		CustomerUUID: new(customerUUID),
		Reference:    new(subRef),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription session link: %s\n", sub.Link)
	fmt.Println("Press Enter after completing the subscription...")
	fmt.Scanln()

	subStatus, err := client.TransactionStatus.GetTransactionStatus(sub.UUID, qbmodels.TransactionTypeCreateSubscription)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription status: %+v\n", subStatus)
	if subStatus.Status != qbmodels.TransactionStatusCompleted {
		fmt.Println("Subscription not completed. Exiting.")
		return
	}

	// The subscription has its own sub@ UUID, distinct from the session UUID: resolve it
	// through the reference set above.
	subscriptionDetails, err := client.Subscriptions.GetSubscriptionByReference(subRef)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Subscription details: %+v\n", subscriptionDetails)

	fmt.Println("Run the subscription job, then press Enter...")
	fmt.Scanln()

	paymentHistory, err := client.Subscriptions.GetPaymentHistory(subscriptionDetails.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Subscription payment history: %+v\n", paymentHistory)

	resp, err := client.Subscriptions.ForceCancel(subscriptionDetails.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Force cancel response: %+v\n", resp)

	//////////////////// Refunds \\\\\\\\\\\\\\\\\\\\

	refunds, err := client.Refunds.GetAll()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Active refunds: %d\n", len(refunds))

	//////////////////// Accounting \\\\\\\\\\\\\\\\\\\\

	to := time.Now().UTC()
	from := to.AddDate(0, -1, 0)
	events, err := client.Accounting.Export(qbf.AccountingExportParams{
		From: from.Format("2006-01-02"),
		To:   to.Format("2006-01-02"),
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Accounting events (last month): %d\n", len(events))
}
