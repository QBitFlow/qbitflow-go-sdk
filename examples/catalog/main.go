// Command catalog creates a product (once: its reference is unique per space), lists the
// products, opens a checkout for the product by its reference, and lists a page of customers.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/catalog
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
	client := newClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	existing, err := client.Products.GetByReference(ctx, "tshirt-blue-m")
	var notFound *qbitflow.NotFoundError
	switch {
	case errors.As(err, &notFound):
		// docs:start products-create
		product, err := client.Products.Create(ctx, &qbitflow.CreateProductParams{
			Name:        "T-shirt",
			Description: "Blue, size M",
			PriceUSD:    "4.99",          // a decimal string, never a float
			Reference:   "tshirt-blue-m", // your own id: unique per space
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("created", product.UUID, "payment link:", product.PaymentLink)
		// docs:end products-create
	case err != nil:
		log.Fatal(err)
	default:
		fmt.Println("tshirt-blue-m already exists:", existing.UUID)
	}

	// docs:start products-list
	// Iterate walks every page (List returns one); hidden products only with IncludeHidden.
	for p, err := range client.Products.Iterate(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %q: %s USD (%s)\n", p.UUID, p.Name, p.PriceUSD, p.Reference)
	}
	// docs:end products-list

	// docs:start checkout-create-payment-product
	session, err := client.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductReference: "tshirt-blue-m", // or ProductUUID
		SuccessURL:       "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:        "https://shop.example.com/orders/cancel",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Redirect the customer to", session.Link)
	// docs:end checkout-create-payment-product

	// docs:start customers-list
	page, err := client.Customers.List(ctx, &qbitflow.CustomerListParams{Limit: 20})
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range page.Items {
		fmt.Println(c.UUID, c.Email)
	}
	if page.HasMore {
		// Pass it back, verbatim, as CustomerListParams.Cursor for the next page.
		fmt.Println("next cursor:", page.NextCursor)
	}
	// docs:end customers-list
}

func newClient() *qbitflow.Client {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		log.Fatal("set QBITFLOW_API_KEY")
	}
	var opts []qbitflow.ClientOption
	if base := os.Getenv("QBITFLOW_BASE_URL"); base != "" {
		opts = append(opts, qbitflow.WithBaseURL(base))
	}
	client, err := qbitflow.NewClient(key, opts...)
	if err != nil {
		log.Fatal(err)
	}
	return client
}
