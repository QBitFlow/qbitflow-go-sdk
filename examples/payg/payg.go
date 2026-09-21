package main

import (
	"fmt"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// NOTE: PAYG session creation is currently disabled on the API.
// This example demonstrates querying and managing existing PAYG subscriptions.
// To create new subscriptions, use client.Subscriptions.CreateSession instead.

func main() {
	client := qbitflow.New("your-api-key-here")

	fmt.Println("=== Pay-As-You-Go Subscription Example ===")
	fmt.Println("Note: PAYG session creation is temporarily disabled.")
	fmt.Println("Existing PAYG subscriptions can still be queried and managed.")

	// Existing PAYG subscriptions can be retrieved and managed:
	// subscriptionUUID := "existing-payg-subscription-uuid-here"

	// Get PAYG subscription details
	/*
		subscription, err := client.PayAsYouGo.GetSubscription(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get subscription: %v", err)
		}
		fmt.Printf("UUID: %s  Status: %s\n", subscription.UUID, subscription.SubscriptionStatus)
	*/

	// Get payment history
	/*
		history, err := client.PayAsYouGo.GetPaymentHistory(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to get history: %v", err)
		}
		fmt.Printf("Payment history: %d entries\n", len(history))
	*/

	// Execute a test billing cycle (test mode only)
	/*
		resp, err := client.PayAsYouGo.ExecuteTestBillingCycle(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to trigger billing: %v", err)
		}
		fmt.Printf("Billing triggered: %s\n", resp.Message)
	*/

	// Force cancel (use with caution)
	/*
		resp, err := client.PayAsYouGo.ForceCancel(subscriptionUUID)
		if err != nil {
			log.Fatalf("Failed to cancel: %v", err)
		}
		fmt.Printf("Cancelled: %s\n", resp.Message)
	*/

	// Retrieve a session by UUID (works for all session types)
	_ = client
	_ = qbf.CreateSessionOptions{}

	fmt.Println("=== Example Complete ===")
}
