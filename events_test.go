package qbitflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// loadEvent parses testdata/events/<type>.json: the docs' example body of each event type
// (the API reference's webhook examples).
func loadEvent(t *testing.T, eventType EventType) *Event {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "events", string(eventType)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent(%s): %v", eventType, err)
	}
	if e.Type != eventType || !strings.HasPrefix(e.ID, "evt_") || e.CreatedAt.IsZero() {
		t.Fatalf("envelope %+v", e)
	}
	return e
}

// strictModel maps each event type to the model its data must decode into with no unknown
// field: every key of the docs' examples is modelled.
var strictModel = map[EventType]func() any{
	EventTypePaymentCompleted:                  func() any { return new(PaymentCompleted) },
	EventTypeSubscriptionCreated:               func() any { return new(Subscription) },
	EventTypeSubscriptionBilled:                func() any { return new(SubscriptionBilled) },
	EventTypeSubscriptionStatusChanged:         func() any { return new(SubscriptionStatusChanged) },
	EventTypeSubscriptionActionRequiredChanged: func() any { return new(SubscriptionActionRequiredChanged) },
	EventTypeSubscriptionBillingFailed:         func() any { return new(SubscriptionBillingFailed) },
	EventTypeSubscriptionUpcomingBill:          func() any { return new(SubscriptionUpcomingBill) },
	EventTypeRefundRequested:                   func() any { return new(Refund) },
	EventTypeRefundApproved:                    func() any { return new(Refund) },
	EventTypeRefundRejected:                    func() any { return new(Refund) },
	EventTypeMemberJoined:                      func() any { return new(MemberJoined) },
	EventTypeMemberRemoved:                     func() any { return new(Member) },
	EventTypeHeldFundsReleased:                 func() any { return new(HeldFundsReleased) },
	EventTypeCheckoutExpired:                   func() any { return new(PaymentSessionData) },
	EventTypeWebhookTest:                       func() any { return new(WebhookTest) },
}

// notModelled lists, per event type, the top-level data keys the API sends but the SDK
// deliberately leaves out (dashboard-only, or constant for an API key): decoding ignores them.
var notModelled = map[EventType][]string{
	EventTypeCheckoutExpired:   {"availableCurrencies", "userName", "organizationName"},
	EventTypeHeldFundsReleased: {"type"},
}

