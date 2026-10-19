package qbitflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

// fieldsOf returns the failing fields of a Validate() result (nil when valid).
func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	if err == nil {
		return nil
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *ValidationError, got %T %v", err, err)
	}
	if ve.StatusCode != 0 {
		t.Errorf("client-side errors have no status")
	}
	var out []string
	for _, f := range ve.FieldErrors {
		if !strings.HasPrefix(f.Message, f.Field+" ") {
			t.Errorf("message %q does not start with its field %q", f.Message, f.Field)
		}
		out = append(out, f.Field)
	}
	return out
}

func check(t *testing.T, name string, err error, want ...string) {
	t.Helper()
	got := fieldsOf(t, err)
	if !slices.Equal(got, want) {
		t.Errorf("%s: failing fields %v, want %v (%v)", name, got, want, err)
	}
}

func TestRuleName(t *testing.T) {
	good := []string{"Pro plan", "O’Brien", "Smith & Co", "prod/backend", "Dr. Who (II)", "Zoë", "李小龙", "ab", strings.Repeat("é", 100)}
	bad := []string{"a", "  ", " \t ", "line\nbreak", "tab\there", "<b>", "x{y}", "a[1]", "back`tick", `back\slash`, "pi|pe", "semi;colon", `quo"te`, "til~de", "car^et", "bidi‮name", "bidi⁦x", "nul\x00x", strings.Repeat("é", 101)}
	for _, s := range good {
		check(t, "name "+s, (&CreateProductParams{Name: s, PriceUSD: "1"}).Validate())
	}
	for _, s := range bad {
		check(t, "name "+s, (&CreateProductParams{Name: s, PriceUSD: "1"}).Validate(), "name")
	}
	check(t, "lastName 1 char", (&CreateCustomerParams{FirstName: "Ada", LastName: "L", Email: "a@b.co"}).Validate())
	check(t, "lastName newline", (&CreateCustomerParams{FirstName: "Ada", LastName: "L\n", Email: "a@b.co"}).Validate(), "lastName")
	check(t, "firstName 1 char", (&CreateCustomerParams{FirstName: "A", Email: "a@b.co"}).Validate(), "firstName")
	check(t, "firstName required", (&CreateCustomerParams{Email: "a@b.co"}).Validate(), "firstName")
	check(t, "update firstName", (&UpdateCustomerParams{FirstName: "A"}).Validate(), "firstName")
	check(t, "update lastName", (&UpdateCustomerParams{LastName: ptr("L")}).Validate())
	check(t, "update lastName bad", (&UpdateCustomerParams{LastName: ptr("bidi\u202ex")}).Validate(), "lastName")
	check(t, "session productName", (&CreatePaymentSessionParams{ProductName: "a", PriceUSD: "1"}).Validate(), "productName")
}

func TestRuleText(t *testing.T) {
	good := []string{"Two\nlines\r\nand\ttab", "Lifetime access (Pro) & more: 100%", "ok"}
	bad := []string{"   ", "\n\n", "<script>", "x\x07bell", strings.Repeat("x", 501), "a"}
	for _, s := range good {
		check(t, "description", (&CreateProductParams{Name: "Pro", PriceUSD: "1", Description: s}).Validate())
	}
	for _, s := range bad {
		check(t, "description", (&CreateProductParams{Name: "Pro", PriceUSD: "1", Description: s}).Validate(), "description")
	}
	// ≤500 without a minimum, and the endpoint description ≤200.
	check(t, "address 1 char", (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Address: "x"}).Validate())
	check(t, "address 501", (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Address: strings.Repeat("x", 501)}).Validate(), "address")
	check(t, "endpoint description 201", (&CreateWebhookEndpointParams{URL: "https://x.io", Description: strings.Repeat("x", 201)}).Validate(), "description")
}

func TestRuleReference(t *testing.T) {
	for _, s := range []string{"order-1042", "INV_2026.10:01@shop", "a", strings.Repeat("r", 100)} {
		check(t, "reference "+s, (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Reference: s}).Validate())
	}
	for _, s := range []string{"has space", "slash/x", "é", "#1", strings.Repeat("r", 101)} {
		check(t, "reference "+s, (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Reference: s}).Validate(), "reference")
	}
	check(t, "session refs", (&CreatePaymentSessionParams{ProductReference: "bad ref", Reference: "o/1", CustomerReference: "c 1"}).Validate(), "reference", "productReference", "customerReference")
	// Every ?reference= filter is checked like a created reference (they are lookups now).
	check(t, "subscription filter", (&SubscriptionListParams{Reference: "bad ref"}).Validate(), "reference")
	check(t, "product filter", (&ProductListParams{Reference: "bad ref"}).Validate(), "reference")
	check(t, "customer filter", (&CustomerListParams{Reference: strings.Repeat("r", 101)}).Validate(), "reference")
	check(t, "payment filter", (&PaymentListParams{Reference: "o/1"}).Validate(), "reference")
	check(t, "dots are references", (&PaymentListParams{Reference: ".."}).Validate())
}

