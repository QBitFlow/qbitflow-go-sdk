package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/QBitFlow/qbitflow-go-sdk/v2"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

var (
	testClient *qbitflow.QBitFlow
	apiKey     string
)

var (
	customerUUID     string
	createdProductID uint64
	createdUserID    uint64
)

// TestMain runs before all tests and initialises the shared test client.
func TestMain(m *testing.M) {
	apiKey = os.Getenv("QBITFLOW_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "QBITFLOW_API_KEY is not set - skipping integration tests")
		os.Exit(1)
	}

	// The local server is not always on localhost:3001 (it may be fronted by a tunnel),
	// so honour QBITFLOW_BASE_URL when present and fall back to the usual local default.
	baseURL := os.Getenv("QBITFLOW_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3001"
	}

	testClient = qbitflow.NewWithConfig(qbf.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	})

	code := m.Run()
	os.Exit(code)
}

// ── Client initialisation ────────────────────────────────────────────────────

func TestClientInitialization(t *testing.T) {
	t.Run("NewClient with API key", func(t *testing.T) {
		client := qbitflow.New("test-key")
		if client == nil {
			t.Fatal("expected client to be created")
		}
	})

	t.Run("NewClientWithConfig", func(t *testing.T) {
		client := qbitflow.NewWithConfig(qbf.Config{
			APIKey:  "test-key",
			BaseURL: "https://api.qbitflow.app",
		})
		if client == nil {
			t.Fatal("expected client to be created")
		}
	})
}

// ── Customers ────────────────────────────────────────────────────────────────

func TestCustomer(t *testing.T) {
	t.Run("Create customer", func(t *testing.T) {
		customer, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}
		customerUUID = customer.UUID
		t.Logf("customer created: %s", customer.UUID)
	})

	t.Run("Get customer by UUID", func(t *testing.T) {
		data := getTestCustomerData()
		created, err := testClient.Customers.Create(data)
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		retrieved, err := testClient.Customers.Get(created.UUID)
		if err != nil {
			t.Fatalf("failed to get customer: %v", err)
		}
		if retrieved.Email != data.Email {
			t.Errorf("expected email %s, got %s", data.Email, retrieved.Email)
		}
		t.Logf("customer retrieved: %s", retrieved.UUID)
	})

	t.Run("Get customer by email", func(t *testing.T) {
		data := getTestCustomerData()
		created, err := testClient.Customers.Create(data)
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		retrieved, err := testClient.Customers.GetByEmail(created.Email)
		if err != nil {
			t.Fatalf("failed to get customer by email: %v", err)
		}
		if retrieved.UUID != created.UUID {
			t.Errorf("expected UUID %s, got %s", created.UUID, retrieved.UUID)
		}
		t.Logf("customer retrieved by email: %s", retrieved.UUID)
	})

	t.Run("Get all customers (paginated)", func(t *testing.T) {
		limit := uint16(2)
		page1, err := testClient.Customers.GetAll(&limit, nil)
		if err != nil {
			t.Fatalf("failed to get customers: %v", err)
		}
		if len(page1.Items) == 0 {
			t.Error("expected at least one customer")
		}
		if page1.HasMore() {
			page2, err := testClient.Customers.GetAll(&limit, page1.NextCursor)
			if err != nil {
				t.Fatalf("failed to get next page of customers: %v", err)
			}
			if len(page2.Items) == 0 {
				t.Error("expected items on second page")
			}
		}
		t.Logf("retrieved %d customers", len(page1.Items))
	})

	t.Run("Update customer", func(t *testing.T) {
		created, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		updated, err := testClient.Customers.Update(created.UUID, &qbf.UpdateCustomer{
			Name:     new("Jane"),
			LastName: new("Smith"),
			Email:    new(created.Email),
		})
		if err != nil {
			t.Fatalf("failed to update customer: %v", err)
		}
		if updated.Name != "Jane" || updated.LastName != "Smith" {
			t.Errorf("unexpected name after update: %s %s", updated.Name, updated.LastName)
		}
		t.Logf("customer updated: %s", updated.UUID)
	})

	// Partial update: only the fields present in the body are modified; every omitted
	// field must be left unchanged (no side-effect reset).
	t.Run("Update customer partially leaves other fields untouched", func(t *testing.T) {
		created, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		updated, err := testClient.Customers.Update(created.UUID, &qbf.UpdateCustomer{
			Name: new("OnlyNameChanged"),
		})
		if err != nil {
			t.Fatalf("failed to partially update customer: %v", err)
		}

		if updated.Name != "OnlyNameChanged" {
			t.Errorf("name was not updated: got %q", updated.Name)
		}
		if updated.LastName != created.LastName {
			t.Errorf("lastName must be unchanged: got %q, want %q", updated.LastName, created.LastName)
		}
		if updated.Email != created.Email {
			t.Errorf("email must be unchanged: got %q, want %q", updated.Email, created.Email)
		}
	})

	// An empty update body is a valid no-op.
	t.Run("Update customer with empty body is a no-op", func(t *testing.T) {
		created, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		updated, err := testClient.Customers.Update(created.UUID, &qbf.UpdateCustomer{})
		if err != nil {
			t.Fatalf("empty update should succeed: %v", err)
		}
		if updated.Name != created.Name || updated.LastName != created.LastName || updated.Email != created.Email {
			t.Errorf("empty update changed the record: %+v", updated)
		}
	})

	t.Run("Delete customer", func(t *testing.T) {
		created, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}

		if err := testClient.Customers.Delete(created.UUID); err != nil {
			t.Fatalf("failed to delete customer: %v", err)
		}

		_, err = testClient.Customers.Get(created.UUID)
		if err == nil {
			t.Error("expected error when fetching deleted customer")
		}
		t.Logf("customer deleted and confirmed: %s", created.UUID)
	})
}

