package qbitflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// loadEvent parses testdata/events/<type>.json: the docs' example body of each event type
// (docs/core webhooks.md, generated from the API's code).
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
	if e.Type != eventType || e.Version != WebhookPayloadVersionV2 || !strings.HasPrefix(e.ID, "evt_") || e.CreatedAt.IsZero() {
		t.Fatalf("envelope %+v", e)
	}
	return e
}

// strictModel maps each event type to the model its data must decode into with no unknown
// field: every key of the docs' examples is modelled.
var strictModel = map[EventType]func() any{
	EventTypePaymentCompleted:                  func() any { return new(PaymentCompleted) },
	EventTypeSubscriptionCreated:               func() any { return new(SubscriptionCreated) },
	EventTypeSubscriptionBilled:                func() any { return new(SubscriptionBilled) },
	EventTypeSubscriptionStatusChanged:         func() any { return new(SubscriptionStatusChanged) },
	EventTypeSubscriptionActionRequiredChanged: func() any { return new(SubscriptionActionRequiredChanged) },
	EventTypeSubscriptionBillingFailed:         func() any { return new(SubscriptionBillingFailed) },
	EventTypeSubscriptionUpcomingBill:          func() any { return new(SubscriptionUpcomingBill) },
	EventTypeRefundRequested:                   func() any { return new(Refund) },
	EventTypeRefundCompleted:                   func() any { return new(Refund) },
	EventTypeRefundDenied:                      func() any { return new(Refund) },
	EventTypeMemberJoined:                      func() any { return new(MemberJoined) },
	EventTypeMemberRemoved:                     func() any { return new(Member) },
	EventTypeHeldFundsReleased:                 func() any { return new(HeldFundsReleased) },
	EventTypeCheckoutExpired:                   func() any { return new(PaymentSessionData) },
	EventTypeWebhookTest:                       func() any { return new(WebhookTest) },
}

