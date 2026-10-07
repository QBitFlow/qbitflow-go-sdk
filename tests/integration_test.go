package tests

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// Integration tests run against a real QBitFlow API.
//
//   - Neither QBITFLOW_API_KEY nor QBITFLOW_BASE_URL set: every integration test skips and
//     the offline suite still runs.
//   - QBITFLOW_API_KEY set without QBITFLOW_BASE_URL: every integration test FAILS. The
//     base URL is never defaulted, so a key can never be sent to an unintended server.
//   - Both set: the tests run against QBITFLOW_BASE_URL.
//
// Every test creates the records it needs (and deletes them afterwards), so any test can
// run on its own with -run. An unexpected error is a failure, never a skip.

var (
	liveClient *qbitflow.QBitFlow
	liveSetup  string // why the live suite cannot run, when a key is set but misconfigured
)

func TestMain(m *testing.M) {
	apiKey := os.Getenv("QBITFLOW_API_KEY")
	baseURL := os.Getenv("QBITFLOW_BASE_URL")

	switch {
	case apiKey != "" && baseURL == "":
		liveSetup = "QBITFLOW_API_KEY is set but QBITFLOW_BASE_URL is not: set it to the server " +
			"to test against (e.g. `set -a; source ../.local.env; set +a`); it is never defaulted"
		fmt.Fprintln(os.Stderr, liveSetup)
	case apiKey != "":
		liveClient = qbitflow.NewWithConfig(qbf.Config{APIKey: apiKey, BaseURL: baseURL})
	default:
		fmt.Fprintln(os.Stderr, "QBITFLOW_API_KEY is not set - integration tests will be skipped")
	}

	os.Exit(m.Run())
}

// requireLive returns the live client, failing the test when the key is set without a
// base URL and skipping it when no key is configured.
func requireLive(t *testing.T) *qbitflow.QBitFlow {
	t.Helper()
	if liveSetup != "" {
		t.Fatal(liveSetup)
	}
	if liveClient == nil {
		t.Skip("QBITFLOW_API_KEY is not set - skipping integration test")
	}
	return liveClient
}

// uniq returns a suffix that keeps references and emails unique across runs.
func uniq() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano()%1_000_000_000, rand.Intn(10000))
}

// mustNotFound fails unless err is a *qberrors.NotFoundError.
func mustNotFound(t *testing.T, err error) {
	t.Helper()
	var notFound *qberrors.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected *qberrors.NotFoundError, got %T: %v", err, err)
	}
}

func newCustomer(t *testing.T, client *qbitflow.QBitFlow) *qbmodels.Customer {
	t.Helper()
	data := getTestCustomerData()
	data.Reference = new("CRM-" + uniq())
	customer, err := client.Customers.Create(data)
	if err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}
	t.Cleanup(func() { _ = client.Customers.Delete(customer.UUID) })
	return customer
}

func newProduct(t *testing.T, client *qbitflow.QBitFlow) *qbmodels.Product {
	t.Helper()
	product, err := client.Products.Create(getTestProductData())
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	t.Cleanup(func() { _ = client.Products.Delete(product.ID) })
	return product
}

func newUser(t *testing.T, client *qbitflow.QBitFlow) *qbmodels.User {
	t.Helper()
	user, err := client.Users.Create(getTestUserData())
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	t.Cleanup(func() { _ = client.Users.Delete(user.ID) })
	return user
}

// ── Client initialisation ────────────────────────────────────────────────────

func TestClientInitialization(t *testing.T) {
	if qbitflow.New("test-key") == nil {
		t.Fatal("expected client to be created")
	}
	client := qbitflow.NewWithConfig(qbf.Config{APIKey: "test-key", BaseURL: "https://api.qbitflow.app/v1/"})
	if client == nil || client.OnBehalfOf(1) == nil {
		t.Fatal("expected client to be created")
	}
}

// ── Customers ────────────────────────────────────────────────────────────────