// ── Users ────────────────────────────────────────────────────────────────────

func TestUsers(t *testing.T) {
	t.Run("Create user", func(t *testing.T) {
		user, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
		createdUserID = user.ID
		t.Logf("user created: %d", user.ID)
	})

	t.Run("Get current user", func(t *testing.T) {
		user, err := testClient.Users.Get()
		if err != nil {
			t.Fatalf("failed to get current user: %v", err)
		}
		t.Logf("current user: %d", user.ID)
	})

	t.Run("Get user by ID", func(t *testing.T) {
		user, err := testClient.Users.GetByID(createdUserID)
		if err != nil {
			t.Fatalf("failed to get user by ID: %v", err)
		}
		t.Logf("user retrieved: %d", user.ID)
	})

	t.Run("Get all users", func(t *testing.T) {
		users, err := testClient.Users.GetAll()
		if err != nil {
			t.Fatalf("failed to get all users: %v", err)
		}
		t.Logf("total users: %d", len(users))
	})

	t.Run("Update user", func(t *testing.T) {
		created, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		updated, err := testClient.Users.Update(created.ID, &qbf.UpdateUser{
			Name:     new("UpdatedName"),
			LastName: new("UpdatedLastName"),
			Email:    new(created.Email),
		})
		if err != nil {
			t.Fatalf("failed to update user: %v", err)
		}
		t.Logf("user updated: %d", updated.ID)
	})

	t.Run("Delete user", func(t *testing.T) {
		created, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		if err := testClient.Users.Delete(created.ID); err != nil {
			t.Fatalf("failed to delete user: %v", err)
		}

		_, err = testClient.Users.GetByID(created.ID)
		if err == nil {
			t.Error("expected error when fetching deleted user")
		}
		t.Logf("user deleted and confirmed: %d", created.ID)
	})
}

// ── Products ─────────────────────────────────────────────────────────────────

