// Command marketplace walks the marketplace flow with an organization API key: invite a seller,
// act in their space once they joined (a product and a checkout of theirs), trust them, and
// read the funds the organization holds for them.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] SELLER_EMAIL=seller@example.com go run ./examples/marketplace
//	QBITFLOW_API_KEY=sk_… MEMBER_UUID=<a member's userUuid> go run ./examples/marketplace
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

	memberUUID := os.Getenv("MEMBER_UUID")
	if memberUUID == "" {
		invite(ctx, client)
		return
	}
	sellAs(ctx, client, memberUUID)
}

// invite invites a seller. They exist in the organization once they accepted.
func invite(ctx context.Context, client *qbitflow.Client) {
	email := os.Getenv("SELLER_EMAIL")
	if email == "" {
		log.Fatal("set SELLER_EMAIL (to invite) or MEMBER_UUID (to act for a member)")
	}
	fee := 5.0 // the organization keeps 5% of the seller's payments
	created, err := client.Invitations.Create(ctx, &qbitflow.CreateInvitationParams{
		Email:                  email,
		TrustLayer:             true, // hold their payments until Members.Trust
		OrganizationFeePercent: &fee,
		RedirectURL:            "https://market.example.com/welcome",
	})
	var conflict *qbitflow.ConflictError
	if errors.As(err, &conflict) && conflict.Code == "already_joined" {
		fmt.Println(email, "is already a member")
		return
	}
	if err != nil {
		log.Fatalf("invite: %v", err)
	}
	fmt.Printf("invitation %s sent; link: %s\n", created.Invitation.UUID, created.Link)

	// Then wait for the member.joined webhook (see examples/webhook_handler):
	//
	//	joined, _ := event.MemberJoined()
	//	// joined.InvitationUUID == created.Invitation.UUID; store joined.UserUUID,
	//	// then run this program with MEMBER_UUID=<joined.UserUUID>.
}

// sellAs acts in a member's space: OnBehalfOf sends On-Behalf-Of on every request.
func sellAs(ctx context.Context, client *qbitflow.Client, memberUUID string) {
	member, err := client.Members.Get(ctx, memberUUID)
	if err != nil {
		log.Fatalf("member: %v", err)
	}
	fmt.Printf("member %s %s, trusted: %v\n", member.Name, member.LastName, member.TrustedAt != nil)

	// Is the seller ready to be paid? Their checkouts need a wallet accepting a currency.
	currencies, err := client.Wallets.ListSupportedCurrencies(ctx, &qbitflow.SupportedCurrenciesParams{UserUUID: memberUUID})
	if err != nil {
		log.Fatalf("currencies: %v", err)
	}
	if len(currencies) == 0 {
		fmt.Println("the seller has no wallet yet: their checkouts would answer 409 merchant_not_ready")
		return
	}

	seller := client.OnBehalfOf(memberUUID)
	product, err := seller.Products.Create(ctx, &qbitflow.CreateProductParams{
		Name: "Handmade mug", Price: 4.5, Reference: fmt.Sprintf("mug-%d", time.Now().Unix()),
	})
	if err != nil {
		log.Fatalf("product: %v", err) // 403 policy_disabled if members.products is off
	}
	session, err := seller.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductUUID: product.UUID,
		SuccessURL:  "https://market.example.com/orders/{{UUID}}",
	})
	if err != nil {
		log.Fatalf("checkout: %v", err)
	}
	fmt.Printf("the buyer pays at %s\n", session.Link)

	// What the organization holds for the seller (their payments while untrusted).
	held, err := client.Members.GetHeldFunds(ctx, memberUUID)
	if err != nil {
		log.Fatalf("held funds: %v", err)
	}
	fmt.Printf("held for them: %.2f USD over %d payments\n", held.TotalAmount, len(held.Ledgers))
	// From the seller's side (their key, or OnBehalfOf): the same total.
	if own, err := seller.Members.GetOwnHeldFunds(ctx); err == nil {
		fmt.Printf("as seen by the seller: %.2f USD\n", own.TotalAmount)
	}

	// Trust them: their new payments go to their wallets directly. What is held stays held
	// until released in the dashboard.
	if member.TrustedAt == nil && os.Getenv("TRUST") == "1" {
		trusted, err := client.Members.Trust(ctx, memberUUID)
		if err != nil {
			log.Fatalf("trust: %v", err)
		}
		fmt.Printf("trusted since %s\n", trusted.TrustedAt.Format(time.RFC3339))
	}
}

func newClient() *qbitflow.Client {
	key := os.Getenv("QBITFLOW_API_KEY")
	if key == "" {
		log.Fatal("set QBITFLOW_API_KEY (an organization key)")
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
