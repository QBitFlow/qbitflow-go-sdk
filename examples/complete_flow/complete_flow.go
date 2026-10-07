// Command complete_flow demonstrates a complete payment workflow, from session creation
// through status polling to payment verification.
//
// Configuration comes from the environment:
//
//	QBITFLOW_API_KEY        required
//	QBITFLOW_BASE_URL       optional (defaults to the production API)
//	QBITFLOW_CUSTOMER_UUID  optional: a customer UUID to pre-associate with the session
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
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

	fmt.Println("=== Complete Payment Flow Example ===")
	fmt.Println()

	// Step 1: Create a payment session.
	//
	// The checkout page substitutes {{UUID}} and {{TRANSACTION_TYPE}} in the redirect URLs
	// with the session's UUID and transaction type, so your success page knows what to
	// verify.
	fmt.Println("Step 1: Creating payment session...")
	session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
		ProductName:  new("Ebook: Go in Practice"),
		Description:  new("Digital download, delivered by email"),
		Price:        new(19.99),
		SuccessURL:   new("https://yoursite.com/success?uuid={{UUID}}&transactionType={{TRANSACTION_TYPE}}"),
		CancelURL:    new("https://yoursite.com/cancel"),
		CustomerUUID: new(os.Getenv("QBITFLOW_CUSTOMER_UUID")), // "" is left out of the request
		Reference:    new("order-2001"),
	})
	if err != nil {
		log.Fatalf("Failed to create payment session: %v", err)
	}

	fmt.Printf("✓ Payment session created!\n")
	fmt.Printf("  Session UUID: %s\n", session.UUID)
	fmt.Printf("  Payment Link: %s\n\n", session.Link)

	// Step 2: Get session details
	fmt.Println("Step 2: Retrieving session details...")
	sessionDetails, err := client.Payments.GetSession(session.UUID, nil)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("✓ Session details retrieved!\n")
	fmt.Printf("  Product: %s\n", sessionDetails.ProductName)
	fmt.Printf("  Description: %s\n", sessionDetails.Description)
	fmt.Printf("  Price: $%.2f USD\n", sessionDetails.Price)
	fmt.Printf("  Organization: %s\n", sessionDetails.OrganizationName)
	if sessionDetails.CustomerUUID != nil {
		fmt.Printf("  Customer UUID: %s\n", *sessionDetails.CustomerUUID)
	} else {
		fmt.Println("  Customer: collected during checkout")
	}
	fmt.Println()

	// Step 3: At this point, you would redirect the user to session.Link
	fmt.Println("Step 3: User payment flow...")
	fmt.Printf("  Redirect user to: %s\n", session.Link)
	fmt.Println("  (In production, use webhooks for real-time updates)")
	fmt.Println()

	// Step 4: Check transaction status (polling example)
	fmt.Println("Step 4: Checking transaction status...")
	var status *qbmodels.TransactionStatus
	for attempt := 1; attempt <= 5; attempt++ {
		fmt.Printf("  Attempt %d/5: Checking status...\n", attempt)

		status, err = client.TransactionStatus.GetTransactionStatus(session.UUID, qbmodels.TransactionTypeOneTimePayment)

		var notFound *qberrors.NotFoundError
		switch {
		case errors.As(err, &notFound):
			// The session exists but nothing has been paid yet.
			fmt.Println("  Not processed yet")
		case err != nil:
			fmt.Printf("  Error checking status: %v\n", err)
		default:
			fmt.Printf("  Current Status: %s\n", status.Status)
		}

		if status != nil && isFinal(status.Status) {
			break
		}
		if attempt < 5 {
			time.Sleep(5 * time.Second)
		}
	}
	fmt.Println()

	// Step 5: On the success page, verify the transaction before fulfilling the order —
	// never trust the redirect alone.
	fmt.Println("Step 5: Verifying the transaction...")
	switch {
	case status == nil:
		fmt.Println("  Transaction not processed yet; wait for the webhook.")
	case status.Status == qbmodels.TransactionStatusCompleted:
		fmt.Printf("  Payment completed, tx hash %s\n", status.TxHash)

		payment, err := client.Payments.GetPaymentByReference("order-2001")
		if err != nil {
			log.Fatalf("Failed to get payment: %v", err)
		}
		fmt.Printf("  Paid $%.2f in %s\n", payment.Amount, payment.Currency.Symbol)
		fulfillOrder(payment)
	case status.Status == qbmodels.TransactionStatusFailed:
		// Message is "" when the API gives no reason.
		fmt.Printf("  Payment failed: %s\n", status.Message)
	default:
		fmt.Printf("  Transaction not completed yet. Status: %s\n", status.Status)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Best practices:")
	fmt.Println("  - Configure webhook URLs in the QBitFlow dashboard and verify every signature")
	fmt.Println("  - Verify the transaction status on success redirects")
	fmt.Println("  - The SDK retries idempotent GETs on network errors and 5xx (1s/2s/4s back-off)")
	fmt.Println(strings.Repeat("=", 50))
}

// isFinal reports whether a transaction status will not change any more.
func isFinal(status qbmodels.TransactionStatusValue) bool {
	switch status {
	case qbmodels.TransactionStatusCompleted, qbmodels.TransactionStatusFailed,
		qbmodels.TransactionStatusCancelled, qbmodels.TransactionStatusExpired:
		return true
	}
	return false
}

// fulfillOrder simulates order fulfillment.
func fulfillOrder(payment *qbmodels.Payment) {
	fmt.Println("  Order fulfillment initiated:")
	fmt.Printf("     - Payment UUID: %s\n", payment.UUID)
	if payment.Reference != nil {
		fmt.Printf("     - Order: %s\n", *payment.Reference)
	}
	fmt.Printf("     - Product: %s\n", payment.Name)
}