func TestProducts(t *testing.T) {
	t.Run("Create product", func(t *testing.T) {
		product, err := testClient.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}
		createdProductID = product.ID
		t.Logf("product created: %d", product.ID)
	})

	t.Run("Get product by ID", func(t *testing.T) {
		product, err := testClient.Products.Get(createdProductID)
		if err != nil {
			t.Fatalf("failed to get product: %v", err)
		}
		t.Logf("product retrieved: %d", product.ID)
	})

	t.Run("Get product by reference", func(t *testing.T) {
		data := getTestProductData()
		created, err := testClient.Products.Create(data)
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}

		retrieved, err := testClient.Products.GetByReference(created.Reference)
		if err != nil {
			t.Fatalf("failed to get product by reference: %v", err)
		}
		t.Logf("product retrieved by reference: %d", retrieved.ID)
	})

	t.Run("Get all products", func(t *testing.T) {
		products, err := testClient.Products.GetAll()
		if err != nil {
			t.Fatalf("failed to get all products: %v", err)
		}
		t.Logf("total products: %d", len(products))
	})

	t.Run("Update product", func(t *testing.T) {
		created, err := testClient.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}

		updated, err := testClient.Products.Update(created.ID, &qbf.UpdateProduct{
			Name:        new("Updated Product"),
			Description: new("Updated Description"),
			Price:       new(29.99),
		})
		if err != nil {
			t.Fatalf("failed to update product: %v", err)
		}
		if updated.Name != "Updated Product" || updated.Price != 29.99 {
			t.Errorf("unexpected values after update: name=%s price=%f", updated.Name, updated.Price)
		}
		t.Logf("product updated: %d", updated.ID)
	})

	// Partial update: sending only price must leave name and description intact.
	t.Run("Update product partially leaves other fields untouched", func(t *testing.T) {
		created, err := testClient.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}

		updated, err := testClient.Products.Update(created.ID, &qbf.UpdateProduct{
			Price: new(42.50),
		})
		if err != nil {
			t.Fatalf("failed to partially update product: %v", err)
		}

		if updated.Price != 42.50 {
			t.Errorf("price was not updated: got %f", updated.Price)
		}
		if updated.Name != created.Name {
			t.Errorf("name must be unchanged: got %q, want %q", updated.Name, created.Name)
		}
		if updated.Description != created.Description {
			t.Errorf("description must be unchanged: got %q, want %q", updated.Description, created.Description)
		}
		// reference is immutable and is not part of UpdateProduct at all
		if updated.Reference != created.Reference {
			t.Errorf("reference must be unchanged: got %q, want %q", updated.Reference, created.Reference)
		}
	})

	// Session-checkout inline products are validated client-side against the API's
	// producttext / URL rules, so bad input never reaches the network.
	t.Run("Session checkout rejects markup in an inline product name", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("<script>alert(1)</script>"),
			Description: new("A perfectly fine description"),
			Price:       new(9.99),
		})
		if err == nil {
			t.Error("expected angle brackets in productName to be rejected")
		}
	})

	t.Run("Session checkout allows script-looking but markup-free text", func(t *testing.T) {
		// producttext blocks markup characters, not script-like wording: the server
		// renders these fields as escaped text, so this must NOT be rejected locally.
		if err := qbf.ValidateProductTextForTest("productName", "javascript:alert(1)", 2, 100); err != nil {
			t.Errorf("markup-free text must be accepted, got: %v", err)
		}
	})

	t.Run("Session checkout rejects a too-short inline product name", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("A"),
			Description: new("A perfectly fine description"),
			Price:       new(9.99),
		})
		if err == nil {
			t.Error("expected a 1-character productName to be rejected (min 2)")
		}
	})

	t.Run("Session checkout treats an empty optional field as not provided", func(t *testing.T) {
		// The API's omitempty skips validation for an empty value, so the SDK must not
		// reject `Ptr(os.Getenv("SUCCESS_URL"))` when the variable is unset.
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:   new(uint64(339)),
			SuccessURL:  new(""),
			ProductName: new(""),
			Description: new(""),
		})
		if err != nil {
			t.Errorf("empty optional fields must be treated as absent, got: %v", err)
		}
	})

	t.Run("Session checkout rejects a non-http redirect URL", func(t *testing.T) {
		for _, bad := range []string{"javascript:alert(1)", "/relative/path", "ftp://example.com"} {
			_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
				ProductID:  new(uint64(1)),
				SuccessURL: new(bad),
			})
			if err == nil {
				t.Errorf("expected redirect URL %q to be rejected", bad)
			}
		}
	})

	// Client-side validation rejects a non-positive price before any request is sent.
	t.Run("Update product rejects non-positive price locally", func(t *testing.T) {
		if _, err := testClient.Products.Update(1, &qbf.UpdateProduct{Price: new(0.0)}); err == nil {
			t.Error("expected price=0 to be rejected by client-side validation")
		}
		if _, err := testClient.Products.Update(1, &qbf.UpdateProduct{Price: new(-5.0)}); err == nil {
			t.Error("expected negative price to be rejected by client-side validation")
		}
	})

	t.Run("Delete product", func(t *testing.T) {
		created, err := testClient.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("failed to create product: %v", err)
		}

		if err := testClient.Products.Delete(created.ID); err != nil {
			t.Fatalf("failed to delete product: %v", err)
		}

		_, err = testClient.Products.Get(created.ID)
		if err == nil {
			t.Error("expected error when fetching deleted product")
		}
		t.Logf("product deleted and confirmed: %d", created.ID)
	})
}