func TestCustomer(t *testing.T) {
	client := requireLive(t)

	t.Run("Create and read back", func(t *testing.T) {
		created := newCustomer(t, client)
		if created.UUID == "" || created.OrganizationID == 0 || created.CreatedAt.IsZero() {
			t.Errorf("created customer = %+v", created)
		}

		byUUID, err := client.Customers.Get(created.UUID)
		if err != nil || byUUID.Email != created.Email {
			t.Fatalf("get by uuid: %+v, %v", byUUID, err)
		}
		byEmail, err := client.Customers.GetByEmail(created.Email)
		if err != nil || byEmail.UUID != created.UUID {
			t.Fatalf("get by email: %+v, %v", byEmail, err)
		}
		byRef, err := client.Customers.GetByReference(created.Reference)
		if err != nil || byRef.UUID != created.UUID {
			t.Fatalf("get by reference: %+v, %v", byRef, err)
		}
		// Not set at creation: plain empty strings.
		if byRef.PhoneNumber != "" || byRef.Address != "" {
			t.Errorf("unset optional fields = %q / %q", byRef.PhoneNumber, byRef.Address)
		}
	})

	t.Run("List (paginated)", func(t *testing.T) {
		newCustomer(t, client)
		newCustomer(t, client)

		limit := uint16(1)
		page1, err := client.Customers.GetAll(&limit, nil)
		if err != nil {
			t.Fatalf("failed to get customers: %v", err)
		}
		if len(page1.Items) != 1 || !page1.HasMore() {
			t.Fatalf("page 1 = %d items, hasMore=%v", len(page1.Items), page1.HasMore())
		}
		page2, err := client.Customers.GetAll(&limit, page1.NextCursor)
		if err != nil {
			t.Fatalf("failed to get the next page: %v", err)
		}
		if len(page2.Items) != 1 || page2.Items[0].UUID == page1.Items[0].UUID {
			t.Errorf("page 2 = %+v", page2.Items)
		}
	})

	t.Run("Partial update leaves other fields untouched", func(t *testing.T) {
		created := newCustomer(t, client)

		updated, err := client.Customers.Update(created.UUID, &qbf.UpdateCustomer{
			Name:        new("OnlyNameChanged"),
			LastName:    new(""), // "" = not provided
			PhoneNumber: new("+33 1 23 45 67 89"),
		})
		if err != nil {
			t.Fatalf("failed to update customer: %v", err)
		}
		if updated.Name != "OnlyNameChanged" || updated.LastName != created.LastName ||
			updated.Email != created.Email || updated.PhoneNumber != "+33 1 23 45 67 89" {
			t.Errorf("updated = %+v", updated)
		}

		same, err := client.Customers.Update(created.UUID, &qbf.UpdateCustomer{})
		if err != nil || same.Name != "OnlyNameChanged" {
			t.Errorf("an empty update must be a no-op: %+v, %v", same, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		data := getTestCustomerData()
		created, err := client.Customers.Create(data)
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}
		if err := client.Customers.Delete(created.UUID); err != nil {
			t.Fatalf("failed to delete customer: %v", err)
		}
		_, err = client.Customers.Get(created.UUID)
		mustNotFound(t, err)
	})
}

// ── Users ────────────────────────────────────────────────────────────────────

func TestUsers(t *testing.T) {
	client := requireLive(t)

	t.Run("Create and read back", func(t *testing.T) {
		user := newUser(t, client)
		if user.ID == 0 || user.Role != qbmodels.RoleUser || user.ClaimedAt != nil {
			t.Errorf("created user = %+v", user)
		}

		byID, err := client.Users.GetByID(user.ID)
		if err != nil || byID.Email != user.Email {
			t.Fatalf("get by id: %+v, %v", byID, err)
		}
		byEmail, err := client.Users.GetByEmail(user.Email)
		if err != nil || byEmail.ID != user.ID {
			t.Fatalf("get by email: %+v, %v", byEmail, err)
		}
		all, err := client.Users.GetAll()
		if err != nil {
			t.Fatalf("get all users: %v", err)
		}
		found := false
		for _, u := range all {
			found = found || u.ID == user.ID
		}
		if !found {
			t.Error("the created user is missing from GetAll")
		}
	})

	t.Run("Current user", func(t *testing.T) {
		me, err := client.Users.Get()
		if err != nil || me.ID == 0 {
			t.Fatalf("current user: %+v, %v", me, err)
		}
	})

	t.Run("Partial update", func(t *testing.T) {
		user := newUser(t, client)
		updated, err := client.Users.Update(user.ID, &qbf.UpdateUser{
			Name:               new("UpdatedName"),
			Email:              new(""), // "" = not provided
			OrganizationFeeBps: new(uint16(250)),
		})
		if err != nil {
			t.Fatalf("failed to update user: %v", err)
		}
		if updated.Name != "UpdatedName" || updated.LastName != user.LastName ||
			updated.Email != user.Email || updated.OrganizationFeeBps != 250 {
			t.Errorf("updated = %+v", updated)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		created, err := client.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
		if err := client.Users.Delete(created.ID); err != nil {
			t.Fatalf("failed to delete user: %v", err)
		}
		_, err = client.Users.GetByID(created.ID)
		mustNotFound(t, err)
	})
}

// ── Products ─────────────────────────────────────────────────────────────────

func TestProducts(t *testing.T) {
	client := requireLive(t)

	t.Run("Create and read back", func(t *testing.T) {
		product := newProduct(t, client)
		if product.ID == 0 || !product.IsActive || product.Reference == "" || product.OrganizationID == 0 {
			t.Errorf("created product = %+v", product)
		}

		byID, err := client.Products.Get(product.ID)
		if err != nil || byID.Name != product.Name {
			t.Fatalf("get by id: %+v, %v", byID, err)
		}
		byRef, err := client.Products.GetByReference(product.Reference)
		if err != nil || byRef.ID != product.ID {
			t.Fatalf("get by reference: %+v, %v", byRef, err)
		}
		all, err := client.Products.GetAll()
		if err != nil {
			t.Fatalf("get all products: %v", err)
		}
		found := false
		for _, p := range all {
			found = found || p.ID == product.ID
		}
		if !found {
			t.Error("the created product is missing from GetAll")
		}
	})

	t.Run("A reference containing a slash is escaped but not routable", func(t *testing.T) {
		_, err := client.Products.GetByReference("ORD/" + uniq())
		mustNotFound(t, err)
	})

	t.Run("Partial update leaves other fields untouched", func(t *testing.T) {
		product := newProduct(t, client)
		updated, err := client.Products.Update(product.ID, &qbf.UpdateProduct{Price: new(42.50)})
		if err != nil {
			t.Fatalf("failed to update product: %v", err)
		}
		if updated.Price != 42.50 || updated.Name != product.Name ||
			updated.Description != product.Description || updated.Reference != product.Reference {
			t.Errorf("updated = %+v", updated)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		created, err := client.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}
		if err := client.Products.Delete(created.ID); err != nil {
			t.Fatalf("failed to delete product: %v", err)
		}
		_, err = client.Products.Get(created.ID)
		mustNotFound(t, err)
	})
}

// ── API Keys ─────────────────────────────────────────────────────────────────

func TestAPIKeys(t *testing.T) {
	client := requireLive(t)

	// Creating and deleting API keys is JWT-only on the API and is not exposed by this SDK.
	keys, err := client.ApiKeys.GetAll()
	if err != nil {
		t.Fatalf("failed to get API keys: %v", err)
	}
	for _, k := range keys {
		if k.ID == 0 || k.CreatedAt.IsZero() {
			t.Errorf("API key = %+v", k)
		}
	}

	user := newUser(t, client)
	userKeys, err := client.ApiKeys.GetForUser(user.ID)
	if err != nil {
		t.Fatalf("failed to get API keys for user: %v", err)
	}
	if userKeys == nil || len(userKeys) != 0 {
		t.Errorf("a new user has no keys, got %#v", userKeys)
	}
}

// ── Payment sessions ─────────────────────────────────────────────────────────

func TestPaymentSession(t *testing.T) {
	client := requireLive(t)

	t.Run("By product ID, with a customer", func(t *testing.T) {
		product := newProduct(t, client)
		customer := newCustomer(t, client)

		session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(product.ID),
			SuccessURL:   new("https://example.com/success"),
			CancelURL:    new("https://example.com/cancel"),
			CustomerUUID: new(customer.UUID),
		})
		if err != nil {
			t.Fatalf("failed to create payment session: %v", err)
		}
		if !strings.HasPrefix(session.UUID, "pay@") || session.Link == "" {
			t.Errorf("session = %+v", session)
		}

		retrieved, err := client.Payments.GetSession(session.UUID, nil)
		if err != nil {
			t.Fatalf("failed to retrieve session: %v", err)
		}
		if retrieved.UUID != session.UUID || !retrieved.IsPayment() || retrieved.IsSubscription() ||
			retrieved.ProductID != product.ID || retrieved.CustomerUUID == nil ||
			*retrieved.CustomerUUID != customer.UUID || retrieved.OrganizationID == 0 {
			t.Errorf("retrieved session = %+v", retrieved)
		}

		// Nothing has been paid: the status route reports the session as not found.
		_, err = client.TransactionStatus.GetTransactionStatus(session.UUID, qbmodels.TransactionTypeOneTimePayment)
		mustNotFound(t, err)
	})

	t.Run("Inline product, by your own references, empty optionals omitted", func(t *testing.T) {
		ref := "order-" + uniq()
		session, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName:  new("Test Product"),
			Description:  new("Test Description"),
			Price:        new(9.99),
			Reference:    new(ref),
			CustomerUUID: new(""), // not provided
			SuccessURL:   new(""),
		})
		if err != nil {
			t.Fatalf("failed to create session with inline product: %v", err)
		}
		retrieved, err := client.Payments.GetSession(session.UUID, nil)
		if err != nil {
			t.Fatalf("failed to retrieve session: %v", err)
		}
		if retrieved.Reference != ref || retrieved.Price != 9.99 || retrieved.CustomerUUID != nil {
			t.Errorf("retrieved session = %+v", retrieved)
		}
	})

	t.Run("Payment lists", func(t *testing.T) {
		limit := uint16(5)
		payments, err := client.Payments.GetAllPayments(&limit, nil)
		if err != nil {
			t.Fatalf("failed to list payments: %v", err)
		}
		if payments.Items == nil || len(payments.Items) > 5 {
			t.Errorf("payments page = %d items", len(payments.Items))
		}
		for _, p := range payments.Items {
			if !strings.HasPrefix(p.UUID, "pay@") || p.Currency.ID != p.CurrencyID || p.CreatedAt.IsZero() {
				t.Errorf("payment = %+v", p)
			}
		}

		combined, err := client.Payments.GetAllCombinedPayments(&limit, nil)
		if err != nil {
			t.Fatalf("failed to list combined payments: %v", err)
		}
		for _, c := range combined.Items {
			if (c.Source != "payment" && c.Source != "subscription_history") || c.Currency.ID != c.CurrencyID {
				t.Errorf("combined item = %+v", c)
			}
		}

		if len(payments.Items) > 0 {
			payment, err := client.Payments.GetPayment(payments.Items[0].UUID)
			if err != nil || payment.UUID != payments.Items[0].UUID {
				t.Errorf("get payment: %+v, %v", payment, err)
			}
		}
	})

	t.Run("Unknown identifiers are not found", func(t *testing.T) {
		const unknown = "pay@00000000-0000-0000-0000-000000000000"
		_, err := client.Payments.GetPayment(unknown)
		mustNotFound(t, err)
		_, err = client.Payments.GetPaymentByReference("missing-" + uniq())
		mustNotFound(t, err)
		_, err = client.Payments.GetCustomerForTransaction(unknown)
		mustNotFound(t, err)
	})
}

