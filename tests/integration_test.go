package tests

import (
	"os"
	"testing"

	"github.com/QBitFlow/qbitflow-go-sdk"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
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

// TestMain runs before all tests
func TestMain(m *testing.M) {
	// Get API key from environment variable
	apiKey = os.Getenv("QBITFLOW_API_KEY")
	if apiKey == "" {
		os.Exit(1)
	}

	// Initialize test client
	testClient = qbitflow.NewWithConfig(qbf.Config{
		APIKey:  apiKey,
		BaseURL: "http://localhost:3001",
	})

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// TestClientInitialization tests client creation
func TestClientInitialization(t *testing.T) {
	t.Run("NewClient with API key", func(t *testing.T) {
		client := qbitflow.New("test-key")
		if client == nil {
			t.Fatal("Expected client to be created")
		}
	})

	t.Run("NewClientWithConfig", func(t *testing.T) {
		client := qbitflow.NewWithConfig(qbf.Config{
			APIKey:  "test-key",
			BaseURL: "https://api.qbitflow.app",
		})
		if client == nil {
			t.Fatal("Expected client to be created")
		}
	})
}

func TestCustomer(t *testing.T) {
	t.Run("Create customer", func(t *testing.T) {
		customer, err := testClient.Customers.Create(getTestCustomerData())
		if err != nil {
			t.Fatalf("Failed to create customer: %v", err)
		}
		t.Logf("Customer created: %s", customer.UUID)

		customerUUID = customer.UUID
	})

	t.Run("Get customer by UUID", func(t *testing.T) {
		customerData := getTestCustomerData()
		createdCustomer, err := testClient.Customers.Create(customerData)
		if err != nil {
			t.Fatalf("Failed to create customer: %v", err)
		}

		retrievedCustomer, err := testClient.Customers.Get(createdCustomer.UUID)
		if err != nil {
			t.Fatalf("Failed to get customer: %v", err)
		}

		if retrievedCustomer.Email != customerData.Email {
			t.Errorf("Expected email %s, got %s", customerData.Email, retrievedCustomer.Email)
		}

		t.Logf("Customer retrieved: %s", retrievedCustomer.UUID)
	})

	t.Run("Get customer by Email", func(t *testing.T) {
		customerData := getTestCustomerData()
		createdCustomer, err := testClient.Customers.Create(customerData)
		if err != nil {
			t.Fatalf("Failed to create customer: %v", err)
		}

		retrievedCustomer, err := testClient.Customers.GetByEmail(createdCustomer.Email)
		if err != nil {
			t.Fatalf("Failed to get customer by email: %v", err)
		}

		if retrievedCustomer.UUID != createdCustomer.UUID {
			t.Errorf("Expected UUID %s, got %s", createdCustomer.UUID, retrievedCustomer.UUID)
		}

		t.Logf("Customer retrieved by email: %s", retrievedCustomer.UUID)
	})

	t.Run("Get all customers", func(t *testing.T) {
		limit := uint16(2)
		customers, err := testClient.Customers.GetAll(&limit, nil)
		if err != nil {
			t.Fatalf("Failed to get all customers: %v", err)
		}

		if len(customers.Items) == 0 {
			t.Error("Expected at least one customer")
		}

		if !customers.HasMore() {
			t.Error("Expected more customers for pagination")
		}

		// Retrieve next page if available
		nextCustomers, err := testClient.Customers.GetAll(&limit, customers.NextCursor)
		if err != nil {
			t.Fatalf("Failed to get next page of customers: %v", err)
		}

		if len(nextCustomers.Items) == 0 {
			t.Error("Expected more customers for pagination")
		}

		t.Logf("Successfully retrieved all customers")
	})

	t.Run("Update customer", func(t *testing.T) {
		customerData := getTestCustomerData()
		createdCustomer, err := testClient.Customers.Create(customerData)
		if err != nil {
			t.Fatalf("Failed to create customer: %v", err)
		}

		updatedData := qbf.UpdateCustomer{
			UUID:     createdCustomer.UUID,
			Name:     "Jane",
			LastName: "Smith",
			Email:    createdCustomer.Email,
		}

		updatedCustomer, err := testClient.Customers.Update(&updatedData)
		if err != nil {
			t.Fatalf("Failed to update customer: %v", err)
		}

		if updatedCustomer.Name != "Jane" || updatedCustomer.LastName != "Smith" {
			t.Errorf("Customer update failed, got name: %s %s", updatedCustomer.Name, updatedCustomer.LastName)
		}

		t.Logf("Customer updated: %s", updatedCustomer.UUID)
	})

	t.Run("Delete customer", func(t *testing.T) {
		customerData := getTestCustomerData()
		createdCustomer, err := testClient.Customers.Create(customerData)
		if err != nil {
			t.Fatalf("Failed to create customer: %v", err)
		}

		err = testClient.Customers.Delete(createdCustomer.UUID)
		if err != nil {
			t.Fatalf("Failed to delete customer: %v", err)
		}

		t.Logf("Customer deleted: %s", createdCustomer.UUID)

		// Try fetching deleted customer
		_, err = testClient.Customers.Get(createdCustomer.UUID)
		if err == nil {
			t.Error("Expected error when fetching deleted customer, got none")
		} else {
			t.Logf("Successfully verified deletion of customer: %v", err)
		}
	})
}

func TestUsers(t *testing.T) {
	t.Run("Create user", func(t *testing.T) {
		user, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		t.Logf("User created: %d", user.ID)

		createdUserID = user.ID
	})

	t.Run("Get current user", func(t *testing.T) {
		user, err := testClient.Users.Get()
		if err != nil {
			t.Fatalf("Failed to get current user: %v", err)
		}
		t.Logf("Current user: %d", user.ID)
	})

	t.Run("Get user by ID", func(t *testing.T) {
		retrievedUser, err := testClient.Users.GetByID(createdUserID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}
		t.Logf("User retrieved by ID: %d", retrievedUser.ID)
	})

	t.Run("Get all users", func(t *testing.T) {
		users, err := testClient.Users.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}
		t.Logf("Total users retrieved: %d", len(users))
	})

	t.Run("Update user", func(t *testing.T) {
		userData := getTestUserData()
		createdUser, err := testClient.Users.Create(userData)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		updatedData := qbf.UpdateUser{
			Name:     "UpdatedName",
			LastName: "UpdatedLastName",
			Email:    createdUser.Email,
		}

		updatedUser, err := testClient.Users.Update(createdUser.ID, &updatedData)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		t.Logf("User updated: %d", updatedUser.ID)
	})

	t.Run("Delete user", func(t *testing.T) {
		userData := getTestUserData()
		createdUser, err := testClient.Users.Create(userData)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		err = testClient.Users.Delete(createdUser.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		t.Logf("User deleted: %d", createdUser.ID)

		// Try fetching deleted user
		_, err = testClient.Users.GetByID(createdUser.ID)
		if err == nil {
			t.Error("Expected error when fetching deleted user, got none")
		} else {
			t.Logf("Successfully verified deletion of user: %v", err)
		}
	})
}

func TestProducts(t *testing.T) {
	t.Run("Create product", func(t *testing.T) {
		product, err := testClient.Products.Create(getTestProductData())
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
		t.Logf("Product created: %d", product.ID)

		createdProductID = product.ID
	})

	t.Run("Get product by ID", func(t *testing.T) {
		retrievedProduct, err := testClient.Products.Get(createdProductID)
		if err != nil {
			t.Fatalf("Failed to get product by ID: %v", err)
		}
		t.Logf("Product retrieved by ID: %d", retrievedProduct.ID)
	})
	t.Run("Get product by Reference", func(t *testing.T) {
		productData := getTestProductData()
		createdProduct, err := testClient.Products.Create(productData)
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}

		retrievedProduct, err := testClient.Products.GetByReference(*createdProduct.Reference)
		if err != nil {
			t.Fatalf("Failed to get product by Reference: %v", err)
		}
		t.Logf("Product retrieved by Reference: %d", retrievedProduct.ID)
	})
	t.Run("Get all products", func(t *testing.T) {
		products, err := testClient.Products.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all products: %v", err)
		}
		t.Logf("Total products retrieved: %d", len(products))
	})

	t.Run("Update product", func(t *testing.T) {
		productData := getTestProductData()
		createdProduct, err := testClient.Products.Create(productData)
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}

		updatedData := qbf.UpdateProduct{
			Name:        "Updated Product",
			Description: "Updated Description",
			Price:       29.99,
		}

		updatedProduct, err := testClient.Products.Update(createdProduct.ID, &updatedData)
		if err != nil {
			t.Fatalf("Failed to update product: %v", err)
		}

		if updatedProduct.Name != "Updated Product" || updatedProduct.Price != 29.99 {
			t.Errorf("Product update failed, got name: %s, price: %f", updatedProduct.Name, updatedProduct.Price)
		} else {
			t.Logf("Product updated: %d", updatedProduct.ID)
		}
	})

	t.Run("Delete product", func(t *testing.T) {
		productData := getTestProductData()
		createdProduct, err := testClient.Products.Create(productData)
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}

		err = testClient.Products.Delete(createdProduct.ID)
		if err != nil {
			t.Fatalf("Failed to delete product: %v", err)
		}

		t.Logf("Product deleted: %d", createdProduct.ID)

		// Try fetching deleted product
		_, err = testClient.Products.Get(createdProduct.ID)
		if err == nil {
			t.Error("Expected error when fetching deleted product, got none")
		} else {
			t.Logf("Successfully verified deletion of product: %v", err)
		}
	})
}