func TestRulePhone(t *testing.T) {
	for _, s := range []string{"+33 6 12 34 56 78", "555 (123) 4567", "06.12.34.56.78", "123456"} {
		check(t, "phone "+s, (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", PhoneNumber: s}).Validate())
	}
	for _, s := range []string{"12345", "+", "phone", "+33 6 12 34 56 7x", "-123456", "123456-", "+" + strings.Repeat("1", 32)} {
		check(t, "phone "+s, (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", PhoneNumber: s}).Validate(), "phoneNumber")
	}
}

func TestRuleEmail(t *testing.T) {
	for _, s := range []string{"a@b.co", "First.Last+tag@Example.COM", "x@sub.domain.io"} {
		check(t, "email "+s, (&CreateCustomerParams{FirstName: "Ada", Email: s}).Validate())
	}
	for _, s := range []string{"no-at", "a@b", "@b.co", "a@.b.co", "a@b.co.", "a b@c.io", "a@@b.co", "a@b@c.io", strings.Repeat("a", 250) + "@b.co"} {
		check(t, "email "+s, (&CreateCustomerParams{FirstName: "Ada", Email: s}).Validate(), "email")
	}
	check(t, "email required", (&CreateCustomerParams{FirstName: "Ada"}).Validate(), "email")
	check(t, "invitation email", (&CreateInvitationParams{Email: "nope"}).Validate(), "email")
	check(t, "filter email", (&CustomerListParams{Email: "nope"}).Validate(), "email")
}

func TestRuleURL(t *testing.T) {
	for _, s := range []string{"https://shop.example.com/thanks?id={{UUID}}&t={{TRANSACTION_TYPE}}", "http://localhost:8080/x", "HTTPS://EXAMPLE.COM"} {
		check(t, "url "+s, (&CreatePaymentSessionParams{ProductUUID: memberUUID, SuccessURL: s, CancelURL: s}).Validate())
	}
	for _, s := range []string{"ftp://x.io", "/relative", "https://", "javascript:alert(1)", "https://x.io/" + strings.Repeat("p", 2040)} {
		check(t, "url "+s, (&CreatePaymentSessionParams{ProductUUID: memberUUID, SuccessURL: s}).Validate(), "successUrl")
	}
	check(t, "redirectUrl", (&CreateInvitationParams{Email: "a@b.co", RedirectURL: "nope"}).Validate(), "redirectUrl")
	check(t, "endpoint url required", (&CreateWebhookEndpointParams{}).Validate(), "url")
	check(t, "endpoint url update", (&UpdateWebhookEndpointParams{URL: "x"}).Validate(), "url")
}

// TestRuleUSD checks the usd rule on every money input: a decimal string, digits with at most 2
// decimals, above 0 and at most 1,000,000, compared exactly (no float).
func TestRuleUSD(t *testing.T) {
	good := []string{"0.01", "0.1", "1", "4.99", "5", "12.50", "19.9", "1000000", "1000000.00", "0000012.5", "999999.99"}
	bad := []string{"", "0", "0.00", "-1", "+1", "1e2", "abc", ".5", "1.", " 1", "1 ", "1.999", "0.001", "0.000001", "1000000.01",
		"1000001", "1000000000", "1,5", "1_000", "NaN", "Infinity", "٣"}
	for _, a := range good {
		check(t, "priceUsd "+a, (&CreateProductParams{Name: "Pro", PriceUSD: a}).Validate())
		check(t, "update priceUsd "+a, (&UpdateProductParams{PriceUSD: a}).Validate())
		check(t, "inline priceUsd "+a, (&CreatePaymentSessionParams{ProductName: "Pro", PriceUSD: a}).Validate())
	}
	for _, a := range bad {
		check(t, "priceUsd "+a, (&CreateProductParams{Name: "Pro", PriceUSD: a}).Validate(), "priceUsd")
		if a != "" {
			check(t, "update priceUsd "+a, (&UpdateProductParams{PriceUSD: a}).Validate(), "priceUsd")
			check(t, "inline priceUsd "+a, (&CreatePaymentSessionParams{ProductName: "Pro", PriceUSD: a}).Validate(), "priceUsd")
			check(t, "subscription inline priceUsd "+a, (&CreateSubscriptionSessionParams{ProductName: "Pro", PriceUSD: a}).Validate(), "priceUsd")
		}
	}
	check(t, "update no price", (&UpdateProductParams{}).Validate())
	err := (&CreateProductParams{Name: "Pro", PriceUSD: "1.999"}).Validate()
	if want := `priceUsd must be an amount in USD as a decimal string (e.g. "12.50"): above 0, at most 1000000, with at most 2 decimals`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("message: %v", err)
	}
	// The exact comparison: no float rounding at the bounds.
	for _, tc := range []struct {
		value string
		limit int
		want  int
	}{
		{"1000000", 1_000_000, 0}, {"999999.99", 1_000_000, -1}, {"1000000.01", 1_000_000, 1}, {"0.5", 1_000_000, -1},
		{"50", 50, 0}, {"50.00", 50, 0}, {"50.01", 50, 1}, {"0050", 50, 0}, {"49.99", 50, -1}, {"100.1", 100, 1}, {"99", 100, -1}, {"101", 100, 1},
	} {
		if got := compareDecimal(tc.value, tc.limit); got != tc.want {
			t.Errorf("compareDecimal(%q, %d) = %d, want %d", tc.value, tc.limit, got, tc.want)
		}
	}
}

func TestRuleFees(t *testing.T) {
	line := FeeItem{Label: "Shipping", Description: "Standard, 3 to 5 days", AmountUSD: "0.75"}
	session := func(fees *CheckoutFees) *CreatePaymentSessionParams {
		return &CreatePaymentSessionParams{ProductUUID: memberUUID, Fees: fees}
	}
	withItem := func(edit func(*FeeItem)) *CreatePaymentSessionParams {
		item := line
		edit(&item)
		return session(&CheckoutFees{Items: []FeeItem{item}})
	}
	lines := func(n int) []FeeItem { return slices.Repeat([]FeeItem{line}, n) }

	check(t, "no fees", session(nil).Validate())
	check(t, "empty fees", session(&CheckoutFees{}).Validate())
	check(t, "processing fee only", session(&CheckoutFees{ProcessingFee: ptr(true)}).Validate())
	check(t, "processing fee off", session(&CheckoutFees{ProcessingFee: ptr(false), Items: lines(1)}).Validate())
	check(t, "10 lines", session(&CheckoutFees{Items: lines(10)}).Validate())
	check(t, "11 lines", session(&CheckoutFees{Items: lines(11)}).Validate(), "fees.items")
	check(t, "11 lines, the last bad", session(&CheckoutFees{Items: append(lines(10), FeeItem{Label: "a|b", AmountUSD: "1"})}).Validate(),
		"fees.items", "fees.items[10].label")

	// label: required, the name rule, 1 to 40 code points.
	for _, label := range []string{"S", "VAT (20%)", strings.Repeat("l", 40), strings.Repeat("😀", 40), "Frais de port é"} {
		check(t, "label "+label, withItem(func(i *FeeItem) { i.Label = label }).Validate())
	}
	for _, label := range []string{"", "   ", strings.Repeat("l", 41), strings.Repeat("😀", 41), "Ship\nping", "Ship\tping", "Ship|ping", "<b>", "bidi‮x"} {
		check(t, "label "+label, withItem(func(i *FeeItem) { i.Label = label }).Validate(), "fees.items[0].label")
	}

	// description: optional, the text rule, at most 200.
	for _, d := range []string{"", "x", strings.Repeat("d", 200), strings.Repeat("😀", 200), "Standard,\n\t3 to 5 days"} {
		check(t, "description "+d, withItem(func(i *FeeItem) { i.Description = d }).Validate())
	}
	for _, d := range []string{"   ", strings.Repeat("d", 201), "nul\x00", "semi;colon", "{x}"} {
		check(t, "description "+d, withItem(func(i *FeeItem) { i.Description = d }).Validate(), "fees.items[0].description")
	}

	// amountUsd: required, the usd rule (a decimal string, above 0, at most 1,000,000, 2 decimals).
	for _, a := range []string{"0.01", "0.75", "4.99", "19.9", "20", "100", "1000000", "12.50"} {
		check(t, fmt.Sprint("amountUsd ", a), withItem(func(i *FeeItem) { i.AmountUSD = a }).Validate())
	}
	for _, a := range []string{"", "0", "-1", "-0.01", "0.001", "1.999", "1000000.01", "1000001", "1e2", "+1", "abc", ".5", "0.30000000000000004"} {
		check(t, fmt.Sprint("amountUsd ", a), withItem(func(i *FeeItem) { i.AmountUSD = a }).Validate(), "fees.items[0].amountUsd")
	}

	// Every failing line is reported, by its index.
	bad := session(&CheckoutFees{Items: []FeeItem{line, {Label: "", AmountUSD: "1"}, {Label: "Tax", Description: " ", AmountUSD: "0"}}})
	check(t, "several lines", bad.Validate(), "fees.items[1].label", "fees.items[2].description", "fees.items[2].amountUsd")
	err := withItem(func(i *FeeItem) { i.AmountUSD = "1.999" }).Validate()
	if want := `fees.items[0].amountUsd must be an amount in USD as a decimal string (e.g. "12.50"): above 0, at most 1000000, with at most 2 decimals`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("message: %v", err)
	}
	// The fees and the session's own fields are reported together.
	check(t, "fees and product", (&CreatePaymentSessionParams{Fees: &CheckoutFees{Items: lines(11)}}).Validate(), "productUuid", "fees.items")

	// A subscription checkout's fees follow the same rules, reported with the terms.
	sub := func(fees *CheckoutFees) *CreateSubscriptionSessionParams {
		return &CreateSubscriptionSessionParams{ProductUUID: memberUUID, Fees: fees}
	}
	check(t, "subscription no fees", sub(nil).Validate())
	check(t, "subscription processing fee only", sub(&CheckoutFees{ProcessingFee: ptr(true)}).Validate())
	check(t, "subscription 10 lines", sub(&CheckoutFees{ProcessingFee: ptr(false), Items: lines(10)}).Validate())
	check(t, "subscription 11 lines", sub(&CheckoutFees{Items: lines(11)}).Validate(), "fees.items")
	check(t, "subscription bad line", sub(&CheckoutFees{Items: []FeeItem{{Label: "a|b", Description: " ", AmountUSD: "0.001"}}}).Validate(),
		"fees.items[0].label", "fees.items[0].description", "fees.items[0].amountUsd")
	check(t, "subscription fees and terms", (&CreateSubscriptionSessionParams{ProductUUID: memberUUID, Frequency: &Duration{Value: 1},
		Fees: &CheckoutFees{Items: []FeeItem{{Label: "Tax", AmountUSD: "-1"}}}}).Validate(), "frequency.unit", "fees.items[0].amountUsd")
	// The fees are reported last, after the session's own fields (metadata included).
	check(t, "fees after metadata", (&CreatePaymentSessionParams{ProductUUID: memberUUID, Metadata: map[string]string{"k": "\x00"},
		Fees: &CheckoutFees{Items: []FeeItem{{Label: "Tax", AmountUSD: "0"}}}}).Validate(), "metadata.k", "fees.items[0].amountUsd")
}

// TestRulePercent checks the percent rule: a decimal string, digits with at most 2 decimals, at
// most 16 characters, from 0 to its maximum, compared exactly.
func TestRulePercent(t *testing.T) {
	for _, p := range []string{"0", "0.00", "0.05", "1.5", "33.33", "50", "50.00", "007.5", "000000000000050"} { // 15 characters
		check(t, "member fee "+p, (&UpdateMemberParams{OrganizationFeePercent: p}).Validate())
		check(t, "invitation fee "+p, (&CreateInvitationParams{Email: "a@b.co", OrganizationFeePercent: p}).Validate())
	}
	for _, p := range []string{"-0.01", "50.01", "51", "1.234", "1.155", "0.30000000000000004", "1e1", ".5", "5.", "50%", " 5", "NaN",
		"Infinity", "0000000000000050"} { // 16 is the limit: 16 digits is fine, 17 not
		want := []string{"organizationFeePercent"}
		if p == "0000000000000050" {
			want = nil
		}
		check(t, "member fee "+p, (&UpdateMemberParams{OrganizationFeePercent: p}).Validate(), want...)
		check(t, "invitation fee "+p, (&CreateInvitationParams{Email: "a@b.co", OrganizationFeePercent: p}).Validate(), want...)
	}
	check(t, "member fee 17 characters", (&UpdateMemberParams{OrganizationFeePercent: "00000000000000050"}).Validate(), "organizationFeePercent")
	check(t, "member fee required", (&UpdateMemberParams{}).Validate(), "organizationFeePercent")
	check(t, "invitation fee default", (&CreateInvitationParams{Email: "a@b.co"}).Validate())
}

// TestRuleMetadata checks the metadata rule on every carrier: at most 50 keys; keys 1 to 40 of
// A-Z a-z 0-9 _ - . : (errors on metadata, the key not echoed); values at most 500 code points
// without control (tab included) or bidirectional characters (errors on metadata.<key>).
func TestRuleMetadata(t *testing.T) {
	keys := func(n int) map[string]string {
		m := map[string]string{}
		for i := range n {
			m[fmt.Sprintf("k%d", i)] = "v"
		}
		return m
	}
	carriers := map[string]func(map[string]string) error{
		"product": func(m map[string]string) error {
			return (&CreateProductParams{Name: "Pro", PriceUSD: "1", Metadata: m}).Validate()
		},
		"product upd": func(m map[string]string) error { return (&UpdateProductParams{Metadata: m}).Validate() },
		"customer": func(m map[string]string) error {
			return (&CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Metadata: m}).Validate()
		},
		"customer upd": func(m map[string]string) error { return (&UpdateCustomerParams{Metadata: m}).Validate() },
		"payment": func(m map[string]string) error {
			return (&CreatePaymentSessionParams{ProductUUID: memberUUID, Metadata: m}).Validate()
		},
		"subscription": func(m map[string]string) error {
			return (&CreateSubscriptionSessionParams{ProductUUID: memberUUID, Metadata: m}).Validate()
		},
		"subscription upd": func(m map[string]string) error { return (&UpdateSubscriptionParams{Metadata: m}).Validate() },
	}
	for name, validate := range carriers {
		check(t, name+" nil", validate(nil))
		check(t, name+" 50 keys", validate(keys(50)))
		check(t, name+" 51 keys", validate(keys(51)), "metadata")
		check(t, name+" good", validate(map[string]string{"orderId": "order-1042", "crm:customer": "C-5521", "a.b_c-d": "é ☃ 😀", "x": ""}))
		check(t, name+" key 40", validate(map[string]string{strings.Repeat("k", 40): "v"}))
		check(t, name+" key 41", validate(map[string]string{strings.Repeat("k", 41): "v"}), "metadata")
		check(t, name+" empty key", validate(map[string]string{"": "v"}), "metadata")
		check(t, name+" key with space", validate(map[string]string{"order id": "v"}), "metadata")
		check(t, name+" key with slash", validate(map[string]string{"a/b": "v"}), "metadata")
		check(t, name+" key non-ASCII", validate(map[string]string{"clé": "v"}), "metadata")
		check(t, name+" value 500", validate(map[string]string{"k": strings.Repeat("😀", 500)}))
		check(t, name+" value 501", validate(map[string]string{"k": strings.Repeat("é", 501)}), "metadata.k")
		check(t, name+" value NUL", validate(map[string]string{"k": "a\x00b"}), "metadata.k")
		check(t, name+" value tab", validate(map[string]string{"k": "a\tb"}), "metadata.k")
		check(t, name+" value newline", validate(map[string]string{"k": "a\nb"}), "metadata.k")
		check(t, name+" value bidi", validate(map[string]string{"k": "a\u202eb"}), "metadata.k")
		check(t, name+" value bidi isolate", validate(map[string]string{"k": "a\u2066b"}), "metadata.k")
		// Several bad keys are one error on metadata (the keys not echoed); bad values each.
		err := validate(map[string]string{"bad key": "v", "worse/key": "v", "a": "\x01", "b": "\x02"})
		check(t, name+" several", err, "metadata.a", "metadata.b", "metadata")
		if err != nil && (strings.Contains(err.Error(), "bad key") || strings.Contains(err.Error(), "worse")) {
			t.Errorf("%s: a bad key is echoed: %v", name, err)
		}
	}
	// On an update a "" value is sent: it deletes the key.
	b, _ := json.Marshal(UpdateCustomerParams{Metadata: map[string]string{"crm": "", "plan": "pro"}})
	if string(b) != `{"metadata":{"crm":"","plan":"pro"}}` {
		t.Errorf("update metadata %s", b)
	}
}

// TestRuleCursor checks the cursor of every paginated list: at most 200 characters, opaque.
func TestRuleCursor(t *testing.T) {
	ok, long := strings.Repeat("c", 200), strings.Repeat("c", 201)
	lists := map[string]func(string) error{
		"products":      func(c string) error { return (&ProductListParams{Cursor: c}).Validate() },
		"customers":     func(c string) error { return (&CustomerListParams{Cursor: c}).Validate() },
		"payments":      func(c string) error { return (&PaymentListParams{Cursor: c}).Validate() },
		"transactions":  func(c string) error { return (&TransactionListParams{Cursor: c}).Validate() },
		"failures":      func(c string) error { return (&FailureListParams{Cursor: c}).Validate() },
		"subscriptions": func(c string) error { return (&SubscriptionListParams{Cursor: c}).Validate() },
		"bills":         func(c string) error { return (&BillListParams{Cursor: c}).Validate() },
		"refunds":       func(c string) error { return (&RefundListParams{Cursor: c}).Validate() },
		"members":       func(c string) error { return (&MemberListParams{Cursor: c}).Validate() },
		"invitations":   func(c string) error { return (&InvitationListParams{Cursor: c}).Validate() },
		"endpoints":     func(c string) error { return (&EndpointListParams{Cursor: c}).Validate() },
		"events":        func(c string) error { return (&EventListParams{Cursor: c}).Validate() },
	}
	for name, validate := range lists {
		check(t, name+" opaque", validate("djF8MDE5Y2FkZmQtODkwMA=="))
		check(t, name+" 200", validate(ok))
		check(t, name+" 200 code points", validate(strings.Repeat("é", 200)))
		check(t, name+" 201", validate(long), "cursor")
	}
}

func TestRuleDuration(t *testing.T) {
	month := Duration{1, DurationUnitMonths}
	cases := []struct {
		name  string
		terms SubscriptionTermsParams
		want  []string
	}{
		{"monthly", SubscriptionTermsParams{Frequency: &month}, nil},
		{"1 year", SubscriptionTermsParams{Frequency: &Duration{1, DurationUnitYears}}, nil},
		{"365 days", SubscriptionTermsParams{Frequency: &Duration{365, DurationUnitDays}}, nil},
		{"52 weeks", SubscriptionTermsParams{Frequency: &Duration{52, DurationUnitWeeks}}, nil},
		{"12 months", SubscriptionTermsParams{Frequency: &Duration{12, DurationUnitMonths}}, nil},
		{"5 seconds (mode minimum is the API's)", SubscriptionTermsParams{Frequency: &Duration{5, DurationUnitSeconds}}, nil},
		{"13 months > 1 year", SubscriptionTermsParams{Frequency: &Duration{13, DurationUnitMonths}}, []string{"subscription.frequency"}},
		{"2 years", SubscriptionTermsParams{Frequency: &Duration{2, DurationUnitYears}}, []string{"subscription.frequency"}},
		{"max uint32 seconds", SubscriptionTermsParams{Frequency: &Duration{math.MaxUint32, DurationUnitYears}}, []string{"subscription.frequency"}},
		{"frequency 0", SubscriptionTermsParams{Frequency: &Duration{0, DurationUnitDays}}, []string{"subscription.frequency.value"}},
		{"no unit", SubscriptionTermsParams{Frequency: &Duration{Value: 3}}, []string{"subscription.frequency.unit"}},
		{"bad unit", SubscriptionTermsParams{Frequency: &Duration{1, "fortnights"}}, []string{"subscription.frequency.unit"}},
		{"frequency required on create", SubscriptionTermsParams{}, []string{"subscription.frequency"}},
		{"trial 0 without unit", SubscriptionTermsParams{Frequency: &month, TrialPeriod: &Duration{}}, nil},
		{"trial 14 days", SubscriptionTermsParams{Frequency: &month, TrialPeriod: &Duration{14, DurationUnitDays}}, nil},
		{"trial no unit", SubscriptionTermsParams{Frequency: &month, TrialPeriod: &Duration{Value: 14}}, []string{"subscription.trialPeriod.unit"}},
		{"trial 0 bad unit", SubscriptionTermsParams{Frequency: &month, TrialPeriod: &Duration{0, "x"}}, []string{"subscription.trialPeriod.unit"}},
		{"minPeriods 1000", SubscriptionTermsParams{Frequency: &month, MinPeriods: ptr(uint32(1000))}, nil},
		{"minPeriods 0", SubscriptionTermsParams{Frequency: &month, MinPeriods: ptr(uint32(0))}, nil},
		{"minPeriods 1001", SubscriptionTermsParams{Frequency: &month, MinPeriods: ptr(uint32(1001))}, []string{"subscription.minPeriods"}},
	}
	for _, tc := range cases {
		terms := tc.terms
		check(t, tc.name, (&CreateProductParams{Name: "Pro", PriceUSD: "1", Subscription: &terms}).Validate(), tc.want...)
	}

	// On a session the terms are top-level and optional.
	check(t, "session no terms", (&CreateSubscriptionSessionParams{ProductUUID: memberUUID}).Validate())
	check(t, "session terms", (&CreateSubscriptionSessionParams{ProductUUID: memberUUID, Frequency: &Duration{Value: 1}, MinPeriods: ptr(uint32(2000))}).Validate(),
		"frequency.unit", "minPeriods")
	// On an update the frequency is optional.
	check(t, "update trial only", (&UpdateProductParams{Subscription: &SubscriptionTermsParams{TrialPeriod: &Duration{}}}).Validate())
}

func TestRuleSession(t *testing.T) {
	cases := []struct {
		name string
		p    CreatePaymentSessionParams
		want []string
	}{
		{"by uuid", CreatePaymentSessionParams{ProductUUID: memberUUID}, nil},
		{"by reference", CreatePaymentSessionParams{ProductReference: "pro-plan"}, nil},
		{"inline", CreatePaymentSessionParams{ProductName: "Pro", PriceUSD: "10"}, nil},
		{"inline + description", CreatePaymentSessionParams{ProductName: "Pro", PriceUSD: "10", Description: "Lifetime"}, nil},
		{"nothing", CreatePaymentSessionParams{}, []string{"productUuid"}},
		{"uuid + reference", CreatePaymentSessionParams{ProductUUID: memberUUID, ProductReference: "x"}, []string{"productUuid"}},
		{"uuid + inline", CreatePaymentSessionParams{ProductUUID: memberUUID, ProductName: "Pro", PriceUSD: "1"}, []string{"productUuid"}},
		{"reference + price", CreatePaymentSessionParams{ProductReference: "pro", PriceUSD: "1"}, []string{"productUuid"}},
		{"uuid + description", CreatePaymentSessionParams{ProductUUID: memberUUID, Description: "x y"}, []string{"productUuid"}},
		{"inline without price", CreatePaymentSessionParams{ProductName: "Pro"}, []string{"priceUsd"}},
		{"inline without name", CreatePaymentSessionParams{PriceUSD: "3"}, []string{"productName"}},
		{"inline without name, bad price", CreatePaymentSessionParams{PriceUSD: "-3"}, []string{"productName", "priceUsd"}},
		{"inline negative price", CreatePaymentSessionParams{ProductName: "Pro", PriceUSD: "-3"}, []string{"priceUsd"}},
		{"bad uuid", CreatePaymentSessionParams{ProductUUID: "42"}, []string{"productUuid"}},
		{"bad customer uuid", CreatePaymentSessionParams{ProductUUID: memberUUID, CustomerUUID: "x"}, []string{"customerUuid"}},
		{"expires 10", CreatePaymentSessionParams{ProductUUID: memberUUID, ExpiresInMinutes: 10}, nil},
		{"expires 1440", CreatePaymentSessionParams{ProductUUID: memberUUID, ExpiresInMinutes: 1440}, nil},
		{"expires 9", CreatePaymentSessionParams{ProductUUID: memberUUID, ExpiresInMinutes: 9}, []string{"expiresInMinutes"}},
		{"expires 1441", CreatePaymentSessionParams{ProductUUID: memberUUID, ExpiresInMinutes: 1441}, []string{"expiresInMinutes"}},
	}
	for _, tc := range cases {
		check(t, tc.name, tc.p.Validate(), tc.want...)
	}
	check(t, "subscription inline", (&CreateSubscriptionSessionParams{ProductName: "Pro", PriceUSD: "9.99", Frequency: &Duration{1, DurationUnitMonths}}).Validate())
	check(t, "subscription mixing", (&CreateSubscriptionSessionParams{ProductReference: "pro", PriceUSD: "9.99"}).Validate(), "productUuid")
	check(t, "nil params", (*CreatePaymentSessionParams)(nil).Validate())
}

func TestRuleIDs(t *testing.T) {
	for _, id := range []string{memberUUID, "019ECA82-5680-7B00-8000-0000000000B1", "pay@" + memberUUID, "sub@" + memberUUID, "payg@" + memberUUID, "sub-hist@" + memberUUID, "refund@" + memberUUID, "transfer@" + memberUUID, "release@" + memberUUID} {
		if !isTxID(id) {
			t.Errorf("isTxID(%q) = false", id)
		}
	}
	for _, id := range []string{"", "pay@", "pay@x", "evt@" + memberUUID, "@" + memberUUID, "PAY@" + memberUUID, memberUUID + "0", "42"} {
		if isTxID(id) {
			t.Errorf("isTxID(%q) = true", id)
		}
	}
	check(t, "txUuid bad", (&RefundListParams{TxUUID: "order-1"}).Validate(), "txUuid")
	check(t, "subscriptionUuid filter", (&FailureListParams{SubscriptionUUID: "sub@nope"}).Validate(), "subscriptionUuid")

	if checkPathUUID("uuid", memberUUID) != nil || checkPathUUID("uuid", "") == nil || checkPathUUID("uuid", "pay@"+memberUUID) == nil {
		t.Error("checkPathUUID")
	}
	if checkPathTxID("uuid", "sub@"+memberUUID) != nil || checkPathTxID("uuid", "") == nil || checkPathTxID("uuid", "x") == nil {
		t.Error("checkPathTxID")
	}
	check(t, "refunds txUuid filter", (&RefundListParams{TxUUID: "pay-1"}).Validate(), "txUuid")
	check(t, "refunds txUuid filter ok", (&RefundListParams{TxUUID: "sub-hist@" + memberUUID}).Validate())
	check(t, "transactions subscriptionUuid", (&TransactionListParams{SubscriptionUUID: "sub@x"}).Validate(), "subscriptionUuid")
	// Path UUIDs accept the nil UUID (the API answers 404); only On-Behalf-Of refuses it.
	if checkPathUUID("uuid", "00000000-0000-0000-0000-000000000000") != nil {
		t.Error("checkPathUUID refuses the nil UUID")
	}
	if checkPathRequired("id", "evt_a/b") != nil || checkPathRequired("id", " ") == nil || checkPathRequired("id", "..") == nil {
		t.Error("checkPathRequired")
	}
}

func TestRuleFilters(t *testing.T) {
	check(t, "exclusive", (&PaymentListParams{IncludeMembers: true, UserUUID: memberUUID}).Validate(), "userUuid")
	check(t, "user only", (&PaymentListParams{UserUUID: memberUUID}).Validate())
	check(t, "bad uuids", (&SubscriptionListParams{CustomerUUID: "x", ProductUUID: "y", UserUUID: "z"}).Validate(), "customerUuid", "productUuid", "userUuid")
	// Subscription status and events type are not checked client-side: a newer value is sent.
	check(t, "subscription status newer", (&SubscriptionListParams{Status: "hibernating"}).Validate())
	check(t, "subscription status ok", (&SubscriptionListParams{Status: SubscriptionStatusCancelled}).Validate())
	check(t, "event type newer", (&EventListParams{Type: "payment.done"}).Validate())
	check(t, "event type ok", (&EventListParams{Type: EventTypeWebhookTest}).Validate())
	// The enum filters the API enforces are checked against the known values.
	check(t, "transactions type", (&TransactionListParams{Type: "subscriptionHistory"}).Validate(), "type")
	check(t, "transactions type refund", (&TransactionListParams{Type: "refund"}).Validate(), "type")
	check(t, "transactions type ok", (&TransactionListParams{Type: TransactionKindBill}).Validate())
	check(t, "transactions exclusive", (&TransactionListParams{IncludeMembers: true, UserUUID: memberUUID}).Validate(), "userUuid")
	check(t, "kind/category", (&FailureListParams{Kind: "refund", Category: "boom"}).Validate(), "kind", "category")
	check(t, "invitation status", (&InvitationListParams{Status: "open"}).Validate(), "status")
	all := []RefundStatus{RefundStatusPending, RefundStatusApproved, RefundStatusRejected, RefundStatusFailed, RefundStatusPending}
	check(t, "refund statuses ok", (&RefundListParams{Status: all}).Validate())
	check(t, "refund 6 statuses", (&RefundListParams{Status: append(all, RefundStatusPending)}).Validate(), "status")
	check(t, "refund unknown status", (&RefundListParams{Status: []RefundStatus{RefundStatusApproved, "completed", ""}}).Validate(), "status[1]", "status[2]")
	check(t, "refund exclusive", (&RefundListParams{IncludeMembers: ptr(true), UserUUID: memberUUID}).Validate(), "userUuid")
	check(t, "refund includeMembers false + user", (&RefundListParams{IncludeMembers: ptr(false), UserUUID: memberUUID}).Validate())
	check(t, "accepted currencies", (&AcceptedCurrenciesParams{UserUUID: "x"}).Validate(), "userUuid")
	for _, p := range []interface{ Validate() error }{(*PaymentListParams)(nil), (*CustomerListParams)(nil), (*EventListParams)(nil), (*RefundListParams)(nil),
		(*ProductListParams)(nil), (*TransactionListParams)(nil), (*EndpointListParams)(nil), (*BillListParams)(nil), (*MemberListParams)(nil), (*AcceptedCurrenciesParams)(nil)} {
		if p.Validate() != nil {
			t.Errorf("nil %T must be valid", p)
		}
	}
}

func TestRuleDates(t *testing.T) {
	var v validator
	v.dateRange("from", "2026-06-01", "to", "2026-06-01")
	v.dateRange("from", "2024-02-29", "to", "2026-12-31") // no window check (D5)
	if v.err() != nil {
		t.Errorf("valid ranges: %v", v.err())
	}
	for _, r := range [][2]string{{"2026-06-02", "2026-06-01"}, {"2026-13-01", "2026-12-01"}, {"2025-02-29", "2025-03-01"}, {"26-01-01", "2026-01-01"}, {"2026-1-1", "2026-01-02"}} {
		var v validator
		v.dateRange("from", r[0], "to", r[1])
		if v.err() == nil {
			t.Errorf("range %v accepted", r)
		}
	}
}

func TestRuleEndpointEvents(t *testing.T) {
	twenty := make([]EventType, 20)
	for i := range twenty {
		twenty[i] = EventTypePaymentCompleted
	}
	check(t, "20 events", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: twenty}).Validate())
	check(t, "21 events", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: append(twenty, EventTypeRefundRejected)}).Validate(), "events")
	check(t, "webhook.test", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{EventTypeRefundRejected, EventTypeWebhookTest}}).Validate(), "events[1]")
	check(t, "empty entry", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{EventTypeRefundApproved, ""}}).Validate(), "events[1]")
	check(t, "update webhook.test", (&UpdateWebhookEndpointParams{Events: []EventType{EventTypeWebhookTest}}).Validate(), "events[0]")
	check(t, "unknown type passes (the API judges)", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{"future.event"}}).Validate())
	check(t, "payloadVersion v1", (&UpdateWebhookEndpointParams{PayloadVersion: "v1"}).Validate(), "payloadVersion")
	check(t, "payloadVersion v2", (&UpdateWebhookEndpointParams{PayloadVersion: WebhookPayloadVersionV2}).Validate())
}

