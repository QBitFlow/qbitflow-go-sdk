package qbitflow

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

// Fixtures with every wire key of each response model (docs/core Go tags), all non-empty.
const (
	fxCurrency = `{"id":8,"name":"USD Coin","symbol":"USDC","decimals":6,"address":"0x8335","mainCurrencyId":3,
		"mainCurrency":{"id":3,"name":"Base","symbol":"BASE","decimals":18,"address":"","mainCurrency":null,"test":false},"test":false}`
	fxSummary  = `{"uuid":"c-1","name":"Ada","lastName":"Lovelace","email":"ada@example.com","reference":"crm-1","deleted":true}`
	fxRefundSm = `{"uuid":"refund@1","status":"pending","initiatedBy":"merchant","refundPercent":50,"amountMinUnits":"5000000","amountUsd":5,
		"createdAt":"2026-10-01T12:00:00Z","respondedAt":null}`
	fxAttempt  = `{"status":"failed","code":"insufficient_funds","message":"Not enough USDC","txHash":"0xa","at":"2026-10-01T12:00:00+02:00"}`
	fxMetadata = `{"feePercent":1.5,"organizationFee":{"organization":"0xorg","feePercent":10},"referralFee":{"referrer":"0xref","feePercent":20,"deadline":"2027-01-01T00:00:00Z"},
		"txMetadata":{"networkFees":{"amount":"1140000000000","unitsConsumed":95000,"l1Fee":"1000"},"blockData":{"number":"52468103","timestamp":1790855998},"mainCurrencyPriceUsd":4512.37},
		"txAmounts":{"usd":{"platform":0.15,"organization":0.98,"referral":0.03,"merchant":8.87,"networkFee":0.0042},
		"minUnits":{"platform":"150000","organization":"985000","referral":"30000","merchant":"8865000","networkFee":"4200"}}}`
	fxTransfer = `"createdAt":"2026-10-01T12:00:00Z","from":"0xfrom","to":"0xto","amount":10,"amountMinUnits":"10000000","currencyId":8,
		"currency":` + fxCurrency + `,"txHash":"0xhash","chain":"BASE","explorerUrl":"https://basescan.org/tx/0xhash","test":false,"userUuid":"m-1"`
)