func TestAPIKeys(t *testing.T) {
	t.Run("Create API Key", func(t *testing.T) {
		apiKey, err := testClient.ApiKeys.Create(&qbf.CreateApiKeyDto{
			Name:   "Test API Key",
			UserID: createdUserID,
			Test:   false,
		})
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}

		t.Logf("API Key created: %s", apiKey.Key)
	})

	t.Run("Get all API Keys for current user", func(t *testing.T) {
		apiKeys, err := testClient.ApiKeys.GetAll()
		if err != nil {
			t.Fatalf("Failed to get API keys: %v", err)
		}
		t.Logf("Total API Keys retrieved: %d", len(apiKeys))
	})

	t.Run("Get API Keys for specific user", func(t *testing.T) {
		user, err := testClient.Users.Create(getTestUserData())
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		apiKeys, err := testClient.ApiKeys.GetForUser(user.ID)
		if err != nil {
			t.Fatalf("Failed to get API keys for user: %v", err)
		}
		t.Logf("Total API Keys retrieved for user %d: %d", user.ID, len(apiKeys))
	})

	t.Run("Delete API Key", func(t *testing.T) {
		apiKey, err := testClient.ApiKeys.Create(&qbf.CreateApiKeyDto{
			Name:   "Test API Key",
			UserID: createdUserID,
			Test:   false,
		})
		if err != nil {
			t.Fatalf("Failed to create API key: %v", err)
		}

		err = testClient.ApiKeys.Delete(apiKey.Data.ID)
		if err != nil {
			t.Fatalf("Failed to delete API key: %v", err)
		}

		t.Logf("API Key deleted: %d", apiKey.Data.ID)
	})
}

