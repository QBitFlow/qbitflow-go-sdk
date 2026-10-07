package tests

import (
	"errors"
	"math"
	"strings"
	"testing"

	qbitflow "github.com/QBitFlow/qbitflow-go-sdk/v2"
	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

// Client-side validation mirrors the API's binding rules — including the behaviour
// verified against the live API — so bad input fails before the round-trip with the same
// *qberrors.ValidationError the API's own 400 produces, and input the API accepts is never
// rejected locally. Every rejection case asserts that no request was issued.

func expectValidationError(t *testing.T, err error, rec *recorder, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error")
	}
	var validationErr *qberrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *qberrors.ValidationError, got %T: %v", err, err)
	}
	if validationErr.StatusCode != 0 {
		t.Errorf("client-side validation must carry status 0, got %d", validationErr.StatusCode)
	}
	if want != "" && !strings.Contains(err.Error(), want) {
		t.Errorf("message %q does not mention %q", err.Error(), want)
	}
	if rec != nil && rec.requests != 0 {
		t.Errorf("the SDK issued %d request(s); it should fail before the round-trip", rec.requests)
	}
}

func expectAccepted(t *testing.T, err error, rec *recorder) {
	t.Helper()
	if err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	if rec != nil && rec.requests != 1 {
		t.Errorf("expected exactly one request, got %d", rec.requests)
	}
}

// ── Customers ────────────────────────────────────────────────────────────────