func TestEventFixturesAreFullyModelled(t *testing.T) {
	if len(strictModel) != len(knownEventTypes) {
		t.Fatalf("%d models for %d types", len(strictModel), len(knownEventTypes))
	}
	for _, eventType := range knownEventTypes {
		e := loadEvent(t, eventType)
		dec := json.NewDecoder(bytes.NewReader(e.Data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(strictModel[eventType]()); err != nil {
			t.Errorf("%s: %v", eventType, err)
		}
	}
}

func TestEventPaymentCompleted(t *testing.T) {
	e := loadEvent(t, EventTypePaymentCompleted)
	p, err := e.PaymentCompleted()
	if err != nil {
		t.Fatal(err)
	}
	if p.UUID != "pay@01a0f755-f200-7000-8000-000000000001" || p.Amount != 10 || p.AmountMinUnits != "10000000" ||
		p.Reference != "order-1042" || p.CustomerReference != "crm-5521" || p.Chain != ChainBASE ||
		p.ManagementPageLink == "" || p.PaidMinUnits != "10004200" || p.PaidUSD == nil || *p.PaidUSD != 10.0042 {
		t.Errorf("payment %+v", p)
	}
	if p.Currency == nil || p.Currency.Symbol != "USDC" || p.Currency.MainCurrency == nil || p.Currency.MainCurrency.MainCurrency != nil ||
		p.Currency.MainCurrencyID == nil || *p.Currency.MainCurrencyID != 3 {
		t.Errorf("currency %+v", p.Currency)
	}
	m := p.Metadata
	if m.FeePercent != 0.99 || m.TxAmounts.MinUnits.Merchant != "9901000" || m.TxAmounts.MinUnits.NetworkFee != "4200" ||
		m.TxAmounts.USD.Merchant != 9.901 || m.TxAmounts.USD.Organization != nil || m.TxAmounts.USD.NetworkFee == nil ||
		m.TxMetadata.BlockData.Timestamp != 1790855998 || m.TxMetadata.NetworkFees.UnitsConsumed != 95000 || m.OrganizationFee != nil ||
		m.TxMetadata.MainCurrencyPriceUSD == nil || *m.TxMetadata.MainCurrencyPriceUSD != 4512.37 {
		t.Errorf("metadata %+v", m)
	}
	// Webhooks never carry the API-only fields.
	if p.Customer != nil || p.Refund != nil || p.Refundable != nil {
		t.Error("API-only fields must decode as absent")
	}
	if p.ConfirmedAt == nil || p.CheckoutOpenedAt == nil {
		t.Error("times")
	}
	// The docs' payment has a price and no fees: amount = price.
	if p.Price != 10 || p.Fees != nil || p.Price != p.Amount {
		t.Errorf("price %v, fees %+v", p.Price, p.Fees)
	}
}

func TestEventSubscriptions(t *testing.T) {
	created, err := loadEvent(t, EventTypeSubscriptionCreated).SubscriptionCreated()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.UUID, "sub@") || created.Frequency != (Duration{1, DurationUnitMonths}) ||
		created.PriceUSD == "" || created.CurrentPeriodEnd == nil || created.NextBillingDate == nil || created.ManagementPageLink == "" {
		t.Errorf("created %+v", created)
	}

	billed, err := loadEvent(t, EventTypeSubscriptionBilled).SubscriptionBilled()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(billed.UUID, "sub-hist@") || !strings.HasPrefix(billed.SubscriptionUUID, "sub@") ||
		billed.SubscriptionStatus == "" || billed.SubscriptionReference == "" || billed.PeriodStart == nil || billed.PeriodEnd == nil {
		t.Errorf("billed %+v", billed)
	}

	changed, err := loadEvent(t, EventTypeSubscriptionStatusChanged).SubscriptionStatusChanged()
	if err != nil || changed.PreviousStatus == "" || changed.Status == changed.PreviousStatus {
		t.Errorf("statusChanged %+v %v", changed, err)
	}

	action, err := loadEvent(t, EventTypeSubscriptionActionRequiredChanged).SubscriptionActionRequiredChanged()
	if err != nil || action.ActionRequired == "" {
		t.Errorf("actionRequiredChanged %+v %v", action, err)
	}

	failed, err := loadEvent(t, EventTypeSubscriptionBillingFailed).SubscriptionBillingFailed()
	if err != nil {
		t.Fatal(err)
	}
	if failed.Reason != BillingFailureReasonInsufficientBalance || failed.AmountUSD != "10" || failed.Attempt != 1 ||
		failed.RemainingAttempts != 4 || failed.BillUUID != "sub-hist@01a0f755-f200-7000-8000-000000000005" ||
		failed.FailureCode != "insufficient_funds" || failed.NextAttemptAt == nil || failed.Status != SubscriptionStatusPastDue {
		t.Errorf("billingFailed %+v", failed)
	}

	upcoming, err := loadEvent(t, EventTypeSubscriptionUpcomingBill).SubscriptionUpcomingBill()
	if err != nil {
		t.Fatal(err)
	}
	if upcoming.AmountUSD == 0 || upcoming.BillingDate.IsZero() || upcoming.BalanceSufficient == nil || upcoming.AllowanceSufficient == nil {
		t.Errorf("upcomingBill %+v", upcoming)
	}
}

func TestEventRefunds(t *testing.T) {
	for eventType, want := range map[EventType]RefundStatus{
		EventTypeRefundRequested: RefundStatusPending,
		EventTypeRefundCompleted: RefundStatusApproved,
		EventTypeRefundDenied:    RefundStatusRejected,
	} {
		e := loadEvent(t, eventType)
		accessor := map[EventType]func() (*Refund, error){
			EventTypeRefundRequested: e.RefundRequested,
			EventTypeRefundCompleted: e.RefundCompleted,
			EventTypeRefundDenied:    e.RefundDenied,
		}[eventType]
		r, err := accessor()
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != want || !strings.HasPrefix(r.UUID, "refund@") || r.TxUUID == "" || r.RefundPercent == 0 || r.InitiatedBy == "" {
			t.Errorf("%s: %+v", eventType, r)
		}
		if (r.RespondedAt == nil) != (want == RefundStatusPending) {
			t.Errorf("%s: respondedAt %v", eventType, r.RespondedAt)
		}
		if r.Approval != nil || r.Customer != nil || r.ProductName != "" {
			t.Errorf("%s: API-only fields present", eventType)
		}
	}
	completed, _ := loadEvent(t, EventTypeRefundCompleted).RefundCompleted()
	if completed.Metadata == nil || completed.TxHash == "" || completed.ExplorerURL == "" {
		t.Errorf("completed refund %+v", completed)
	}
}

func TestEventMembersAndFunds(t *testing.T) {
	joined, err := loadEvent(t, EventTypeMemberJoined).MemberJoined()
	if err != nil {
		t.Fatal(err)
	}
	if joined.UserUUID == "" || joined.InvitationUUID == "" || joined.SpaceUUID == "" || joined.JoinedAt.IsZero() {
		t.Errorf("joined %+v", joined)
	}
	removed, err := loadEvent(t, EventTypeMemberRemoved).MemberRemoved()
	if err != nil || removed.UserUUID == "" {
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
	if !released.Received || released.Type != TransferTypeHeldFundsRelease || len(released.Ledgers) == 0 ||
		released.TxMetadata.MainCurrencyPriceUSD == nil || released.Currency == nil {
		t.Errorf("released %+v", released)
	}
	first := released.Ledgers[0]
	if first.Type != LedgerEntryTypePayment || first.Metadata == nil || first.OwedMinUnits != "99010000" || first.OwedUSD != 99.01 {
		t.Errorf("ledger %+v", first)
	}
	for _, l := range released.Ledgers {
		if l.Type == LedgerEntryTypeRefund && (l.Metadata != nil || l.RefundedTxUUID == "" || !strings.HasPrefix(l.OwedMinUnits, "-")) {
			t.Errorf("refund line %+v", l)
		}
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
	if p.TxType != TransactionTypePayment || p.Price == nil || *p.Price != 10 || len(p.AvailableCurrencyIDs) != 3 ||
		p.OrganizationName != "Example Shop" || p.ExpiresAt == nil {
		t.Errorf("payment session %+v", p)
	}

	if p.Fees != nil || p.Amount != nil {
		t.Errorf("the docs' session has no fees: %+v %v", p.Fees, p.Amount)
	}

	withFees := &Event{Type: EventTypeCheckoutExpired, Version: "v2", Data: json.RawMessage(`{
		"uuid":"pay@2","txType":"payment","organizationName":"Shop","price":3.99,"amount":4.81,
		"fees":[{"type":"custom","label":"Shipping","description":"Standard, 3 to 5 days","amountUsd":"0.75"},
		{"type":"processingFee","label":"Processing fee","amountUsd":"0.07"}]}`)}
	ce, err = withFees.CheckoutExpired()
	if err != nil || ce.Payment == nil {
		t.Fatalf("session with fees: %+v %v", ce, err)
	}
	if f := ce.Payment.Fees; len(f) != 2 || f[0].Label != "Shipping" || f[0].Description == "" || f[0].AmountUSD != "0.75" ||
		f[1].Type != FeeLineTypeProcessingFee || f[1].Description != "" || ce.Payment.Amount == nil || *ce.Payment.Amount != 4.81 {
		t.Errorf("session fees %+v amount %v", ce.Payment.Fees, ce.Payment.Amount)
	}

	sub := &Event{Type: EventTypeCheckoutExpired, Version: "v2", Data: json.RawMessage(`{
		"uuid":"sub@1","txType":"createSubscription","organizationName":"Shop","test":true,"availableCurrencyIds":[8],
		"frequency":{"value":1,"unit":"weeks"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":3}`)}
	ce, err = sub.CheckoutExpired()
	if err != nil || ce.Payment != nil || ce.Subscription == nil {
		t.Fatalf("subscription session: %+v %v", ce, err)
	}
	s := ce.Subscription
	if s.UUID != "sub@1" || s.Frequency != (Duration{1, DurationUnitWeeks}) || s.TrialPeriod == nil || *s.MinPeriods != 3 {
		t.Errorf("subscription session %+v", s)
	}

	unknown := &Event{Type: EventTypeCheckoutExpired, Version: "v2", Data: json.RawMessage(`{"uuid":"x@1","txType":"payAsYouGo","organizationName":"Shop"}`)}
	ce, err = unknown.CheckoutExpired()
	if err != nil || ce.Payment == nil || ce.Payment.TxType != "payAsYouGo" {
		t.Errorf("unknown txType: %+v %v", ce, err)
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
	if _, err := e.RefundDenied(); err == nil {
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
	// The amountUsd clash: a float target refuses the billingFailed string.
	if _, err := e.DecodeData[SubscriptionUpcomingBill](); !errors.As(err, new(*ValidationError)) {
		t.Errorf("wrong shape: %v", err)
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
	body := `{"id":"evt_1","type":"refund.denied","version":"v2","createdAt":"2026-10-01T12:00:00Z","test":false,
		"data":{"uuid":"refund@1","status":"rejected"},
		"deliveries":[{"endpointUuid":"e1","url":"https://x.io/hook","delivered":false,"attempts":[
			{"uuid":"a1","eventId":"evt_1","eventType":"refund.denied","endpointUuid":"e1","attempt":1,"delivered":false,
			 "statusCode":500,"error":"500 Internal","durationMs":120,"attemptedAt":"2026-10-01T12:00:01Z"},
			{"uuid":"a2","eventId":"evt_1","eventType":"refund.denied","endpointUuid":"e1","attempt":2,"delivered":false,
			 "error":"timeout","durationMs":30000,"attemptedAt":"2026-10-01T12:01:01Z"}]}]}`
	d, err := decodeResponse[EventDetail](&response{status: 200, body: []byte(body)})
	if err != nil {
		t.Fatal(err)
	}
	r, err := d.RefundDenied()
	if err != nil || r.Status != RefundStatusRejected {
		t.Errorf("embedded event accessor: %+v %v", r, err)
	}
	a := d.Deliveries[0].Attempts
	if len(a) != 2 || a[0].StatusCode == nil || *a[0].StatusCode != 500 || a[1].StatusCode != nil || a[1].DurationMs != 30000 {
		t.Errorf("attempts %+v", a)
	}
}