// ── Subscriptions ────────────────────────────────────────────────────────────

func TestSubscriptionSession(t *testing.T) {
	client := requireLive(t)

	t.Run("Create with a trial and read back", func(t *testing.T) {
		product := newProduct(t, client)

		session, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID:   new(product.ID),
			Frequency:   qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			TrialPeriod: &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays},
			MinPeriods:  new(uint32(0)), // no minimum: left out
			SuccessURL:  new("https://example.com/success"),
		})
		if err != nil {
			t.Fatalf("failed to create subscription session: %v", err)
		}
		if !strings.HasPrefix(session.UUID, "sub@") {
			t.Errorf("session = %+v", session)
		}

		retrieved, err := client.Subscriptions.GetSession(session.UUID, nil)
		if err != nil {
			t.Fatalf("failed to retrieve subscription session: %v", err)
		}
		if !retrieved.IsSubscription() || retrieved.IsPayment() || retrieved.Frequency == 0 || retrieved.TrialPeriod == 0 {
			t.Errorf("retrieved session = %+v", retrieved)
		}
	})

	t.Run("Inline product with a zero trial (accepted by the API)", func(t *testing.T) {
		_, err := client.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductName: new("Monthly Plan"),
			Description: new("Monthly subscription"),
			Price:       new(9.99),
			Frequency:   qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			TrialPeriod: &qbmodels.Duration{Value: 0, Unit: qbmodels.DurationUnitDays},
			Reference:   new("sub-" + uniq()),
		})
		if err != nil {
			t.Fatalf("failed to create subscription session: %v", err)
		}
	})

	t.Run("Unknown subscriptions", func(t *testing.T) {
		const unknown = "sub@00000000-0000-0000-0000-000000000000"
		_, err := client.Subscriptions.GetSubscription(unknown)
		mustNotFound(t, err)
		_, err = client.Subscriptions.GetSubscriptionByReference("missing-" + uniq())
		mustNotFound(t, err)

		// The public history route answers an empty list, not a 404.
		history, err := client.Subscriptions.GetPaymentHistory(unknown)
		if err != nil || history == nil || len(history) != 0 {
			t.Errorf("history = %#v, %v", history, err)
		}
	})
}

