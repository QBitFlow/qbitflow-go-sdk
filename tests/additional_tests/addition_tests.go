package main

import (
	"fmt"
	"os"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

const CUSTOMER_UUID = "your-customer-uuid-here"

func main() {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		panic("QBITFLOW_API_KEY environment variable is not set")
	}

	client := qbitflow.NewWithConfig(qbf.Config{
		APIKey:  key,
		BaseURL: "http://localhost:3001",
	})

	//////////////////// One-time Payment \\\\\\\\\\\\\\\\\\\\

	payment, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductID:    new(uint64(1)),
		CustomerUUID: new(CUSTOMER_UUID),
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

	fmt.Printf("Payment status: %v\n", status)
	if status.Status != qbmodels.TransactionStatusCompleted {
		fmt.Println("Payment not completed. Exiting.")
		return
	}

	paymentDetails, err := client.Payments.GetPayment(payment.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Payment details: %+v\n", paymentDetails)

	// Customer for transaction
	customer, err := client.Payments.GetCustomerForTransaction(payment.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Customer: %s %s (%s)\n", customer.Name, customer.LastName, customer.Email)

	//////////////////// Subscription \\\\\\\\\\\\\\\\\\\\

	sub, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductID:    new(uint64(1)),
		Frequency:    qbmodels.Duration{Unit: qbmodels.DurationUnitMonths, Value: 1},
		CustomerUUID: new(CUSTOMER_UUID),
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

	fmt.Printf("Subscription status: %v\n", subStatus)
	if subStatus.Status != qbmodels.TransactionStatusCompleted {
		fmt.Println("Subscription not completed. Exiting.")
		return
	}

	subscriptionDetails, err := client.Subscriptions.GetSubscription(sub.UUID)
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

	events, err := client.Accounting.Export(qbf.AccountingExportParams{
		From: "2025-01-01",
		To:   "2025-12-31",
	})
	if err != nil {
		fmt.Printf("Accounting export error (may be empty): %v\n", err)
	} else {
		fmt.Printf("Accounting events: %d\n", len(events))
	}

	// NOTE: PAYG session creation is currently disabled.
	// Existing PAYG subscriptions can be managed via client.PayAsYouGo.GetSubscription etc.
}
