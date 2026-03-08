package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/QBitFlow/qbitflow-go-sdk"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

// WebhookServer handles incoming webhooks from QBitFlow
type WebhookServer struct {
	client *qbitflow.QBitFlow
}

func main() {
	// Initialize the QBitFlow client
	client := qbitflow.New("your-api-key-here")

	server := &WebhookServer{
		client: client,
	}

	// Set up webhook handler
	http.HandleFunc("/webhook", server.handleWebhook)

	// Set up success/cancel redirect handlers
	http.HandleFunc("/success", server.handleSuccess)
	http.HandleFunc("/cancel", server.handleCancel)

	fmt.Println("=== QBitFlow Webhook Handler Example ===")
	fmt.Println("Starting webhook server on :8080...")
	fmt.Println("\nEndpoints:")
	fmt.Println("  POST /webhook - Webhook handler")
	fmt.Println("  GET  /success - Success redirect handler")
	fmt.Println("  GET  /cancel  - Cancel redirect handler")
	fmt.Println()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// handleWebhook processes incoming webhook events
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📨 Received webhook event")

	// 1. Parse the webhook payload
	var webhook qbmodels.SessionWebhookResponse
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// 2. Extract the signature and timestamp from headers
	signature := r.Header.Get(s.client.Webhooks.GetSignatureHeader())
	timestamp := r.Header.Get(s.client.Webhooks.GetTimestampHeader())
	if signature == "" || timestamp == "" {
		fmt.Println("❌ Missing required webhook headers for verification")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. Verify the webhook authenticity
	valid, err := s.client.Webhooks.Verify(webhook, signature, timestamp)
	if err != nil || !valid {
		fmt.Printf("❌ Webhook verification failed: %v\n", err)
		// Sending a >= 400 response will cause QBitFlow to retry the webhook
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fmt.Printf("📋 Webhook Details:\n")
	fmt.Printf("   UUID: %s\n", webhook.UUID)
	fmt.Printf("   Status: %s\n", webhook.Status.Status)
	fmt.Printf("   Type: %s\n", webhook.Status.Type)

	if webhook.Status.TxHash != nil {
		fmt.Printf("   Transaction Hash: %s\n", *webhook.Status.TxHash)
	}

	if webhook.Status.Message != nil {
		fmt.Printf("   Message: %s\n", *webhook.Status.Message)
	}

	// Process webhook based on status
	switch webhook.Status.Status {
	case qbmodels.TransactionStatusCompleted:
		s.handleCompletedTransaction(webhook)

	case qbmodels.TransactionStatusFailed:
		s.handleFailedTransaction(webhook)

	case qbmodels.TransactionStatusCancelled:
		s.handleCancelledTransaction(webhook)

	case qbmodels.TransactionStatusPending:
		s.handlePendingTransaction(webhook)

	case qbmodels.TransactionStatusWaitingConfirmation:
		s.handleWaitingConfirmation(webhook)

	default:
		fmt.Printf("⚠️  Unhandled status: %s\n", webhook.Status.Status)
	}

	// Always respond with 200 OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"uuid":   webhook.UUID,
	})
}

// handleCompletedTransaction processes completed transactions
func (s *WebhookServer) handleCompletedTransaction(webhook qbmodels.SessionWebhookResponse) {
	fmt.Println("✅ Transaction completed successfully!")

	// Process based on transaction type
	switch webhook.Status.Type {
	case qbmodels.TransactionTypeOneTimePayment:
		fmt.Println("   Type: One-time payment")
		// Update order status, send confirmation email, etc.
		s.fulfillOrder(webhook.Session.CustomerUUID, webhook.UUID)

	case qbmodels.TransactionTypeCreateSubscription:
		fmt.Println("   Type: Subscription creation")
		// Activate subscription, grant access, send welcome email
		s.activateSubscription(webhook.Session.CustomerUUID, webhook.UUID)

	case qbmodels.TransactionTypeCreatePAYGSubscription:
		fmt.Println("   Type: PAYG subscription creation")
		// Activate PAYG subscription
		s.activatePAYGSubscription(webhook.Session.CustomerUUID, webhook.UUID)

	case qbmodels.TransactionTypeExecuteSubscriptionPayment:
		fmt.Println("   Type: Subscription renewal payment")
		// Handle recurring payment
		s.handleSubscriptionRenewal(webhook.Session.CustomerUUID, webhook.UUID)
	}

	fmt.Printf("   Customer UUID: %s\n", webhook.Session.CustomerUUID)
	fmt.Printf("   Product: %s\n", webhook.Session.ProductName)
	fmt.Printf("   Amount: $%.2f\n", webhook.Session.Price)
}

// handleFailedTransaction processes failed transactions
func (s *WebhookServer) handleFailedTransaction(webhook qbmodels.SessionWebhookResponse) {
	fmt.Println("❌ Transaction failed")

	// Log failure, send notification to customer
	fmt.Printf("   Customer UUID: %s\n", webhook.Session.CustomerUUID)
	fmt.Printf("   Product: %s\n", webhook.Session.ProductName)

	if webhook.Status.Message != nil {
		fmt.Printf("   Failure reason: %s\n", *webhook.Status.Message)
	}

	// Send email to customer about failed payment
	s.notifyPaymentFailed(webhook.Session.CustomerUUID, webhook.UUID)
}

