package qbitflow

import (
	"encoding/json"
	"errors"
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
		check(t, "name "+s, (&CreateProductParams{Name: s, Price: 1}).Validate())
	}
	for _, s := range bad {
		check(t, "name "+s, (&CreateProductParams{Name: s, Price: 1}).Validate(), "name")
	}
	check(t, "lastName 1 char", (&CreateCustomerParams{Name: "Ada", LastName: "L", Email: "a@b.co"}).Validate())
	check(t, "lastName newline", (&CreateCustomerParams{Name: "Ada", LastName: "L\n", Email: "a@b.co"}).Validate(), "lastName")
}

func TestRuleText(t *testing.T) {
	good := []string{"Two\nlines\r\nand\ttab", "Lifetime access (Pro) & more: 100%", "ok"}
	bad := []string{"   ", "\n\n", "<script>", "x\x07bell", strings.Repeat("x", 501), "a"}
	for _, s := range good {
		check(t, "description", (&CreateProductParams{Name: "Pro", Price: 1, Description: s}).Validate())
	}
	for _, s := range bad {
		check(t, "description", (&CreateProductParams{Name: "Pro", Price: 1, Description: s}).Validate(), "description")
	}
	// ≤500 without a minimum, and the endpoint description ≤200.
	check(t, "address 1 char", (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", Address: "x"}).Validate())
	check(t, "address 501", (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", Address: strings.Repeat("x", 501)}).Validate(), "address")
	check(t, "endpoint description 201", (&CreateWebhookEndpointParams{URL: "https://x.io", Description: strings.Repeat("x", 201)}).Validate(), "description")
	check(t, "refund reason", (&InitiateRefundParams{TxUUID: "pay@" + memberUUID, Reason: "Broken, can you refund?", MerchantMessage: "Sorry\nTeam"}).Validate())
	check(t, "refund reason bad", (&InitiateRefundParams{TxUUID: "pay@" + memberUUID, Reason: "{x}", MerchantMessage: strings.Repeat("m", 501)}).Validate(), "reason", "merchantMessage")
}

func TestRuleReference(t *testing.T) {
	for _, s := range []string{"order-1042", "INV_2026.10:01@shop", "a", strings.Repeat("r", 100)} {
		check(t, "reference "+s, (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", Reference: s}).Validate())
	}
	for _, s := range []string{"has space", "slash/x", "é", "#1", strings.Repeat("r", 101)} {
		check(t, "reference "+s, (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", Reference: s}).Validate(), "reference")
	}
	check(t, "session refs", (&CreatePaymentSessionParams{ProductReference: "bad ref", Reference: "o/1", CustomerReference: "c 1"}).Validate(), "reference", "productReference", "customerReference")
	check(t, "subscription filter", (&SubscriptionListParams{Reference: "bad ref"}).Validate(), "reference")
}

func TestRulePhone(t *testing.T) {
	for _, s := range []string{"+33 6 12 34 56 78", "555 (123) 4567", "06.12.34.56.78", "123456"} {
		check(t, "phone "+s, (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", PhoneNumber: s}).Validate())
	}
	for _, s := range []string{"12345", "+", "phone", "+33 6 12 34 56 7x", "-123456", "123456-", "+" + strings.Repeat("1", 32)} {
		check(t, "phone "+s, (&CreateCustomerParams{Name: "Ada", Email: "a@b.co", PhoneNumber: s}).Validate(), "phoneNumber")
	}
}

func TestRuleEmail(t *testing.T) {
	for _, s := range []string{"a@b.co", "First.Last+tag@Example.COM", "x@sub.domain.io"} {
		check(t, "email "+s, (&CreateCustomerParams{Name: "Ada", Email: s}).Validate())
	}
	for _, s := range []string{"no-at", "a@b", "@b.co", "a@.b.co", "a@b.co.", "a b@c.io", "a@@b.co", "a@b@c.io", strings.Repeat("a", 250) + "@b.co"} {
		check(t, "email "+s, (&CreateCustomerParams{Name: "Ada", Email: s}).Validate(), "email")
	}
	check(t, "email required", (&CreateCustomerParams{Name: "Ada"}).Validate(), "email")
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

func TestRulePrice(t *testing.T) {
	for _, p := range []float64{0.01, 5, 1e6} {
		check(t, "price", (&CreateProductParams{Name: "Pro", Price: p}).Validate())
	}
	for _, p := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		check(t, "price", (&CreateProductParams{Name: "Pro", Price: p}).Validate(), "price")
	}
	check(t, "update price", (&UpdateProductParams{Price: ptr(0.0)}).Validate(), "price")
	check(t, "update no price", (&UpdateProductParams{}).Validate())
}

func TestRulePercent(t *testing.T) {
	for _, p := range []float64{0.01, 1.5, 33.33, 100} {
		check(t, "refund", (&InitiateRefundParams{TxUUID: memberUUID, RefundPercent: &p}).Validate())
	}
	tenth := 0.1
	for _, p := range []float64{0, -1, 100.01, 1.155, tenth + 0.2, math.NaN(), math.Inf(1)} { // 0.1+0.2 at run time is 0.30000000000000004
		check(t, "refund", (&InitiateRefundParams{TxUUID: memberUUID, RefundPercent: &p}).Validate(), "refundPercent")
	}
	for _, p := range []float64{0, 0.05, 1.5, 50} {
		check(t, "member fee", (&UpdateMemberParams{OrganizationFeePercent: p}).Validate())
		check(t, "invitation fee", (&CreateInvitationParams{Email: "a@b.co", OrganizationFeePercent: &p}).Validate())
	}
	for _, p := range []float64{-0.01, 50.01, 1.234} {
		check(t, "member fee", (&UpdateMemberParams{OrganizationFeePercent: p}).Validate(), "organizationFeePercent")
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
		check(t, tc.name, (&CreateProductParams{Name: "Pro", Price: 1, Subscription: &terms}).Validate(), tc.want...)
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
		{"inline", CreatePaymentSessionParams{ProductName: "Pro", Price: 10}, nil},
		{"inline + description", CreatePaymentSessionParams{ProductName: "Pro", Price: 10, Description: "Lifetime"}, nil},
		{"nothing", CreatePaymentSessionParams{}, []string{"productUuid"}},
		{"uuid + reference", CreatePaymentSessionParams{ProductUUID: memberUUID, ProductReference: "x"}, []string{"productUuid"}},
		{"uuid + inline", CreatePaymentSessionParams{ProductUUID: memberUUID, ProductName: "Pro", Price: 1}, []string{"productUuid"}},
		{"uuid + description", CreatePaymentSessionParams{ProductUUID: memberUUID, Description: "x y"}, []string{"productUuid"}},
		{"inline without price", CreatePaymentSessionParams{ProductName: "Pro"}, []string{"price"}},
		{"inline without name", CreatePaymentSessionParams{Price: 3}, []string{"productName"}},
		{"inline negative price", CreatePaymentSessionParams{ProductName: "Pro", Price: -3}, []string{"price"}},
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
	check(t, "subscription inline", (&CreateSubscriptionSessionParams{ProductName: "Pro", Price: 9.99, Frequency: &Duration{1, DurationUnitMonths}}).Validate())
	check(t, "subscription mixing", (&CreateSubscriptionSessionParams{ProductReference: "pro", Price: 9.99}).Validate(), "productUuid")
	check(t, "nil params", (*CreatePaymentSessionParams)(nil).Validate())
}

func TestRuleIDs(t *testing.T) {
	for _, id := range []string{memberUUID, "019ECA82-5680-7B00-8000-0000000000B1", "pay@" + memberUUID, "sub@" + memberUUID, "payg@" + memberUUID, "sub-hist@" + memberUUID, "refund@" + memberUUID, "transfer@" + memberUUID} {
		if !isTxID(id) {
			t.Errorf("isTxID(%q) = false", id)
		}
	}
	for _, id := range []string{"", "pay@", "pay@x", "evt@" + memberUUID, "@" + memberUUID, "PAY@" + memberUUID, memberUUID + "0", "42"} {
		if isTxID(id) {
			t.Errorf("isTxID(%q) = true", id)
		}
	}
	check(t, "txUuid required", (&InitiateRefundParams{}).Validate(), "txUuid")
	check(t, "txUuid bad", (&InitiateRefundParams{TxUUID: "order-1"}).Validate(), "txUuid")
	check(t, "subscriptionUuid filter", (&FailureListParams{SubscriptionUUID: "sub@nope"}).Validate(), "subscriptionUuid")

	if checkPathUUID("uuid", memberUUID) != nil || checkPathUUID("uuid", "") == nil || checkPathUUID("uuid", "pay@"+memberUUID) == nil {
		t.Error("checkPathUUID")
	}
	if checkPathTxID("uuid", "sub@"+memberUUID) != nil || checkPathTxID("uuid", "") == nil || checkPathTxID("uuid", "x") == nil {
		t.Error("checkPathTxID")
	}
	if checkPathRequired("reference", "a/b") != nil || checkPathRequired("reference", " ") == nil {
		t.Error("checkPathRequired")
	}
}

func TestRuleFilters(t *testing.T) {
	check(t, "exclusive", (&PaymentListParams{IncludeMembers: true, UserUUID: memberUUID}).Validate(), "userUuid")
	check(t, "user only", (&PaymentListParams{UserUUID: memberUUID}).Validate())
	check(t, "bad uuids", (&SubscriptionListParams{CustomerUUID: "x", ProductUUID: "y", UserUUID: "z"}).Validate(), "customerUuid", "productUuid", "userUuid")
	check(t, "status", (&SubscriptionListParams{Status: "hibernating"}).Validate(), "status")
	check(t, "status ok", (&SubscriptionListParams{Status: SubscriptionStatusCancelled}).Validate())
	check(t, "source", (&CombinedPaymentListParams{Source: "subscription_history"}).Validate(), "source")
	check(t, "kind/category", (&FailureListParams{Kind: "refund", Category: "boom"}).Validate(), "kind", "category")
	check(t, "invitation status", (&InvitationListParams{Status: "open"}).Validate(), "status")
	check(t, "event type", (&EventListParams{Type: "payment.done"}).Validate(), "type")
	check(t, "event type ok", (&EventListParams{Type: EventTypeWebhookTest}).Validate())
	check(t, "refund exclusive", (&RefundListParams{IncludeMembers: ptr(true), UserUUID: memberUUID}).Validate(), "userUuid")
	check(t, "refund includeMembers false + user", (&RefundListParams{IncludeMembers: ptr(false), UserUUID: memberUUID}).Validate())
	check(t, "supported currencies", (&SupportedCurrenciesParams{UserUUID: "x"}).Validate(), "userUuid")
	for _, p := range []interface{ Validate() error }{(*PaymentListParams)(nil), (*CustomerListParams)(nil), (*EventListParams)(nil), (*RefundListParams)(nil)} {
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
	check(t, "21 events", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: append(twenty, EventTypeRefundDenied)}).Validate(), "events")
	check(t, "webhook.test", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{EventTypeRefundDenied, EventTypeWebhookTest}}).Validate(), "events[1]")
	check(t, "update webhook.test", (&UpdateWebhookEndpointParams{Events: []EventType{EventTypeWebhookTest}}).Validate(), "events[0]")
	check(t, "unknown type passes (the API judges)", (&CreateWebhookEndpointParams{URL: "https://x.io", Events: []EventType{"future.event"}}).Validate())
	check(t, "payloadVersion v1", (&UpdateWebhookEndpointParams{PayloadVersion: WebhookPayloadVersionV1}).Validate(), "payloadVersion")
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
		{UpdateCustomerParams{PhoneNumber: ptr(""), Address: ptr("")}, `{"phoneNumber":"","address":""}`},
		{UpdateCustomerParams{Name: "Ada", PhoneNumber: ptr("+33 6 12 34 56 78")}, `{"name":"Ada","phoneNumber":"+33 6 12 34 56 78"}`},
		{UpdateProductParams{}, `{}`},
		{UpdateProductParams{Description: ptr("")}, `{"description":""}`},
		{UpdateProductParams{Price: ptr(9.99), IsActive: ptr(false), RemoveSubscription: true}, `{"price":9.99,"isActive":false,"removeSubscription":true}`},
		{UpdateWebhookEndpointParams{}, `{}`},
		{UpdateWebhookEndpointParams{Description: ptr(""), Events: []EventType{}}, `{"events":[],"description":""}`},
		{UpdateWebhookEndpointParams{Enabled: ptr(false), PayloadVersion: WebhookPayloadVersionV2}, `{"payloadVersion":"v2","enabled":false}`},
		{UpdateMemberParams{}, `{"organizationFeePercent":0}`},
		{CreateInvitationParams{Email: "a@b.co"}, `{"email":"a@b.co","role":"user","trustLayer":false}`},
		{CreateInvitationParams{Email: "a@b.co", TrustLayer: true, OrganizationFeePercent: ptr(2.5), RedirectURL: "https://x.io"}, `{"email":"a@b.co","role":"user","trustLayer":true,"organizationFeePercent":2.5,"redirectUrl":"https://x.io"}`},
		{CreateProductParams{Name: "Pro", Price: 10}, `{"name":"Pro","price":10}`},
		{CreateProductParams{Name: "Pro", Price: 10, Subscription: &SubscriptionTermsParams{Frequency: &Duration{1, DurationUnitMonths}, TrialPeriod: &Duration{}, MinPeriods: ptr(uint32(0))}},
			`{"name":"Pro","price":10,"subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":0},"minPeriods":0}}`},
		{CreatePaymentSessionParams{ProductUUID: memberUUID}, `{"productUuid":"` + memberUUID + `"}`},
		{CreateSubscriptionSessionParams{ProductName: "Pro", Price: 9.99, Frequency: &Duration{1, DurationUnitWeeks}, ExpiresInMinutes: 30},
			`{"productName":"Pro","price":9.99,"expiresInMinutes":30,"frequency":{"value":1,"unit":"weeks"}}`},
		{InitiateRefundParams{TxUUID: "pay@x"}, `{"txUuid":"pay@x"}`},
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
	check(t, "clear phone/address", (&UpdateCustomerParams{PhoneNumber: ptr(""), Address: ptr("")}).Validate())
	check(t, "bad phone", (&UpdateCustomerParams{PhoneNumber: ptr("abc")}).Validate(), "phoneNumber")
	check(t, "blank address", (&UpdateCustomerParams{Address: ptr("   ")}).Validate(), "address")
	check(t, "clear description", (&UpdateProductParams{Description: ptr("")}).Validate())
	check(t, "short description", (&UpdateProductParams{Description: ptr("x")}).Validate(), "description")
	check(t, "remove + set", (&UpdateProductParams{RemoveSubscription: true, Subscription: &SubscriptionTermsParams{}}).Validate(), "removeSubscription")
	check(t, "clear endpoint description", (&UpdateWebhookEndpointParams{Description: ptr("")}).Validate())
	check(t, "nil update", (*UpdateCustomerParams)(nil).Validate())
}

func TestValidationCollectsAllFields(t *testing.T) {
	err := (&CreateCustomerParams{Name: "x", LastName: "<", Email: "nope", PhoneNumber: "1", Address: " ", Reference: "a b"}).Validate()
	check(t, "all", err, "name", "lastName", "email", "phoneNumber", "address", "reference")
	if !strings.HasPrefix(err.Error(), "validation failed; name: name must be 2 to 100 characters") {
		t.Errorf("message %q", err.Error())
	}
}
