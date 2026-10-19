package qbitflow

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
)

// Fixtures with every wire key of each response model (docs/api generated Go types), all
// non-empty, shaped like the docs' examples (a 10 USD price paid in BASE:USDC, 0.99 % fee).
const (
	fxCurrency = `{"id":8,"code":"BASE:USDC","symbol":"USDC","name":"USD Coin","chain":"BASE","decimals":6,` +
		`"address":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","mainCurrencyId":3,"test":false}`
	fxNative   = `{"id":1,"code":"ETH","symbol":"ETH","name":"Ethereum","chain":"ETH","decimals":18,"test":false}`
	fxSummary  = `{"uuid":"c-1","firstName":"Ada","lastName":"Lovelace","email":"ada@example.com","reference":"crm-1","deleted":true}`
	fxRefundSm = `{"uuid":"refund@1","status":"approved","initiatedBy":"merchant","refundPercent":"50","amount":"5.0021",` +
		`"amountMinUnits":"5002100","amountUsd":"5.0021","createdAt":"2026-10-01T12:00:00Z","respondedAt":"2026-10-01T13:00:00Z"}`
	fxAttempt    = `{"status":"failed","code":"insufficient_funds","message":"Not enough USDC","txHash":"0xa","at":"2026-10-01T12:00:00Z"}`
	fxTx         = `{"blockNumber":"52468103","blockTime":"2026-10-01T11:59:58Z","gasUsed":95000,"fee":{"amount":"0.00000114","amountMinUnits":"1140000000000","amountUsd":"0.005144"},"l1Fee":{"amount":"0.0000001","amountMinUnits":"100000000000","amountUsd":"0.00045"},"nativePriceUsd":"4512.37"}`
	fxSettlement = `{"feePercent":"0.99","organizationFee":{"address":"0xorg","feePercent":"10"},` +
		`"referralFee":{"address":"0xref","feePercent":"20","deadline":"2027-01-01T00:00:00Z"},` +
		`"split":{"platform":{"amount":"0.0792","amountMinUnits":"79200","amountUsd":"0.0792"},` +
		`"organization":{"amount":"0.9901","amountMinUnits":"990100","amountUsd":"0.9901"},` +
		`"referral":{"amount":"0.0198","amountMinUnits":"19800","amountUsd":"0.0198"},` +
		`"merchant":{"amount":"8.9109","amountMinUnits":"8910900","amountUsd":"8.9109"},` +
		`"networkFee":{"amount":"0.0042","amountMinUnits":"4200","amountUsd":"0.0042"}},"transaction":` + fxTx + `}`
	fxTransfer = `"createdAt":"2026-10-01T12:00:00Z","from":"0xfrom","to":"0xto","amount":"10","amountMinUnits":"10000000","amountUsd":"10",` +
		`"currency":` + fxCurrency + `,"txHash":"0xhash","chain":"BASE","explorerUrl":"https://basescan.org/tx/0xhash","test":false,"userUuid":"m-1"`
	fxPaid      = `"confirmedAt":"2026-10-01T11:59:58Z","paid":"10.0042","paidMinUnits":"10004200","paidUsd":"10.0042"`
	fxRefunding = `"refund":` + fxRefundSm + `,"refundable":false,"notRefundableReason":"refundExists"`
	fxFees      = `[{"type":"custom","label":"VAT (20%)","description":"French VAT","amountUsd":"1.30"},{"type":"processingFee","label":"Processing fee","amountUsd":"0.20"}]`
	fxSub       = `{"uuid":"sub@1","reference":"customer-7731","createdAt":"2026-09-01T12:00:00Z","updatedAt":"2026-10-01T12:00:00Z","from":"0xf","to":"0xt",` +
		`"productUuid":"p-1","subscriptionHash":"0x2","currency":` + fxCurrency + `,"frequency":{"value":1,"unit":"months"},` +
		`"status":"stopped","actionRequired":"raiseMaximum","currentPeriodEnd":"2026-10-31T12:00:00Z","priceUsd":"9.99","fees":` + fxFees + `,` +
		`"allowance":{"amount":"110","amountMinUnits":"110000000","amountUsd":"110","remainingPeriods":0,"maxPerPeriod":"30","maxPerPeriodMinUnits":"30000000"},` +
		`"lastBilledAt":"2026-10-01T12:00:00Z","nextBillingAt":"2026-10-31T12:00:00Z","cancelledAt":"2026-10-02T12:00:00Z","cancellationReason":"merchant",` +
		`"cancellableAt":"2026-12-01T12:00:00Z","metadata":{"plan":"pro","crm:account":"A-1042"},"managementPageLink":"https://qbitflow.app/user/s/manage?uuid=sub@1",` +
		`"test":false,"userUuid":"m-1","customerUuid":"c-1","customerReference":"crm-1","customer":` + fxSummary + `,` +
		`"dunning":{"failedAttempts":1,"remainingAttempts":3,"failingSince":"2026-10-01T12:00:00Z","nextAttemptAt":"2026-10-02T12:00:00Z","awaitingMaximumSince":"2026-10-01T12:00:00Z"}}`
	fxEndpoint = `"uuid":"e-1","test":true,"url":"https://x.io/hook","events":["payment.completed"],"includeMembers":true,"payloadVersion":"v2",` +
		`"description":"d","createdAt":"2026-10-01T12:00:00Z","rotatedAt":"2026-10-01T12:00:00Z","disabledAt":"2026-10-02T12:00:00Z",` +
		`"disabledReason":"failing","failingSince":"2026-09-29T12:00:00Z","lastDeliveredAt":"2026-09-28T12:00:00Z"`
	fxSession = `"uuid":"sub@1","reference":"order-1044","productUuid":"p-1","productReference":"tshirt","productName":"T-shirt",` +
		`"description":"Blue, size M","priceUsd":"3.99","fees":[{"type":"custom","label":"Shipping","description":"Standard","amountUsd":"0.75"},` +
		`{"type":"processingFee","label":"Processing fee","amountUsd":"0.07"}],"amountUsd":"4.81","successUrl":"https://x/ok","cancelUrl":"https://x/ko",` +
		`"redirectUrl":"https://x/thanks","organizationName":"Shop","userName":"Ada","test":true,"txType":"createSubscription",` +
		`"availableCurrencies":[` + fxCurrency + `,` + fxNative + `],"metadata":{"orderId":"order-1044"},` +
		`"createdAt":"2026-10-01T12:00:00Z","expiresAt":"2026-10-01T13:00:00Z"`
)

