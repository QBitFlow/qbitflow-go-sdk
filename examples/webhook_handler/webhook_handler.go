// Command webhook_handler is a small HTTP server that receives QBitFlow webhooks and the
// checkout redirects.
//
// Configuration comes from the environment:
//
//	QBITFLOW_API_KEY         required (used for remote verification and status checks)
//	QBITFLOW_BASE_URL        optional (defaults to the production API)
//	QBITFLOW_WEBHOOK_SECRET  optional: when set, webhooks are verified locally (no API call)
//
// Point the dashboard's Transaction webhook at /webhook/transaction and the Subscription
// status webhook at /webhook/subscription.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// WebhookServer handles incoming webhooks from QBitFlow
type WebhookServer struct {
	client *qbitflow.QBitFlow
	secret string
}

// successPage is rendered with html/template, which escapes the query values it echoes —
// writing them into HTML verbatim would make the page an XSS vector.
var successPage = template.Must(template.New("success").Parse(`<html>
<head><title>Payment Successful</title></head>
<body>
	<h1>Payment Successful!</h1>
	<p>Thank you for your payment.</p>
	<p>Transaction UUID: {{.UUID}}</p>
	<p>Status: {{.Status}}</p>
	<p><a href="/">Return to Home</a></p>
</body>
</html>
`))

func main() {
	apiKey := os.Getenv("QBITFLOW_API_KEY")
	if apiKey == "" {
		log.Fatal("set QBITFLOW_API_KEY to run this example")
	}

	server := &WebhookServer{
		client: qbitflow.NewWithConfig(qbf.Config{
			APIKey:  apiKey,
			BaseURL: os.Getenv("QBITFLOW_BASE_URL"), // "" = production
		}),
		secret: os.Getenv("QBITFLOW_WEBHOOK_SECRET"),
	}

	http.HandleFunc("/webhook/transaction", server.handleTransactionWebhook)
	http.HandleFunc("/webhook/subscription", server.handleSubscriptionWebhook)
	http.HandleFunc("/success", server.handleSuccess)
	http.HandleFunc("/cancel", server.handleCancel)

	fmt.Println("=== QBitFlow Webhook Handler Example ===")
	fmt.Println("Starting webhook server on :8080...")
	fmt.Println("  POST /webhook/transaction   - Transaction webhook")
	fmt.Println("  POST /webhook/subscription  - Subscription status webhook")
	fmt.Println("  GET  /success               - Success redirect handler")
	fmt.Println("  GET  /cancel                - Cancel redirect handler")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// verify authenticates a webhook and returns its raw body, or writes the error response
// and returns ok=false.
//
// Answering >= 400 makes QBitFlow retry the delivery, which is what you want for a
// transient failure (the API was unreachable) but not for a forged request — both are
// rejected here, with different statuses so your logs tell them apart.
func (s *WebhookServer) verify(w http.ResponseWriter, r *http.Request) ([]byte, qbf.WebhookHeaders, bool) {
	headers := qbf.ExtractWebhookHeaders(r.Header)

	if s.secret != "" {
		// Local verification: needs the webhook secret, no network call. It reads at most
		// qbf.MaxWebhookBodySize bytes and checks the 5-minute replay window.
		body, err := qbf.VerifyWebhookRequest(r, s.secret, nil)
		if err != nil {
			log.Printf("local webhook verification failed: %v", err)
			http.Error(w, "invalid webhook", http.StatusUnauthorized)
			return nil, headers, false
		}
		return body, headers, true
	}

	// Remote verification: no secret needed, one API round-trip per webhook.
	body, err := readBody(r)
	if err != nil {
		http.Error(w, "invalid webhook", http.StatusBadRequest)
		return nil, headers, false
	}
	valid, err := s.client.Webhooks.Verify(body, headers.Signature, headers.Timestamp)
	if err != nil {
		var validation *qberrors.ValidationError
		if errors.As(err, &validation) && validation.StatusCode == 0 {
			// Missing headers or an empty/invalid body: not a genuine delivery.
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return nil, headers, false
		}
		// Outage, expired key, ...: not a verdict on the signature — let QBitFlow retry.
		log.Printf("webhook verification unavailable: %v", err)
		http.Error(w, "verification unavailable", http.StatusServiceUnavailable)
		return nil, headers, false
	}
	if !valid {
		http.Error(w, "invalid webhook", http.StatusUnauthorized)
		return nil, headers, false
	}
	return body, headers, true
}

// readBody reads a request body with the same bound VerifyWebhookRequest uses.
func readBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, qbf.MaxWebhookBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > qbf.MaxWebhookBodySize {
		return nil, fmt.Errorf("webhook body exceeds %d bytes", qbf.MaxWebhookBodySize)
	}
	return body, nil
}