// withoutKeys returns data (a JSON object) without the given top-level keys (when present).
func withoutKeys(t *testing.T, data []byte, keys []string) []byte {
	t.Helper()
	if len(keys) == 0 {
		return data
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		delete(m, k)
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEventFixturesAreFullyModelled(t *testing.T) {
	if len(strictModel) != len(knownEventTypes) || len(knownEventTypes) != 15 {
		t.Fatalf("%d models for %d types", len(strictModel), len(knownEventTypes))
	}
	files, _ := filepath.Glob(filepath.Join("testdata", "events", "*.json"))
	if len(files) != len(knownEventTypes) {
		t.Errorf("%d fixtures for %d types", len(files), len(knownEventTypes))
	}
	for _, eventType := range knownEventTypes {
		e := loadEvent(t, eventType)
		dec := json.NewDecoder(bytes.NewReader(withoutKeys(t, e.Data, notModelled[eventType])))
		dec.DisallowUnknownFields()
		if err := dec.Decode(strictModel[eventType]()); err != nil {
			t.Errorf("%s: %v", eventType, err)
		}
		// validateEventData (the router's check) accepts every docs example.
		if err := validateEventData(e); err != nil {
			t.Errorf("%s: validateEventData: %v", eventType, err)
		}
	}
}

func TestEventPaymentCompleted(t *testing.T) {
	e := loadEvent(t, EventTypePaymentCompleted)
	p, err := e.PaymentCompleted()
	if err != nil {
		t.Fatal(err)
	}
	if p.UUID != "pay@01a0f755-f200-7000-8000-000000000001" || p.Amount != "10" || p.AmountMinUnits != "10000000" || p.AmountUSD != "10" ||
		p.Reference != "order-1042" || p.CustomerReference != "crm-5521" || p.Chain != ChainBASE || p.ProductName != "Pro plan (lifetime)" ||
		p.ManagementPageLink == "" || p.Paid != "10.0042" || p.PaidMinUnits != "10004200" || p.PaidUSD != "10.0042" ||
		p.Metadata["orderId"] != "order-1042" || p.Metadata["crm:customer"] != "C-5521" {
		t.Errorf("payment %+v", p)
	}
	if p.Currency.Code != "BASE:USDC" || p.Currency.Symbol != "USDC" || p.Currency.Chain != ChainBASE || p.Currency.MainCurrencyID == nil ||
		*p.Currency.MainCurrencyID != 3 || p.Currency.Address == "" {
		t.Errorf("currency %+v", p.Currency)
	}
	st := p.Settlement
	if st.FeePercent != "0.99" || st.Split == nil || st.Split.Merchant.AmountMinUnits != "9901000" || st.Split.Platform.Amount != "0.099" ||
		st.Split.NetworkFee == nil || st.Split.NetworkFee.Amount != "0.0042" || st.Split.Organization != nil || st.Split.Referral != nil ||
		st.OrganizationFee != nil || st.ReferralFee != nil || st.Transaction == nil || st.Transaction.BlockNumber != "52468103" ||
		st.Transaction.GasUsed == nil || *st.Transaction.GasUsed != 95000 || st.Transaction.NativePriceUSD != "4512.37" ||
		st.Transaction.L1Fee != nil || st.Transaction.Fee.AmountUSD != "0.005144" {
		t.Errorf("settlement %+v", st)
	}
	// Webhooks never carry the API-only fields.
	if p.Customer != nil || p.Refund != nil || p.Refundable != nil || p.NotRefundableReason != "" {
		t.Error("API-only fields must decode as absent")
	}
	if p.ConfirmedAt == nil || p.CheckoutOpenedAt == nil {
		t.Error("times")
	}
	// The docs' payment has a price and no fees: amountUsd = priceUsd.
	if p.PriceUSD != "10" || p.Fees == nil || len(p.Fees) != 0 || p.PriceUSD != p.AmountUSD {
		t.Errorf("price %v, fees %+v", p.PriceUSD, p.Fees)
	}
}

func TestEventSubscriptions(t *testing.T) {
	created, err := loadEvent(t, EventTypeSubscriptionCreated).SubscriptionCreated()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.UUID, "sub@") || created.Frequency != (Duration{1, DurationUnitMonths}) || created.Status != SubscriptionStatusActive ||
		created.PriceUSD != "10" || created.CurrentPeriodEnd == nil || created.NextBillingAt == nil || created.LastBilledAt == nil ||
		created.ManagementPageLink == "" || created.Metadata["plan"] != "pro" || created.Reference != "customer-7731" ||
		created.Allowance == nil || created.Allowance.Amount != "110" || *created.Allowance.RemainingPeriods != 11 ||
		created.Allowance.MaxPerPeriodMinUnits != "30000000" || created.Currency.Code != "BASE:USDC" || created.Dunning != nil {
		t.Errorf("created %+v", created)
	}
	if !created.HasAccess(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)) || created.HasAccess(*created.CurrentPeriodEnd) {
		t.Error("HasAccess on the created subscription")
	}

	billed, err := loadEvent(t, EventTypeSubscriptionBilled).SubscriptionBilled()
	if err != nil {
		t.Fatal(err)
	}
	b, sub := billed.Bill, billed.Subscription
	if !strings.HasPrefix(b.UUID, "sub-hist@") || b.SubscriptionUUID != sub.UUID || b.Amount != "10" || b.ProductName != "Pro plan" ||
		b.PeriodStart == nil || b.PeriodEnd == nil || b.Settlement.Split == nil || b.Settlement.FeePercent != "0.99" || b.Paid != "10.0042" {
		t.Errorf("billed bill %+v", b)
	}
	if sub.Status != SubscriptionStatusActive || sub.Reference != "customer-7731" || sub.Metadata["crm:account"] != "A-1042" ||
		!sub.NextBillingAt.Equal(*b.PeriodEnd) {
		t.Errorf("billed subscription %+v", sub)
	}

	changed, err := loadEvent(t, EventTypeSubscriptionStatusChanged).SubscriptionStatusChanged()
	if err != nil || changed.PreviousStatus != SubscriptionStatusPastDue || changed.Status != SubscriptionStatusActive ||
		changed.ManagementPageLink == "" {
		t.Errorf("statusChanged %+v %v", changed, err)
	}

	action, err := loadEvent(t, EventTypeSubscriptionActionRequiredChanged).SubscriptionActionRequiredChanged()
	if err != nil || action.ActionRequired != ActionRequiredTopUpAllowance || action.PreviousActionRequired != "" ||
		action.Allowance == nil || *action.Allowance.RemainingPeriods != 0 {
		t.Errorf("actionRequiredChanged %+v %v", action, err)
	}

	failed, err := loadEvent(t, EventTypeSubscriptionBillingFailed).SubscriptionBillingFailed()
	if err != nil {
		t.Fatal(err)
	}
	if failed.Reason != BillingFailureReasonInsufficientBalance || failed.AmountUSD != "10" || failed.Attempt != 1 ||
		failed.RemainingAttempts != 4 || failed.BillUUID != "sub-hist@01a0f755-f200-7000-8000-000000000005" ||
		failed.FailureCode != "insufficient_funds" || failed.NextAttemptAt == nil || failed.Status != SubscriptionStatusPastDue ||
		failed.ManagementPageLink == "" {
		t.Errorf("billingFailed %+v", failed)
	}

	upcoming, err := loadEvent(t, EventTypeSubscriptionUpcomingBill).SubscriptionUpcomingBill()
	if err != nil {
		t.Fatal(err)
	}
	if upcoming.AmountUSD != "10" || !upcoming.BillingAt.Equal(time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)) || upcoming.TrialEnding ||
		upcoming.BalanceSufficient == nil || !*upcoming.BalanceSufficient || upcoming.AllowanceSufficient == nil {
		t.Errorf("upcomingBill %+v", upcoming)
	}
}

