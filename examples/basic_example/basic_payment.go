package main

import (
	"fmt"
	"log"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func main() {
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Basic One-Time Payment Example ===")

	// Example 1: Create a payment session using a product ID
	fmt.Println("1. Creating payment session with product ID...")
	session1, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductID:    new(uint64(1)),
		SuccessURL:   new("https://yoursite.com/success"),
		CancelURL:    new("https://yoursite.com/cancel"),
		CustomerUUID: new("customer-uuid-123"),
	})
	if err != nil {
		log.Fatalf("Failed to create payment session: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session1.UUID)
	fmt.Printf("Payment Link: %s\n", session1.Link)
	fmt.Println()

	// Example 2: Create a payment session with inline product details
	fmt.Println("2. Creating payment session with custom product...")
	session2, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductName:  new("Premium Membership"),
		Description:  new("One-time payment for premium membership"),
		Price:        new(99.99),
		SuccessURL:   new("https://yoursite.com/success"),
		CancelURL:    new("https://yoursite.com/cancel"),
		CustomerUUID: new("customer-uuid-456"),
	})
	if err != nil {
		log.Fatalf("Failed to create custom payment session: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session2.UUID)
	fmt.Printf("Payment Link: %s\n", session2.Link)
	fmt.Println()

	// Example 3: Retrieve session details
	fmt.Println("3. Retrieving payment session details...")
	sessionDetails, err := client.Payments.GetSession(session1.UUID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Product Name: %s\n", sessionDetails.ProductName)
	fmt.Printf("Price: $%.2f\n", sessionDetails.Price)
	fmt.Printf("Organization: %s\n", sessionDetails.OrganizationName)
	fmt.Printf("Is payment: %v\n", sessionDetails.IsPayment())
	fmt.Println()

	// Example 4: Get all payments with pagination
	fmt.Println("4. Getting all payments...")
	limit := uint16(10)
	payments, err := client.Payments.GetAllPayments(&limit, nil)
	if err != nil {
		log.Fatalf("Failed to get payments: %v", err)
	}

	fmt.Printf("Retrieved %d payments, has more: %v\n", len(payments.Items), payments.HasMore())
	for i, payment := range payments.Items {
		if i >= 3 {
			break
		}
		fmt.Printf("  [%d] UUID: %s  Amount: $%.2f\n", i+1, payment.UUID, payment.Amount)
	}
	fmt.Println()

	// Example 5: Retrieve the customer associated with a transaction
	fmt.Println("5. Getting customer for a transaction...")
	if len(payments.Items) > 0 {
		customer, err := client.Payments.GetCustomerForTransaction(payments.Items[0].UUID)
		if err != nil {
			log.Printf("Failed to get customer for transaction: %v", err)
		} else {
			fmt.Printf("Customer: %s %s (%s)\n", customer.Name, customer.LastName, customer.Email)
		}
	}

	fmt.Println("\n=== Example Complete ===")
}