var modelFixtures = map[string]struct {
	json   string
	decode func([]byte) (any, error)
}{
	"Currency":        {fxCurrency, strict[Currency]},
	"Currency/native": {fxNative, strict[Currency]},
	"Payment": {`{"uuid":"pay@1",` + fxTransfer + `,"reference":"order-1","priceUsd":"8.5","fees":` + fxFees + `,"productName":"Pro",` +
		`"productDescription":"Lifetime","metadata":{"orderId":"order-1","crm:customer":"C-5521"},"productUuid":"p-1","customerUuid":"c-1",` +
		`"customerReference":"crm-1","note":"thanks","customer":` + fxSummary + `,"settlement":` + fxSettlement + `,` + fxPaid + `,` + fxRefunding + `,` +
		`"checkoutOpenedAt":"2026-10-01T11:00:00Z"}`, strict[Payment]},
	"PaymentCompleted": {`{"uuid":"pay@1",` + fxTransfer + `,"priceUsd":"10","fees":[],"productName":"Pro","metadata":{},"settlement":` + fxSettlement + `,` +
		`"managementPageLink":"https://qbitflow.app/user/s/manage?uuid=pay@1"}`, strict[PaymentCompleted]},
	"SubscriptionSessionData": {`{` + fxSession + `,"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":14,"unit":"days"},` +
		`"minPeriods":3,"upgradingFromTrial":true}`, strict[SubscriptionSessionData]},
	"Bill": {`{"uuid":"sub-hist@1",` + fxTransfer + `,"productName":"Pro","productDescription":"Monthly","productUuid":"p-1","subscriptionUuid":"sub@1",` +
		`"customerUuid":"c-1","customerReference":"crm-1","customer":` + fxSummary + `,"settlement":` + fxSettlement + `,` +
		`"periodStart":"2026-10-01T12:00:00Z","periodEnd":"2026-10-31T12:00:00Z",` + fxPaid + `,` + fxRefunding + `}`, strict[Bill]},
	"Transaction": {`{"type":"bill","uuid":"sub-hist@1","createdAt":"2026-10-01T12:00:00Z","from":"0xf","to":"0xt","productName":"Pro",` +
		`"productDescription":"d","priceUsd":"8.5","fees":` + fxFees + `,"amount":"10","amountMinUnits":"10000000","amountUsd":"10",` +
		`"currency":` + fxCurrency + `,"productUuid":"p-1","txHash":"0xh","customerUuid":"c-1","customer":` + fxSummary + `,` +
		`"customerReference":"crm-1","subscriptionUuid":"sub@1","periodStart":"2026-10-01T12:00:00Z","periodEnd":"2026-10-31T12:00:00Z",` +
		`"reference":"r","subscriptionReference":"sr","metadata":{"k":"v"},"chain":"SOL","explorerUrl":"https://x","test":true,"userUuid":"m-1",` +
		`"settlement":` + fxSettlement + `,` + fxPaid + `,` + fxRefunding + `}`, strict[Transaction]},
	"Failure": {`{"uuid":"f-1","kind":"bill","txUuid":"sub-hist@1","attempt":2,"subscriptionUuid":"sub@1","code":"insufficient_funds",` +
		`"category":"insufficientBalance","message":"m","attempted":"9.99","attemptedMinUnits":"9990000","attemptedUsd":"9.99",` +
		`"currency":` + fxCurrency + `,"from":"0xf","txHash":"0xh","customerUuid":"c-1","productUuid":"p-1","createdAt":"2026-10-01T12:00:00Z",` +
		`"customer":` + fxSummary + `,"test":false,"userUuid":"m-1"}`, strict[Failure]},
	"CreatedCheckoutSession": {`{"link":"https://qbitflow.app/checkout?uuid=pay@1","uuid":"pay@1","expiresAt":"2026-10-02T12:00:00Z"}`, strict[CreatedCheckoutSession]},
	"CheckoutSessionStatus": {`{"uuid":"pay@1","status":"created","txHash":"0xh","message":"m","lastAttempt":` + fxAttempt + `,` +
		`"expiresAt":"2026-10-02T12:00:00Z"}`, strict[CheckoutSessionStatus]},
	"Subscription": {fxSub, strict[Subscription]},
	"TestBill": {`{"billUuid":"sub-hist@019c-1","stage":"waiting","outcome":"paid","attempts":1,"nextAttemptAt":"2026-10-02T12:00:00Z",` +
		`"txHash":"0xh","failureCode":"insufficient_allowance"}`, strict[TestBill]},
	"Refund": {`{"uuid":"refund@1","txUuid":"pay@1","reason":"r","status":"approved","createdAt":"2026-10-01T12:00:00Z","merchantMessage":"m",` +
		`"respondedAt":"2026-10-01T13:00:00Z","test":false,"userUuid":"m-1","txHash":"0xh","initiatedBy":"customer","held":true,` +
		`"paid":"10.0042","paidMinUnits":"10004200","paidUsd":"10.0042","refundPercent":"50","amount":"5.0021","amountMinUnits":"5002100",` +
		`"amountUsd":"5.0021","currency":` + fxCurrency + `,"settlement":{"transaction":` + fxTx + `},"chain":"BASE","explorerUrl":"https://e",` +
		`"approval":{"status":"waitingConfirmation","txHash":"0xa","lastAttempt":` + fxAttempt + `},"customerUuid":"c-1","productName":"Pro",` +
		`"customer":` + fxSummary + `}`, strict[Refund]},
	"Customer": {`{"uuid":"c-1","firstName":"Ada","lastName":"Lovelace","email":"ada@example.com","verified":true,"phoneNumber":"+33 6 12 34 56 78",` +
		`"address":"1 rue X","reference":"crm-1","metadata":{"crm":"42"},"createdAt":"2026-10-01T12:00:00Z","test":true,"userUuid":"m-1"}`, strict[Customer]},
	"Product": {`{"uuid":"p-1","name":"Pro","description":"d","priceUsd":"9.99","createdAt":"2026-10-01T12:00:00Z","isActive":true,"reference":"pro",` +
		`"metadata":{"sku":"PRO-1"},"subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":3},` +
		`"paymentLink":"https://qbitflow.app/pay/p-1","test":false,"userUuid":"m-1"}`, strict[Product]},
	"Member": {`{"userUuid":"m-1","firstName":"Ada","lastName":"L","email":"a@b.co","test":false,"organizationFeePercent":"7.5",` +
		`"trustedAt":"2026-10-03T12:00:00Z","joinedAt":"2026-10-01T12:00:00Z","acceptedCurrencies":[` + fxCurrency + `],"spaceUuid":"s-1"}`, strict[Member]},
	"MemberJoined": {`{"userUuid":"m-1","firstName":"Ada","email":"a@b.co","test":false,"organizationFeePercent":"0",` +
		`"joinedAt":"2026-10-01T12:00:00Z","acceptedCurrencies":[],"spaceUuid":"s-1","invitationUuid":"i-1"}`, strict[MemberJoined]},
	"HeldFunds": {`{"ledgers":[{"txUuid":"pay@1","type":"payment","description":"Payment pay@1","amountUsd":"10","settlement":` + fxSettlement + `,` +
		`"currency":` + fxCurrency + `,"owed":"8.9109","owedMinUnits":"8910900","owedUsd":"8.9109","createdAt":"2026-10-01T12:00:00Z"},` +
		`{"txUuid":"refund@1","type":"refund","description":"Refund of payment pay@1","refundedTxUuid":"pay@1","amountUsd":"10.0042",` +
		`"currency":` + fxCurrency + `,"owed":"-10.0042","owedMinUnits":"-10004200","owedUsd":"-10.0042","createdAt":"2026-10-02T12:00:00Z"}],` +
		`"totalUsd":"-1.0933"}`, strict[HeldFunds]},
	"MemberHeldFunds": {`{"userUuid":"m-1","totalUsd":"98.5","count":2,"oldestAt":"2026-09-28T12:00:00Z"}`, strict[MemberHeldFunds]},
	"HeldFundsReleased": {`{"uuid":"transfer@1",` + fxTransfer + `,"received":true,"type":"heldFundsRelease","settlement":{"transaction":` + fxTx + `},` +
		`"ledgers":[{"txUuid":"sub-hist@1","type":"bill","description":"Subscription bill sub-hist@1","amountUsd":"10","currency":` + fxCurrency + `,` +
		`"owed":"9.901","owedMinUnits":"9901000","owedUsd":"9.901","createdAt":"2026-10-01T12:00:00Z"}]}`, strict[HeldFundsReleased]},
	"Wallet": {`{"uuid":"w-1","createdAt":"2026-10-01T12:00:00Z","address":"0xpk","test":false,"userUuid":"m-1","tokenWallets":[` +
		`{"uuid":"tw-1","createdAt":"2026-10-01T12:00:00Z","currency":` + fxCurrency + `,"walletUuid":"w-1",` +
		`"balance":{"currency":` + fxCurrency + `,"amount":"12.5","amountMinUnits":"12500000","amountUsd":"12.5"}}],"currency":` + fxNative + `}`, strict[Wallet]},
	"Invitation": {`{"uuid":"i-1","test":false,"email":"a@b.co","role":"user","holdFunds":true,"organizationFeePercent":"2.5",` +
		`"redirectUrl":"https://x.io/welcome","expiresAt":"2026-10-08T12:00:00Z","acceptedByUserUuid":"m-1","acceptedAt":"2026-10-02T12:00:00Z",` +
		`"revokedAt":"2026-10-03T12:00:00Z","createdAt":"2026-10-01T12:00:00Z","status":"accepted"}`, strict[Invitation]},
	"CreatedInvitation": {`{"uuid":"i-1","test":true,"email":"a@b.co","role":"user","holdFunds":false,"organizationFeePercent":"0",` +
		`"redirectUrl":"https://x.io/welcome","expiresAt":"2026-10-08T12:00:00Z","acceptedByUserUuid":"m-1","acceptedAt":"2026-10-02T12:00:00Z",` +
		`"revokedAt":"2026-10-03T12:00:00Z","createdAt":"2026-10-01T12:00:00Z","status":"pending","link":"https://qbitflow.app/invite?token=x"}`, strict[CreatedInvitation]},
	"AccountingEvent": {`{"txTimeUtc":"2026-09-13T19:23:26.62Z","type":"refund","reference":"r","relatedReference":"r0",` +
		`"productName":"Pro","customerReference":"crm-1","grossAmountUsd":"-10.0042","platformFeeUsd":"0.099","referralFeeUsd":"0.0198",` +
		`"organizationFeeUsd":"0.9901","netAmountUsd":"-10.0042","chain":"BASE","currency":"USDC","currencyDecimals":6,"currencyPriceUsd":"1.00002",` +
		`"grossAmount":"-10.0042","grossAmountMinUnits":"-10004200","platformFee":"0.099","platformFeeMinUnits":"99000","referralFee":"0.0198",` +
		`"referralFeeMinUnits":"19800","organizationFee":"0.9901","organizationFeeMinUnits":"990100","netAmount":"-10.0042","netAmountMinUnits":"-10004200",` +
		`"networkFeesPaidBy":"merchant","networkFees":"0.0000012","networkFeesMinUnits":"1200000000000","networkFeesCurrency":"ETH",` +
		`"networkFeesCurrencyDecimals":18,"networkFeesCurrencyPriceUsd":"4512.37","networkFeesUsd":"0.00541","platformFeePercent":"0.99",` +
		`"referralFeePercent":"20","organizationFeePercent":"10","fromAddress":"0xf","toAddress":"0xt","txHash":"0xh","blockNumberOrSlot":"52468103",` +
		`"tokenContractOrMint":"0x8335","explorerUrl":"https://e","receiptUrl":"https://r","txUuid":"refund@1","relatedTxUuid":"pay@0",` +
		`"productUuid":"p-1","productReference":"pro","productDescription":"d","customerUuid":"c-1","userUuid":"m-1"}`, strict[AccountingEvent]},
	"WebhookEndpointCreated": {`{` + fxEndpoint + `,"secret":"whsec_x"}`, strict[WebhookEndpointCreated]},
	"EventDetail": {`{"id":"evt_1","type":"payment.completed","version":"v2","createdAt":"2026-10-01T12:00:00.123Z","test":false,"userUuid":"m-1",` +
		`"data":{"uuid":"pay@1"},"deliveries":[{"endpointUuid":"e-1","url":"https://x.io/hook","delivered":true,"attempts":[{"uuid":"a-1",` +
		`"eventId":"evt_1","eventType":"payment.completed","endpointUuid":"e-1","attempt":2,"delivered":false,"skipped":true,"statusCode":500,` +
		`"error":"500 Internal Server Error","durationMs":120,"attemptedAt":"2026-10-01T12:00:01Z"}]}]}`, strict[EventDetail]},
	"Me": {`{"credential":"apiKey","apiKeyUuid":"k-1","userUuid":"u-1","role":"user","onBehalfOfUserUuid":"m-1","space":{"uuid":"s-1",` +
		`"organizationUuid":"o-1","organizationName":"Shop","userUuid":"m-1","test":true,"role":"admin",` +
		`"member":{"userUuid":"m-1","firstName":"Ada","lastName":"L","email":"a@b.co"},"organizationFeePercent":"7.5",` +
		`"trustedAt":"2026-10-01T12:00:00Z","createdAt":"2026-09-01T12:00:00Z"}}`, strict[Me]},
	"Page": {`{"items":[` + fxSummary + `],"hasMore":true,"nextCursor":"djF8MDE5Y2FkZmQ"}`, strict[Page[CustomerSummary]]},
}