func TestEventRefunds(t *testing.T) {
	for eventType, want := range map[EventType]RefundStatus{
		EventTypeRefundRequested: RefundStatusPending,
		EventTypeRefundApproved:  RefundStatusApproved,
		EventTypeRefundRejected:  RefundStatusRejected,
	} {
		e := loadEvent(t, eventType)
		accessor := map[EventType]func() (*Refund, error){
			EventTypeRefundRequested: e.RefundRequested,
			EventTypeRefundApproved:  e.RefundApproved,
			EventTypeRefundRejected:  e.RefundRejected,
		}[eventType]
		r, err := accessor()
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != want || !strings.HasPrefix(r.UUID, "refund@") || !strings.HasPrefix(r.TxUUID, "pay@") || r.RefundPercent == "" ||
			r.InitiatedBy != RefundInitiatorCustomer || r.Paid != "10.0042" || r.Currency.Code != "BASE:USDC" || r.Reason == "" {
			t.Errorf("%s: %+v", eventType, r)
		}
		if (r.RespondedAt == nil) != (want == RefundStatusPending) {
			t.Errorf("%s: respondedAt %v", eventType, r.RespondedAt)
		}
		if r.Approval != nil || r.Customer != nil || r.CustomerUUID != "" || r.ProductName != "" {
			t.Errorf("%s: API-only fields present", eventType)
		}
	}
	approved, _ := loadEvent(t, EventTypeRefundApproved).RefundApproved()
	if approved.RefundPercent != "50" || approved.Amount != "5.0021" || approved.Settlement == nil || approved.Settlement.Transaction == nil ||
		approved.TxHash == "" || approved.ExplorerURL == "" || approved.MerchantMessage != "" {
		t.Errorf("approved refund %+v", approved)
	}
	rejected, _ := loadEvent(t, EventTypeRefundRejected).RefundRejected()
	if rejected.MerchantMessage == "" || rejected.Settlement != nil || rejected.TxHash != "" {
		t.Errorf("rejected refund %+v", rejected)
	}
	requested, _ := loadEvent(t, EventTypeRefundRequested).RefundRequested()
	if requested.RefundPercent != "100" || requested.Amount != requested.Paid {
		t.Errorf("requested refund %+v", requested)
	}
}