func TestCreateCustomerValidation(t *testing.T) {
	valid := func() *qbf.CreateCustomer {
		return &qbf.CreateCustomer{Name: "Jane", LastName: "O'Neil-Smith", Email: "jane@example.com"}
	}

	accepted := []struct {
		name   string
		mutate func(c *qbf.CreateCustomer)
	}{
		{"unicode letters", func(c *qbf.CreateCustomer) { c.Name = "Zoë"; c.LastName = "Müller Ștefan" }},
		{"whitespace-only name (the API accepts it)", func(c *qbf.CreateCustomer) { c.Name = "   " }},
		{"Unicode decimal digits", func(c *qbf.CreateCustomer) { c.Name = "Jane ٣" }},
		{"all allowed punctuation", func(c *qbf.CreateCustomer) { c.LastName = "a-b_c'd.e f" }},
	}
	for _, tc := range accepted {
		t.Run("accepts "+tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{"uuid":"c1"}`)
			c := valid()
			tc.mutate(c)
			_, err := client.Customers.Create(c)
			expectAccepted(t, err, rec)
		})
	}

	rejected := []struct {
		name   string
		mutate func(c *qbf.CreateCustomer)
		want   string
	}{
		{"empty name", func(c *qbf.CreateCustomer) { c.Name = "" }, "Name is required"},
		{"name too short", func(c *qbf.CreateCustomer) { c.Name = "J" }, "Name must be between 2 and 100"},
		{"name too long", func(c *qbf.CreateCustomer) { c.Name = strings.Repeat("a", 101) }, "Name must be between 2 and 100"},
		{"superscript digit (not Nd)", func(c *qbf.CreateCustomer) { c.Name = "Jo²" }, "Name may only contain"},
		{"name with markup", func(c *qbf.CreateCustomer) { c.Name = "<b>Jane</b>" }, "Name may only contain"},
		{"name with comma", func(c *qbf.CreateCustomer) { c.Name = "Jane, Doe" }, "Name may only contain"},
		{"last name with @", func(c *qbf.CreateCustomer) { c.LastName = "Doe@x" }, "LastName may only contain"},
		{"missing email", func(c *qbf.CreateCustomer) { c.Email = "" }, "Email is required"},
		{"email without @", func(c *qbf.CreateCustomer) { c.Email = "jane.example.com" }, "Email must be a valid email"},
		{"email without domain dot", func(c *qbf.CreateCustomer) { c.Email = "jane@localhost" }, "Email must be a valid email"},
		{"email with two @", func(c *qbf.CreateCustomer) { c.Email = "a@b@c.com" }, "Email must be a valid email"},
		{"email with whitespace", func(c *qbf.CreateCustomer) { c.Email = "ja ne@example.com" }, "Email must be a valid email"},
		{"email with a trailing dot", func(c *qbf.CreateCustomer) { c.Email = "jane@example.com." }, "Email must be a valid email"},
	}
	for _, tc := range rejected {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			c := valid()
			tc.mutate(c)
			_, err := client.Customers.Create(c)
			expectValidationError(t, err, rec, tc.want)
		})
	}

	t.Run("empty optional fields are left out of the body", func(t *testing.T) {
		client, rec := clientRecording(t, `{"uuid":"c1"}`)
		c := valid()
		c.PhoneNumber, c.Address, c.Reference = new(""), new(""), new("CRM-1")
		_, err := client.Customers.Create(c)
		expectAccepted(t, err, rec)

		body := decodedBody(t, rec)
		if _, ok := body["phoneNumber"]; ok {
			t.Errorf("an empty phoneNumber was sent: %s", rec.body)
		}
		if _, ok := body["address"]; ok {
			t.Errorf("an empty address was sent: %s", rec.body)
		}
		if body["reference"] != "CRM-1" || body["email"] != "jane@example.com" {
			t.Errorf("body = %s", rec.body)
		}
	})

	t.Run("the address is not normalised", func(t *testing.T) {
		client, rec := clientRecording(t, `{"uuid":"c1"}`)
		c := valid()
		c.Email = "Jane.Doe@Example.COM"
		_, err := client.Customers.Create(c)
		expectAccepted(t, err, rec)
		if decodedBody(t, rec)["email"] != "Jane.Doe@Example.COM" {
			t.Errorf("email was altered: %s", rec.body)
		}
	})
}

func TestUpdateCustomerValidatesOnlyProvidedFields(t *testing.T) {
	t.Run("empty update is accepted", func(t *testing.T) {
		client, rec := clientRecording(t, `{"uuid":"c1"}`)
		_, err := client.Customers.Update("c1", &qbf.UpdateCustomer{})
		expectAccepted(t, err, rec)
		if rec.body != "{}" {
			t.Errorf("body = %s, want {}", rec.body)
		}
	})
	t.Run("empty strings mean not provided and are left out", func(t *testing.T) {
		client, rec := clientRecording(t, `{"uuid":"c1"}`)
		_, err := client.Customers.Update("c1", &qbf.UpdateCustomer{
			Name: new(""), LastName: new(""), Email: new(""), PhoneNumber: new(""), Address: new("1 Main St"),
		})
		expectAccepted(t, err, rec)
		if rec.body != `{"address":"1 Main St"}` {
			t.Errorf("body = %s", rec.body)
		}
	})
	t.Run("a provided invalid field is rejected", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Customers.Update("c1", &qbf.UpdateCustomer{Email: new("not-an-email")})
		expectValidationError(t, err, rec, "Email must be a valid email")
		if strings.Contains(err.Error(), "not-an-email") {
			t.Errorf("error message leaks the input value: %q", err.Error())
		}
	})
	t.Run("missing uuid is rejected", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Customers.Update("", &qbf.UpdateCustomer{Name: new("Jane")})
		expectValidationError(t, err, rec, "Customer UUID is required")
	})
}

// ── Users ────────────────────────────────────────────────────────────────────

func TestCreateUserValidation(t *testing.T) {
	valid := func() *qbf.CreateUser {
		return &qbf.CreateUser{Name: "Alice", LastName: "Smith", Email: "alice@example.com", Role: qbmodels.RoleUser}
	}

	t.Run("accepts admin and user roles, including the untyped string form", func(t *testing.T) {
		for _, role := range []qbmodels.UserRole{qbmodels.RoleAdmin, "user"} {
			client, rec := clientRecording(t, `{"id":1}`)
			u := valid()
			u.Role = role
			u.OrganizationFeeBps = 5000
			_, err := client.Users.Create(u)
			expectAccepted(t, err, rec)
		}
	})

	cases := []struct {
		name   string
		mutate func(u *qbf.CreateUser)
		want   string
	}{
		{"owner role is not assignable", func(u *qbf.CreateUser) { u.Role = qbmodels.RoleOwner }, `Role must be "admin" or "user"`},
		{"handle role is not assignable", func(u *qbf.CreateUser) { u.Role = qbmodels.RoleHandle }, `Role must be "admin" or "user"`},
		{"empty role", func(u *qbf.CreateUser) { u.Role = "" }, `Role must be "admin" or "user"`},
		{"fee above 5000 bps", func(u *qbf.CreateUser) { u.OrganizationFeeBps = 5001 }, "OrganizationFeeBps must be between 0 and 5000"},
		{"name too short", func(u *qbf.CreateUser) { u.Name = "A" }, "Name must be between 2 and 100"},
		{"missing email", func(u *qbf.CreateUser) { u.Email = "" }, "Email is required"},
		{"invalid email", func(u *qbf.CreateUser) { u.Email = "alice" }, "Email must be a valid email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			u := valid()
			tc.mutate(u)
			_, err := client.Users.Create(u)
			expectValidationError(t, err, rec, tc.want)
		})
	}
}

func TestUpdateUserValidation(t *testing.T) {
	t.Run("zero id is rejected before any request", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Users.Update(0, &qbf.UpdateUser{Name: new("Alicia")})
		expectValidationError(t, err, rec, "User ID must be positive")
	})
	t.Run("fee above 5000 bps is rejected", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Users.Update(1, &qbf.UpdateUser{OrganizationFeeBps: new(uint16(6000))})
		expectValidationError(t, err, rec, "OrganizationFeeBps must be between 0 and 5000")
	})
	t.Run("explicit zero fee is sent", func(t *testing.T) {
		client, rec := clientRecording(t, `{"id":1}`)
		_, err := client.Users.Update(1, &qbf.UpdateUser{OrganizationFeeBps: new(uint16(0))})
		expectAccepted(t, err, rec)
		if rec.body != `{"organizationFeeBps":0}` {
			t.Errorf("body = %s", rec.body)
		}
	})
	t.Run("empty strings mean not provided and are left out", func(t *testing.T) {
		client, rec := clientRecording(t, `{"id":1}`)
		_, err := client.Users.Update(1, &qbf.UpdateUser{Name: new(""), LastName: new("Smith"), Email: new("")})
		expectAccepted(t, err, rec)
		if rec.body != `{"lastName":"Smith"}` {
			t.Errorf("body = %s", rec.body)
		}
	})
}

// ── Email lookups use the email rule ─────────────────────────────────────────

func TestEmailLookupsUseTheEmailRule(t *testing.T) {
	cases := []struct {
		name string
		call func(c *qbitflow.QBitFlow, email string) error
	}{
		{"customers.GetByEmail", func(c *qbitflow.QBitFlow, e string) error { _, err := c.Customers.GetByEmail(e); return err }},
		{"users.GetByEmail", func(c *qbitflow.QBitFlow, e string) error { _, err := c.Users.GetByEmail(e); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			expectValidationError(t, tc.call(client, "a@b"), rec, "must be a valid email")

			client, rec = clientRecording(t, `{}`)
			expectAccepted(t, tc.call(client, "a@b.co"), rec)
		})
	}
}

// ── Products ─────────────────────────────────────────────────────────────────

func TestProductValidation(t *testing.T) {
	t.Run("update guards the id", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Products.Update(0, &qbf.UpdateProduct{Price: new(9.99)})
		expectValidationError(t, err, rec, "Product ID must be positive")
	})

	t.Run("price must be a finite number greater than 0", func(t *testing.T) {
		for _, price := range []float64{0, -5, math.NaN(), math.Inf(1), math.Inf(-1)} {
			client, rec := clientRecording(t, `{}`)
			_, err := client.Products.Update(1, &qbf.UpdateProduct{Price: new(price)})
			expectValidationError(t, err, rec, "Product price must be a finite number greater than 0")

			client, rec = clientRecording(t, `{}`)
			_, err = client.Products.Create(&qbf.CreateProduct{Name: "Widget", Description: "A widget", Price: price})
			expectValidationError(t, err, rec, "Product price must be a finite number greater than 0")
		}
	})

	t.Run("an empty name or description on update is rejected (the API rejects it too)", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Products.Update(1, &qbf.UpdateProduct{Name: new("")})
		expectValidationError(t, err, rec, "Product name must not be blank")

		client, rec = clientRecording(t, `{}`)
		_, err = client.Products.Update(1, &qbf.UpdateProduct{Description: new("   ")})
		expectValidationError(t, err, rec, "Product description must not be blank")
	})

	t.Run("an empty reference on create is left out", func(t *testing.T) {
		client, rec := clientRecording(t, `{"id":1}`)
		_, err := client.Products.Create(&qbf.CreateProduct{Name: "Widget", Description: "A widget", Price: 1, Reference: new("")})
		expectAccepted(t, err, rec)
		if _, ok := decodedBody(t, rec)["reference"]; ok {
			t.Errorf("an empty reference was sent: %s", rec.body)
		}
	})

	t.Run("producttext rejects control characters but allows newlines and tabs", func(t *testing.T) {
		if err := qbf.ValidateProductTextForTest("name", "line one\nline two\tend", 2, 100); err != nil {
			t.Errorf("newline/tab must be accepted: %v", err)
		}
		if err := qbf.ValidateProductTextForTest("name", "bell"+string(rune(7)), 2, 100); err == nil {
			t.Error("a control character must be rejected")
		}
		if err := qbf.ValidateProductTextForTest("name", "　　", 2, 100); err == nil {
			t.Error("text that is blank after Unicode whitespace trimming must be rejected")
		}
	})
}

// ── Session checkout ─────────────────────────────────────────────────────────

func TestSessionCheckoutValidation(t *testing.T) {
	t.Run("rejects markup in an inline product name", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("<script>alert(1)</script>"),
			Description: new("A perfectly fine description"),
			Price:       new(9.99),
		})
		expectValidationError(t, err, rec, "productName")
	})

	t.Run("allows script-looking but markup-free text", func(t *testing.T) {
		if err := qbf.ValidateProductTextForTest("productName", "javascript:alert(1)", 2, 100); err != nil {
			t.Errorf("markup-free text must be accepted, got: %v", err)
		}
	})

	t.Run("rejects a too-short inline product name", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("A"),
			Description: new("A perfectly fine description"),
			Price:       new(9.99),
		})
		expectValidationError(t, err, rec, "productName must be between 2 and 100")
	})

	t.Run("empty optional strings are not provided and are left out of the body", func(t *testing.T) {
		// `new(os.Getenv("SUCCESS_URL"))` must work when the variable is unset — and an
		// empty customerUUID sent to the API would be a 400.
		client, rec := clientRecording(t, `{"link":"l","uuid":"pay@1"}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductID:         new(uint64(339)),
			SuccessURL:        new(""),
			CancelURL:         new(""),
			ProductName:       new(""),
			Description:       new(""),
			CustomerUUID:      new(""),
			Reference:         new(""),
			ProductReference:  new(""),
			CustomerReference: new(""),
		})
		expectAccepted(t, err, rec)
		if rec.body != `{"productId":339}` {
			t.Errorf("body = %s, want only the product id", rec.body)
		}
	})

	t.Run("redirect URLs must be absolute http(s) URLs; the scheme is case-insensitive", func(t *testing.T) {
		for _, bad := range []string{"javascript:alert(1)", "/relative/path", "ftp://example.com", "https://"} {
			client, rec := clientRecording(t, `{}`)
			_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(1)), SuccessURL: new(bad)})
			expectValidationError(t, err, rec, "successUrl")
		}
		for _, good := range []string{"https://shop.example.com/ok?o=1", "HTTPS://shop.example.com/", "Http://localhost:8080/x"} {
			client, rec := clientRecording(t, `{"link":"l","uuid":"pay@1"}`)
			_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(1)), CancelURL: new(good)})
			expectAccepted(t, err, rec)
		}
	})

	t.Run("requires a product selection", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{})
		expectValidationError(t, err, rec, "either ProductID, ProductReference, or ProductName")

		// An incomplete inline triplet is not a selection either.
		client, rec = clientRecording(t, `{}`)
		_, err = client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductName: new("Widget"), Price: new(1.0)})
		expectValidationError(t, err, rec, "either ProductID, ProductReference, or ProductName")

		// An empty reference does not select a product.
		client, rec = clientRecording(t, `{}`)
		_, err = client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductReference: new("")})
		expectValidationError(t, err, rec, "either ProductID, ProductReference, or ProductName")
	})

	t.Run("rejects a zero product id", func(t *testing.T) {
		client, rec := clientRecording(t, `{}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(0))})
		expectValidationError(t, err, rec, "ProductID must be positive")
	})

	t.Run("an inline price must be finite and greater than 0 (the API rejects 0)", func(t *testing.T) {
		for _, price := range []float64{0, -10, math.NaN(), math.Inf(1)} {
			client, rec := clientRecording(t, `{}`)
			_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
				ProductName: new("Test"), Description: new("Test"), Price: new(price),
			})
			expectValidationError(t, err, rec, "price must be a finite number greater than 0")
		}

		client, rec := clientRecording(t, `{"link":"l","uuid":"pay@1"}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{
			ProductName: new("Sample"), Description: new("Costs a cent"), Price: new(0.01),
		})
		expectAccepted(t, err, rec)
	})

	t.Run("customerUUID must be a bare UUID", func(t *testing.T) {
		for _, bad := range []string{"customer-uuid-123", "cus@3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f", "3f2b1c1e8d1a4c2b9b0e1a2b3c4d5e6f", " 3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f"} {
			client, rec := clientRecording(t, `{}`)
			_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(1)), CustomerUUID: new(bad)})
			expectValidationError(t, err, rec, "CustomerUUID must be a UUID")
		}
		for _, good := range []string{"3f2b1c1e-8d1a-4c2b-9b0e-1a2b3c4d5e6f", "3F2B1C1E-8D1A-4C2B-9B0E-1A2B3C4D5E6F"} {
			client, rec := clientRecording(t, `{"link":"l","uuid":"pay@1"}`)
			_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductID: new(uint64(1)), CustomerUUID: new(good)})
			expectAccepted(t, err, rec)
			if decodedBody(t, rec)["customerUUID"] != good {
				t.Errorf("customerUUID not sent: %s", rec.body)
			}
		}
	})

	t.Run("accepts a product reference alone", func(t *testing.T) {
		client, rec := clientRecording(t, `{"link":"l","uuid":"pay@1"}`)
		_, err := client.Payments.CreateSession(&qbf.CreateSessionOptions{ProductReference: new("PROD-1")})
		expectAccepted(t, err, rec)
	})
}

