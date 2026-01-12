package main

import (
	"fmt"
	"log"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/utils"
)

func main() {
	// Initialize the QBitFlow client
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Pay-As-You-Go Subscription Example ===\n")

	// Example 1: Create a PAYG subscription with free credits
	fmt.Println("1. Creating PAYG subscription with free credits...")
	session, err := client.PayAsYouGo.CreateSession(&qbitflow.CreatePAYGSessionOptions{
		ProductID: 1,
		Frequency: qbmodels.Duration{
			Value: 1,
			Unit:  qbmodels.DurationUnitMonths,
		},
		FreeCredits:  utils.Float64Ptr(100.0), // Give 100 free credits to start
		SuccessURL:   utils.StringPtr("https://yoursite.com/payg/success"),
		CancelURL:    utils.StringPtr("https://yoursite.com/payg/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create PAYG subscription: %v", err)
	}

	fmt.Printf("✓ PAYG subscription session created!\n")
	fmt.Printf("  Session UUID: %s\n", session.UUID)
	fmt.Printf("  Subscription Link: %s\n", session.Link)
	fmt.Println()

	// Example 2: Create a PAYG subscription without free credits
	fmt.Println("2. Creating basic PAYG subscription...")
	session2, err := client.PayAsYouGo.CreateSession(&qbitflow.CreatePAYGSessionOptions{
		ProductID: 1,
		Frequency: qbmodels.Duration{
			Value: 1,
			Unit:  qbmodels.DurationUnitMonths,
		},
		SuccessURL:   utils.StringPtr("https://yoursite.com/payg/success"),
		CancelURL:    utils.StringPtr("https://yoursite.com/payg/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-456"),
	})
	if err != nil {
		log.Fatalf("Failed to create PAYG subscription: %v", err)
	}

	fmt.Printf("✓ PAYG subscription session created!\n")
	fmt.Printf("  Session UUID: %s\n", session2.UUID)
	fmt.Println()

	// Example 3: Get PAYG session details
	fmt.Println("3. Retrieving PAYG session details...")
	sessionDetails, err := client.PayAsYouGo.GetSession(session.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("✓ Session retrieved!\n")
	fmt.Printf("  Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("  Price per unit: $%.2f\n", sessionDetails.Price)
	if sessionDetails.Options != nil {
		if sessionDetails.Options.FreeCredits != nil {
			fmt.Printf("  Free Credits: %.2f\n", *sessionDetails.Options.FreeCredits)
		}
		if sessionDetails.Options.MinPeriods != nil {
			fmt.Printf("  Minimum Periods: %d\n", *sessionDetails.Options.MinPeriods)
		}
	}
	fmt.Println()

	// Note: The following examples would work after the subscription is completed
	// subscriptionUUID := "completed-payg-subscription-uuid-here"

	// Example 4: Get PAYG subscription details
	fmt.Println("4. Getting PAYG subscription details...")
	fmt.Println("   (This would work with a completed subscription UUID)")

	// Uncomment when you have a real subscription UUID:
	/*
		subscription, err := client.PayAsYouGo.GetSubscription(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get subscription: %v", err)
		}

		fmt.Printf("✓ PAYG Subscription details:\n")
		fmt.Printf("  UUID: %s\n", subscription.UUID)
		fmt.Printf("  Status: %s\n", subscription.Status)
		fmt.Printf("  Current Units: %.2f\n", subscription.CurrentUnits)
		fmt.Printf("  Max Units: %.2f\n", subscription.MaxUnits)
		fmt.Printf("  Usage: %.1f%%\n", (subscription.CurrentUnits/subscription.MaxUnits)*100)
		fmt.Printf("  Next Billing Date: %s\n", subscription.NextBillingDate.Format("2006-01-02"))

		if subscription.FreeCredits != nil {
			fmt.Printf("  Free Credits: %.2f\n", *subscription.FreeCredits)
		}
		if subscription.MinPeriods != nil {
			fmt.Printf("  Minimum Periods: %d\n", *subscription.MinPeriods)
		}
	*/

	// Example 5: Increase units for current billing period
	fmt.Println("\n5. Increasing units for current period...")
	fmt.Println("   (This would work with a completed subscription UUID)")

	// Uncomment when you have a real subscription UUID:
	/*
		increaseAmount := 50.0
		updatedSubscription, err := client.PayAsYouGo.IncreaseUnitsCurrentPeriod(
			subscriptionUUID,
			increaseAmount,
		)
		if err != nil {
			log.Fatalf("Failed to increase units: %v", err)
		}

		fmt.Printf("✓ Units increased successfully!\n")
		fmt.Printf("  Previous Current Units: %.2f\n", subscription.CurrentUnits)
		fmt.Printf("  New Current Units: %.2f\n", updatedSubscription.CurrentUnits)
		fmt.Printf("  Increase Amount: %.2f\n", increaseAmount)
		fmt.Printf("  Max Units: %.2f\n", updatedSubscription.MaxUnits)
	*/

	// Example 6: Get payment history
	fmt.Println("\n6. Getting payment history...")
	fmt.Println("   (This would work with a completed subscription UUID)")

	// Uncomment when you have a real subscription UUID:
	/*
		history, err := client.PayAsYouGo.GetPaymentHistory(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get payment history: %v", err)
		}

		fmt.Printf("✓ Payment history retrieved: %d payments\n", len(history))
		for i, payment := range history {
			fmt.Printf("\n  Payment #%d:\n", i+1)
			fmt.Printf("    UUID: %s\n", payment.UUID)
			fmt.Printf("    Status: %s\n", payment.Status)
			fmt.Printf("    Units Used: %.2f\n", payment.CurrentUnits)
			fmt.Printf("    Date: %s\n", payment.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	*/

	// Example 7: Usage monitoring patterns
	fmt.Println("\n7. Usage Monitoring Patterns...\n")

	fmt.Println("   Pattern 1: Monitor usage threshold")
	fmt.Println("   ```go")
	fmt.Println("   if subscription.CurrentUnits >= subscription.MaxUnits * 0.8 {")
	fmt.Println("       // Alert customer - 80% usage reached")
	fmt.Println("       // Consider automatically increasing allowance")
	fmt.Println("   }")
	fmt.Println("   ```")

	fmt.Println("\n   Pattern 2: Proactive unit increase")
	fmt.Println("   ```go")
	fmt.Println("   if subscription.CurrentUnits >= subscription.MaxUnits * 0.9 {")
	fmt.Println("       // Automatically increase allowance by 20%")
	fmt.Println("       increase := subscription.MaxUnits * 0.2")
	fmt.Println("       client.PayAsYouGo.IncreaseUnitsCurrentPeriod(uuid, increase)")
	fmt.Println("   }")
	fmt.Println("   ```")

	// Example 8: Execute test billing cycle (TEST MODE ONLY)
	fmt.Println("\n8. Executing test billing cycle...")
	fmt.Println("   (This only works in test mode)")

	// Uncomment in test mode:
	/*
		response, err := client.PayAsYouGo.ExecuteTestBillingCycle(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to execute billing: %v", err)
		}

		fmt.Printf("✓ Billing cycle initiated!\n")
		fmt.Printf("  Message: %s\n", response.Message)
		fmt.Printf("  Status Link: %s\n", response.StatusLink)
	*/

	// Example 9: Force cancel (USE WITH CAUTION!)
	fmt.Println("\n9. Force cancelling PAYG subscription...")
	fmt.Println("   ⚠️  This immediately cancels the subscription!")
	fmt.Println("   (Example code commented out for safety)")

	// Uncomment only when you really need to cancel:
	/*
		response, err := client.PayAsYouGo.ForceCancel(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to cancel subscription: %v", err)
		}

		fmt.Printf("✓ PAYG Subscription cancelled!\n")
		fmt.Printf("  Message: %s\n", response.Message)
	*/

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\n💡 Best Practices for PAYG:")
	fmt.Println("   - Monitor usage regularly and alert customers at thresholds (e.g., 80%, 90%)")
	fmt.Println("   - Consider auto-increasing allowance when nearing limits")
	fmt.Println("   - Use webhooks to track billing events and usage updates")
	fmt.Println("   - Provide customers with usage analytics in your dashboard")
	fmt.Println("   - Set appropriate minimum periods to ensure commitment")
	fmt.Println("   - Offer free credits for new customers to encourage adoption")
}