func TestEventMembersAndFunds(t *testing.T) {
	joined, err := loadEvent(t, EventTypeMemberJoined).MemberJoined()
	if err != nil {
		t.Fatal(err)
	}
	if joined.UserUUID == "" || joined.InvitationUUID == "" || joined.SpaceUUID == "" || joined.JoinedAt.IsZero() || joined.FirstName != "Alice" ||
		joined.OrganizationFeePercent != "5" || joined.TrustedAt != nil || len(joined.AcceptedCurrencies) != 1 || joined.AcceptedCurrencies[0].ID != 8 {
		t.Errorf("joined %+v", joined)
	}
	removed, err := loadEvent(t, EventTypeMemberRemoved).MemberRemoved()
	if err != nil || removed.UserUUID == "" || removed.TrustedAt == nil || removed.LastName != "Martin" {
		t.Errorf("removed %+v %v", removed, err)
	}

	e := loadEvent(t, EventTypeHeldFundsReleased)
	if e.UserUUID != "019eca82-5680-7b00-8000-0000000000b1" {
		t.Errorf("envelope userUuid %q", e.UserUUID)
	}
	released, err := e.HeldFundsReleased()
	if err != nil {
		t.Fatal(err)
	}
	if !released.Received || len(released.Ledgers) != 3 || released.Amount != "98.9068" ||
		released.AmountUSD != "98.9068" || released.Settlement.Transaction == nil || released.Currency.Code != "BASE:USDC" ||
		!strings.HasPrefix(released.UUID, "transfer@") {
		t.Errorf("released %+v", released)
	}
	first := released.Ledgers[0]
	if first.Type != LedgerEntryTypePayment || first.Settlement == nil || first.Settlement.Split == nil || first.Owed != "99.01" ||
		first.OwedMinUnits != "99010000" || first.OwedUSD != "99.01" || first.AmountUSD != "100" {
		t.Errorf("ledger %+v", first)
	}
	refund := released.Ledgers[2]
	if refund.Type != LedgerEntryTypeRefund || refund.Settlement != nil || refund.RefundedTxUUID == "" || !strings.HasPrefix(refund.OwedMinUnits, "-") {
		t.Errorf("refund line %+v", refund)
	}
	// amount is the sum of the lines' owed (exact decimal sum, in min units).
	var sum int64
	for _, l := range released.Ledgers {
		n, _ := strconv.ParseInt(l.OwedMinUnits, 10, 64)
		sum += n
	}
	if strconv.FormatInt(sum, 10) != released.AmountMinUnits {
		t.Errorf("sum of owed %d, amount %s", sum, released.AmountMinUnits)
	}
}

