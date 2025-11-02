package main

import (
	"fmt"
	"log"

	qbmodels "github.com/qbitflow/qbitflow-go-sdk/pkg/models"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/utils"
)

func main() {
	// Initialize the QBitFlow client
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Subscription Management Example ===\n")

	// Example 1: Create a subscription with a trial period
	fmt.Println("1. Creating subscription with trial period...")
	session, err := client.Subscriptions.CreateSession(&qbitflow.CreateSubscriptionSessionOptions{
		ProductID: 1,
		Frequency: qbmodels.Duration{
			Value: 1,
			Unit:  qbmodels.DurationUnitMonths,
		},
		TrialPeriod: &qbmodels.Duration{
			Value: 7,
			Unit:  qbmodels.DurationUnitDays,
		},
		SuccessURL:   utils.StringPtr("https://yoursite.com/subscription/success"),
		CancelURL:    utils.StringPtr("https://yoursite.com/subscription/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("✓ Subscription session created!\n")
	fmt.Printf("  Session UUID: %s\n", session.UUID)
	fmt.Printf("  Subscription Link: %s\n", session.Link)
	fmt.Println()

	// Example 2: Create a subscription without trial period
	fmt.Println("2. Creating subscription without trial...")
	session2, err := client.Subscriptions.CreateSession(&qbitflow.CreateSubscriptionSessionOptions{
		ProductID: 2,
		Frequency: qbmodels.Duration{
			Value: 1,
			Unit:  qbmodels.DurationUnitMonths,
		},
		SuccessURL:   utils.StringPtr("https://yoursite.com/subscription/success"),
		CancelURL:    utils.StringPtr("https://yoursite.com/subscription/cancel"),
		CustomerUUID: utils.StringPtr("customer-uuid-456"),
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}

	fmt.Printf("✓ Subscription session created!\n")
	fmt.Printf("  Session UUID: %s\n", session2.UUID)
	fmt.Printf("  Subscription Link: %s\n", session2.Link)
	fmt.Println()

	// Example 3: Get subscription session details
	fmt.Println("3. Retrieving subscription session...")
	sessionDetails, err := client.Subscriptions.GetSession(session.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("✓ Session retrieved!\n")
	fmt.Printf("  Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("  Price: $%.2f\n", sessionDetails.Price)
	if sessionDetails.Options != nil {
		fmt.Printf("  Frequency: %d\n", sessionDetails.Options.Frequency)
		if sessionDetails.Options.TrialPeriod != nil {
			fmt.Printf("  Trial Period: %d periods\n", *sessionDetails.Options.TrialPeriod)
		}
	}
	fmt.Println()

	// Note: The following examples would work after the subscription is completed
	// For demonstration purposes, we'll use placeholder UUIDs

	// subscriptionUUID := "completed-subscription-uuid-here"

	// Example 4: Get subscription details
	fmt.Println("4. Getting subscription details...")
	fmt.Println("   (This would work with a completed subscription UUID)")

	// Uncomment when you have a real subscription UUID:
	/*
		subscription, err := client.Subscriptions.GetSubscription(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get subscription: %v", err)
		}

		fmt.Printf("✓ Subscription details:\n")
		fmt.Printf("  UUID: %s\n", subscription.UUID)
		fmt.Printf("  Status: %s\n", subscription.Status)
		fmt.Printf("  Product ID: %d\n", subscription.ProductID)
		fmt.Printf("  Customer UUID: %s\n", subscription.CustomerUUID)
		fmt.Printf("  Next Billing Date: %s\n", subscription.NextBillingDate.Format("2006-01-02"))
		fmt.Printf("  Created At: %s\n", subscription.CreatedAt.Format("2006-01-02 15:04:05"))
	*/

	// Example 5: Get subscription payment history
	fmt.Println("\n5. Getting payment history...")
	fmt.Println("   (This would work with a completed subscription UUID)")

	// Uncomment when you have a real subscription UUID:
	/*
		history, err := client.Subscriptions.GetPaymentHistory(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get payment history: %v", err)
		}

		fmt.Printf("✓ Payment history retrieved: %d payments\n", len(history))
		for i, payment := range history {
			fmt.Printf("\n  Payment #%d:\n", i+1)
			fmt.Printf("    UUID: %s\n", payment.UUID)
			fmt.Printf("    Status: %s\n", payment.Status)
			fmt.Printf("    Date: %s\n", payment.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	*/

	// Example 6: Execute test billing cycle (TEST MODE ONLY)
	fmt.Println("\n6. Executing test billing cycle...")
	fmt.Println("   (This only works in test mode)")

	// Uncomment in test mode:
	/*
		response, err := client.Subscriptions.ExecuteTestBillingCycle(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to execute billing: %v", err)
		}

		fmt.Printf("✓ Billing cycle initiated!\n")
		fmt.Printf("  Message: %s\n", response.Message)
		fmt.Printf("  Status Link: %s\n", response.StatusLink)
	*/

	// Example 7: Force cancel subscription (USE WITH CAUTION!)
	fmt.Println("\n7. Force cancelling subscription...")
	fmt.Println("   ⚠️  This immediately cancels the subscription!")
	fmt.Println("   (Example code commented out for safety)")

	// Uncomment only when you really need to cancel:
	/*
		response, err := client.Subscriptions.ForceCancel(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to cancel subscription: %v", err)
		}

		fmt.Printf("✓ Subscription cancelled!\n")
		fmt.Printf("  Message: %s\n", response.Message)
	*/

	// Example 8: Different subscription frequencies
	fmt.Println("\n8. Examples of different subscription frequencies...")

	frequencies := []qbmodels.Duration{
		{Value: 1, Unit: qbmodels.DurationUnitDays},    // Daily
		{Value: 1, Unit: qbmodels.DurationUnitWeeks},   // Weekly
		{Value: 2, Unit: qbmodels.DurationUnitWeeks},   // Bi-weekly
		{Value: 1, Unit: qbmodels.DurationUnitMonths},  // Monthly
		{Value: 3, Unit: qbmodels.DurationUnitMonths},  // Quarterly
		{Value: 6, Unit: qbmodels.DurationUnitMonths},  // Semi-annually
		{Value: 12, Unit: qbmodels.DurationUnitMonths}, // Annually
	}

	for _, freq := range frequencies {
		fmt.Printf("   %d %s\n", freq.Value, freq.Unit)
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\n💡 Tips:")
	fmt.Println("   - Always use webhooks to track subscription events")
	fmt.Println("   - Store subscription UUIDs in your database")
	fmt.Println("   - Test billing cycles in test mode before going live")
	fmt.Println("   - Monitor next billing dates for proactive customer communication")
}