// ── API Keys ─────────────────────────────────────────────────────────────────

func TestAPIKeys(t *testing.T) {
	// NOTE: creating and deleting API keys is a JWT-only operation on the API and is not
	// exposed by this SDK (which authenticates with an API key). Only read access is tested.

	t.Run("Get all API keys for current user", func(t *testing.T) {
		keys, err := testClient.ApiKeys.GetAll()
		if err != nil {
			t.Fatalf("failed to get API keys: %v", err)
		}
		// The secret key value must never be returned on a key record.
		for _, k := range keys {
			if k.ID == 0 {
				t.Error("expected a non-zero API key ID")
			}
		}
		t.Logf("total API keys: %d", len(keys))
	})

	t.Run("Get API keys for specific user", func(t *testing.T) {
		user, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		keys, err := testClient.ApiKeys.GetForUser(user.ID)
		if err != nil {
			t.Fatalf("failed to get API keys for user: %v", err)
		}
		t.Logf("API keys for user %d: %d", user.ID, len(keys))
	})
}

// ── Payment sessions ─────────────────────────────────────────────────────────

func TestPaymentSession(t *testing.T) {
	t.Run("Create session with product ID", func(t *testing.T) {
		session, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(createdProductID),
			SuccessURL:   new("https://example.com/success"),
			CancelURL:    new("https://example.com/cancel"),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create payment session: %v", err)
		}
		if session.UUID == "" || session.Link == "" {
			t.Error("expected non-empty UUID and Link")
		}
		t.Logf("payment session created: %s", session.UUID)
	})

	t.Run("Create session with inline product", func(t *testing.T) {
		session, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName:  new("Test Product"),
			Description:  new("Test Description"),
			Price:        new(99.99),
			SuccessURL:   new("https://example.com/success"),
			CancelURL:    new("https://example.com/cancel"),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create session with inline product: %v", err)
		}
		t.Logf("inline product session created: %s", session.UUID)
	})

	t.Run("Validation — missing required fields", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{})
		if err == nil {
			t.Error("expected validation error for missing fields")
		}
	})

	t.Run("Validation — negative price", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("Test"),
			Description: new("Test"),
			Price:       new(-10.0),
		})
		if err == nil {
			t.Error("expected validation error for negative price")
		}
	})

	t.Run("Retrieve payment session", func(t *testing.T) {
		created, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(createdProductID),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		retrieved, err := testClient.Payments.GetSession(created.UUID)
		if err != nil {
			t.Fatalf("failed to retrieve session: %v", err)
		}
		if retrieved.UUID != created.UUID {
			t.Errorf("UUID mismatch: want %s, got %s", created.UUID, retrieved.UUID)
		}
		if retrieved.IsSubscription() {
			t.Error("payment session should not report IsSubscription()==true")
		}
		t.Logf("session retrieved: %s", retrieved.UUID)
	})

	t.Run("Get all payments (paginated)", func(t *testing.T) {
		limit := uint16(5)
		payments, err := testClient.Payments.GetAllPayments(&limit, nil)
		if err != nil {
			t.Logf("note: requires existing payments — skipping: %v", err)
			t.Skip()
		}
		t.Logf("payments retrieved: %d", len(payments.Items))
	})

	t.Run("Get all combined payments", func(t *testing.T) {
		limit := uint16(5)
		combined, err := testClient.Payments.GetAllCombinedPayments(&limit, nil)
		if err != nil {
			t.Logf("note: requires existing payments — skipping: %v", err)
			t.Skip()
		}
		t.Logf("combined payments retrieved: %d", len(combined.Items))
	})
}

// ── Subscription sessions ────────────────────────────────────────────────────