func TestSubscriptionSessionValidation(t *testing.T) {
	base := func() *qbf.CreateSubscriptionSessionOptions {
		return &qbf.CreateSubscriptionSessionOptions{
			ProductID: new(uint64(1)),
			Frequency: qbmodels.Duration{Value: 1, Unit: qbmodels.DurationUnitMonths},
		}
	}

	t.Run("accepts every documented unit", func(t *testing.T) {
		for _, unit := range []qbmodels.DurationUnit{
			qbmodels.DurationUnitSeconds, qbmodels.DurationUnitMinutes, qbmodels.DurationUnitHours,
			qbmodels.DurationUnitDays, qbmodels.DurationUnitWeeks, qbmodels.DurationUnitMonths,
			qbmodels.DurationUnitYears,
		} {
			client, rec := clientRecording(t, `{"link":"l","uuid":"sub@1"}`)
			opts := base()
			opts.Frequency.Unit = unit
			opts.TrialPeriod = &qbmodels.Duration{Value: 7, Unit: qbmodels.DurationUnitDays}
			opts.MinPeriods = new(uint32(3))
			_, err := client.Subscriptions.CreateSession(opts)
			expectAccepted(t, err, rec)
		}
	})

	t.Run("accepts the uint32 bounds and a zero trial / minPeriods (the API accepts them)", func(t *testing.T) {
		client, rec := clientRecording(t, `{"link":"l","uuid":"sub@1"}`)
		opts := base()
		opts.Frequency.Value = math.MaxUint32
		opts.TrialPeriod = &qbmodels.Duration{Value: 0, Unit: qbmodels.DurationUnitDays}
		opts.MinPeriods = new(uint32(0))
		_, err := client.Subscriptions.CreateSession(opts)
		expectAccepted(t, err, rec)

		body := decodedBody(t, rec)
		if _, ok := body["minPeriods"]; ok {
			t.Errorf("minPeriods 0 must be left out: %s", rec.body)
		}
		if _, ok := body["trialPeriod"]; !ok {
			t.Errorf("a zero trial period is still sent: %s", rec.body)
		}
	})

	cases := []struct {
		name   string
		mutate func(o *qbf.CreateSubscriptionSessionOptions)
		want   string
	}{
		{"zero frequency", func(o *qbf.CreateSubscriptionSessionOptions) { o.Frequency.Value = 0 }, "Frequency.Value must be between 1 and 4294967295"},
		{"negative frequency", func(o *qbf.CreateSubscriptionSessionOptions) { o.Frequency.Value = -1 }, "Frequency.Value must be between 1 and 4294967295"},
		{"frequency above uint32", func(o *qbf.CreateSubscriptionSessionOptions) { o.Frequency.Value = math.MaxUint32 + 1 }, "Frequency.Value must be between 1 and 4294967295"},
		{"missing frequency", func(o *qbf.CreateSubscriptionSessionOptions) { o.Frequency = qbmodels.Duration{} }, "Frequency.Value"},
		{"unknown frequency unit", func(o *qbf.CreateSubscriptionSessionOptions) { o.Frequency.Unit = "fortnights" }, "Frequency.Unit must be one of"},
		{"negative trial period", func(o *qbf.CreateSubscriptionSessionOptions) {
			o.TrialPeriod = &qbmodels.Duration{Value: -1, Unit: qbmodels.DurationUnitDays}
		}, "TrialPeriod.Value must be between 0 and 4294967295"},
		{"trial period above uint32", func(o *qbf.CreateSubscriptionSessionOptions) {
			o.TrialPeriod = &qbmodels.Duration{Value: math.MaxUint32 + 1, Unit: qbmodels.DurationUnitDays}
		}, "TrialPeriod.Value must be between 0 and 4294967295"},
		{"trial period with an unknown unit", func(o *qbf.CreateSubscriptionSessionOptions) {
			o.TrialPeriod = &qbmodels.Duration{Value: 7, Unit: "day"}
		}, "TrialPeriod.Unit must be one of"},
		{"no product", func(o *qbf.CreateSubscriptionSessionOptions) { o.ProductID = nil }, "either ProductID, ProductReference, or ProductName"},
		{"bad customer uuid", func(o *qbf.CreateSubscriptionSessionOptions) { o.CustomerUUID = new("sub@1") }, "CustomerUUID must be a UUID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `{}`)
			opts := base()
			tc.mutate(opts)
			_, err := client.Subscriptions.CreateSession(opts)
			expectValidationError(t, err, rec, tc.want)
		})
	}
}

