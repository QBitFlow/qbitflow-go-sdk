// Command currencies lists the currencies customers can pay with.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/currencies
package main

import (
	"context"
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

	// docs:start currencies-list
	// A public route limited to 60 requests a minute per IP: cache the list at start-up.
	// One bounded list (a slice): the live currencies; Test: the test networks'; Main: the
	// chains' native coins only.
	currencies, err := client.Currencies.List(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range currencies {
		fmt.Printf("%d: %s (%s on %s), %d decimals\n", c.ID, c.Code, c.Name, c.Chain, c.Decimals)
	}
	// docs:end currencies-list
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