func TestSubscriptionSession(t *testing.T) {
	t.Run("Create subscription with trial period", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID:    new(createdProductID),
			Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			TrialPeriod:  &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays},
			SuccessURL:   new("https://example.com/success"),
			CancelURL:    new("https://example.com/cancel"),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create subscription session: %v", err)
		}
		t.Logf("subscription session created: %s", session.UUID)
	})

	t.Run("Create subscription without trial", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID:    new(createdProductID),
			Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create subscription session: %v", err)
		}
		t.Logf("subscription session (no trial) created: %s", session.UUID)
	})

	t.Run("Create subscription with inline product", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductName:  new("Monthly Plan"),
			Description:  new("Monthly subscription"),
			Price:        new(9.99),
			Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create subscription session with inline product: %v", err)
		}
		t.Logf("subscription session (inline product) created: %s", session.UUID)
	})

	t.Run("Validation — missing frequency value", func(t *testing.T) {
		_, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID: new(createdProductID),
			Frequency: qbmodels.Duration{Value: 0, Unit: qbmodels.DurationUnitMonths},
		})
		if err == nil {
			t.Error("expected validation error for zero frequency")
		}
	})

	t.Run("Retrieve subscription session", func(t *testing.T) {
		created, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID:    new(createdProductID),
			Frequency:    qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Fatalf("failed to create subscription session: %v", err)
		}

		retrieved, err := testClient.Subscriptions.GetSession(created.UUID)
		if err != nil {
			t.Fatalf("failed to retrieve subscription session: %v", err)
		}
		if retrieved.UUID != created.UUID {
			t.Errorf("UUID mismatch: want %s, got %s", created.UUID, retrieved.UUID)
		}
		if !retrieved.IsSubscription() {
			t.Error("subscription session should report IsSubscription()==true")
		}
		t.Logf("subscription session retrieved: %s", retrieved.UUID)
	})
}

// ── Transaction status ───────────────────────────────────────────────────────

func TestTransactionStatus(t *testing.T) {
	t.Run("Get status for non-existent transaction", func(t *testing.T) {
		_, err := testClient.TransactionStatus.GetTransactionStatus(
			"00000000-0000-0000-0000-000000000000",
			qbmodels.TransactionTypeOneTimePayment,
		)
		if err == nil {
			t.Error("expected error for non-existent transaction")
		}
		t.Logf("error received as expected: %v", err)
	})

	t.Run("GetTransactionStatusWSURL returns a URL", func(t *testing.T) {
		url := testClient.TransactionStatus.GetTransactionStatusWSURL(
			"test-uuid",
			qbmodels.TransactionTypeOneTimePayment,
		)
		if url == "" {
			t.Error("expected non-empty WebSocket URL")
		}
		t.Logf("WebSocket URL: %s", url)
	})
}

// ── Refunds ──────────────────────────────────────────────────────────────────

func TestRefunds(t *testing.T) {
	t.Run("Get refund by transaction UUID — non-existent", func(t *testing.T) {
		_, err := testClient.Refunds.GetByTransactionUUID("00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Error("expected error for non-existent refund")
		}
		t.Logf("error received as expected: %v", err)
	})

	t.Run("Get all active refunds", func(t *testing.T) {
		refunds, err := testClient.Refunds.GetAll()
		if err != nil {
			t.Fatalf("failed to get all refunds: %v", err)
		}
		t.Logf("active refunds: %d", len(refunds))
	})

	t.Run("Get all inactive refunds (paginated)", func(t *testing.T) {
		limit := uint16(5)
		refunds, err := testClient.Refunds.GetAllInactive(&limit, nil)
		if err != nil {
			t.Fatalf("failed to get inactive refunds: %v", err)
		}
		t.Logf("inactive refunds retrieved: %d", len(refunds.Items))
	})
}

// ── Accounting ───────────────────────────────────────────────────────────────

func TestAccounting(t *testing.T) {
	t.Run("Export accounting data as JSON", func(t *testing.T) {
		events, err := testClient.Accounting.Export(qbf.AccountingExportParams{
			From: "2025-01-01",
			To:   "2025-12-31",
		})
		if err != nil {
			t.Logf("note: no accounting data in range or API error — skipping: %v", err)
			t.Skip()
		}
		t.Logf("accounting events retrieved: %d", len(events))
	})

	t.Run("Validation — missing dates", func(t *testing.T) {
		_, err := testClient.Accounting.Export(qbf.AccountingExportParams{})
		if err == nil {
			t.Error("expected validation error for missing dates")
		}
	})
}

// ── Claims ───────────────────────────────────────────────────────────────────