// strict decodes with unknown fields refused: every fixture key has a field.
func strict[T any](data []byte) (any, error) {
	var v T
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	err := dec.Decode(&v)
	return v, err
}

func TestModelsMatchTheWire(t *testing.T) {
	for name, fx := range modelFixtures {
		v, err := fx.decode([]byte(fx.json))
		if err != nil {
			t.Errorf("%s: strict decode: %v", name, err)
			continue
		}
		// Round trip: the model writes back exactly the fixture's keys (no field without a
		// wire key, no wire key lost), at every level.
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var want, got any
		_ = json.Unmarshal([]byte(fx.json), &want)
		_ = json.Unmarshal(out, &got)
		if path, ok := sameKeys(want, got, name); !ok {
			t.Errorf("keys differ at %s\nfixture: %s\nmodel:   %s", path, fx.json, out)
		}
	}
}

// sameKeys compares the object keys of two decoded JSON trees.
func sameKeys(a, b any, path string) (string, bool) {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return path, false
		}
		ak, bk := slices.Sorted(maps.Keys(av)), slices.Sorted(maps.Keys(bv))
		if !slices.Equal(ak, bk) {
			return path + " " + jsonString(ak) + " vs " + jsonString(bk), false
		}
		for k := range av {
			if p, ok := sameKeys(av[k], bv[k], path+"."+k); !ok {
				return p, false
			}
		}
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return path, false
		}
		for i := range av {
			if p, ok := sameKeys(av[i], bv[i], path+"[]"); !ok {
				return p, false
			}
		}
	}
	return path, true
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func fixture[T any](t *testing.T, name string) T {
	t.Helper()
	v, err := modelFixtures[name].decode([]byte(modelFixtures[name].json))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return v.(T)
}

