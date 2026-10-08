// Command client builds a client from an API key and checks what the key is, then builds one
// from the environment with an explicit option.
//
//	QBITFLOW_API_KEY=sk_… go run ./examples/client
//	QBITFLOW_API_KEY=sk_… QBITFLOW_BASE_URL=… [QBITFLOW_ON_BEHALF_OF=<a member's userUuid>] go run ./examples/client
//
// The first client uses the default base URL; the second one reads QBITFLOW_BASE_URL too.
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if os.Getenv("QBITFLOW_BASE_URL") == "" {
		initClient(ctx)
	}
	fromEnv(ctx)
}

// initClient builds a client from the key alone, with the default base URL.
func initClient(ctx context.Context) {
	// docs:start client-init
	// One client per API key, shared by the whole program: it is safe for concurrent use.
	client, err := qbitflow.NewClient(os.Getenv("QBITFLOW_API_KEY"))
	if err != nil {
		log.Fatal(err) // a *qbitflow.ValidationError: no key, or not an sk_… key
	}

	// The recommended start-up check: what is this key, and which mode is it in?
	me, err := client.Me(ctx)
	if err != nil {
		log.Fatal(err) // a *qbitflow.AuthenticationError for an unknown or revoked key
	}
	if me.Space != nil {
		fmt.Printf("%s, role %s, test mode %v\n", me.Space.OrganizationName, me.Role, me.Space.Test)
	}
	// docs:end client-init
}

// fromEnv builds a client from the environment.
func fromEnv(ctx context.Context) {
	// docs:start config-from-env
	// QBITFLOW_API_KEY (required), QBITFLOW_BASE_URL and QBITFLOW_ON_BEHALF_OF (when set);
	// the options given here override them.
	client, err := qbitflow.NewClientFromEnv(qbitflow.WithTimeout(10 * time.Second))
	if err != nil {
		log.Fatal(err) // a *qbitflow.ValidationError naming QBITFLOW_API_KEY when it is not set
	}
	// docs:end config-from-env

	me, err := client.Me(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("from the environment: role %s, on behalf of %q\n", me.Role, me.OnBehalfOf)
}