// ── Refunds ──────────────────────────────────────────────────────────────────

func TestRefunds(t *testing.T) {
	client := requireLive(t)

	_, err := client.Refunds.GetByTransactionUUID("pay@00000000-0000-0000-0000-000000000000")
	mustNotFound(t, err)

	refunds, err := client.Refunds.GetAll()
	if err != nil {
		t.Fatalf("failed to get refunds: %v", err)
	}
	for _, r := range refunds {
		if r.RespondedAt != nil || !strings.HasPrefix(r.UUID, "refund@") {
			t.Errorf("an active refund must await a response: %+v", r)
		}
	}

	limit := uint16(5)
	inactive, err := client.Refunds.GetAllInactive(&limit, nil)
	if err != nil {
		t.Fatalf("failed to get inactive refunds: %v", err)
	}
	for _, r := range inactive.Items {
		if r.RespondedAt == nil {
			t.Errorf("a handled refund must have RespondedAt: %+v", r)
		}
	}
}

// ── Accounting ───────────────────────────────────────────────────────────────

func TestAccounting(t *testing.T) {
	client := requireLive(t)

	to := time.Now().UTC()
	params := qbf.AccountingExportParams{From: to.AddDate(0, -1, 0).Format("2006-01-02"), To: to.Format("2006-01-02")}

	events, err := client.Accounting.Export(params)
	if err != nil {
		t.Fatalf("failed to export accounting data: %v", err)
	}
	if events == nil {
		t.Error("the export must be a (possibly empty) list")
	}

	csv, err := client.Accounting.ExportCSV(params)
	if err != nil {
		t.Fatalf("failed to export accounting CSV: %v", err)
	}
	if !strings.HasPrefix(csv, "paymentId,") {
		t.Errorf("CSV must start with the header row, got %q", csv[:min(len(csv), 40)])
	}
}