func TestModelValues(t *testing.T) {
	p := fixture[Payment](t, "Payment")
	if p.Amount != "10" || p.AmountMinUnits != "10000000" || p.AmountUSD != "10" || p.PriceUSD != "8.5" || p.ProductName != "Pro" ||
		p.Currency.Code != "BASE:USDC" || p.Currency.Chain != ChainBASE || p.Currency.MainCurrencyID == nil || *p.Currency.MainCurrencyID != 3 ||
		p.Metadata["orderId"] != "order-1" || p.Paid != "10.0042" || p.PaidUSD != "10.0042" {
		t.Errorf("payment %+v", p)
	}
	if p.Customer == nil || !p.Customer.Deleted || p.Customer.FirstName != "Ada" || p.Refund == nil || p.Refund.Status != RefundStatusApproved ||
		p.Refund.RefundPercent != "50" || p.Refund.Amount != "5.0021" || p.Refund.RespondedAt == nil ||
		p.Refundable == nil || *p.Refundable || p.NotRefundableReason != NotRefundableReasonRefundExists {
		t.Errorf("payment customer and refund %+v %+v", p.Customer, p.Refund)
	}
	st := p.Settlement
	if st.FeePercent != "0.99" || st.OrganizationFee == nil || st.OrganizationFee.Address != "0xorg" || st.OrganizationFee.FeePercent != "10" ||
		st.ReferralFee == nil || st.ReferralFee.FeePercent != "20" || st.Split == nil || st.Split.Merchant.Amount != "8.9109" ||
		st.Split.Platform.AmountMinUnits != "79200" || st.Split.NetworkFee == nil || st.Split.NetworkFee.AmountUSD != "0.0042" ||
		st.Transaction == nil || st.Transaction.BlockNumber != "52468103" || st.Transaction.GasUsed == nil || *st.Transaction.GasUsed != 95000 ||
		st.Transaction.Fee.AmountMinUnits != "1140000000000" || st.Transaction.L1Fee == nil || st.Transaction.NativePriceUSD != "4512.37" ||
		st.Transaction.BlockTime == nil || !st.Transaction.BlockTime.Equal(time.Date(2026, 10, 1, 11, 59, 58, 0, time.UTC)) {
		t.Errorf("settlement %+v", st)
	}
	if len(p.Fees) != 2 || p.Fees[0] != (FeeLine{Type: FeeLineTypeCustom, Label: "VAT (20%)", Description: "French VAT", AmountUSD: "1.30"}) ||
		p.Fees[1] != (FeeLine{Type: FeeLineTypeProcessingFee, Label: "Processing fee", AmountUSD: "0.20"}) {
		t.Errorf("payment fees %+v", p.Fees)
	}

	native := fixture[Currency](t, "Currency/native")
	if native.Address != "" || native.MainCurrencyID != nil || native.Code != "ETH" || native.Decimals != 18 {
		t.Errorf("native coin %+v", native)
	}

	ss := fixture[SubscriptionSessionData](t, "SubscriptionSessionData")
	if ss.AmountUSD != "4.81" || ss.PriceUSD != "3.99" || len(ss.Fees) != 2 || ss.TxType != TransactionTypeCreateSubscription ||
		len(ss.AvailableCurrencies) != 2 || ss.AvailableCurrencies[1].Code != "ETH" || ss.Metadata["orderId"] != "order-1044" ||
		ss.Frequency != (Duration{1, DurationUnitMonths}) || *ss.MinPeriods != 3 || !ss.UpgradingFromTrial {
		t.Errorf("session %+v", ss)
	}

	sb := fixture[Subscription](t, "Subscription")
	if sb.PriceUSD != "9.99" || sb.Allowance == nil || sb.Allowance.Amount != "110" || sb.Allowance.RemainingPeriods == nil ||
		*sb.Allowance.RemainingPeriods != 0 || sb.Allowance.MaxPerPeriod != "30" || sb.Allowance.MaxPerPeriodMinUnits != "30000000" ||
		sb.LastBilledAt == nil || sb.NextBillingAt == nil || sb.CancellableAt == nil || sb.Metadata["plan"] != "pro" ||
		sb.ManagementPageLink == "" || sb.Dunning == nil || sb.Dunning.RemainingAttempts != 3 || sb.Currency.Symbol != "USDC" {
		t.Errorf("subscription %+v", sb)
	}

	tx := fixture[Transaction](t, "Transaction")
	if tx.Type != TransactionKindBill || tx.PriceUSD != "8.5" || tx.PeriodEnd == nil || tx.Settlement.Split == nil || tx.Metadata["k"] != "v" {
		t.Errorf("transaction %+v", tx)
	}

	tb := fixture[TestBill](t, "TestBill")
	if !strings.HasPrefix(tb.BillUUID, "sub-hist@") || tb.Stage != BillingStageWaiting || tb.NextAttemptAt == nil {
		t.Errorf("test bill %+v", tb)
	}

	r := fixture[Refund](t, "Refund")
	if r.Paid != "10.0042" || r.RefundPercent != "50" || r.Amount != "5.0021" || r.Settlement == nil || r.Settlement.Transaction == nil ||
		r.Settlement.Transaction.BlockNumber != "52468103" || r.Currency.ID != 8 || r.CustomerUUID != "c-1" {
		t.Errorf("refund %+v", r)
	}

	ae := fixture[AccountingEvent](t, "AccountingEvent")
	if ae.Currency != "USDC" || ae.CurrencyPriceUSD != "1.00002" || ae.NetworkFeesPaidBy != NetworkFeesPaidByMerchant ||
		ae.NetworkFeesCurrency != "ETH" || ae.NetworkFeesCurrencyDecimals == nil || *ae.NetworkFeesCurrencyDecimals != 18 ||
		ae.NetworkFeesCurrencyPriceUSD != "4512.37" || ae.NetAmount != "-10.0042" || ae.GrossAmountMinUnits != "-10004200" ||
		ae.TxUUID != "refund@1" || ae.RelatedTxUUID != "pay@0" || ae.PlatformFeePercent != "0.99" ||
		!ae.TxTimeUTC.Equal(time.Date(2026, 9, 13, 19, 23, 26, 620000000, time.UTC)) {
		t.Errorf("accounting event %+v", ae)
	}
	// A referralFee row: only what the referrer earned; no network fees, no payer; an absent
	// fee is "", never "0".
	rf, err := decodeJSON[AccountingEvent]([]byte(`{"txTimeUtc":"2026-09-13T19:23:26Z","type":"referralFee","referralFeeUsd":"0.02",` +
		`"chain":"SOL","currency":"SOL","currencyDecimals":9,"currencyPriceUsd":"150.5","referralFee":"0.000132","referralFeeMinUnits":"132000",` +
		`"referralFeePercent":"20","blockNumberOrSlot":"1"}`))
	if err != nil || rf.Type != AccountingEventTypeReferralFee || rf.NetworkFeesPaidBy != "" || rf.NetworkFeesCurrencyDecimals != nil ||
		rf.NetworkFees != "" || rf.ReferralFee != "0.000132" || rf.Currency != "SOL" || rf.GrossAmountUSD != "" || rf.OrganizationFee != "" ||
		rf.TokenContractOrMint != "" || rf.TxUUID != "" {
		t.Errorf("referral fee row %+v %v", rf, err)
	}

	inv := fixture[CreatedInvitation](t, "CreatedInvitation")
	if inv.UUID != "i-1" || inv.Test == nil || !*inv.Test || inv.HoldFunds == nil || *inv.HoldFunds || inv.Link == "" ||
		inv.OrganizationFeePercent != "0" {
		t.Errorf("created invitation %+v", inv)
	}
	// A team invitation: mode-free, so test, holdFunds and the fee are absent.
	team, err := decodeJSON[Invitation]([]byte(`{"uuid":"i-2","email":"a@b.co","role":"admin","expiresAt":"2026-10-08T12:00:00Z",` +
		`"createdAt":"2026-10-01T12:00:00Z","status":"pending"}`))
	if err != nil || team.Test != nil || team.HoldFunds != nil || team.OrganizationFeePercent != "" || team.Role != RoleAdmin || team.AcceptedAt != nil {
		t.Errorf("team invitation %+v %v", team, err)
	}

	hf := fixture[HeldFunds](t, "HeldFunds")
	if hf.TotalUSD != "-1.0933" || hf.Ledgers[0].Settlement == nil || hf.Ledgers[1].Settlement != nil ||
		hf.Ledgers[1].Type != LedgerEntryTypeRefund || hf.Ledgers[1].Owed != "-10.0042" || hf.Ledgers[1].RefundedTxUUID != "pay@1" {
		t.Errorf("held funds %+v", hf)
	}

	w := fixture[Wallet](t, "Wallet")
	if w.Address != "0xpk" || w.Currency.Code != "ETH" || w.TokenWallets[0].Currency.Code != "BASE:USDC" || w.TokenWallets[0].Balance == nil ||
		w.TokenWallets[0].Balance.Amount != "12.5" || w.TokenWallets[0].Balance.Currency.ID != 8 {
		t.Errorf("wallet %+v", w)
	}

	m := fixture[Member](t, "Member")
	if m.FirstName != "Ada" || m.OrganizationFeePercent != "7.5" || len(m.AcceptedCurrencies) != 1 || m.AcceptedCurrencies[0].Code != "BASE:USDC" {
		t.Errorf("member %+v", m)
	}

	me := fixture[Me](t, "Me")
	if me.OnBehalfOfUserUUID != "m-1" || me.Space == nil || me.Space.Role != RoleAdmin || me.Space.OrganizationFeePercent != "7.5" ||
		me.Space.Member == nil || me.Space.Member.FirstName != "Ada" || me.Space.TrustedAt == nil {
		t.Errorf("me %+v", me)
	}
	// The session-only fields (user, spaces, reauthenticatedUntil) are not modelled: ignored.
	if me2, err := decodeJSON[Me]([]byte(`{"credential":"apiKey","spaces":[],"user":{"uuid":"u"}}`)); err != nil || me2.Credential != CredentialAPIKey {
		t.Errorf("me with session fields %+v %v", me2, err)
	}

	page := fixture[Page[CustomerSummary]](t, "Page")
	if !page.HasMore || page.NextCursor != "djF8MDE5Y2FkZmQ" || len(page.Items) != 1 {
		t.Errorf("page %+v", page)
	}
}

