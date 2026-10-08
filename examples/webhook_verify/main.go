// Command webhook_verify receives QBitFlow webhooks without the router: it reads the raw body,
// verifies its QBitFlow-Signature header and parses the event with ConstructEvent, then
// answers. examples/webhook_handler does the same with a WebhookRouter.
//
//	QBITFLOW_WEBHOOK_SECRET=whsec_… go run ./examples/webhook_verify
package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/QBitFlow/qbitflow-go-sdk/v3"
)

func main() {
	if os.Getenv("QBITFLOW_WEBHOOK_SECRET") == "" {
		log.Fatal("set QBITFLOW_WEBHOOK_SECRET")
	}
	http.HandleFunc("POST /webhooks/qbitflow", handleWebhook)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// docs:start webhook-verify
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// The raw body, exactly as received: the signature covers its bytes.
	rawBody, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	event, err := qbitflow.ConstructEvent(rawBody, r.Header.Get(qbitflow.SignatureHeader), os.Getenv("QBITFLOW_WEBHOOK_SECRET"))
	var sigErr *qbitflow.WebhookSignatureError
	if errors.As(err, &sigErr) {
		log.Printf("rejected: %s", sigErr.Reason) // e.g. noMatchingSignature, timestampOutsideTolerance
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest) // not a v2 event
		return
	}

	if event.Type == qbitflow.EventTypePaymentCompleted {
		payment, err := event.PaymentCompleted()
		if err != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		// Deliveries are at least once: fulfil the order once per event.ID.
		log.Printf("event %s: order %q paid", event.ID, payment.Reference)
	}
	w.WriteHeader(http.StatusOK) // to the types you ignore too
}

// docs:end webhook-verify
