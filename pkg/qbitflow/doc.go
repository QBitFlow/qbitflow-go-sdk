// Package qbf provides the Go SDK for QBitFlow - Next Generation Crypto Payment Processing.
//
// The QBitFlow SDK allows you to easily integrate cryptocurrency payments into your
// Go applications. It provides a simple and intuitive interface for creating payment
// sessions, managing subscriptions, and tracking transaction status.
//
// # Installation
//
//	go get github.com/QBitFlow/qbitflow-go-sdk/v2
//
// # Quick Start
//
//	package main
//
//	import (
//		"fmt"
//		"log"
//
//		"github.com/QBitFlow/qbitflow-go-sdk/v2"
//		qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
//	)
//
//	func main() {
//		// Initialize the client
//		client := qbitflow.New("your-api-key-here")
//
//		// Create a payment session for an existing product
//		session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
//			ProductID: new(uint64(1)),
//		})
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		fmt.Printf("Payment link: %s\n", session.Link)
//	}
//
// # Features
//
//   - One-time payment sessions
//   - Recurring subscriptions
//   - Transaction status: webhooks (recommended) or polling GetTransactionStatus
//   - Webhook verification, locally (HMAC) or via the API
//   - Customers, products, users, API keys (read), account claims
//   - Refunds and accounting exports (JSON and CSV)
//   - Acting on behalf of a user, per client (QBitFlow.OnBehalfOf) or per service
//   - Client-side validation mirroring the API's rules
//   - Automatic retries for idempotent requests
//   - Typed errors and strongly typed responses
//
// # Services
//
// The SDK is organized into services exposed on the qbitflow.QBitFlow client:
// Payments, Subscriptions, TransactionStatus, Customers, Products, Users, ApiKeys,
// Webhooks, Currencies, Refunds, Accounting and Claims.
//
// # Error Handling
//
// Every failure is one of the types in package qberrors, mapped from the HTTP status:
//
//   - ValidationError: 400/422, and client-side validation before a request (StatusCode 0)
//   - AuthenticationError: 401
//   - ForbiddenError: 403
//   - NotFoundError: 404
//   - ConflictError: 409
//   - RateLimitError: 429 (with RetryAfter)
//   - ServerError: 5xx, a 3xx (redirects are never followed), or a 2xx whose body is empty
//     or not the expected JSON
//   - NetworkError: no response received
//   - QBitFlowError: the base type, returned directly for any other 4xx
//
// Every type unwraps to *qberrors.QBitFlowError, so one errors.As target reads the status
// of any failure.
//
// Example:
//
//	import qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
//
//	payment, err := client.Payments.GetPayment("pay@...")
//	var notFound *qberrors.NotFoundError
//	if errors.As(err, &notFound) {
//		fmt.Println("Payment not found")
//	}
//
// # Configuration
//
// You can customize the client configuration:
//
//	client := qbitflow.NewWithConfig(qbf.Config{
//		APIKey:     "your-api-key",
//		BaseURL:    "https://api.qbitflow.app/v1",
//		Timeout:    30 * time.Second,
//		MaxRetries: 3, // idempotent GETs only; 0 = default (3), negative or DisableRetries = off
//	})
//
// For more information, see the full documentation at https://qbitflow.app/docs
package qbf
