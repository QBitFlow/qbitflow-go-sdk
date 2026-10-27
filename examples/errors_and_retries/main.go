// Command errors_and_retries shows how the SDK reports errors and retries: the error classes
// with errors.As, IsRetryable, and a stable Idempotency-Key across processes.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/errors_and_retries
//
// The last step opens a checkout for the order order-1042 (unique per space) with an idempotency
// key: once that order is paid, it answers 409 unique_violation.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v3"
)

func main() {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		log.Fatal("set QBITFLOW_API_KEY")
	}
	// Reads and the 6 idempotent creates are retried on network errors, 5xx and 429 (1 s, 2 s,
	// 4 s…); WithMaxRetries(0) turns that off. WithTimeout bounds each attempt.
	opts := []qbitflow.ClientOption{qbitflow.WithMaxRetries(3), qbitflow.WithTimeout(15 * time.Second)}
	if base := os.Getenv("QBITFLOW_BASE_URL"); base != "" {
		opts = append(opts, qbitflow.WithBaseURL(base))
	}
	client, err := qbitflow.NewClient(key, opts...)
	if err != nil {
		log.Fatal(err) // a *ValidationError: a bad key format or option
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// docs:start errors-handling
	payment, err := client.Payments.GetByReference(ctx, "order-1042")
	var (
		validation *qbitflow.ValidationError
		notFound   *qbitflow.NotFoundError
		apiErr     *qbitflow.APIError
	)
	switch {
	case err == nil:
		fmt.Println("order-1042 paid by", payment.UUID)
	case errors.As(err, &validation):
		// StatusCode 0: refused before sending; 400: refused by the API.
		for _, f := range validation.FieldErrors {
			fmt.Printf("%s: %s\n", f.Field, f.Message)
		}
	case errors.As(err, &notFound):
		fmt.Println("no payment for order-1042")
	case errors.As(err, &apiErr):
		// Every SDK error embeds an *APIError: branch on Code, quote RequestID to support.
		fmt.Printf("QBitFlow error %d %s (request %s), retryable: %v\n",
			apiErr.StatusCode, apiErr.Code, apiErr.RequestID, qbitflow.IsRetryable(err))
	default:
		log.Fatal(err)
	}
	// docs:end errors-handling

	// 1. Client-side validation: nothing is sent.
	_, err = client.Products.Create(ctx, &qbitflow.CreateProductParams{Name: "x", PriceUSD: "-1"})
	describe("invalid params", err)

	// 2. A 404 from the API.
	_, err = client.CheckoutSessions.Get(ctx, "pay@019eca82-5680-7b00-8000-00000000dead")
	describe("unknown session", err)

	// 3. An idempotent create with your own key: the same key replays the first result instead
	//    of creating a second customer, even from another process after a crash. Reusing the key
	//    with another body is a 422 (*IdempotencyError).
	idempotencyKey := "signup-" + time.Now().Format("20060102-150405")
	params := &qbitflow.CreateCustomerParams{
		FirstName: "Ada",
		Email:     "ada+" + idempotencyKey + "@example.com",
		Reference: "crm-" + idempotencyKey,
	}
	first, err := client.Customers.Create(ctx, params, qbitflow.WithIdempotencyKey(idempotencyKey), qbitflow.WithRequestID("signup-42"))
	if err != nil {
		describe("create", err)
		return
	}
	again, err := client.Customers.Create(ctx, params, qbitflow.WithIdempotencyKey(idempotencyKey))
	if err == nil {
		fmt.Printf("same customer both times: %v (%s)\n", first.UUID == again.UUID, first.UUID)
	}
	params.FirstName = "Grace"
	_, err = client.Customers.Create(ctx, params, qbitflow.WithIdempotencyKey(idempotencyKey))
	describe("same key, other body", err)

	// 4. A unique value taken twice: a 409 unique_violation, details.field naming it (a reference
	//    is unique per space; an email is not).
	_, err = client.Customers.Create(ctx, &qbitflow.CreateCustomerParams{FirstName: "Ada", Email: first.Email, Reference: first.Reference})
	describe("reference taken", err)

	_ = client.Customers.Delete(ctx, first.UUID) // clean up

	checkoutOnce(ctx)
}

// checkoutOnce opens the checkout of an order with a client that retries, under a key that
// makes a retried job return the same session.
func checkoutOnce(ctx context.Context) {
	// docs:start retries-idempotency
	// Reads and creates are retried on network errors, 5xx and 429 (1 s, 2 s, 4 s…);
	// WithMaxRetries(0) turns that off. The client reads QBITFLOW_API_KEY.
	client, err := qbitflow.NewClientFromEnv(qbitflow.WithMaxRetries(5))
	if err != nil {
		log.Fatal(err)
	}

	// Each create sends an Idempotency-Key, the same on its retries. Derive your own from the
	// order to retry across processes: the same key returns the same session, never a second one.
	orderID := "order-1042"
	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductName: "T-shirt",
		Description: "Blue, size M",
		PriceUSD:    "4.99",
		Reference:   orderID,
		SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:   "https://shop.example.com/orders/cancel",
	}, qbitflow.WithIdempotencyKey("checkout-"+orderID))
	var reused *qbitflow.IdempotencyError
	if errors.As(err, &reused) {
		log.Fatal("this key was already used with other params") // 422 idempotency_key_reused
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Redirect the customer to", session.Link)
	// docs:end retries-idempotency
}

// describe prints what an error is, the way an application would branch on it.
func describe(what string, err error) {
	var (
		validation *qbitflow.ValidationError
		notFound   *qbitflow.NotFoundError
		conflict   *qbitflow.ConflictError
		idem       *qbitflow.IdempotencyError
		rateLimit  *qbitflow.RateLimitError
		auth       *qbitflow.AuthenticationError
		denied     *qbitflow.PermissionDeniedError
		network    *qbitflow.NetworkError
		server     *qbitflow.ServerError
		api        *qbitflow.APIError
	)
	fmt.Printf("%s: ", what)
	switch {
	case err == nil:
		fmt.Println("ok")
	case errors.As(err, &validation):
		// StatusCode 0: refused before sending; 400: refused by the API.
		fmt.Printf("invalid input (status %d)\n", validation.StatusCode)
		for _, f := range validation.FieldErrors {
			fmt.Printf("  %s: %s\n", f.Field, f.Message)
		}
	case errors.As(err, &notFound):
		fmt.Printf("not found (request %s)\n", notFound.RequestID)
	case errors.As(err, &conflict):
		fmt.Printf("conflict %s, details %v\n", conflict.Code, conflict.Details) // e.g. unique_violation, details.field
	case errors.As(err, &idem):
		fmt.Println("idempotency key reused with another request: use a new key")
	case errors.As(err, &rateLimit):
		fmt.Printf("rate limited: retry after %s\n", rateLimit.RetryAfter)
	case errors.As(err, &auth):
		fmt.Println("bad API key")
	case errors.As(err, &denied):
		fmt.Printf("not allowed: %s\n", denied.Code) // forbidden, or policy_disabled (details.policy)
	case errors.As(err, &network), errors.As(err, &server):
		fmt.Printf("transient (retryable: %v): %v\n", qbitflow.IsRetryable(err), err)
	case errors.As(err, &api):
		fmt.Printf("API error %d %s\n", api.StatusCode, api.Code)
	default:
		fmt.Println(err)
	}
}
