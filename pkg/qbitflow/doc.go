// Package qbitflow provides a Go SDK for the QBitFlow - Next Generation Crypto Payment Processing.
//
// The QBitFlow SDK allows you to easily integrate cryptocurrency payments into your
// Go applications. It provides a simple and intuitive interface for creating payment
// sessions, managing subscriptions, and tracking transaction status.
//
// # Installation
//
//	go get github.com/qbitflow/qbitflow-go-sdk
//
// # Quick Start
//
//	package main
//
//	import (
//	    "fmt"
//	    "log"
//	    "github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
//	)
//
//	func main() {
//	    // Initialize the client
//	    client := qbitflow.New("your-api-key-here")
//
//	    // Create a payment session
//	    session, err := client.Payments.CreateSession(qbitflow.CreateSessionOptions{
//	        ProductID:    intPtr(1),
//	        CustomerUUID: stringPtr("customer-123"),
//	    })
//
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    fmt.Printf("Payment link: %s\n", session.Link)
//	}
//
// # Features
//
//   - One-time payment sessions
//   - Recurring subscriptions
//   - Pay-as-you-go subscriptions
//   - Transaction status tracking
//   - Webhook support
//   - Comprehensive error handling
//   - Type-safe API
//
// # Services
//
// The SDK is organized into several services:
//
//   - Payments: Handle one-time payment operations
//   - Subscriptions: Manage recurring subscriptions
//   - PayAsYouGo: Handle usage-based subscriptions
//   - TransactionStatus: Track transaction status
//
// # Error Handling
//
// The SDK provides custom error types for better error handling:
//
//   - QBitFlowError: General API errors with status codes
//   - NotFoundError: 404 not found errors
//   - ValidationError: Client-side validation errors
//
// Example:
//
//	import qberrors "github.com/qbitflow/qbitflow-go-sdk/pkg/errors"
//
//	payment, err := client.Payments.GetPayment("invalid-uuid")
//	if err != nil {
//	    if _, ok := err.(*qberrors.NotFoundError); ok {
//	        fmt.Println("Payment not found")
//	    }
//	}
//
// # Configuration
//
// You can customize the client configuration:
//
//	client := qbitflow.NewWithConfig(qbitflow.Config{
//	    APIKey:  "your-api-key",
//	    BaseURL: "https://api.qbitflow.app",
//	    Timeout: 30 * time.Second,
//	})
//
// For more information, see the full documentation at https://qbitflow.app/docs
package qbf