// handleTransactionWebhook processes transaction webhooks (a checkout completed).
func (s *WebhookServer) handleTransactionWebhook(w http.ResponseWriter, r *http.Request) {
	body, headers, ok := s.verify(w, r)
	if !ok {
		return
	}

	// The dashboard's "Test webhook" button sends a probe to check the endpoint is
	// reachable. It is checked *after* verification on purpose: the probe is signed like
	// any other delivery, so this proves your whole verification setup works end to end.
	if headers.IsTest() {
		log.Println("connectivity test webhook received (signature verified)")
		w.WriteHeader(http.StatusOK)
		return
	}

	var webhook qbmodels.SessionWebhookResponse
	if err := json.Unmarshal(body, &webhook); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	log.Printf("transaction webhook: uuid=%s type=%s", webhook.UUID, webhook.TxType)

	// Status is nullable: acknowledge a delivery without one rather than rejecting it,
	// which would only make QBitFlow retry it.
	if webhook.Status != nil && webhook.Status.Status == qbmodels.TransactionStatusCompleted {
		customer := "collected at checkout"
		if webhook.Session.CustomerUUID != nil {
			customer = *webhook.Session.CustomerUUID
		}

		switch webhook.TxType {
		case qbmodels.TransactionTypeOneTimePayment:
			// Session.Reference echoes the reference you set when creating the session.
			log.Printf("payment completed for order %q (customer %s), tx %s",
				webhook.Session.Reference, customer, webhook.Status.TxHash)
		case qbmodels.TransactionTypeCreateSubscription:
			// The first period is already paid. webhook.UUID is the *session* UUID; the
			// subscription's own sub@ UUID arrives with the subscription webhooks.
			log.Printf("subscription started for customer %s (first period billed)", customer)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionWebhook processes subscription webhooks: status transitions and
// billing of a renewed period.
func (s *WebhookServer) handleSubscriptionWebhook(w http.ResponseWriter, r *http.Request) {
	body, headers, ok := s.verify(w, r)
	if !ok {
		return
	}
	if headers.IsTest() {
		w.WriteHeader(http.StatusOK)
		return
	}

	var event qbmodels.SubscriptionWebhook
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case qbmodels.SubscriptionWebhookTypeStatusTransition:
		transition, err := event.StatusTransition()
		if err != nil {
			http.Error(w, "invalid webhook payload", http.StatusBadRequest)
			return
		}
		log.Printf("subscription %s: %s -> %s", event.SubscriptionUUID,
			transition.PreviousStatus, transition.CurrentStatus)

		switch transition.CurrentStatus {
		case qbmodels.SubscriptionStatusActive:
			// Trial converted or payment recovered — grant/keep access.
		case qbmodels.SubscriptionStatusPastDue, qbmodels.SubscriptionStatusLowOnFunds:
			// Prompt the customer to top up their allowance.
		case qbmodels.SubscriptionStatusCancelled, qbmodels.SubscriptionStatusTrialExpired:
			// Revoke access.
		}

	case qbmodels.SubscriptionWebhookTypeBilling:
		billing, err := event.Billing()
		if err != nil {
			http.Error(w, "invalid webhook payload", http.StatusBadRequest)
			return
		}
		log.Printf("subscription %s renewed: %s, $%.2f", event.SubscriptionUUID, billing.UUID, billing.Amount)

	default:
		// A type this SDK does not know yet: acknowledge it so it is not retried forever.
		log.Printf("unhandled subscription webhook type %q", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// handleSuccess handles successful payment redirects.
func (s *WebhookServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")                       // from the {{UUID}} placeholder
	transactionType := r.URL.Query().Get("transactionType") // from the {{TRANSACTION_TYPE}} placeholder

	page := struct{ UUID, Status string }{UUID: uuid, Status: "unknown"}

	// Never trust the redirect alone: ask the API for the transaction's actual state.
	if uuid != "" && transactionType != "" {
		status, err := s.client.TransactionStatus.GetTransactionStatus(uuid, qbmodels.TransactionType(transactionType))
		var notFound *qberrors.NotFoundError
		switch {
		case errors.As(err, &notFound):
			page.Status = "not processed yet"
		case err != nil:
			log.Printf("error getting transaction status: %v", err)
			http.Error(w, "Failed to get transaction status", http.StatusInternalServerError)
			return
		default:
			page.Status = string(status.Status)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := successPage.Execute(w, page); err != nil {
		log.Printf("rendering success page: %v", err)
	}
}

// handleCancel handles payment cancellation redirects.
func (s *WebhookServer) handleCancel(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<html>
<head><title>Payment Cancelled</title></head>
<body>
	<h1>Payment Cancelled</h1>
	<p>Your payment has been cancelled.</p>
	<p><a href="/">Return to Home</a></p>
</body>
</html>
`)
}