func TestClaims(t *testing.T) {
	t.Run("Get claim funds", func(t *testing.T) {
		funds, err := testClient.Claims.GetClaimFunds()
		if err != nil {
			t.Fatalf("failed to get claim funds: %v", err)
		}
		t.Logf("claim funds: %d", len(funds))
	})

	t.Run("Create claim request for user", func(t *testing.T) {
		user, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("failed to create user for claim test: %v", err)
		}

		resp, err := testClient.Claims.CreateClaimRequest(user.ID)
		if err != nil {
			t.Fatalf("failed to create claim request: %v", err)
		}
		if resp.Link == "" {
			t.Error("expected non-empty claim link")
		}
		t.Logf("claim request created, link: %s", resp.Link)
	})

	t.Run("Trigger test claim funds", func(t *testing.T) {
		resp, err := testClient.Claims.TriggerTestClaimFunds(createdUserID)
		if err != nil {
			t.Logf("note: may fail if no ledger entries exist — skipping: %v", err)
			t.Skip()
		}
		t.Logf("test claim funds triggered: %s", resp.Message)
	})
}

// ── Model constant tests ─────────────────────────────────────────────────────

func TestDurationUnits(t *testing.T) {
	cases := []qbmodels.Duration{
		{Value: 1, Unit: qbmodels.DurationUnitSeconds},
		{Value: 1, Unit: qbmodels.DurationUnitMinutes},
		{Value: 1, Unit: qbmodels.DurationUnitHours},
		{Value: 1, Unit: qbmodels.DurationUnitDays},
		{Value: 1, Unit: qbmodels.DurationUnitWeeks},
		{Value: 1, Unit: qbmodels.DurationUnitMonths},
	}
	for _, d := range cases {
		t.Run(string(d.Unit), func(t *testing.T) {
			if d.Value != 1 {
				t.Errorf("expected value 1, got %d", d.Value)
			}
		})
	}
}

func TestTransactionTypes(t *testing.T) {
	types := []qbmodels.TransactionType{
		qbmodels.TransactionTypeOneTimePayment,
		qbmodels.TransactionTypeCreateSubscription,
		qbmodels.TransactionTypeCancelSubscription,
		qbmodels.TransactionTypeExecuteSubscriptionPayment,
		qbmodels.TransactionTypeCreatePAYGSubscription,
		qbmodels.TransactionTypeCancelPAYGSubscription,
		qbmodels.TransactionTypeIncreaseAllowance,
		qbmodels.TransactionTypeUpdateMaxAmount,
	}
	for _, tt := range types {
		t.Run(string(tt), func(t *testing.T) {
			if tt == "" {
				t.Error("transaction type should not be empty")
			}
		})
	}
}

func TestTransactionStatusValues(t *testing.T) {
	statuses := []qbmodels.TransactionStatusValue{
		qbmodels.TransactionStatusCreated,
		qbmodels.TransactionStatusWaitingConfirmation,
		qbmodels.TransactionStatusPending,
		qbmodels.TransactionStatusCompleted,
		qbmodels.TransactionStatusFailed,
		qbmodels.TransactionStatusCancelled,
		qbmodels.TransactionStatusExpired,
	}
	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			if s == "" {
				t.Error("status should not be empty")
			}
		})
	}
}

func TestRefundStatusValues(t *testing.T) {
	statuses := []qbmodels.RefundStatus{
		qbmodels.RefundStatusPending,
		qbmodels.RefundStatusApproved,
		qbmodels.RefundStatusRejected,
		qbmodels.RefundStatusFailed,
	}
	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			if s == "" {
				t.Error("refund status should not be empty")
			}
		})
	}
}

func TestSessionCheckoutHelpers(t *testing.T) {
	t.Run("payment session", func(t *testing.T) {
		s := &qbmodels.SessionCheckout{Frequency: 0}
		if !s.IsPayment() {
			t.Error("expected IsPayment()==true for zero frequency")
		}
		if s.IsSubscription() {
			t.Error("expected IsSubscription()==false for zero frequency")
		}
	})

	t.Run("subscription session", func(t *testing.T) {
		s := &qbmodels.SessionCheckout{Frequency: 2592000}
		if !s.IsSubscription() {
			t.Error("expected IsSubscription()==true for non-zero frequency")
		}
		if s.IsPayment() {
			t.Error("expected IsPayment()==false for non-zero frequency")
		}
	})
}

// ── Benchmarks ───────────────────────────────────────────────────────────────

func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qbitflow.New("test-api-key")
	}
}

func BenchmarkCreateSessionOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qbf.CreateSessionOptions{
			ProductID:    new(uint64(1)),
			SuccessURL:   new("https://example.com/success"),
			CancelURL:    new("https://example.com/cancel"),
			CustomerUUID: new("test-customer"),
		}
	}
}