// TestPaymentSession tests payment session creation
func TestPaymentSession(t *testing.T) {
	t.Run("Create payment session with product ID", func(t *testing.T) {
		session, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(uint64(1)),
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			WebhookURL:   new(string("https://example.com/webhook")),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key and may fail in CI")
			t.Logf("Error: %v", err)
			t.Skip("Skipping due to API requirements")
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		if session.UUID == "" {
			t.Error("Expected session UUID to be set")
		}

		if session.Link == "" {
			t.Error("Expected session link to be set")
		}

		t.Logf("Session created: %s", session.UUID)
	})

	t.Run("Create payment session with custom product", func(t *testing.T) {
		session, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName:  new(string("Test Product")),
			Description:  new(string("Test Description")),
			Price:        new(float64(99.99)),
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		t.Logf("Custom product session created: %s", session.UUID)
	})

	t.Run("Validation error - missing required fields", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			// Missing all required fields
		})

		if err == nil {
			t.Error("Expected validation error")
		}
	})

	t.Run("Validation error - negative price", func(t *testing.T) {
		_, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new(string("Test")),
			Description: new(string("Test")),
			Price:       new(float64(-10.0)), // Negative price
		})

		if err == nil {
			t.Error("Expected validation error for negative price")
		}
	})

	t.Run("Get payment session", func(t *testing.T) {
		session, err := testClient.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:    new(uint64(1)),
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key and may fail in CI")
			t.Errorf("Error: %v", err)
		}

		retrievedSession, err := testClient.Payments.GetSession(session.UUID)
		if err != nil {
			t.Fatalf("Failed to get payment session: %v", err)
		}

		if retrievedSession.UUID != session.UUID {
			t.Errorf("Expected session UUID %s, got %s", session.UUID, retrievedSession.UUID)
		}

		t.Logf("Payment session retrieved: %s", retrievedSession.UUID)
	})

	t.Run("Get all payments", func(t *testing.T) {
		limit := uint16(5)
		payments, err := testClient.Payments.GetAllPayments(&limit, nil)
		if err != nil {
			t.Logf("Note: This test requires a valid API key and existing payments")
			t.Errorf("Error: %v", err)
		}

		if len(payments.Items) == 0 {
			t.Error("Expected at least one payment")
		}

		t.Logf("Total payments retrieved: %d", len(payments.Items))
	})

	t.Run("Get all combined payments", func(t *testing.T) {
		limit := uint16(5)
		payments, err := testClient.Payments.GetAllCombinedPayments(&limit, nil)
		if err != nil {
			t.Logf("Note: This test requires a valid API key and existing payments")
			t.Errorf("Error: %v", err)
		}

		if len(payments.Items) == 0 {
			t.Error("Expected at least one combined payment")
		}

		t.Logf("Total combined payments retrieved: %d", len(payments.Items))
	})
}

