package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/QBitFlow/qbitflow-go-sdk/pkg/utils"
)

func main() {
	// Initialize the QBitFlow client
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Complete Payment Flow Example ===\n")
	fmt.Println("This example demonstrates a complete payment workflow from")
	fmt.Println("session creation through status checking and payment verification.\n")

	// Step 1: Create a payment session
	fmt.Println("Step 1: Creating payment session...")
	session, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
		ProductID:    utils.Uint64Ptr(1),
		SuccessURL:   utils.StringPtr("https://yoursite.com/success?uuid={{UUID}}&type={{TRANSACTION_TYPE}}"),
		CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create payment session: %v", err)
	}

	fmt.Printf("✓ Payment session created!\n")
	fmt.Printf("  Session UUID: %s\n", session.UUID)
	fmt.Printf("  Payment Link: %s\n\n", session.Link)

	// Step 2: Get session details
	fmt.Println("Step 2: Retrieving session details...")
	sessionDetails, err := client.Payments.GetSession(session.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("✓ Session details retrieved!\n")
	fmt.Printf("  Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("  Description: %s\n", sessionDetails.Description)
	fmt.Printf("  Price: $%.2f USD\n", sessionDetails.Price)
	fmt.Printf("  Organization: %s\n", sessionDetails.OrganizationName)
	fmt.Printf("  Customer UUID: %s\n\n", sessionDetails.CustomerUUID)

	// Step 3: At this point, you would redirect the user to session.Link
	fmt.Println("Step 3: User payment flow...")
	fmt.Printf("  🌐 Redirect user to: %s\n", session.Link)
	fmt.Println("  👤 User selects cryptocurrency and completes payment")
	fmt.Println("  ⏳ Waiting for payment confirmation...\n")

	// Simulate waiting for payment (in real scenario, this would be async via webhook)
	fmt.Println("  (In production, use webhooks for real-time updates)\n")

	// Step 4: Check transaction status (polling example)
	fmt.Println("Step 4: Checking transaction status...")
	fmt.Println("  (This demonstrates status checking - in production use webhooks)")

	// Example of polling transaction status
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("  Attempt %d/%d: Checking status...\n", attempt, maxAttempts)

		status, err := client.TransactionStatus.GetTransactionStatus(
			session.UUID,
			qbmodels.TransactionTypeOneTimePayment,
		)

		if err != nil {
			fmt.Printf("  ⚠️  Error checking status: %v\n", err)
			// In production, handle NotFoundError specifically
			// (Transaction might not be created yet if user hasn't started payment)
		} else {
			fmt.Printf("  Current Status: %s\n", status.Status)

			// Check if payment is completed
			if status.Status == qbmodels.TransactionStatusCompleted {
				fmt.Println("  ✅ Payment completed successfully!")

				if status.TxHash != nil {
					fmt.Printf("  📝 Transaction Hash: %s\n", *status.TxHash)
				}

				break
			} else if status.Status == qbmodels.TransactionStatusFailed {
				fmt.Println("  ❌ Payment failed!")
				if status.Message != nil {
					fmt.Printf("  Reason: %s\n", *status.Message)
				}
				break
			} else if status.Status == qbmodels.TransactionStatusCancelled {
				fmt.Println("  🚫 Payment cancelled by user")
				break
			}
		}

		if attempt < maxAttempts {
			time.Sleep(5 * time.Second)
		}
	}
	fmt.Println()

	// Step 5: Handle success redirect (example)
	fmt.Println("Step 5: Handling success redirect...")
	fmt.Println("  User is redirected to success URL with UUID and transaction type")
	fmt.Println("  Example: https://yoursite.com/success?uuid=XXX&type=payment")
	fmt.Println()

	// When user lands on success page, verify the transaction
	fmt.Println("  Verifying transaction on success page...")
	status, err := client.TransactionStatus.GetTransactionStatus(
		session.UUID,
		qbmodels.TransactionTypeOneTimePayment,
	)

	if err != nil {
		fmt.Printf("  ❌ Error verifying transaction: %v\n", err)
	} else {
		fmt.Printf("  Transaction Status: %s\n", status.Status)

		if status.Status == qbmodels.TransactionStatusCompleted {
			// Step 6: Retrieve payment details
			fmt.Println("\nStep 6: Retrieving payment details...")

			// Note: GetPayment only works after payment is processed
			// In this example, we'll show the session details instead
			fmt.Println("  Payment confirmed! Session details:")
			fmt.Printf("    Product: %s\n", sessionDetails.ProductName)
			fmt.Printf("    Amount: $%.2f\n", sessionDetails.Price)
			fmt.Printf("    Customer: %s\n", sessionDetails.CustomerUUID)

			// Step 7: Fulfill order
			fmt.Println("\nStep 7: Fulfilling order...")
			fulfillOrder(sessionDetails.CustomerUUID, session.UUID, sessionDetails.ProductName)
		} else {
			fmt.Printf("  ⚠️  Transaction not completed yet. Status: %s\n", status.Status)
		}
	}

	// Step 8: Best practices summary
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎯 Best Practices Summary:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\n1. Session Creation:")
	fmt.Println("   ✓ Always provide webhook URL for real-time updates")
	fmt.Println("   ✓ Use success/cancel URLs with {{UUID}} and {{TRANSACTION_TYPE}} placeholders")
	fmt.Println("   ✓ Store session UUID in your database")

	fmt.Println("\n2. Status Monitoring:")
	fmt.Println("   ✓ Use webhooks instead of polling for production")
	fmt.Println("   ✓ Implement proper error handling for all status checks")
	fmt.Println("   ✓ Handle all transaction statuses (pending, failed, cancelled, etc.)")

	fmt.Println("\n3. Security:")
	fmt.Println("   ✓ Validate webhook signatures (if implemented)")
	fmt.Println("   ✓ Verify transaction status on success redirects")
	fmt.Println("   ✓ Never trust client-side data alone")

	fmt.Println("\n4. User Experience:")
	fmt.Println("   ✓ Show clear payment instructions")
	fmt.Println("   ✓ Provide status updates during confirmation")
	fmt.Println("   ✓ Handle cancellations gracefully")
	fmt.Println("   ✓ Send confirmation emails after successful payments")

	fmt.Println("\n5. Error Handling:")
	fmt.Println("   ✓ Handle network errors")
	fmt.Println("   ✓ Implement retry logic for transient failures")
	fmt.Println("   ✓ Log all errors for debugging")
	fmt.Println("   ✓ Provide helpful error messages to users")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("\n=== Complete Flow Example Finished ===")
}

// fulfillOrder simulates order fulfillment
func fulfillOrder(customerUUID, orderUUID, productName string) {
	fmt.Println("  📦 Order fulfillment initiated:")
	fmt.Printf("     - Customer UUID: %s\n", customerUUID)
	fmt.Printf("     - Order UUID: %s\n", orderUUID)
	fmt.Printf("     - Product: %s\n", productName)
	fmt.Println("     - Updating database...")
	fmt.Println("     - Sending confirmation email...")
	fmt.Println("     - Granting access/delivering product...")
	fmt.Println("  ✅ Order fulfilled successfully!")
}