// TestModelOptionalFields decodes the minimal shapes the docs allow: every O field absent
// decodes to its zero value ("" or nil), every L field absent to an empty slice.
func TestModelOptionalFields(t *testing.T) {
	sub, err := decodeJSON[Subscription]([]byte(`{"uuid":"sub@1","to":"0xt","status":"trial","priceUsd":"9.99","fees":[],"metadata":{},` +
		`"currency":` + fxCurrency + `,"frequency":{"value":1,"unit":"months"},"currentPeriodEnd":"2026-10-15T12:00:00Z"}`))
	if err != nil || sub.From != "" || sub.ProductUUID != "" || sub.SubscriptionHash != "" || sub.Allowance != nil || sub.LastBilledAt != nil ||
		sub.NextBillingAt != nil || sub.CancellableAt != nil || sub.Dunning != nil || sub.ActionRequired != "" || sub.CurrentPeriodEnd == nil {
		t.Errorf("trial %+v %v", sub, err)
	}
	pay, err := decodeJSON[Payment]([]byte(`{"uuid":"pay@1","amount":"10","priceUsd":"10","fees":[],"metadata":{},"settlement":{"feePercent":"0.99"}}`))
	if err != nil || pay.Settlement.Split != nil || pay.Settlement.Transaction != nil || pay.ConfirmedAt != nil || pay.Paid != "" ||
		pay.Refund != nil || pay.Refundable != nil || pay.Customer != nil || pay.ProductDescription != "" {
		t.Errorf("minimal payment %+v %v", pay, err)
	}
	ref, err := decodeJSON[Refund]([]byte(`{"uuid":"refund@1","txUuid":"pay@1","status":"pending","paid":"10","paidMinUnits":"10000000",` +
		`"refundPercent":"100","amount":"10","amountMinUnits":"10000000"}`))
	if err != nil || ref.Reason != "" || ref.MerchantMessage != "" || ref.RespondedAt != nil || ref.TxHash != "" || ref.PaidUSD != "" ||
		ref.AmountUSD != "" || ref.Settlement != nil || ref.Approval != nil {
		t.Errorf("pending refund %+v %v", ref, err)
	}
	// Lenient on absence and null (behaviour §6): zero values, never an error.
	nulls, err := decodeJSON[Product]([]byte(`{"uuid":"p-1","description":null,"metadata":null,"subscription":null,"priceUsd":null}`))
	if err != nil || nulls.Description != "" || nulls.Metadata != nil || nulls.Subscription != nil || nulls.PriceUSD != "" {
		t.Errorf("nulls %+v %v", nulls, err)
	}
}