// handleCancelledTransaction processes cancelled transactions
func (s *WebhookServer) handleCancelledTransaction(webhook qbmodels.SessionWebhookResponse) {
	fmt.Println("🚫 Transaction cancelled")

	// Log cancellation
	fmt.Printf("   Customer UUID: %s\n", webhook.Session.CustomerUUID)
	fmt.Printf("   Product: %s\n", webhook.Session.ProductName)

	// Update analytics, maybe send follow-up email
	s.handleCancellation(webhook.Session.CustomerUUID, webhook.UUID)
}

// handlePendingTransaction processes pending transactions
func (s *WebhookServer) handlePendingTransaction(webhook qbmodels.SessionWebhookResponse) {
	fmt.Println("⏳ Transaction pending")
	fmt.Printf("   UUID: %s\n", webhook.UUID)

	// Monitor transaction, maybe show pending status to customer
}

// handleWaitingConfirmation processes transactions waiting for confirmation
func (s *WebhookServer) handleWaitingConfirmation(webhook qbmodels.SessionWebhookResponse) {
	fmt.Println("⏰ Waiting for blockchain confirmation")
	fmt.Printf("   UUID: %s\n", webhook.UUID)

	if webhook.Status.TxHash != nil {
		fmt.Printf("   Tx Hash: %s\n", *webhook.Status.TxHash)
	}
}

// handleSuccess handles successful payment redirects
func (s *WebhookServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	transactionType := r.URL.Query().Get("transactionType")

	fmt.Printf("✅ Success redirect received\n")
	fmt.Printf("   UUID: %s\n", uuid)
	fmt.Printf("   Type: %s\n", transactionType)

	// Get transaction status
	if uuid != "" && transactionType != "" {
		txType := qbmodels.TransactionType(transactionType)
		status, err := s.client.TransactionStatus.GetTransactionStatus(uuid, txType)
		if err != nil {
			fmt.Printf("❌ Error getting transaction status: %v\n", err)
			http.Error(w, "Failed to get transaction status", http.StatusInternalServerError)
			return
		}

		fmt.Printf("   Status: %s\n", status.Status)

		// If completed, fetch session details
		if status.Status == qbmodels.TransactionStatusCompleted {
			session, err := s.client.Payments.GetSession(uuid)
			if err != nil {
				fmt.Printf("❌ Error getting session: %v\n", err)
			} else {
				fmt.Printf("   Product: %s\n", session.ProductName)
				fmt.Printf("   Price: $%.2f\n", session.Price)
			}
		}
	}

	// Render success page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
		<head><title>Payment Successful</title></head>
		<body>
			<h1>✅ Payment Successful!</h1>
			<p>Thank you for your payment.</p>
			<p>Transaction UUID: %s</p>
			<p><a href="/">Return to Home</a></p>
		</body>
		</html>
	`, uuid)
}

// handleCancel handles payment cancellation redirects
func (s *WebhookServer) handleCancel(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🚫 Cancel redirect received")

	// Render cancel page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
		<head><title>Payment Cancelled</title></head>
		<body>
			<h1>Payment Cancelled</h1>
			<p>Your payment has been cancelled.</p>
			<p><a href="/">Return to Home</a></p>
		</body>
		</html>
	`)
}

// Business logic functions (implement based on your needs)

func (s *WebhookServer) fulfillOrder(customerUUID, orderUUID string) {
	fmt.Printf("   📦 Fulfilling order for customer %s\n", customerUUID)
	// Your order fulfillment logic here:
	// - Update database
	// - Send confirmation email
	// - Trigger delivery process
	// - Update inventory
}

func (s *WebhookServer) activateSubscription(customerUUID, subscriptionUUID string) {
	fmt.Printf("   🔓 Activating subscription for customer %s\n", customerUUID)
	// Your subscription activation logic here:
	// - Grant access to premium features
	// - Update user permissions
	// - Send welcome email
	// - Schedule next billing reminder
}

func (s *WebhookServer) activatePAYGSubscription(customerUUID, subscriptionUUID string) {
	fmt.Printf("   🔓 Activating PAYG subscription for customer %s\n", customerUUID)
	// Your PAYG subscription activation logic here:
	// - Set up usage tracking
	// - Initialize usage counters
	// - Grant access
}

func (s *WebhookServer) handleSubscriptionRenewal(customerUUID, paymentUUID string) {
	fmt.Printf("   🔄 Processing subscription renewal for customer %s\n", customerUUID)
	// Your renewal logic here:
	// - Extend subscription period
	// - Send renewal confirmation
	// - Update analytics
}

func (s *WebhookServer) notifyPaymentFailed(customerUUID, transactionUUID string) {
	fmt.Printf("   📧 Sending payment failure notification to customer %s\n", customerUUID)
	// Your notification logic here:
	// - Send email
	// - Update retry schedule
	// - Log for support team
}

func (s *WebhookServer) handleCancellation(customerUUID, transactionUUID string) {
	fmt.Printf("   📊 Logging cancellation for customer %s\n", customerUUID)
	// Your cancellation logic here:
	// - Update analytics
	// - Maybe send survey
	// - Log for analysis
}
