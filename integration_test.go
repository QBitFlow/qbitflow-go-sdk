//go:build integration

// Live checks against a QBitFlow API. They never run with `go test ./...`:
//
//	QBITFLOW_API_KEY=sk_… QBITFLOW_BASE_URL=https://… go test -tags integration -run Live -v .
//
// The read-only group only reads. The write group also needs QBITFLOW_LIVE_WRITES=1 and a
// test-mode key (or QBITFLOW_ALLOW_LIVE_MODE_WRITES=1, for a live-mode key on a disposable
// server such as one running on local chains); it creates a product, a customer, checkout
// sessions (one with fees) and a webhook endpoint, and deletes (or expires) each of them.
package qbitflow_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v3"
)

// liveClient builds a client from the environment: no key → skip; a key without a base URL →
// fail (never aim the checks at production by accident).
func liveClient(t *testing.T) *qbitflow.Client {
	t.Helper()
	key, base := os.Getenv("QBITFLOW_API_KEY"), os.Getenv("QBITFLOW_BASE_URL")
	if key == "" {
		t.Skip("QBITFLOW_API_KEY not set: live checks skipped")
	}
	if base == "" {
		t.Fatal("QBITFLOW_API_KEY is set but QBITFLOW_BASE_URL is not: set the API's base URL explicitly")
	}
	c, err := qbitflow.NewClient(key, qbitflow.WithBaseURL(base))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func liveContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// isOrganizationKey reports whether me is an organization key acting for itself.
func isOrganizationKey(me *qbitflow.Me) bool {
	return me.Role == qbitflow.RoleAdmin && me.OnBehalfOf == ""
}

func TestLiveReadOnly(t *testing.T) {
	c := liveClient(t)
	ctx := liveContext(t)

	me, err := c.Me(ctx)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.Space == nil || me.Space.UUID == "" {
		t.Fatalf("Me: no space in %+v", me)
	}
	t.Logf("key: role %s, test mode %v", me.Role, me.Space.Test)

	t.Run("Products.List", func(t *testing.T) {
		if _, err := c.Products.List(ctx, &qbitflow.ProductListParams{IncludeHidden: true}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Customers", func(t *testing.T) {
		page, err := c.Customers.List(ctx, &qbitflow.CustomerListParams{Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) > 2 {
			t.Errorf("%d items for limit 2", len(page.Items))
		}
		n := 0
		for cust, err := range c.Customers.Iterate(ctx, &qbitflow.CustomerListParams{Limit: 2}) {
			if err != nil {
				t.Fatal(err)
			}
			if cust.UUID == "" {
				t.Error("customer without uuid")
			}
			if n++; n == 3 { // the first page and one more row at most
				break
			}
		}
	})
	t.Run("Payments", func(t *testing.T) {
		if _, err := c.Payments.List(ctx, &qbitflow.PaymentListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
		since := time.Now().Add(-30 * 24 * time.Hour)
		if _, err := c.Payments.ListCombined(ctx, &qbitflow.CombinedPaymentListParams{Limit: 5, CreatedAfter: &since}); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Failures.List(ctx, &qbitflow.FailureListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Subscriptions.List", func(t *testing.T) {
		if _, err := c.Subscriptions.List(ctx, &qbitflow.SubscriptionListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Refunds", func(t *testing.T) {
		if _, err := c.Refunds.List(ctx, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Refunds.ListInactive(ctx, &qbitflow.RefundListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Members", func(t *testing.T) {
		if !isOrganizationKey(me) {
			t.Skip("organization key only")
		}
		if _, err := c.Members.List(ctx, &qbitflow.MemberListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Members.ListHeldFunds(ctx); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Invitations.List", func(t *testing.T) {
		if !isOrganizationKey(me) {
			t.Skip("organization key only")
		}
		if _, err := c.Invitations.List(ctx, &qbitflow.InvitationListParams{Limit: 5}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Wallets", func(t *testing.T) {
		if _, err := c.Wallets.List(ctx, &qbitflow.WalletListParams{WithBalances: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Wallets.ListSupportedCurrencies(ctx, nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Currencies", func(t *testing.T) {
		currencies, err := c.Currencies.ListAvailable(ctx, &qbitflow.CurrencyListParams{Test: me.Space.Test})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Currencies.ListMain(ctx, nil); err != nil {
			t.Fatal(err)
		}
		if len(currencies) > 0 {
			cur, err := c.Currencies.Get(ctx, currencies[0].ID)
			if err != nil || cur.ID != currencies[0].ID {
				t.Fatalf("Get(%d): %+v %v", currencies[0].ID, cur, err)
			}
		}
	})
	t.Run("Accounting", func(t *testing.T) {
		to := time.Now().UTC()
		from := to.AddDate(0, 0, -30)
		events, err := c.Accounting.ExportJSON(ctx, from.Format(time.DateOnly), to.Format(time.DateOnly))
		if err != nil {
			t.Fatal(err)
		}
		csv, err := c.Accounting.ExportCSV(ctx, from.Format(time.DateOnly), to.Format(time.DateOnly))
		if err != nil {
			t.Fatal(err)
		}
		if csv == "" {
			t.Error("empty CSV (a header line is expected)")
		}
		t.Logf("%d accounting events", len(events))
	})
	t.Run("Webhooks", func(t *testing.T) {
		if _, err := c.Webhooks.Endpoints.List(ctx); err != nil {
			t.Fatal(err)
		}
		page, err := c.Webhooks.Events.List(ctx, &qbitflow.EventListParams{Limit: 5})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) > 0 {
			if _, err := c.Webhooks.Events.Get(ctx, page.Items[0].ID); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("CheckoutSessions.GetStatus unknown", func(t *testing.T) {
		_, err := c.CheckoutSessions.GetStatus(ctx, "pay@019eca82-5680-7b00-8000-00000000dead")
		if !errors.As(err, new(*qbitflow.NotFoundError)) {
			t.Fatalf("want *NotFoundError, got %T %v", err, err)
		}
	})
}

// liveWriter is the client for the write group: it needs QBITFLOW_LIVE_WRITES=1 and a
// test-mode key, unless QBITFLOW_ALLOW_LIVE_MODE_WRITES=1 explicitly allows a live-mode one.
func liveWriter(t *testing.T) (*qbitflow.Client, context.Context) {
	t.Helper()
	if os.Getenv("QBITFLOW_LIVE_WRITES") != "1" {
		t.Skip("QBITFLOW_LIVE_WRITES=1 not set: write checks skipped")
	}
	c := liveClient(t)
	ctx := liveContext(t)
	me, err := c.Me(ctx)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if (me.Space == nil || !me.Space.Test) && os.Getenv("QBITFLOW_ALLOW_LIVE_MODE_WRITES") != "1" {
		t.Skip("write checks run on a test-mode key only (QBITFLOW_ALLOW_LIVE_MODE_WRITES=1 allows a live-mode key)")
	}
	return c, ctx
}

func suffix() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

// cleanup runs f at the end of the test with a fresh context; a 404 (already gone) is fine.
func cleanup(t *testing.T, what string, f func(ctx context.Context) error) {
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := f(ctx); err != nil && !errors.As(err, new(*qbitflow.NotFoundError)) {
			t.Errorf("cleanup %s: %v", what, err)
		}
	})
}

func TestLiveWritesProduct(t *testing.T) {
	c, ctx := liveWriter(t)
	ref := "sdk-go-test-" + suffix()
	p, err := c.Products.Create(ctx, &qbitflow.CreateProductParams{Name: "SDK Go test", Price: 1, Reference: ref})
	if err != nil {
		t.Fatal(err)
	}
	cleanup(t, "product", func(ctx context.Context) error { return c.Products.Delete(ctx, p.UUID) })

	got, err := c.Products.Get(ctx, p.UUID)
	if err != nil || got.Reference != ref {
		t.Fatalf("Get: %+v %v", got, err)
	}
	if byRef, err := c.Products.GetByReference(ctx, ref); err != nil || byRef.UUID != p.UUID {
		t.Fatalf("GetByReference: %+v %v", byRef, err)
	}
	price := 2.0
	updated, err := c.Products.Update(ctx, p.UUID, &qbitflow.UpdateProductParams{Price: &price, Description: ptr("Updated by the Go SDK")})
	if err != nil || updated.Price != 2 || updated.Description == "" {
		t.Fatalf("Update: %+v %v", updated, err)
	}
	if err := c.Products.Delete(ctx, p.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Products.Get(ctx, p.UUID); !errors.As(err, new(*qbitflow.NotFoundError)) {
		t.Errorf("Get after Delete: %v", err)
	}
}

func TestLiveWritesCustomer(t *testing.T) {
	c, ctx := liveWriter(t)
	s := suffix()
	cust, err := c.Customers.Create(ctx, &qbitflow.CreateCustomerParams{
		Name: "Ada", LastName: "Test", Email: "sdk-go-test-" + s + "@example.com", PhoneNumber: "+33 6 12 34 56 78", Reference: "sdk-go-" + s,
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanup(t, "customer", func(ctx context.Context) error { return c.Customers.Delete(ctx, cust.UUID) })
	if cust.PhoneNumber == "" {
		t.Errorf("created without its phone: %+v", cust)
	}

	updated, err := c.Customers.Update(ctx, cust.UUID, &qbitflow.UpdateCustomerParams{PhoneNumber: ptr("")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.PhoneNumber != "" || updated.Email != cust.Email {
		t.Errorf("Update (clear phone): %+v", updated)
	}
	if err := c.Customers.Delete(ctx, cust.UUID); err != nil {
		t.Fatal(err)
	}
}

func TestLiveWritesCheckout(t *testing.T) {
	c, ctx := liveWriter(t)
	session, err := c.CheckoutSessions.CreatePayment(ctx, &qbitflow.CreatePaymentSessionParams{
		ProductName: "SDK Go test", Price: 1, Reference: "sdk-go-order-" + suffix(),
		SuccessURL: "https://example.com/ok?id={{UUID}}", CancelURL: "https://example.com/cancel",
	})
	if err != nil {
		var nr *qbitflow.ConflictError
		if errors.As(err, &nr) && nr.Code == "merchant_not_ready" {
			t.Skip("the space accepts no currency (merchant_not_ready)")
		}
		t.Fatal(err)
	}
	cleanup(t, "checkout session", func(ctx context.Context) error {
		_, err := c.CheckoutSessions.Expire(ctx, session.UUID)
		var ce *qbitflow.ConflictError
		if errors.As(err, &ce) {
			return nil // already final
		}
		return err
	})
	if session.Link == "" || session.ExpiresAt.IsZero() {
		t.Errorf("session %+v", session)
	}

	status, err := c.CheckoutSessions.GetStatus(ctx, session.UUID)
	if err != nil || status.Status != qbitflow.CheckoutSessionStatusCreated {
		t.Fatalf("GetStatus: %+v %v", status, err)
	}
	expired, err := c.CheckoutSessions.Expire(ctx, session.UUID)
	if err != nil || expired.Status != qbitflow.CheckoutSessionStatusExpired {
		t.Fatalf("Expire: %+v %v", expired, err)
	}

	// H3: an expired session is final, so WaitForCompletion answers at its first poll.
	waited, err := c.CheckoutSessions.WaitForCompletion(ctx, session.UUID, &qbitflow.WaitOptions{Timeout: 10 * time.Second, Interval: time.Second})
	if err != nil || waited.Status != qbitflow.CheckoutSessionStatusExpired {
		t.Fatalf("WaitForCompletion: %+v %v", waited, err)
	}
}

// TestLiveWritesCheckoutFees opens a payment checkout with fees (the catalog's feesCheckout
// values: a line and the processing fee), reads its status and expires it. The session's fees
// and amount are only readable from the checkout page's route (not the SDK's), so the check is
// the round trip; in test mode it also checks the $5 cap counts the fees.
func TestLiveWritesCheckoutFees(t *testing.T) {
	c, ctx := liveWriter(t)
	params := &qbitflow.CreatePaymentSessionParams{
		ProductName: "T-shirt", Description: "Blue, size M", Price: 3.99, Reference: "order-1044-" + suffix(),
		SuccessURL: "https://shop.example.com/orders/success?uuid=" + qbitflow.PlaceholderUUID,
		CancelURL:  "https://shop.example.com/orders/cancel",
		Fees: &qbitflow.CheckoutFees{
			Items:         []qbitflow.FeeItem{{Label: "Shipping", Description: "Standard, 3 to 5 days", AmountUSD: 0.75}},
			ProcessingFee: new(true),
		},
	}
	session, err := c.CheckoutSessions.CreatePayment(ctx, params)
	if err != nil {
		var nr *qbitflow.ConflictError
		if errors.As(err, &nr) && nr.Code == "merchant_not_ready" {
			t.Skip("the space accepts no currency (merchant_not_ready)")
		}
		t.Fatal(err)
	}
	expire := func(uuid string) {
		cleanup(t, "checkout session "+uuid, func(ctx context.Context) error {
			_, err := c.CheckoutSessions.Expire(ctx, uuid)
			var ce *qbitflow.ConflictError
			if errors.As(err, &ce) {
				return nil // already final
			}
			return err
		})
	}
	expire(session.UUID)
	if session.Link == "" || session.ExpiresAt.IsZero() {
		t.Errorf("session %+v", session)
	}
	status, err := c.CheckoutSessions.GetStatus(ctx, session.UUID)
	if err != nil || status.Status != qbitflow.CheckoutSessionStatusCreated {
		t.Fatalf("GetStatus: %+v %v", status, err)
	}

	// Test mode caps the amount, fees included, at $5: $3.99 plus a $1.50 line is refused.
	me, err := c.Me(ctx)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.Space != nil && me.Space.Test {
		over := *params
		over.Reference = "order-1044-cap-" + suffix()
		over.Fees = &qbitflow.CheckoutFees{Items: []qbitflow.FeeItem{{Label: "Shipping", AmountUSD: 1.5}}}
		s, err := c.CheckoutSessions.CreatePayment(ctx, &over)
		if err == nil {
			expire(s.UUID)
			t.Fatalf("a $5.49 checkout was accepted in test mode: %+v", s)
		}
		var ve *qbitflow.ValidationError
		if !errors.As(err, &ve) || ve.StatusCode != 400 {
			t.Fatalf("over the test-mode cap: %v, want a 400 ValidationError", err)
		}
	}
}

func TestLiveWritesWebhookEndpoint(t *testing.T) {
	c, ctx := liveWriter(t)
	created, err := c.Webhooks.Endpoints.Create(ctx, &qbitflow.CreateWebhookEndpointParams{
		URL: "https://example.com/qbitflow-sdk-test", Events: []qbitflow.EventType{qbitflow.EventTypePaymentCompleted},
		Description: "Go SDK test " + suffix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanup(t, "webhook endpoint", func(ctx context.Context) error { return c.Webhooks.Endpoints.Delete(ctx, created.UUID) })
	if created.Secret == "" {
		t.Error("no secret on create")
	}

	// H1/H2: a delivery signed with SignWebhook passes the API's own check (VerifyRemote) and
	// the router answers it 200 after running its handler. Nothing is written.
	body := []byte(`{"createdAt":"` + time.Now().UTC().Format(time.RFC3339) + `","data":{"endpointUuid":"` + created.UUID +
		`","message":"Go SDK router check"},"id":"evt_` + created.UUID + `","test":true,"type":"webhook.test","version":"v2"}`)
	header := qbitflow.SignWebhook(body, created.Secret, time.Now())
	if err := c.Webhooks.VerifyRemote(ctx, created.UUID, body, header); err != nil {
		t.Fatalf("VerifyRemote of a SignWebhook header: %v", err)
	}
	handled := false
	router := qbitflow.NewWebhookRouter(created.Secret).OnWebhookTest(func(_ context.Context, data *qbitflow.WebhookTest, _ *qbitflow.Event) error {
		handled = data.EndpointUUID == created.UUID
		return nil
	})
	if res := router.Handle(ctx, body, header); res.Status != 200 || !handled {
		t.Fatalf("router: %+v (handled %v)", res, handled)
	}

	got, err := c.Webhooks.Endpoints.Get(ctx, created.UUID)
	if err != nil || got.URL != created.URL {
		t.Fatalf("Get: %+v %v", got, err)
	}
	disabled := false
	updated, err := c.Webhooks.Endpoints.Update(ctx, created.UUID, &qbitflow.UpdateWebhookEndpointParams{Description: ptr(""), Enabled: &disabled})
	if err != nil || updated.Description != "" || updated.DisabledAt == nil {
		t.Fatalf("Update: %+v %v", updated, err)
	}
	if err := c.Webhooks.Endpoints.Delete(ctx, created.UUID); err != nil {
		t.Fatal(err)
	}
}

func ptr[T any](v T) *T { return &v }
