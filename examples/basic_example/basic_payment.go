// Command basic_example walks through one-time payments: creating checkout sessions,
// reading a session back, listing payments and resolving the customer behind a payment.
//
// Configuration comes from the environment:
//
//	QBITFLOW_API_KEY        required
//	QBITFLOW_BASE_URL       optional (defaults to the production API)
//	QBITFLOW_CUSTOMER_UUID  optional: a customer UUID to pre-associate with the session
//	QBITFLOW_PRODUCT_ID     optional: an existing product ID to sell
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
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

	// An empty CustomerUUID is simply left out of the request, so the customer is asked
	// for their details during checkout when QBITFLOW_CUSTOMER_UUID is not set.
	customerUUID := os.Getenv("QBITFLOW_CUSTOMER_UUID")

	fmt.Println("=== Basic One-Time Payment Example ===")

	// Example 1: Create a payment session with an inline product definition
	fmt.Println("1. Creating payment session with an inline product...")
	session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductName:  new("Premium Membership"),
		Description:  new("One-time payment for premium membership"),
		Price:        new(99.99),
		SuccessURL:   new("https://yoursite.com/success"),
		CancelURL:    new("https://yoursite.com/cancel"),
		CustomerUUID: new(customerUUID),
		Reference:    new("order-1001"), // your own order ID, echoed back on the payment
	})
	if err != nil {
		log.Fatalf("Failed to create payment session: %v", err)
	}

	fmt.Printf("Session UUID: %s\n", session.UUID)
	fmt.Printf("Payment Link: %s\n", session.Link)
	fmt.Println()

	// Example 2: Create a payment session for an existing product
	if raw := os.Getenv("QBITFLOW_PRODUCT_ID"); raw != "" {
		productID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			log.Fatalf("QBITFLOW_PRODUCT_ID must be a positive integer: %v", err)
		}

		fmt.Println("2. Creating payment session with an existing product...")
		byID, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(productID),
			SuccessURL:   new("https://yoursite.com/success"),
			CancelURL:    new("https://yoursite.com/cancel"),
			CustomerUUID: new(customerUUID),
		})
		if err != nil {
			log.Fatalf("Failed to create product payment session: %v", err)
		}
		fmt.Printf("Session UUID: %s\n", byID.UUID)
		fmt.Printf("Payment Link: %s\n", byID.Link)
		fmt.Println()
	}

	// Example 3: Retrieve session details
	fmt.Println("3. Retrieving payment session details...")
	sessionDetails, err := client.Payments.GetSession(session.UUID, nil)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Product Name: %s\n", sessionDetails.ProductName)
	fmt.Printf("Price: $%.2f\n", sessionDetails.Price)
	fmt.Printf("Organization: %s\n", sessionDetails.OrganizationName)
	fmt.Printf("Accepted currency IDs: %v\n", sessionDetails.AvailableCurrencies)
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
		fmt.Printf("  [%d] UUID: %s  Amount: $%.2f %s\n",
			i+1, payment.UUID, payment.Amount, payment.Currency.Symbol)
	}
	fmt.Println()

	// Example 5: Retrieve the customer associated with a payment
	fmt.Println("5. Getting customer for a transaction...")
	if len(payments.Items) > 0 {
		payment := payments.Items[0]
		if payment.CustomerUUID == nil {
			fmt.Println("The latest payment has no customer attached")
		} else {
			customer, err := client.Payments.GetCustomerForTransaction(payment.UUID)
			if err != nil {
				log.Printf("Failed to get customer for transaction: %v", err)
			} else {
				fmt.Printf("Customer: %s %s (%s)\n", customer.Name, customer.LastName, customer.Email)
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
}
