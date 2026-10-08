// Command accounting exports a year of accounting events, as JSON rows and as a CSV file
// (qbitflow-2026.csv in the current directory).
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] go run ./examples/accounting
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// docs:start accounting-export
	// The API exports at most 95 days at a time: the …Range methods split any range into
	// windows, request them in order and join the results (the CSV header once).
	events, err := client.Accounting.ExportJSONRange(ctx, "2026-01-01", "2026-12-31")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range events {
		fmt.Println(e.Type, e.PaymentUUID, e.TxTimeUTC.Format(time.DateOnly), e.ProductName)
	}

	csv, err := client.Accounting.ExportCSVRange(ctx, "2026-01-01", "2026-12-31")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("qbitflow-2026.csv", []byte(csv), 0o600); err != nil {
		log.Fatal(err)
	}
	// docs:end accounting-export
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