func TestEventCheckoutExpired(t *testing.T) {
	e := loadEvent(t, EventTypeCheckoutExpired)
	ce, err := e.CheckoutExpired()
	if err != nil {
		t.Fatal(err)
	}
	if ce.Subscription != nil || ce.Payment == nil {
		t.Fatalf("payment session expected: %+v", ce)
	}
	p := ce.Payment
	if p.TxType != TransactionTypePayment || p.PriceUSD != "10" || p.AmountUSD != "10" ||
		p.Metadata["orderId"] != "order-1043" || p.ExpiresAt == nil || p.CreatedAt == nil || p.Reference != "order-1043" {
		t.Errorf("payment session %+v", p)
	}
	if p.Fees == nil || len(p.Fees) != 0 {
		t.Errorf("the docs' session has no fees: %+v", p.Fees)
	}

	withFees := &Event{Type: EventTypeCheckoutExpired, Data: json.RawMessage(`{
		"uuid":"pay@2","txType":"payment","organizationName":"Shop","priceUsd":"3.99","amountUsd":"4.81",
		"fees":[{"type":"custom","label":"Shipping","description":"Standard, 3 to 5 days","amountUsd":"0.75"},
		{"type":"processingFee","label":"Processing fee","amountUsd":"0.07"}],"availableCurrencies":[],"metadata":{}}`)}
	ce, err = withFees.CheckoutExpired()
	if err != nil || ce.Payment == nil {
		t.Fatalf("session with fees: %+v %v", ce, err)
	}
	if f := ce.Payment.Fees; len(f) != 2 || f[0].Label != "Shipping" || f[0].Description == "" || f[0].AmountUSD != "0.75" ||
		f[1].Type != FeeLineTypeProcessingFee || f[1].Description != "" || ce.Payment.AmountUSD != "4.81" {
		t.Errorf("session fees %+v amount %v", ce.Payment.Fees, ce.Payment.AmountUSD)
	}

	sub := &Event{Type: EventTypeCheckoutExpired, Data: json.RawMessage(`{
		"uuid":"sub@1","txType":"createSubscription","organizationName":"Shop","test":true,"priceUsd":"9.99","amountUsd":"9.99","fees":[],
		"availableCurrencies":[` + fxCurrency + `],"metadata":{"plan":"pro"},
		"frequency":{"value":1,"unit":"weeks"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":3,"upgradingFromTrial":false}`)}
	ce, err = sub.CheckoutExpired()
	if err != nil || ce.Payment != nil || ce.Subscription == nil {
		t.Fatalf("subscription session: %+v %v", ce, err)
	}
	s := ce.Subscription
	if s.UUID != "sub@1" || s.Frequency != (Duration{1, DurationUnitWeeks}) || s.TrialPeriod == nil || *s.MinPeriods != 3 ||
		s.UpgradingFromTrial || s.Metadata["plan"] != "pro" {
		t.Errorf("subscription session %+v", s)
	}

	unknown := &Event{Type: EventTypeCheckoutExpired, Data: json.RawMessage(`{"uuid":"x@1","txType":"payAsYouGo","organizationName":"Shop"}`)}
	ce, err = unknown.CheckoutExpired()
	if err != nil || ce.Payment == nil || ce.Payment.TxType != "payAsYouGo" {
		t.Errorf("unknown txType: %+v %v", ce, err)
	}

	// The session's amounts are decimal strings: a number is refused.
	numeric := &Event{Type: EventTypeCheckoutExpired, Data: json.RawMessage(`{"uuid":"pay@3","txType":"payment","priceUsd":10}`)}
	if _, err := numeric.CheckoutExpired(); !errors.As(err, new(*ValidationError)) {
		t.Errorf("numeric priceUsd: %v", err)
	}
}

func TestEventWebhookTest(t *testing.T) {
	w, err := loadEvent(t, EventTypeWebhookTest).WebhookTest()
	if err != nil || w.EndpointUUID == "" || w.Message == "" {
		t.Errorf("%+v %v", w, err)
	}
}

func TestEventAccessorTypeMismatch(t *testing.T) {
	e := loadEvent(t, EventTypePaymentCompleted)
	_, err := e.SubscriptionCreated()
	var ve *ValidationError
	if !errors.As(err, &ve) || !strings.Contains(err.Error(), "payment.completed") {
		t.Errorf("mismatch: %v", err)
	}
	if _, err := e.RefundRejected(); err == nil {
		t.Error("refund accessor on a payment event")
	}
	if _, err := e.CheckoutExpired(); err == nil {
		t.Error("checkout accessor on a payment event")
	}
}