// TestSubscriptionSession tests subscription session creation
func TestSubscriptionSession(t *testing.T) {
	t.Run("Create subscription with trial period", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			TrialPeriod: &qbmodels.Duration{
				Value: 7,
				Unit:  qbmodels.DurationUnitDays,
			},
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			WebhookURL:   new(string("https://example.com/webhook")),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key")
			t.Errorf("Error: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		t.Logf("Subscription session created: %s", session.UUID)
	})

	t.Run("Create subscription without trial", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		t.Logf("Subscription session (no trial) created: %s", session.UUID)
	})

	t.Run("Get subscription session", func(t *testing.T) {
		session, err := testClient.Subscriptions.CreateSession(&qbf.CreateSubscriptionSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key")
			t.Errorf("Error: %v", err)
		}

		retrievedSession, err := testClient.Subscriptions.GetSession(session.UUID)
		if err != nil {
			t.Fatalf("Failed to get subscription session: %v", err)
		}

		if retrievedSession.UUID != session.UUID {
			t.Errorf("Expected session UUID %s, got %s", session.UUID, retrievedSession.UUID)
		}

		t.Logf("Subscription session retrieved: %s", retrievedSession.UUID)
	})
}

// TestPAYGSubscription tests pay-as-you-go subscription creation
func TestPAYGSubscription(t *testing.T) {
	t.Run("Create PAYG subscription with free credits", func(t *testing.T) {
		session, err := testClient.PayAsYouGo.CreateSession(&qbf.CreatePAYGSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			FreeCredits:  new(float64(100.0)),
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			WebhookURL:   new(string("https://example.com/webhook")),
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key")
			t.Errorf("Error: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		t.Logf("PAYG session created: %s", session.UUID)
	})

	t.Run("Create basic PAYG subscription", func(t *testing.T) {
		session, err := testClient.PayAsYouGo.CreateSession(&qbf.CreatePAYGSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		if session == nil {
			t.Fatal("Expected session to be created")
		}

		t.Logf("Basic PAYG session created: %s", session.UUID)
	})

	t.Run("Get PAYG subscription session", func(t *testing.T) {
		session, err := testClient.PayAsYouGo.CreateSession(&qbf.CreatePAYGSessionOptions{
			ProductID: createdProductID,
			Frequency: qbmodels.Duration{
				Value: 1,
				Unit:  qbmodels.DurationUnitMonths,
			},
			CustomerUUID: &customerUUID,
		})
		if err != nil {
			t.Logf("Note: This test requires a valid API key")
			t.Errorf("Error: %v", err)
		}

		retrievedSession, err := testClient.PayAsYouGo.GetSession(session.UUID)
		if err != nil {
			t.Fatalf("Failed to get PAYG session: %v", err)
		}

		if retrievedSession.UUID != session.UUID {
			t.Errorf("Expected session UUID %s, got %s", session.UUID, retrievedSession.UUID)
		}

		t.Logf("PAYG session retrieved: %s", retrievedSession.UUID)
	})
}

// TestTransactionStatus tests transaction status checking
func TestTransactionStatus(t *testing.T) {
	t.Run("Get non-existent transaction status", func(t *testing.T) {
		_, err := testClient.TransactionStatus.GetTransactionStatus(
			"non-existent-uuid",
			qbmodels.TransactionTypeOneTimePayment,
		)

		if err == nil {
			t.Error("Expected error for non-existent transaction")
		}

		t.Logf("Error (expected): %v", err)
	})
}

// TestGetPayments tests payment retrieval
func TestGetPayments(t *testing.T) {
	t.Run("Get all payments with pagination", func(t *testing.T) {
		limit := uint16(10)
		payments, err := testClient.Payments.GetAllPayments(&limit, nil)
		if err != nil {
			t.Logf("Note: This test requires a valid API key and existing payments")
			t.Logf("Error: %v", err)
			t.Skip("Skipping due to API requirements")
		}

		if payments == nil {
			t.Fatal("Expected payments result")
		}

		t.Logf("Retrieved %d payments", len(payments.Items))
		t.Logf("Has more: %v", payments.HasMore())
	})

	t.Run("Get all combined payments", func(t *testing.T) {
		limit := uint16(10)
		payments, err := testClient.Payments.GetAllCombinedPayments(&limit, nil)
		if err != nil {
			t.Logf("Note: This test requires a valid API key")
			t.Skip("Skipping due to API requirements")
		}

		if payments == nil {
			t.Fatal("Expected payments result")
		}

		t.Logf("Retrieved %d combined payments", len(payments.Items))
	})
}

// TestDurationUnits tests different duration units
func TestDurationUnits(t *testing.T) {
	durations := []qbmodels.Duration{
		{Value: 1, Unit: qbmodels.DurationUnitSeconds},
		{Value: 1, Unit: qbmodels.DurationUnitMinutes},
		{Value: 1, Unit: qbmodels.DurationUnitHours},
		{Value: 1, Unit: qbmodels.DurationUnitDays},
		{Value: 1, Unit: qbmodels.DurationUnitWeeks},
		{Value: 1, Unit: qbmodels.DurationUnitMonths},
	}

	for _, duration := range durations {
		t.Run(string(duration.Unit), func(t *testing.T) {
			if duration.Value != 1 {
				t.Errorf("Expected value 1, got %d", duration.Value)
			}
		})
	}
}

// TestTransactionTypes tests transaction type constants
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

	for _, txType := range types {
		t.Run(string(txType), func(t *testing.T) {
			if txType == "" {
				t.Error("Transaction type should not be empty")
			}
		})
	}
}

// TestTransactionStatusValues tests transaction status value constants
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

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			if status == "" {
				t.Error("Status should not be empty")
			}
		})
	}
}

// BenchmarkClientCreation benchmarks client creation
func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qbitflow.New("test-api-key")
	}
}

// BenchmarkCreateSessionOptions benchmarks creating session options
func BenchmarkCreateSessionOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qbf.CreateSessionOptions{
			ProductID:    new(uint64(1)),
			SuccessURL:   new(string("https://example.com/success")),
			CancelURL:    new(string("https://example.com/cancel")),
			WebhookURL:   new(string("https://example.com/webhook")),
			CustomerUUID: new(string("test-customer")),
		}
	}
}