// TestModelStrictTypes refuses a present value of the wrong JSON type: money, token amounts
// and rates are decimal strings (a JSON number is refused, never parsed into a float), and an
// object field refuses a list. Integral floats are accepted into integer fields.
func TestModelStrictTypes(t *testing.T) {
	for name, decode := range map[string]func() error{
		"payment amount as a number":     func() error { _, err := decodeJSON[Payment]([]byte(`{"amount":10}`)); return err },
		"payment priceUsd as a number":   func() error { _, err := decodeJSON[Payment]([]byte(`{"priceUsd":8.5}`)); return err },
		"fee amountUsd as a number":      func() error { _, err := decodeJSON[Payment]([]byte(`{"fees":[{"amountUsd":2}]}`)); return err },
		"fees as an object":              func() error { _, err := decodeJSON[Payment]([]byte(`{"fees":{"type":"custom"}}`)); return err },
		"settlement feePercent a number": func() error { _, err := decodeJSON[Payment]([]byte(`{"settlement":{"feePercent":0.99}}`)); return err },
		"product priceUsd as a number":   func() error { _, err := decodeJSON[Product]([]byte(`{"priceUsd":1e2}`)); return err },
		"member fee percent a number":    func() error { _, err := decodeJSON[Member]([]byte(`{"organizationFeePercent":5}`)); return err },
		"refund percent as a number":     func() error { _, err := decodeJSON[Refund]([]byte(`{"refundPercent":100}`)); return err },
		"held funds total as a number":   func() error { _, err := decodeJSON[HeldFunds]([]byte(`{"totalUsd":-10}`)); return err },
		"balance amount as a number":     func() error { _, err := decodeJSON[Balance]([]byte(`{"amount":12.5}`)); return err },
		"accounting gross as a number":   func() error { _, err := decodeJSON[AccountingEvent]([]byte(`{"grossAmount":-10.0042}`)); return err },
		"currency as an id":              func() error { _, err := decodeJSON[Payment]([]byte(`{"currency":8}`)); return err },
		"allowance as a string":          func() error { _, err := decodeJSON[Subscription]([]byte(`{"allowance":"110000000"}`)); return err },
		"page hasMore as a string": func() error {
			_, err := decodeJSON[Page[Product]]([]byte(`{"items":[],"hasMore":"false"}`))
			return err
		},
		"metadata value as a number": func() error { _, err := decodeJSON[Customer]([]byte(`{"metadata":{"k":1}}`)); return err },
	} {
		if decode() == nil {
			t.Errorf("%s: decoded, want an error", name)
		}
	}

	// Integral floats into integer fields (1.0 → 1).
	f, err := decodeJSON[Failure]([]byte(`{"attempt":2.0,"currency":{"id":8.0,"decimals":6.0}}`))
	if err != nil || f.Attempt != 2 || f.Currency.ID != 8 || f.Currency.Decimals != 6 {
		t.Errorf("failure %+v %v", f, err)
	}
	s, err := decodeJSON[Subscription]([]byte(`{"allowance":{"amount":"1","amountMinUnits":"1","remainingPeriods":11.0}}`))
	if err != nil || s.Allowance == nil || *s.Allowance.RemainingPeriods != 11 {
		t.Errorf("subscription %+v %v", s, err)
	}
}

// TestModelFees decodes the checkout fees on a payment: an empty list, a line without
// description, an unknown line type kept.
func TestModelFees(t *testing.T) {
	none, err := decodeJSON[Payment]([]byte(`{"uuid":"pay@1","amount":"10","priceUsd":"10","fees":[]}`))
	if err != nil || none.PriceUSD != "10" || len(none.Fees) != 0 {
		t.Errorf("without fees: %+v %v", none, err)
	}
	absent, err := decodeJSON[Payment]([]byte(`{"uuid":"pay@1","fees":null}`))
	if err != nil || absent.PriceUSD != "" || len(absent.Fees) != 0 {
		t.Errorf("price absent, fees null: %+v %v", absent, err)
	}

	p, err := decodeJSON[Payment]([]byte(`{"amountUsd":"12.03","priceUsd":"10","fees":[{"type":"tip","label":"Tip","amountUsd":"2"},` +
		`{"type":"processingFee","label":"Processing fee","description":null,"amountUsd":"0.03"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.Fees[0].Type != "tip" || p.Fees[0].AmountUSD != "2" || p.Fees[1].Description != "" || p.Fees[1].Type != FeeLineTypeProcessingFee {
		t.Errorf("fees %+v", p.Fees)
	}
}
