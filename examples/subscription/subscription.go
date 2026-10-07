// Command subscription walks through recurring subscriptions: creating subscription
// checkout sessions, reading a session back, and managing an existing subscription.
//
// Configuration comes from the environment:
//
//	QBITFLOW_API_KEY            required
//	QBITFLOW_BASE_URL           optional (defaults to the production API)
//	QBITFLOW_CUSTOMER_UUID      optional: a customer UUID to pre-associate with the session
//	QBITFLOW_SUBSCRIPTION_UUID  optional: an existing subscription (sub@…) to manage
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
	apiKey := os.Getenv("QBITFLOW_API_KEY")
	if apiKey == "" {
		log.Fatal("set QBITFLOW_API_KEY to run this example")
	}
	client := qbitflow.NewWithConfig(qbf.Config{
		APIKey:  apiKey,
		BaseURL: os.Getenv("QBITFLOW_BASE_URL"), // "" = production
	})
	customerUUID := os.Getenv("QBITFLOW_CUSTOMER_UUID") // "" is left out of the request

	fmt.Println("=== Subscription Management Example ===")
	fmt.Println()

	// Example 1: Create a subscription with a trial period
	fmt.Println("1. Creating subscription with trial period...")
	session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductName:  new("Pro Plan"),
		Description:  new("Monthly access to every Pro feature"),
		Price:        new(9.99),
		Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
		TrialPeriod:  &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays},
		SuccessURL:   new("https://yoursite.com/subscription/success"),
		CancelURL:    new("https://yoursite.com/subscription/cancel"),
		CustomerUUID: new(customerUUID),
		Reference:    new("plan-pro-customer-42"), // your own reference, usable with GetSubscriptionByReference
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session.UUID)
	fmt.Printf("Subscription Link: %s\n", session.Link)
	fmt.Println()

	// Example 2: Create a subscription without trial, with a minimum commitment
	fmt.Println("2. Creating subscription without trial...")
	session2, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductName:  new("Team Plan"),
		Description:  new("Quarterly billing for the whole team"),
		Price:        new(49.0),
		Frequency:    qbmodels.Duration{Value: 3, Unit: qbmodels.DurationUnitMonths},
		MinPeriods:   new(uint32(2)),
		SuccessURL:   new("https://yoursite.com/subscription/success"),
		CancelURL:    new("https://yoursite.com/subscription/cancel"),
		CustomerUUID: new(customerUUID),
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session2.UUID)
	fmt.Printf("Subscription Link: %s\n", session2.Link)
	fmt.Println()

	// Example 3: Retrieve session details
	fmt.Println("3. Retrieving subscription session...")
	sessionDetails, err := client.Subscriptions.GetSession(session.UUID, nil)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("Price: $%.2f\n", sessionDetails.Price)
	fmt.Printf("Frequency (seconds): %d\n", sessionDetails.Frequency)
	fmt.Printf("Trial period (seconds): %d\n", sessionDetails.TrialPeriod)
	fmt.Printf("Is subscription: %v\n", sessionDetails.IsSubscription())
	fmt.Println()

	// The subscription itself exists only once the customer completes the checkout, and
	// it has its own sub@ UUID (different from the session UUID above). You receive it in
	// the subscription webhook (SubscriptionWebhook.SubscriptionUUID), or look the
	// subscription up by the Reference you set.
	subscriptionUUID := os.Getenv("QBITFLOW_SUBSCRIPTION_UUID")
	if subscriptionUUID == "" {
		sub, err := client.Subscriptions.GetSubscriptionByReference("plan-pro-customer-42")
		if err == nil {
			subscriptionUUID = sub.UUID
		}
	}

	if subscriptionUUID != "" {
		// Example 4: Get subscription details
		fmt.Println("4. Getting subscription details...")
		subscription, err := client.Subscriptions.GetSubscription(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get subscription: %v", err)
		}
		fmt.Printf("UUID: %s  Status: %s  Currency: %s  Next billing: %s\n",
			subscription.UUID, subscription.SubscriptionStatus, subscription.Currency.Symbol,
			subscription.NextBillingDate.Format("2006-01-02"))
		if subscription.LastBillingDate.IsZero() {
			fmt.Println("Not billed yet")
		}
		fmt.Println()

		// Example 5: Billing history
		fmt.Println("5. Billing history...")
		history, err := client.Subscriptions.GetPaymentHistory(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get billing history: %v", err)
		}
		for _, entry := range history {
			fmt.Printf("   %s — $%.2f on %s\n", entry.UUID, entry.Amount, entry.CreatedAt.Format("2006-01-02"))
		}
		fmt.Println()

		// Example 6: Execute a test billing cycle (test subscriptions only).
		//   response, err := client.Subscriptions.ExecuteTestBillingCycle(subscriptionUUID)
		//   (a subscription that is not due yet answers *qberrors.ConflictError)
		//
		// Example 7: Force cancel (use with caution).
		//   response, err := client.Subscriptions.ForceCancel(subscriptionUUID)
	} else {
		fmt.Println("4-7. Set QBITFLOW_SUBSCRIPTION_UUID (or complete the checkout above) to manage a subscription.")
		fmt.Println()
	}

	// Supported subscription frequencies
	fmt.Println("Supported subscription frequencies:")
	frequencies := []qbmodels.Duration{
		{Value: 1, Unit: qbmodels.DurationUnitDays},
		{Value: 1, Unit: qbmodels.DurationUnitWeeks},
		{Value: 2, Unit: qbmodels.DurationUnitWeeks},
		{Value: 1, Unit: qbmodels.DurationUnitMonths},
		{Value: 3, Unit: qbmodels.DurationUnitMonths},
		{Value: 6, Unit: qbmodels.DurationUnitMonths},
		{Value: 1, Unit: qbmodels.DurationUnitYears},
	}
	for _, freq := range frequencies {
		fmt.Printf("   %d %s\n", freq.Value, freq.Unit)
	}

	fmt.Println("\n=== Example Complete ===")
}