func TestRuleHeaders(t *testing.T) {
	for _, k := range []string{"a", "order-1042", "~!@#$%^&*()_+{}|:<>?", strings.Repeat("k", 255)} {
		if !isIdempotencyKey(k) {
			t.Errorf("isIdempotencyKey(%q) = false", k)
		}
	}
	for _, k := range []string{"", "a b", "tab\t", "é", "del\x7f", strings.Repeat("k", 256)} {
		if isIdempotencyKey(k) {
			t.Errorf("isIdempotencyKey(%q) = true", k)
		}
	}
	for _, id := range []string{"a", "sdk-probe.123:abc", "A_b-C.d:E", strings.Repeat("r", 128)} {
		if !isRequestID(id) {
			t.Errorf("isRequestID(%q) = false", id)
		}
	}
	for _, id := range []string{"", "a b", "a/b", "a@b", strings.Repeat("r", 129)} {
		if isRequestID(id) {
			t.Errorf("isRequestID(%q) = true", id)
		}
	}
}

// TestUpdateSemantics checks what update params send: nil is omitted, a pointer to "" is sent
// (clears), the other "" values are omitted.
func TestUpdateSemantics(t *testing.T) {
	cases := []struct {
		params any
		want   string
	}{
		{UpdateCustomerParams{}, `{}`},
		{UpdateCustomerParams{LastName: ptr(""), PhoneNumber: ptr(""), Address: ptr("")}, `{"lastName":"","phoneNumber":"","address":""}`},
		{UpdateCustomerParams{FirstName: "Ada", PhoneNumber: ptr("+33 6 12 34 56 78")}, `{"firstName":"Ada","phoneNumber":"+33 6 12 34 56 78"}`},
		{UpdateCustomerParams{Metadata: map[string]string{"crm": ""}}, `{"metadata":{"crm":""}}`},
		{UpdateProductParams{}, `{}`},
		{UpdateProductParams{Description: ptr("")}, `{"description":""}`},
		{UpdateProductParams{PriceUSD: "9.99", IsActive: ptr(false), RemoveSubscription: true}, `{"priceUsd":"9.99","isActive":false,"removeSubscription":true}`},
		{UpdateSubscriptionParams{}, `{}`},
		{UpdateSubscriptionParams{Metadata: map[string]string{"plan": "", "seats": "5"}}, `{"metadata":{"plan":"","seats":"5"}}`},
		{UpdateWebhookEndpointParams{}, `{}`},
		{UpdateWebhookEndpointParams{Description: ptr(""), Events: []EventType{}}, `{"events":[],"description":""}`},
		{UpdateWebhookEndpointParams{Enabled: ptr(false), PayloadVersion: WebhookPayloadVersionV2}, `{"payloadVersion":"v2","enabled":false}`},
		{UpdateMemberParams{OrganizationFeePercent: "0"}, `{"organizationFeePercent":"0"}`},
		{CreateInvitationParams{Email: "a@b.co"}, `{"email":"a@b.co","role":"user","holdFunds":false}`},
		{CreateInvitationParams{Email: "a@b.co", HoldFunds: true, OrganizationFeePercent: "2.5", RedirectURL: "https://x.io"}, `{"email":"a@b.co","role":"user","holdFunds":true,"organizationFeePercent":"2.5","redirectUrl":"https://x.io"}`},
		{CreateProductParams{Name: "Pro", PriceUSD: "10"}, `{"name":"Pro","priceUsd":"10"}`},
		{CreateProductParams{Name: "Pro", PriceUSD: "10.50", Subscription: &SubscriptionTermsParams{Frequency: &Duration{1, DurationUnitMonths}, TrialPeriod: &Duration{}, MinPeriods: ptr(uint32(0))}},
			`{"name":"Pro","priceUsd":"10.50","subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":0},"minPeriods":0}}`},
		{CreatePaymentSessionParams{ProductUUID: memberUUID}, `{"productUuid":"` + memberUUID + `"}`},
		{CreatePaymentSessionParams{ProductUUID: memberUUID, Fees: &CheckoutFees{Items: []FeeItem{{Label: "Tax", AmountUSD: "1.30"}}}},
			`{"productUuid":"` + memberUUID + `","fees":{"items":[{"label":"Tax","amountUsd":"1.30"}]}}`},
		{CreateSubscriptionSessionParams{ProductName: "Pro", PriceUSD: "9.99", Frequency: &Duration{1, DurationUnitWeeks}, ExpiresInMinutes: 30},
			`{"productName":"Pro","priceUsd":"9.99","expiresInMinutes":30,"frequency":{"value":1,"unit":"weeks"}}`},
		{CreateCustomerParams{FirstName: "Ada", Email: "a@b.co", Metadata: map[string]string{"k": "v"}}, `{"firstName":"Ada","email":"a@b.co","metadata":{"k":"v"}}`},
		{CreateWebhookEndpointParams{URL: "https://x.io", IncludeMembers: ptr(false)}, `{"url":"https://x.io","includeMembers":false}`},
		{verifyWebhookBody{EndpointUUID: "e", Body: "{}", Signature: "t=1,v1=x"}, `{"endpointUuid":"e","body":"{}","signature":"t=1,v1=x"}`},
	}
	for i, tc := range cases {
		b, err := json.Marshal(tc.params)
		if err != nil || string(b) != tc.want {
			t.Errorf("case %d (%T): %s, want %s (%v)", i, tc.params, b, tc.want, err)
		}
	}

	// Validation follows the same semantics: a clearing "" passes, nil passes.
	check(t, "clear lastName/phone/address", (&UpdateCustomerParams{LastName: ptr(""), PhoneNumber: ptr(""), Address: ptr("")}).Validate())
	check(t, "bad phone", (&UpdateCustomerParams{PhoneNumber: ptr("abc")}).Validate(), "phoneNumber")
	check(t, "blank address", (&UpdateCustomerParams{Address: ptr("   ")}).Validate(), "address")
	check(t, "clear description", (&UpdateProductParams{Description: ptr("")}).Validate())
	check(t, "short description", (&UpdateProductParams{Description: ptr("x")}).Validate(), "description")
	check(t, "remove + set", (&UpdateProductParams{RemoveSubscription: true, Subscription: &SubscriptionTermsParams{}}).Validate(), "removeSubscription")
	check(t, "clear endpoint description", (&UpdateWebhookEndpointParams{Description: ptr("")}).Validate())
	check(t, "nil update", (*UpdateCustomerParams)(nil).Validate())
}

func TestValidationCollectsAllFields(t *testing.T) {
	err := (&CreateCustomerParams{FirstName: "x", LastName: "<", Email: "nope", PhoneNumber: "1", Address: " ", Reference: "a b",
		Metadata: map[string]string{"k": "\x7f"}}).Validate()
	check(t, "all", err, "firstName", "lastName", "email", "phoneNumber", "address", "reference", "metadata.k")
	if !strings.HasPrefix(err.Error(), "validation failed; firstName: firstName must be 2 to 100 characters") {
		t.Errorf("message %q", err.Error())
	}
}