// ── Claims ───────────────────────────────────────────────────────────────────

func TestClaims(t *testing.T) {
	client := requireLive(t)

	funds, err := client.Claims.GetClaimFunds()
	if err != nil || funds == nil {
		t.Fatalf("failed to get claim funds: %#v, %v", funds, err)
	}

	user := newUser(t, client)
	created, err := client.Claims.CreateClaimRequest(user.ID)
	if err != nil || created.Link == "" {
		t.Fatalf("failed to create claim request: %+v, %v", created, err)
	}
	existing, err := client.Claims.GetClaimRequestByUserID(user.ID)
	if err != nil || existing.Link == "" {
		t.Fatalf("failed to get claim request: %+v, %v", existing, err)
	}

	// The trigger exists only in test mode: a live-scope key gets the API's 400.
	resp, err := client.Claims.TriggerTestClaimFunds(user.ID)
	var validationErr *qberrors.ValidationError
	switch {
	case err == nil:
		if resp.Message == "" {
			t.Error("expected a message")
		}
	case errors.As(err, &validationErr) && validationErr.StatusCode == 400:
		// live-scope key
	default:
		t.Fatalf("unexpected error from the test trigger: %T: %v", err, err)
	}
}

// ── Currencies, webhooks, on-behalf-of ───────────────────────────────────────

