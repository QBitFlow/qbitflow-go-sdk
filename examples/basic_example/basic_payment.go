package main

import (
	"fmt"
	"log"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
	"github.com/qbitflow/qbitflow-go-sdk/pkg/utils"
)

func main() {
	// Initialize the QBitFlow client
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Basic One-Time Payment Example ===")

	// Example 1: Create a payment session using a product ID
	fmt.Println("1. Creating payment session with product ID...")
	session1, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
		ProductID:    utils.Uint64Ptr(1),
		SuccessURL:   utils.StringPtr("https://yoursite.com/success?uuid={{UUID}}&type={{TRANSACTION_TYPE}}"),
		CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
		WebhookURL:   utils.StringPtr("https://yoursite.com/webhook"),
		CustomerUUID: utils.StringPtr("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create payment session: %v", err)
	}

	fmt.Printf("✓ Payment session created successfully!\n")
	fmt.Printf("  Session UUID: %s\n", session1.UUID)
	fmt.Printf("  Payment Link: %s\n", session1.Link)
	if session1.ExpiresAt != nil {
		fmt.Printf("  Expires At: %d\n", *session1.ExpiresAt)
	}
	fmt.Println()

	// Example 2: Create a payment session with custom product details
	fmt.Println("2. Creating payment session with custom product...")
	session2, err := client.Payments.CreateSession(&qbitflow.CreateSessionOptions{
		ProductName:  utils.StringPtr("Premium Membership"),
		Description:  utils.StringPtr("One-time payment for premium membership"),
		Price:        utils.Float64Ptr(99.99),
		SuccessURL:   utils.StringPtr("https://yoursite.com/success"),
		CancelURL:    utils.StringPtr("https://yoursite.com/cancel"),
		CustomerUUID: utils.StringPtr("customer-uuid-456"),
	})
	if err != nil {
		log.Fatalf("Failed to create custom payment session: %v", err)
	}

	fmt.Printf("✓ Custom payment session created!\n")
	fmt.Printf("  Session UUID: %s\n", session2.UUID)
	fmt.Printf("  Payment Link: %s\n", session2.Link)
	fmt.Println()

	// Example 3: Get payment session details
	fmt.Println("3. Retrieving payment session details...")
	sessionDetails, err := client.Payments.GetSession(session1.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("✓ Session retrieved successfully!\n")
	fmt.Printf("  Product Name: %s\n", sessionDetails.ProductName)
	fmt.Printf("  Description: %s\n", sessionDetails.Description)
	fmt.Printf("  Price: $%.2f\n", sessionDetails.Price)
	fmt.Printf("  Organization: %s\n", sessionDetails.OrganizationName)
	fmt.Printf("  Customer UUID: %s\n", sessionDetails.CustomerUUID)
	fmt.Println()

	// Example 4: Get all payments (with pagination)
	fmt.Println("4. Getting all payments...")
	limit := 10
	payments, err := client.Payments.GetAllPayments(&limit, nil)
	if err != nil {
		log.Fatalf("Failed to get payments: %v", err)
	}

	fmt.Printf("✓ Retrieved %d payments\n", len(payments.Items))
	fmt.Printf("  Has More: %v\n", payments.HasMore())
	if payments.NextCursor != nil {
		fmt.Printf("  Next Cursor: %s\n", *payments.NextCursor)
	}

	// Display first few payments
	for i, payment := range payments.Items {
		if i >= 3 {
			break
		}
		fmt.Printf("\n  Payment #%d:\n", i+1)
		fmt.Printf("    UUID: %s\n", payment.UUID)
		fmt.Printf("    Amount: $%.2f\n", payment.Amount)
		fmt.Printf("    Created: %s\n", payment.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	// Example 5: Pagination - get next page if available
	if payments.HasMore() && payments.NextCursor != nil {
		fmt.Println("5. Getting next page of payments...")
		nextPage, err := client.Payments.GetAllPayments(&limit, payments.NextCursor)
		if err != nil {
			log.Fatalf("Failed to get next page: %v", err)
		}
		fmt.Printf("✓ Retrieved %d more payments\n", len(nextPage.Items))
	}

	fmt.Println("\n=== Example Complete ===")
}