func TestEventDecodeData(t *testing.T) {
	e := loadEvent(t, EventTypeSubscriptionBillingFailed)
	// Any shape, whatever the type: e.g. the base Subscription only.
	s, err := e.DecodeData[Subscription]()
	if err != nil || s.Status != SubscriptionStatusPastDue {
		t.Errorf("DecodeData[Subscription]: %+v %v", s, err)
	}
	m, err := e.DecodeData[map[string]any]()
	if err != nil || m["amountUsd"] != "10" {
		t.Errorf("DecodeData[map]: %v %v", m, err)
	}
	// A shape that does not fit: an amountUsd as a number where the model wants a decimal string.
	numeric := &Event{Type: EventTypeSubscriptionBillingFailed, Data: json.RawMessage(`{"amountUsd":10}`)}
	if _, err := numeric.SubscriptionBillingFailed(); !errors.As(err, new(*ValidationError)) {
		t.Errorf("wrong shape: %v", err)
	}
	if _, err := e.DecodeData[[]Subscription](); !errors.As(err, new(*ValidationError)) {
		t.Errorf("object into a list: %v", err)
	}

	for _, data := range []string{``, `null`, ` null `} {
		empty := &Event{Type: EventTypeWebhookTest, Data: json.RawMessage(data)}
		w, err := empty.WebhookTest()
		if err != nil || *w != (WebhookTest{}) {
			t.Errorf("data %q: %+v %v", data, w, err)
		}
	}
}

func TestParseEventUnknownAndInvalid(t *testing.T) {
	e, err := ParseEvent([]byte(`{"id":"evt_9","type":"invoice.paid","version":"v2","createdAt":"2026-10-01T14:00:00.5+02:00","test":true,"userUuid":"u-1","data":{"anything":[1,2]},"extra":1}`))
	if err != nil {
		t.Fatalf("unknown type must parse: %v", err)
	}
	if e.Type != "invoice.paid" || string(e.Data) != `{"anything":[1,2]}` || e.UserUUID != "u-1" || !e.Test ||
		!e.CreatedAt.Equal(time.Date(2026, 10, 1, 12, 0, 0, 5e8, time.UTC)) {
		t.Errorf("event %+v", e)
	}
	if _, err := e.PaymentCompleted(); err == nil {
		t.Error("accessor on an unknown type")
	}

	bad := map[string]string{
		"empty":         ``,
		"array":         `[{"version":"v2"}]`,
		"string":        `"v2"`,
		"not json":      `{"version":`,
		"v1":            `{"id":"evt_1","type":"payment.completed","version":"v1","data":{}}`,
		"no version":    `{"id":"evt_1","type":"payment.completed","data":{}}`,
		"wrong type":    `{"id":"evt_1","type":"payment.completed","version":"v2","test":"yes"}`,
		"bad createdAt": `{"id":"evt_1","type":"webhook.test","version":"v2","createdAt":"today"}`,
	}
	for name, body := range bad {
		_, err := ParseEvent([]byte(body))
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want *ValidationError, got %v", name, err)
		}
	}
}

func TestEventDetailDecoding(t *testing.T) {
	body := `{"id":"evt_1","type":"refund.rejected","version":"v2","createdAt":"2026-10-01T12:00:00Z","test":false,
		"data":{"uuid":"refund@1","status":"rejected"},
		"deliveries":[{"endpointUuid":"e1","url":"https://x.io/hook","delivered":false,"attempts":[
			{"uuid":"a1","eventId":"evt_1","eventType":"refund.rejected","endpointUuid":"e1","attempt":1,"delivered":false,
			 "statusCode":500,"error":"500 Internal","durationMs":120,"attemptedAt":"2026-10-01T12:00:01Z"},
			{"uuid":"a2","eventId":"evt_1","eventType":"refund.rejected","endpointUuid":"e1","attempt":2,"delivered":false,"skipped":true,
			 "error":"timeout","durationMs":30000,"attemptedAt":"2026-10-01T12:01:01Z"}]}]}`
	d, err := decodeResponse[EventDetail](&response{status: 200, body: []byte(body)})
	if err != nil {
		t.Fatal(err)
	}
	r, err := d.RefundRejected()
	if err != nil || r.Status != RefundStatusRejected {
		t.Errorf("embedded event accessor: %+v %v", r, err)
	}
	a := d.Deliveries[0].Attempts
	if len(a) != 2 || a[0].StatusCode == nil || *a[0].StatusCode != 500 || a[1].StatusCode != nil || a[1].DurationMs != 30000 ||
		a[0].Skipped || !a[1].Skipped {
		t.Errorf("attempts %+v", a)
	}
}