func TestCurrencies(t *testing.T) {
	client := requireLive(t)

	available, err := client.Currencies.GetAllAvailable(false)
	if err != nil {
		t.Fatalf("failed to get currencies: %v", err)
	}
	mains, err := client.Currencies.GetAllMain(false)
	if err != nil {
		t.Fatalf("failed to get main currencies: %v", err)
	}
	if len(mains) > len(available) {
		t.Errorf("main currencies (%d) must be a subset of the available ones (%d)", len(mains), len(available))
	}
	for _, c := range mains {
		if c.MainCurrency != nil || c.Test {
			t.Errorf("main live currency = %+v", c)
		}
	}
}

func TestWebhookVerifyRejectsAForgedSignature(t *testing.T) {
	client := requireLive(t)

	ok, err := client.Webhooks.Verify([]byte(`{"uuid":"pay@1","txType":"payment"}`),
		"sha256=0000000000000000000000000000000000000000000000000000000000000000",
		fmt.Sprint(time.Now().Unix()))
	if err != nil || ok {
		t.Fatalf("a forged signature must yield (false, nil), got (%v, %v)", ok, err)
	}
}

func TestOnBehalfOfUser(t *testing.T) {
	client := requireLive(t)
	user := newUser(t, client)

	scoped := client.OnBehalfOf(user.ID)
	me, err := scoped.Users.Get()
	if err != nil || me.ID != user.ID {
		t.Fatalf("GET /user/ on behalf of %d: %+v, %v", user.ID, me, err)
	}
	products, err := scoped.Products.GetAll()
	if err != nil || products == nil || len(products) != 0 {
		t.Errorf("a new user has no products, got %#v, %v", products, err)
	}

	// The organization-level client still sees organization-level data.
	orgMe, err := client.Users.Get()
	if err != nil || orgMe.ID == user.ID {
		t.Errorf("the base client must be unaffected: %+v, %v", orgMe, err)
	}
}

// ── Model constant tests ─────────────────────────────────────────────────────

func TestRefundStatusValues(t *testing.T) {
	want := map[qbmodels.RefundStatus]string{
		qbmodels.RefundStatusPending:  "pending",
		qbmodels.RefundStatusApproved: "approved",
		qbmodels.RefundStatusRefused:  "refused",
		qbmodels.RefundStatusFailed:   "failed",
	}
	for status, expected := range want {
		if string(status) != expected {
			t.Errorf("wire value = %q, want %q", string(status), expected)
		}
	}
}

func TestUserRoleWireValues(t *testing.T) {
	want := map[qbmodels.UserRole]string{
		qbmodels.RoleHandle: "handle",
		qbmodels.RoleUser:   "user",
		qbmodels.RoleAdmin:  "admin",
		qbmodels.RoleOwner:  "owner",
	}
	for role, expected := range want {
		if string(role) != expected {
			t.Errorf("wire value = %q, want %q", string(role), expected)
		}
	}
}

// ── Benchmarks ───────────────────────────────────────────────────────────────

func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qbitflow.New("test-api-key")
	}
}