// ── Transaction status guards ────────────────────────────────────────────────

func TestTransactionStatusGuardsItsArguments(t *testing.T) {
	client, rec := clientRecording(t, `{}`)
	_, err := client.TransactionStatus.GetTransactionStatus("", qbmodels.TransactionTypeOneTimePayment)
	expectValidationError(t, err, rec, "Transaction UUID is required")

	client, rec = clientRecording(t, `{}`)
	_, err = client.TransactionStatus.GetTransactionStatus("pay@1", "")
	expectValidationError(t, err, rec, "Transaction type is required")

	client, rec = clientRecording(t, `{}`)
	_, err = client.Subscriptions.GetSubscriptionByReference("")
	expectValidationError(t, err, rec, "Subscription reference is required")
}

// ── Accounting export ────────────────────────────────────────────────────────

func TestAccountingExportValidation(t *testing.T) {
	t.Run("accepts a long window: the maximum span is the API's to enforce", func(t *testing.T) {
		// The live API accepts spans of up to 95 days; the SDK does not second-guess it.
		client, rec := clientRecording(t, `[]`)
		_, err := client.Accounting.Export(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-04-05"})
		expectAccepted(t, err, rec)
		if !strings.Contains(rec.rawQuery, "format=json") || !strings.Contains(rec.rawQuery, "from=2026-01-01") {
			t.Errorf("query = %q", rec.rawQuery)
		}

		client, rec = clientRecording(t, "paymentId,type\n")
		csv, err := client.Accounting.ExportCSV(qbf.AccountingExportParams{From: "2026-01-01", To: "2026-01-01"})
		expectAccepted(t, err, rec)
		if csv != "paymentId,type\n" || !strings.Contains(rec.rawQuery, "format=csv") {
			t.Errorf("csv = %q query = %q", csv, rec.rawQuery)
		}
	})

	cases := []struct {
		name     string
		from, to string
		want     string
	}{
		{"missing dates", "", "", "From and To dates are required"},
		{"missing to", "2026-01-01", "", "From and To dates are required"},
		{"bad from format", "01/01/2026", "2026-02-01", "From must be a date formatted YYYY-MM-DD"},
		{"bad to format", "2026-01-01", "2026-2-1", "To must be a date formatted YYYY-MM-DD"},
		{"impossible date", "2026-02-30", "2026-03-01", "From must be a date formatted YYYY-MM-DD"},
		{"to before from (the API rejects it)", "2026-03-01", "2026-02-01", "To must not be before From"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, rec := clientRecording(t, `[]`)
			_, err := client.Accounting.Export(qbf.AccountingExportParams{From: tc.from, To: tc.to})
			expectValidationError(t, err, rec, tc.want)

			client, rec = clientRecording(t, ``)
			_, err = client.Accounting.ExportCSV(qbf.AccountingExportParams{From: tc.from, To: tc.to})
			expectValidationError(t, err, rec, tc.want)
		})
	}
}
