// Command marketplace walks the marketplace flow with an organization API key: list the members
// and the pending invitations, invite a seller (and revoke the invitation), then act for a member
// who joined: read them and their wallets, sell in their space, read the funds held for them,
// and, on request, change their fee, trust them or remove them.
//
//	QBITFLOW_API_KEY=sk_… [QBITFLOW_BASE_URL=…] [INVITE=1 [REVOKE=1]] go run ./examples/marketplace
//	QBITFLOW_API_KEY=sk_… MEMBER_UUID=<a member's userUuid> [FEE=1] [TRUST=1] [REMOVE=1] go run ./examples/marketplace
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

	overview(ctx, client)
	if os.Getenv("INVITE") == "1" {
		invite(ctx, client)
	}

	memberUUID := os.Getenv("MEMBER_UUID")
	if memberUUID == "" {
		fmt.Println("set MEMBER_UUID to act for a member")
		return
	}
	sellAs(ctx, client, memberUUID)
	manage(ctx, client, memberUUID)
}

// overview lists the members and the pending invitations.
func overview(ctx context.Context, client *qbitflow.Client) {
	// docs:start members-list
	page, err := client.Members.List(ctx, &qbitflow.MemberListParams{Limit: 20})
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range page.Items {
		fmt.Printf("%s %s %s, fee %s %%, trusted: %v\n", m.UserUUID, m.FirstName, m.Email, m.OrganizationFeePercent, m.TrustedAt != nil)
	}
	// docs:end members-list

	// docs:start invitations-list
	invitations, err := client.Invitations.List(ctx, &qbitflow.InvitationListParams{
		Status: qbitflow.InvitationStatusPending,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, inv := range invitations.Items {
		fmt.Println(inv.UUID, inv.Email, "expires", inv.ExpiresAt.Format(time.RFC3339))
	}
	// docs:end invitations-list
}

// invite invites a seller, and revokes the invitation on request. They exist in the
// organization once they accepted.
func invite(ctx context.Context, client *qbitflow.Client) {
	// docs:start members-invite
	created, err := client.Invitations.Create(ctx, &qbitflow.CreateInvitationParams{
		Email:                  "seller@example.com",
		HoldFunds:              true, // hold their payments until Members.Trust (always sent)
		OrganizationFeePercent: "10", // your commission on their payments: "0" to "50" %
		RedirectURL:            "https://shop.example.com/sellers/welcome",
	})
	if err != nil {
		log.Fatal(err) // 409 already_joined for a member
	}
	// The link is also emailed. The seller exists once they accepted: wait for member.joined.
	fmt.Println("invitation", created.UUID, "link:", created.Link)
	// docs:end members-invite

	// Then wait for the member.joined webhook (see examples/webhook_handler):
	//
	//	joined, _ := event.MemberJoined()
	//	// joined.InvitationUUID == created.UUID; store joined.UserUUID,
	//	// then run this program with MEMBER_UUID=<joined.UserUUID>.

	if os.Getenv("REVOKE") != "1" {
		return
	}
	invitationUUID := created.UUID

	// docs:start invitations-revoke
	// 204: its link stops working.
	if err := client.Invitations.Revoke(ctx, invitationUUID); err != nil {
		log.Fatal(err) // 409 once accepted or revoked
	}
	fmt.Println("revoked", invitationUUID)
	// docs:end invitations-revoke
}

// sellAs reads a member and acts in their space.
func sellAs(ctx context.Context, client *qbitflow.Client, memberUUID string) {
	// docs:start members-get
	member, err := client.Members.Get(ctx, memberUUID) // their userUuid
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %s <%s>, fee %s %%, trusted: %v\n",
		member.FirstName, member.LastName, member.Email, member.OrganizationFeePercent, member.TrustedAt != nil)
	for _, c := range member.AcceptedCurrencies {
		fmt.Println("  accepts", c.Code)
	}
	// docs:end members-get

	// docs:start members-wallets
	wallets, err := client.Wallets.ListForMember(ctx, memberUUID)
	if err != nil {
		log.Fatal(err)
	}
	for _, w := range wallets {
		fmt.Println(w.Currency.Chain, w.Address)
	}
	// docs:end members-wallets

	// docs:start client-on-behalf-of
	// A client acting in the member's space (organization key): every request sends
	// On-Behalf-Of. It shares the connections and configuration of client.
	seller := client.OnBehalfOf(memberUUID)
	products, err := seller.Products.List(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(products.Items), "products on the first page of the seller's space")
	// docs:end client-on-behalf-of

	// Is the seller ready to be paid? Their checkouts need a wallet accepting a currency.
	currencies, err := client.Wallets.ListAcceptedCurrencies(ctx, &qbitflow.AcceptedCurrenciesParams{UserUUID: memberUUID})
	if err != nil {
		log.Fatal(err)
	}
	if len(currencies) == 0 {
		fmt.Println("the seller has no wallet yet: their checkouts would answer 409 merchant_not_ready")
	} else {
		product, err := seller.Products.Create(ctx, &qbitflow.CreateProductParams{
			Name: "Handmade mug", PriceUSD: "4.50", Reference: fmt.Sprintf("mug-%d", time.Now().Unix()),
		})
		if err != nil {
			log.Fatalf("product: %v", err) // 403 policy_disabled if members.products is off
		}
		session, err := seller.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
			ProductUUID: product.UUID,
			SuccessURL:  "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		})
		if err != nil {
			log.Fatalf("checkout: %v", err)
		}
		fmt.Printf("the buyer pays at %s\n", session.Link)
	}

	// docs:start members-held-funds
	// What the organization holds for its members (their payments while untrusted).
	// The longest held first.
	all, err := client.Members.ListHeldFunds(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range all {
		fmt.Printf("%s: %s USD over %d lines\n", s.UserUUID, s.TotalUSD, s.Count)
	}

	held, err := client.Members.GetHeldFunds(ctx, memberUUID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("owed to %s: %s USD over %d lines\n", memberUUID, held.TotalUSD, len(held.Ledgers))
	// docs:end members-held-funds

	// docs:start members-own-held-funds
	// As the member: with their own key, or the organization's acting for them.
	own, err := client.OnBehalfOf(memberUUID).Members.GetOwnHeldFunds(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("held for me: %s USD over %d lines\n", own.TotalUSD, len(own.Ledgers))
	// docs:end members-own-held-funds
}

// manage changes a member's fee, trusts them or removes them, on request.
func manage(ctx context.Context, client *qbitflow.Client, memberUUID string) {
	if os.Getenv("FEE") == "1" {
		// docs:start members-update
		// The organization's commission on their payments, a percent string ("0" to "50"; a
		// checkout already created keeps its fee).
		member, err := client.Members.Update(ctx, memberUUID, &qbitflow.UpdateMemberParams{
			OrganizationFeePercent: "10",
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("fee now", member.OrganizationFeePercent, "%")
		// docs:end members-update
	}

	if os.Getenv("TRUST") == "1" {
		// docs:start members-trust
		// Their new payments go to their own wallets from now on. What is held stays held until
		// you release it in the dashboard (heldFunds.released tells you).
		member, err := client.Members.Trust(ctx, memberUUID)
		if err != nil {
			log.Fatal(err)
		}
		if member.TrustedAt != nil {
			fmt.Println("trusted since", member.TrustedAt.Format(time.RFC3339))
		}
		// docs:end members-trust
	}

	if os.Getenv("REMOVE") == "1" {
		// docs:start members-remove
		err := client.Members.Remove(ctx, memberUUID)
		var conflict *qbitflow.ConflictError
		switch {
		case errors.As(err, &conflict) && conflict.Code == "held_funds_pending":
			fmt.Println("not removed: release the funds held for them first")
		case err != nil:
			log.Fatal(err)
		default:
			fmt.Println("removed: their keys stop working and their checkouts close")
		}
		// docs:end members-remove
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
