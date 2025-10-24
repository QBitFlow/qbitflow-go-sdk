package main

import (
	"fmt"
	"os"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/utils"
)

const CUSTOMER_UUID = "your-customer-uuid-here"

func main() {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		panic("QBITFLOW_API_KEY environment variable is not set")
	}

	client := qbitflow.NewWithConfig(qbitflow.Config{
		APIKey:  key,
		BaseURL: "http://localhost:3001",
	})

	//////////////////// One-time Payment \\\\\\\\\\\\\\\\\\\\

	payment, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
		ProductID:    utils.Uint64Ptr(1),
		CustomerUUID: utils.StringPtr(CUSTOMER_UUID),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment session link: %s\n", payment.Link)

	// Wait for the user to press Enter before exiting
	fmt.Println("Press Enter after completing the payment...")
	fmt.Scanln()

	// Retrieve the payment status
	status, err := client.TransactionStatus.GetTransactionStatus(payment.UUID, models.TransactionTypeOneTimePayment)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment status: %v\n", status)
	if status.Status != models.TransactionStatusCompleted {
		fmt.Println("Payment not completed. Exiting.")
		return
	}

	// Retrieve the payment details
	paymentDetails, err := client.Payments.GetPayment(payment.UUID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Payment details: %+v\n", paymentDetails)

	//////////////////// Subscription \\\\\\\\\\\\\\\\\\\\

	sub, err := client.Subscriptions.CreateSession(&qbitflow.CreateSubscriptionSessionOptions{
		ProductID: 1,
		Frequency: models.Duration{
			Unit:  models.DurationUnitMonths,
			Value: 1,
		},
		CustomerUUID: utils.StringPtr(CUSTOMER_UUID),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription session link: %s\n", sub.Link)

	// Wait for the user to press Enter before exiting
	fmt.Println("Press Enter after completing the subscription...")
	fmt.Scanln()

	// Retrieve the subscription status
	subStatus, err := client.TransactionStatus.GetTransactionStatus(sub.UUID, models.TransactionTypeCreateSubscription)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription status: %v\n", subStatus)
	if subStatus.Status != models.TransactionStatusCompleted {
		fmt.Println("Subscription not completed. Exiting.")
		return
	}

	// Retrieve the subscription details
	subscriptionDetails, err := client.Subscriptions.GetSubscription(sub.UUID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription details: %+v\n", subscriptionDetails)

	// Wait for the user to run the subscription job and press Enter
	fmt.Println("Run the subscription job to process the subscription payment, then press Enter...")
	fmt.Scanln()

	// Retrieve the payment history for the subscription
	paymentHistory, err := client.Subscriptions.GetPaymentHistory(subscriptionDetails.UUID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Subscription payment history: %+v\n", paymentHistory)

	// Force cancel a subscription
	resp, err := client.Subscriptions.ForceCancel(subscriptionDetails.UUID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Force cancel response: %+v\n", resp)

	// Try and retrieve the subscription again
	subscriptionDetails, err = client.Subscriptions.GetSubscription(sub.UUID)
	if err == nil {
		panic("expected error when retrieving canceled subscription, got none")
	}

	fmt.Printf("Subscription details after force cancel: %+v\n", subscriptionDetails)

	//////////////////// Pay-as-you-go \\\\\\\\\\\\\\\\\\\\

	paygSub, err := client.PayAsYouGo.CreateSession(&qbitflow.CreatePAYGSessionOptions{
		ProductID:    1,
		Frequency:    models.Duration{Unit: models.DurationUnitMonths, Value: 1},
		CustomerUUID: utils.StringPtr(CUSTOMER_UUID),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("PAYG Subscription session link: %s\n", paygSub.Link)

	// Wait for the user to press Enter before exiting
	fmt.Println("Press Enter after completing the PAYG subscription...")
	fmt.Scanln()

	// Retrieve the PAYG subscription status
	paygSubStatus, err := client.TransactionStatus.GetTransactionStatus(paygSub.UUID, models.TransactionTypeCreatePAYGSubscription)
	if err != nil {
		panic(err)
	}

	fmt.Printf("PAYG Subscription status: %v\n", paygSubStatus)
	if paygSubStatus.Status != models.TransactionStatusCompleted {
		fmt.Println("PAYG Subscription not completed. Exiting.")
		return
	}

	// Retrieve the PAYG subscription details
	paygSubscriptionDetails, err := client.PayAsYouGo.GetSubscription(paygSub.UUID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("PAYG Subscription details: %+v\n", paygSubscriptionDetails)

	// Increase the units used
	_, err = client.PayAsYouGo.IncreaseUnitsCurrentPeriod(paygSubscriptionDetails.UUID, 5.0) // Increase by 5 units. Product price is 9.99 per unit
	if err != nil {
		panic(err)
	}
}
