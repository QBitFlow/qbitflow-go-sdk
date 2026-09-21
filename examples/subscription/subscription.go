package main

import (
	"fmt"
	"log"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Subscription Management Example ===")
	fmt.Println()

	// Example 1: Create a subscription with a trial period
	fmt.Println("1. Creating subscription with trial period...")
	session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductID:    new(uint64(1)),
		Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
		TrialPeriod:  &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays},
		SuccessURL:   new("https://yoursite.com/subscription/success"),
		CancelURL:    new("https://yoursite.com/subscription/cancel"),
		CustomerUUID: new("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session.UUID)
	fmt.Printf("Subscription Link: %s\n", session.Link)
	fmt.Println()

	// Example 2: Create a subscription without trial
	fmt.Println("2. Creating subscription without trial...")
	session2, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
		ProductID:    new(uint64(2)),
		Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
		SuccessURL:   new("https://yoursite.com/subscription/success"),
		CancelURL:    new("https://yoursite.com/subscription/cancel"),
		CustomerUUID: new("customer-uuid-456"),
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session2.UUID)
	fmt.Printf("Subscription Link: %s\n", session2.Link)
	fmt.Println()

	// Example 3: Retrieve session details
	fmt.Println("3. Retrieving subscription session...")
	sessionDetails, err := client.Subscriptions.GetSession(session.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("Price: $%.2f\n", sessionDetails.Price)
	fmt.Printf("Frequency (seconds): %d\n", sessionDetails.Frequency)
	fmt.Printf("Trial period (seconds): %d\n", sessionDetails.TrialPeriod)
	fmt.Printf("Is subscription: %v\n", sessionDetails.IsSubscription())
	fmt.Println()

	// Note: the following examples require a completed subscription UUID.
	// subscriptionUUID := "completed-subscription-uuid-here"

	// Example 4: Get subscription details (uncomment with a real UUID)
	/*
		subscription, err := client.Subscriptions.GetSubscription(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get subscription: %v", err)
		}
		fmt.Printf("UUID: %s  Status: %s  Next billing: %s\n",
			subscription.UUID, subscription.SubscriptionStatus,
			subscription.NextBillingDate.Format("2006-01-02"))
	*/

	// Example 5: Execute test billing cycle (test mode only)
	/*
		response, err := client.Subscriptions.ExecuteTestBillingCycle(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to execute billing: %v", err)
		}
		fmt.Printf("Billing triggered: %s\n", response.Message)
	*/

	// Example 6: Force cancel (use with caution)
	/*
		response, err := client.Subscriptions.ForceCancel(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to cancel subscription: %v", err)
		}
		fmt.Printf("Subscription cancelled: %s\n", response.Message)
	*/

	// Example 7: Supported subscription frequencies
	fmt.Println("7. Supported subscription frequencies:")
	frequencies := []qbmodels.Duration{
		{Value: 1, Unit: qbmodels.DurationUnitDays},
		{Value: 1, Unit: qbmodels.DurationUnitWeeks},
		{Value: 2, Unit: qbmodels.DurationUnitWeeks},
		{Value: 1, Unit: qbmodels.DurationUnitMonths},
		{Value: 3, Unit: qbmodels.DurationUnitMonths},
		{Value: 6, Unit: qbmodels.DurationUnitMonths},
		{Value: 12, Unit: qbmodels.DurationUnitMonths},
	}
	for _, freq := range frequencies {
		fmt.Printf("   %d %s\n", freq.Value, freq.Unit)
	}

	fmt.Println("\n=== Example Complete ===")
}