var modelFixtures = map[string]struct {
	json   string
	decode func([]byte) (any, error)
}{
	"Currency": {fxCurrency, strict[Currency]},
	"Payment": {`{"uuid":"pay@1",` + fxTransfer + `,"reference":"order-1","name":"Pro","description":"Lifetime","productUuid":"p-1",
		"customerUuid":"c-1","customerReference":"crm-1","note":"thanks","customer":` + fxSummary + `,"metadata":` + fxMetadata + `,
		"confirmedAt":"2026-10-01T11:59:58Z","paidMinUnits":"10004200","paidUsd":10.0042,"refund":` + fxRefundSm + `,"refundable":false,
		"notRefundableReason":"refundExists","checkoutOpenedAt":"2026-10-01T11:00:00Z"}`, strict[Payment]},
	"Bill": {`{"uuid":"sub-hist@1",` + fxTransfer + `,"name":"Pro","description":"Monthly","productUuid":"p-1","subscriptionUuid":"sub@1",
		"customerUuid":"c-1","customerReference":"crm-1","customer":` + fxSummary + `,"metadata":` + fxMetadata + `,
		"periodStart":"2026-10-01T12:00:00Z","periodEnd":"2026-10-31T12:00:00Z","confirmedAt":"2026-10-01T11:59:58Z","paidMinUnits":"1",
		"paidUsd":1,"refund":` + fxRefundSm + `,"refundable":true,"notRefundableReason":"heldFundsReleased"}`, strict[Bill]},
	"CombinedPayment": {`{"source":"subscriptionHistory","uuid":"sub-hist@1","createdAt":"2026-10-01T12:00:00Z","from":"0xf","to":"0xt","name":"Pro",
		"description":"d","amount":10,"amountMinUnits":"10000000","currencyId":8,"currency":` + fxCurrency + `,"productUuid":"p-1","txHash":"0xh",
		"customerUuid":"c-1","customer":` + fxSummary + `,"customerReference":"crm-1","subscriptionUuid":"sub@1","reference":"r","subscriptionReference":"sr",
		"chain":"SOL","explorerUrl":"https://x","test":true,"userUuid":"m-1","refund":` + fxRefundSm + `,"refundable":false,
		"notRefundableReason":"refundExists","metadata":` + fxMetadata + `}`, strict[CombinedPayment]},
	"Failure": {`{"uuid":"f-1","kind":"bill","txUuid":"sub-hist@1","attempt":2,"subscriptionUuid":"sub@1","code":"insufficient_funds",
		"category":"insufficientBalance","message":"m","attemptedUsd":9.99,"attemptedMinUnits":"9990000","currencyId":8,"from":"0xf","txHash":"0xh",
		"customerUuid":"c-1","productUuid":"p-1","createdAt":"2026-10-01T12:00:00Z","customer":` + fxSummary + `,"test":false,"userUuid":"m-1"}`, strict[Failure]},
	"CheckoutSession":       {`{"link":"https://qbitflow.app/checkout?uuid=pay@1","uuid":"pay@1","expiresAt":"2026-10-02T12:00:00Z"}`, strict[CheckoutSession]},
	"CheckoutSessionStatus": {`{"uuid":"pay@1","status":"created","txHash":"0xh","message":"m","lastAttempt":` + fxAttempt + `}`, strict[CheckoutSessionStatus]},
	"Subscription": {`{"uuid":"sub@1","reference":"r","createdAt":"2026-09-01T12:00:00Z","updatedAt":"2026-10-01T12:00:00Z","from":"0xf","to":"0xt",
		"productUuid":"p-1","subscriptionHash":"0x2","currencyId":8,"currency":` + fxCurrency + `,"frequency":{"value":1,"unit":"months"},
		"allowance":"110000000","status":"stopped","actionRequired":"raiseMaximum","currentPeriodEnd":"2026-10-31T12:00:00Z","priceUsd":"9.99",
		"maxAmountPerPeriod":"30000000","lastBillingDate":"2026-10-01T12:00:00Z","nextBillingDate":"2026-10-31T12:00:00Z",
		"cancelledAt":"2026-10-02T12:00:00Z","cancellationReason":"merchant","minimumCancellationDate":"2026-12-01T12:00:00Z","test":false,
		"userUuid":"m-1","customerUuid":"c-1","customerReference":"crm-1","customer":` + fxSummary + `,
		"dunning":{"failedAttempts":1,"remainingAttempts":3,"failingSince":"2026-10-01T12:00:00Z","nextAttemptAt":"2026-10-02T12:00:00Z","awaitingMaximumSince":"2026-10-01T12:00:00Z"}}`, strict[Subscription]},
	"BillingState": {`{"billUuid":"019c-1","stage":"done","outcome":"paid","attempts":0,"nextAttempt":"2026-10-02T12:00:00Z","txHash":"0xh","failureCode":"x"}`, strict[BillingState]},
	"Refund": {`{"uuid":"refund@1","txUuid":"pay@1","reason":"r","status":"pending","createdAt":"2026-10-01T12:00:00Z","merchantMessage":"m",
		"respondedAt":"2026-10-01T13:00:00Z","test":false,"userUuid":"m-1","txHash":"0xh","initiatedBy":"customer","held":true,"paidMinUnits":"10004200",
		"paidUsd":10.0042,"refundPercent":100,"amountMinUnits":"10004200","amountUsd":10.0042,"currencyId":8,
		"metadata":{"networkFees":{"amount":"1","unitsConsumed":1},"blockData":{"number":"1","timestamp":1}},"chain":"ETH","explorerUrl":"https://e",
		"approval":{"status":"waitingConfirmation","txHash":"0xa","lastAttempt":` + fxAttempt + `},"productName":"Pro","customer":` + fxSummary + `}`, strict[Refund]},
	"Customer": {`{"uuid":"c-1","name":"Ada","lastName":"Lovelace","email":"ada@example.com","verified":true,"phoneNumber":"+33 6 12 34 56 78",
		"address":"1 rue X","reference":"crm-1","createdAt":"2026-10-01T12:00:00Z","test":true,"userUuid":"m-1"}`, strict[Customer]},
	"Product": {`{"uuid":"p-1","name":"Pro","description":"d","price":9.99,"createdAt":"2026-10-01T12:00:00Z","isActive":true,"reference":"pro",
		"subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":3},
		"paymentLink":"https://qbitflow.app/pay/p-1","test":false,"userUuid":"m-1"}`, strict[Product]},
	"Member": {`{"userUuid":"m-1","name":"Ada","lastName":"L","email":"a@b.co","test":false,"organizationFeePercent":0,"trustedAt":null,
		"joinedAt":"2026-10-01T12:00:00Z","acceptedCurrencyIds":[2,5,8],"spaceUuid":"s-1"}`, strict[Member]},
	"HeldFunds": {`{"ledgers":[{"txUuid":"refund@1","type":"refund","description":"Refund of payment pay@1","refundedTxUuid":"pay@1","amount":10,
		"metadata":null,"currencyId":8,"owedMinUnits":"-10004200","owedUsd":-10.0042,"createdAt":"2026-10-01T12:00:00Z"}],"totalAmount":-10.0042}`, strict[HeldFunds]},
	"MemberHeldFundsSummary": {`{"userUuid":"m-1","totalAmount":98.5,"count":2,"oldestAt":"2026-09-28T12:00:00Z"}`, strict[MemberHeldFundsSummary]},
	"Wallet": {`{"uuid":"w-1","createdAt":"2026-10-01T12:00:00Z","publicKey":"0xpk","test":false,"userUuid":"m-1","tokenWallets":[
		{"uuid":"tw-1","createdAt":"2026-10-01T12:00:00Z","tokenId":8,"token":` + fxCurrency + `,"walletUuid":"w-1",
		"balance":{"currency":"BASE:USDC","balance":"1000000","balanceUsd":1}}],"currencyId":3,"currency":` + fxCurrency + `}`, strict[Wallet]},
	"Invitation": {`{"uuid":"i-1","test":false,"email":"a@b.co","role":"user","trustLayer":true,"organizationFeePercent":2.5,
		"redirectUrl":"https://x.io/welcome","expiresAt":"2026-10-08T12:00:00Z","acceptedByUserUuid":"m-1","acceptedAt":"2026-10-02T12:00:00Z",
		"revokedAt":null,"createdAt":"2026-10-01T12:00:00Z","status":"accepted"}`, strict[Invitation]},
	"InvitationCreated": {`{"invitation":{"uuid":"i-1","test":null,"email":"a@b.co","role":"admin","trustLayer":null,"organizationFeePercent":0,
		"expiresAt":"2026-10-08T12:00:00Z","acceptedAt":null,"revokedAt":null,"createdAt":"2026-10-01T12:00:00Z","status":"pending"},
		"link":"https://qbitflow.app/invite?token=x"}`, strict[InvitationCreated]},
	"AccountingEvent": {`{"paymentUuid":"pay@1","paymentReference":"r","type":"refund","txTimeUtc":"2026-09-13T21:23:26.620071+02:00","receiptUrl":"https://r",
		"relatedPaymentUuid":"pay@0","relatedPaymentReference":"r0","userUuid":"m-1","productUuid":"p-1","productReference":"pro","productName":"Pro",
		"productDescription":"d","customerUuid":"c-1","customerReference":"crm-1","chain":"BASE","blockNumberOrSlot":"52468103","txHash":"0xh",
		"fromAddress":"0xf","toAddress":"0xt","tokenSymbol":"USDC","currencyDecimals":6,"tokenContractOrMint":"0x8335","explorerUrl":"https://e",
		"grossAmount":"10000000","grossAmountUsd":10,"platformFeePercent":1.5,"platformFeeUsd":0.15,"platformFee":"150000",
		"organizationFeePercent":0,"organizationFeeUsd":0,"organizationFee":"0","referralFeePercent":20,"referralFeeUsd":0.03,"referralFee":"30000",
		"networkFeesUsd":0.0042,"networkFees":"4200","netAmountUsd":9.85,"netAmount":"9850000","userName":"Ada","userLastName":"L",
		"customerName":"Bob","customerLastName":"B"}`, strict[AccountingEvent]},
	"WebhookEndpointCreated": {`{"uuid":"e-1","test":true,"url":"https://x.io/hook","events":["payment.completed"],"includeMembers":true,
		"payloadVersion":"v2","description":"d","createdAt":"2026-10-01T12:00:00Z","rotatedAt":"2026-10-01T12:00:00Z","disabledAt":"2026-10-02T12:00:00Z",
		"disabledReason":"failing","failingSince":"2026-09-29T12:00:00Z","lastDeliveredAt":"2026-09-28T12:00:00Z","secret":"whsec_x"}`, strict[WebhookEndpointCreated]},
	"Me": {`{"credential":"apiKey","apiKeyUuid":"k-1","userUuid":"u-1","role":"admin","onBehalfOf":"m-1","space":{"uuid":"s-1","organizationUuid":"o-1",
		"organizationName":"Shop","userUuid":"m-1","member":{"userUuid":"m-1","name":"Ada","lastName":"L","email":"a@b.co"},"test":true}}`, strict[Me]},
	"Page": {`{"items":[` + fxSummary + `],"nextCursor":"c-1"}`, strict[Page[CustomerSummary]]},
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

func TestModelValues(t *testing.T) {
	v, _ := modelFixtures["Payment"].decode([]byte(modelFixtures["Payment"].json))
	p := v.(Payment)
	if p.Customer == nil || p.Customer.Deleted == nil || !*p.Customer.Deleted || p.Refund == nil || p.Refund.RespondedAt != nil ||
		p.Refundable == nil || *p.Refundable || p.NotRefundableReason != NotRefundableReasonRefundExists ||
		p.Metadata.OrganizationFee == nil || p.Metadata.OrganizationFee.FeePercent != 10 || *p.Metadata.TxAmounts.USD.Organization != 0.98 {
		t.Errorf("payment %+v", p)
	}
	v, _ = modelFixtures["InvitationCreated"].decode([]byte(modelFixtures["InvitationCreated"].json))
	inv := v.(InvitationCreated)
	if inv.Invitation.Test != nil || inv.Invitation.TrustLayer != nil || inv.Invitation.Role != RoleAdmin || inv.Link == "" {
		t.Errorf("team invitation %+v", inv)
	}
	v, _ = modelFixtures["HeldFunds"].decode([]byte(modelFixtures["HeldFunds"].json))
	hf := v.(HeldFunds)
	if hf.TotalAmount >= 0 || hf.Ledgers[0].Metadata != nil || hf.Ledgers[0].OwedMinUnits != "-10004200" {
		t.Errorf("held funds %+v", hf)
	}
	// Integral floats into integer fields; JSON ints into percent floats.
	m, err := decodeJSON[Member]([]byte(`{"organizationFeePercent":2,"acceptedCurrencyIds":[8.0]}`))
	if err != nil || m.OrganizationFeePercent != 2 || m.AcceptedCurrencyIDs[0] != 8 {
		t.Errorf("member %+v %v", m, err)
	}
}
