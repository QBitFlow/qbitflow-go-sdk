package qbitflow

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// Ids used by the service tests.
const (
	uuidA   = "019eca82-5680-7b00-8000-0000000000c1"
	uuidB   = "019eca82-5680-7b00-8000-0000000000c2"
	payID   = "pay@019eca82-5680-7b00-8000-0000000000d1"
	subID   = "sub@019eca82-5680-7b00-8000-0000000000d2"
	billID  = "sub-hist@019eca82-5680-7b00-8000-0000000000d3"
	eventID = "evt_bd1a913d-188e-5ac5-a2d6-5fd25cd67301"
	ref     = "ord:1042@shop" // a reference, sent as the ?reference= filter
	refEs   = "ord%3A1042%40shop"
	cursorA = "djF8MDE5ZWNhODI" // opaque cursors
	cursorB = "djF8MDE5ZWNhODM"
)

// Instants with a "+" offset: the query must carry it as %2B.
var (
	after  = time.Date(2026, 9, 1, 12, 0, 0, 0, time.FixedZone("", 2*3600))
	before = time.Date(2026, 10, 1, 0, 0, 0, 500_000_000, time.UTC)
)

const (
	afterQ  = "2026-09-01T12%3A00%3A00%2B02%3A00"
	beforeQ = "2026-10-01T00%3A00%3A00.5Z"
)

func fx(name string) string { return modelFixtures[name].json }

// page is a page of a paginated list that has more.
func page(item string) string {
	return `{"items":[` + item + `],"hasMore":true,"nextCursor":"` + cursorB + `"}`
}

// bounded is a bounded list's (or a last page's) envelope.
// withReference returns the fixture JSON object with its reference set to ref (the getByReference
// helpers return only the item carrying the reference they asked for).
func withReference(t *testing.T, fixture, ref string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(fixture), &m); err != nil {
		t.Fatal(err)
	}
	m["reference"] = ref
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func bounded(items ...string) string {
	return `{"items":[` + strings.Join(items, ",") + `],"hasMore":false}`
}

func eventFixture(t *testing.T) string {
	data, err := os.ReadFile("testdata/events/webhook.test.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// is asserts v's dynamic type and returns it.
func is[T any](t *testing.T, v any) T {
	t.Helper()
	got, ok := v.(T)
	if !ok {
		t.Fatalf("result is %T, want %T", v, *new(T))
	}
	return got
}

// routeCase is one method call and the request it must send.
type routeCase struct {
	name string
	call func(ctx context.Context, c *Client, opts ...RequestOption) (any, error)

	method, path, query string
	body                string // the JSON body expected ("" = no body)
	idempotent          bool
	accept              string // expected Accept (default application/json)

	status      int    // reply status (default 200)
	reply       string // reply body
	contentType string // reply Content-Type (default application/json)

	check func(t *testing.T, v any)
}

// routeCases covers every service method (the 54 routes of surface.md §2, plus Me).
func routeCases(t *testing.T) []routeCase {
	evt := eventFixture(t)
	no := false
	yes := true
	return []routeCase{
		// Products.
		{name: "Products.List", method: "GET", path: "/products", query: "cursor=" + cursorA + "&includeHidden=true&limit=50&subscription=false",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.List(ctx, &ProductListParams{Limit: 50, Cursor: cursorA, IncludeHidden: true, Subscription: &no}, o...)
			},
			reply: page(fx("Product")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Product]](t, v); len(p.Items) != 1 || p.Items[0].UUID != "p-1" || p.Items[0].Subscription == nil ||
					p.Items[0].PriceUSD != "9.99" || !p.HasMore || p.NextCursor != cursorB {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Products.List nil", method: "GET", path: "/products",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.List(ctx, nil, o...)
			},
			reply: bounded()},
		{name: "Products.Create", method: "POST", path: "/products", idempotent: true, status: 201,
			body: `{"name":"Pro","priceUsd":"9.99","reference":"pro","subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":0},"minPeriods":3},"metadata":{"sku":"PRO-1"}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Create(ctx, &CreateProductParams{Name: "Pro", PriceUSD: "9.99", Reference: "pro",
					Subscription: &SubscriptionTermsParams{Frequency: &Duration{1, DurationUnitMonths}, TrialPeriod: &Duration{}, MinPeriods: ptr[uint32](3)},
					Metadata:     map[string]string{"sku": "PRO-1"}}, o...)
			},
			reply: fx("Product"),
			check: func(t *testing.T, v any) {
				if p := is[*Product](t, v); p.UUID != "p-1" || p.PriceUSD != "9.99" || p.Metadata["sku"] != "PRO-1" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Products.Get", method: "GET", path: "/products/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Get(ctx, uuidA, o...)
			},
			reply: fx("Product")},
		{name: "Products.GetByReference", method: "GET", path: "/products", query: "reference=" + refEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.GetByReference(ctx, ref, o...)
			},
			reply: bounded(withReference(t, fx("Product"), ref)),
			check: func(t *testing.T, v any) {
				if p := is[*Product](t, v); p.UUID != "p-1" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Products.Update", method: "PATCH", path: "/products/" + uuidA,
			body: `{"description":"","priceUsd":"12.50","isActive":false,"metadata":{"sku":"","tier":"gold"}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Update(ctx, uuidA, &UpdateProductParams{Description: ptr(""), PriceUSD: "12.50", IsActive: &no,
					Metadata: map[string]string{"sku": "", "tier": "gold"}}, o...)
			},
			reply: fx("Product")},
		{name: "Products.Update remove subscription", method: "PATCH", path: "/products/" + uuidA, body: `{"removeSubscription":true}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Update(ctx, uuidA, &UpdateProductParams{RemoveSubscription: true}, o...)
			},
			reply: fx("Product")},
		{name: "Products.Delete", method: "DELETE", path: "/products/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Products.Delete(ctx, uuidA, o...)
			},
			status: 204},

		// Customers.
		{name: "Customers.Create", method: "POST", path: "/customers", idempotent: true, status: 201,
			body: `{"firstName":"Ada","lastName":"Lovelace","email":"ada@example.com","phoneNumber":"+33 6 12 34 56 78","reference":"crm-1","metadata":{"crm":"42"}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Create(ctx, &CreateCustomerParams{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com",
					PhoneNumber: "+33 6 12 34 56 78", Reference: "crm-1", Metadata: map[string]string{"crm": "42"}}, o...)
			},
			reply: fx("Customer"),
			check: func(t *testing.T, v any) {
				if c := is[*Customer](t, v); c.UUID != "c-1" || !c.Verified || c.UserUUID != "m-1" || c.FirstName != "Ada" || c.Metadata["crm"] != "42" {
					t.Errorf("%+v", c)
				}
			}},
		{name: "Customers.Update", method: "PATCH", path: "/customers/" + uuidA,
			body: `{"email":"new@example.com","lastName":"","phoneNumber":"","address":"","metadata":{"crm":""}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Update(ctx, uuidA, &UpdateCustomerParams{Email: "new@example.com", LastName: ptr(""), PhoneNumber: ptr(""),
					Address: ptr(""), Metadata: map[string]string{"crm": ""}}, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.Update first name", method: "PATCH", path: "/customers/" + uuidA, body: `{"firstName":"Augusta","lastName":"King"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Update(ctx, uuidA, &UpdateCustomerParams{FirstName: "Augusta", LastName: ptr("King")}, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.List", method: "GET", path: "/customers",
			query: "cursor=" + cursorA + "&email=ada%2B1%40example.com&limit=5&verified=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.List(ctx, &CustomerListParams{Limit: 5, Cursor: cursorA, Email: "ada+1@example.com", Verified: &yes}, o...)
			},
			reply: page(fx("Customer")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Customer]](t, v); len(p.Items) != 1 || !p.HasMore || p.NextCursor != cursorB {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Customers.Get", method: "GET", path: "/customers/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Get(ctx, uuidA, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.GetByReference", method: "GET", path: "/customers", query: "reference=" + refEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.GetByReference(ctx, ref, o...)
			},
			reply: bounded(withReference(t, fx("Customer"), ref))},
		{name: "Customers.Delete", method: "DELETE", path: "/customers/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Customers.Delete(ctx, uuidA, o...)
			},
			status: 204},

		// Checkout sessions.
		{name: "CheckoutSessions.CreatePayment", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"reference":"order-1","productName":"T-shirt","description":"Blue","priceUsd":"4.50","successUrl":"https://shop.example/ok?id={{UUID}}",` +
				`"cancelUrl":"https://shop.example/ko","customerReference":"crm-1","expiresInMinutes":30,"metadata":{"orderId":"order-1"}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{Reference: "order-1", ProductName: "T-shirt",
					Description: "Blue", PriceUSD: "4.50", SuccessURL: "https://shop.example/ok?id={{UUID}}", CancelURL: "https://shop.example/ko",
					CustomerReference: "crm-1", ExpiresInMinutes: 30, Metadata: map[string]string{"orderId": "order-1"}}, o...)
			},
			reply: fx("CreatedCheckoutSession"),
			check: func(t *testing.T, v any) {
				if s := is[*CreatedCheckoutSession](t, v); s.UUID != "pay@1" || s.Link == "" || s.ExpiresAt.IsZero() {
					t.Errorf("%+v", s)
				}
			}},
		{name: "CheckoutSessions.CreatePayment with fees", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"reference":"order-1044","productName":"T-shirt","description":"Blue, size M","priceUsd":"3.99",` +
				`"fees":{"processingFee":true,"items":[{"label":"Shipping","description":"Standard, 3 to 5 days","amountUsd":"0.75"},{"label":"VAT (20%)","amountUsd":"20"}]}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{Reference: "order-1044", ProductName: "T-shirt",
					Description: "Blue, size M", PriceUSD: "3.99", Fees: &CheckoutFees{ProcessingFee: &yes, Items: []FeeItem{
						{Label: "Shipping", Description: "Standard, 3 to 5 days", AmountUSD: "0.75"}, {Label: "VAT (20%)", AmountUSD: "20"},
					}}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreatePayment processing fee off", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","fees":{"processingFee":false,"items":[{"label":"Shipping","amountUsd":"19.90"}]}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductUUID: uuidA,
					Fees: &CheckoutFees{ProcessingFee: &no, Items: []FeeItem{{Label: "Shipping", AmountUSD: "19.90"}}}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreatePayment fees without processingFee", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"productReference":"pro","fees":{"items":[{"label":"Tax","amountUsd":"1000000"}]}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductReference: "pro",
					Fees: &CheckoutFees{Items: []FeeItem{{Label: "Tax", AmountUSD: "1000000"}}}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreatePayment processing fee only", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","fees":{"processingFee":true}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductUUID: uuidA, Fees: &CheckoutFees{ProcessingFee: &yes}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreatePayment empty fees", method: "POST", path: "/checkout-sessions/payment", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","fees":{}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductUUID: uuidA, Fees: &CheckoutFees{}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreateSubscription", method: "POST", path: "/checkout-sessions/subscription", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","customerUuid":"` + uuidB + `","frequency":{"value":1,"unit":"weeks"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":0}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductUUID: uuidA, CustomerUUID: uuidB,
					Frequency: &Duration{1, DurationUnitWeeks}, TrialPeriod: &Duration{14, DurationUnitDays}, MinPeriods: ptr[uint32](0)}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreateSubscription with fees", method: "POST", path: "/checkout-sessions/subscription", idempotent: true, status: 201,
			body: `{"productName":"Pro","priceUsd":"9.99","frequency":{"value":1,"unit":"months"},"metadata":{"plan":"pro"},` +
				`"fees":{"processingFee":true,"items":[{"label":"VAT (20%)","description":"French VAT","amountUsd":"2"}]}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductName: "Pro", PriceUSD: "9.99",
					Frequency: &Duration{1, DurationUnitMonths}, Metadata: map[string]string{"plan": "pro"}, Fees: &CheckoutFees{ProcessingFee: &yes, Items: []FeeItem{
						{Label: "VAT (20%)", Description: "French VAT", AmountUSD: "2"},
					}}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.CreateSubscription processing fee off", method: "POST", path: "/checkout-sessions/subscription", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","fees":{"processingFee":false}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductUUID: uuidA, Fees: &CheckoutFees{ProcessingFee: &no}}, o...)
			},
			reply: fx("CreatedCheckoutSession")},
		{name: "CheckoutSessions.Get", method: "GET", path: "/checkout-sessions/" + payID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.Get(ctx, payID, o...)
			},
			reply: fx("CheckoutSessionStatus"),
			check: func(t *testing.T, v any) {
				if s := is[*CheckoutSessionStatus](t, v); s.Status != CheckoutSessionStatusCreated || s.LastAttempt == nil || s.ExpiresAt == nil {
					t.Errorf("%+v", s)
				}
			}},
		{name: "CheckoutSessions.Expire", method: "POST", path: "/checkout-sessions/" + subID + "/expire",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.Expire(ctx, subID, o...)
			},
			reply: `{"uuid":"` + subID + `","status":"expired","expiresAt":"2026-10-01T12:00:00Z"}`,
			check: func(t *testing.T, v any) {
				if s := is[*CheckoutSessionStatus](t, v); s.Status != CheckoutSessionStatusExpired || s.ExpiresAt == nil {
					t.Errorf("%+v", s)
				}
			}},

		// Payments, transactions and failures.
		{name: "Payments.List", method: "GET", path: "/payments",
			query: "createdAfter=" + afterQ + "&createdBefore=" + beforeQ + "&cursor=" + cursorA + "&customerUuid=" + uuidB +
				"&limit=50&productUuid=" + uuidA + "&refunded=false&userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.List(ctx, &PaymentListParams{Limit: 50, Cursor: cursorA, CustomerUUID: uuidB, ProductUUID: uuidA,
					CreatedAfter: &after, CreatedBefore: &before, UserUUID: memberUUID, Refunded: &no}, o...)
			},
			reply: page(fx("Payment")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Payment]](t, v); len(p.Items) != 1 || p.Items[0].UUID != "pay@1" || p.Items[0].Currency.Code != "BASE:USDC" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Payments.Get", method: "GET", path: "/payments/" + payID, query: "includeMembers=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.Get(ctx, payID, &ReadParams{IncludeMembers: true}, o...)
			},
			reply: fx("Payment"),
			check: func(t *testing.T, v any) {
				if p := is[*Payment](t, v); p.Reference != "order-1" || p.Settlement.OrganizationFee == nil || p.Amount != "10" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Payments.Get nil params", method: "GET", path: "/payments/" + strings.TrimPrefix(payID, "pay@"),
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.Get(ctx, strings.TrimPrefix(payID, "pay@"), nil, o...)
			},
			reply: fx("Payment")},
		{name: "Payments.GetByReference", method: "GET", path: "/payments", query: "reference=" + refEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.GetByReference(ctx, ref, o...)
			},
			reply: bounded(withReference(t, fx("Payment"), ref))},
		{name: "Transactions.List", method: "GET", path: "/transactions",
			query: "includeMembers=true&refunded=true&subscriptionUuid=sub%40019eca82-5680-7b00-8000-0000000000d2&type=bill",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Transactions.List(ctx, &TransactionListParams{IncludeMembers: true, Refunded: &yes,
					Type: TransactionKindBill, SubscriptionUUID: subID}, o...)
			},
			reply: page(fx("Transaction")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Transaction]](t, v); p.Items[0].Type != TransactionKindBill || p.Items[0].PeriodEnd == nil {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Failures.List", method: "GET", path: "/payment-failures",
			query: "category=allowanceExhausted&kind=bill&limit=3&subscriptionUuid=sub%40019eca82-5680-7b00-8000-0000000000d2",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Failures.List(ctx, &FailureListParams{Limit: 3, Kind: FailureKindBill, Category: FailureCategoryAllowanceExhausted, SubscriptionUUID: subID}, o...)
			},
			reply: page(fx("Failure")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Failure]](t, v); p.Items[0].Kind != FailureKindBill || p.Items[0].Attempted != "9.99" {
					t.Errorf("%+v", p)
				}
			}},

		// Subscriptions.
		{name: "Subscriptions.List", method: "GET", path: "/subscriptions",
			query: "createdAfter=" + afterQ + "&customerUuid=" + uuidB + "&reference=crm%3A42&status=pastDue",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.List(ctx, &SubscriptionListParams{CustomerUUID: uuidB, CreatedAfter: &after,
					Status: SubscriptionStatusPastDue, Reference: "crm:42"}, o...)
			},
			reply: page(fx("Subscription")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Subscription]](t, v); p.Items[0].Status != SubscriptionStatusStopped || p.Items[0].Dunning == nil {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Subscriptions.Get", method: "GET", path: "/subscriptions/" + subID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Get(ctx, subID, nil, o...)
			},
			reply: fx("Subscription")},
		{name: "Subscriptions.GetByReference", method: "GET", path: "/subscriptions", query: "reference=" + refEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.GetByReference(ctx, ref, o...)
			},
			reply: bounded(withReference(t, fx("Subscription"), ref))},
		{name: "Subscriptions.Update", method: "PATCH", path: "/subscriptions/" + subID, body: `{"metadata":{"plan":"","seats":"5"}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Update(ctx, subID, &UpdateSubscriptionParams{Metadata: map[string]string{"plan": "", "seats": "5"}}, o...)
			},
			reply: fx("Subscription"),
			check: func(t *testing.T, v any) {
				if s := is[*Subscription](t, v); s.UUID != "sub@1" || s.Metadata["plan"] != "pro" {
					t.Errorf("%+v", s)
				}
			}},
		{name: "Subscriptions.Update empty", method: "PATCH", path: "/subscriptions/" + subID, body: `{}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Update(ctx, subID, &UpdateSubscriptionParams{}, o...)
			},
			reply: fx("Subscription")},
		{name: "Subscriptions.ListBills", method: "GET", path: "/subscriptions/" + subID + "/bills", query: "cursor=" + cursorA + "&limit=100",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.ListBills(ctx, subID, &BillListParams{Limit: 100, Cursor: cursorA}, o...)
			},
			reply: page(fx("Bill")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Bill]](t, v); p.Items[0].UUID != "sub-hist@1" || p.Items[0].SubscriptionUUID != "sub@1" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Subscriptions.GetBill", method: "GET", path: "/bills/" + billID, query: "includeMembers=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.GetBill(ctx, billID, &ReadParams{IncludeMembers: true}, o...)
			},
			reply: fx("Bill")},
		{name: "Subscriptions.Cancel 202", method: "POST", path: "/subscriptions/" + subID + "/cancel", query: "immediate=false",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Cancel(ctx, subID, &CancelSubscriptionParams{Immediate: &no}, o...)
			},
			status: 202, reply: fx("Subscription"),
			check: func(t *testing.T, v any) {
				if s := is[*SubscriptionCancellation](t, v); !s.Pending || s.Subscription.UUID != "sub@1" {
					t.Errorf("%+v", s)
				}
			}},
		{name: "Subscriptions.Cancel 200", method: "POST", path: "/subscriptions/" + subID + "/cancel",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Cancel(ctx, subID, nil, o...)
			},
			reply: `{"uuid":"sub@1","status":"cancelled","cancellationReason":"merchant","fees":[],"metadata":{}}`,
			check: func(t *testing.T, v any) {
				if s := is[*SubscriptionCancellation](t, v); s.Pending || s.Subscription.Status != SubscriptionStatusCancelled {
					t.Errorf("%+v", s)
				}
			}},
		{name: "Subscriptions.TestBill", method: "POST", path: "/subscriptions/" + subID + "/test-bill",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.TestBill(ctx, subID, o...)
			},
			reply: fx("TestBill"),
			check: func(t *testing.T, v any) {
				if s := is[*TestBill](t, v); s.Stage != BillingStageWaiting || s.Outcome != BillingOutcomePaid || !strings.HasPrefix(s.BillUUID, "sub-hist@") {
					t.Errorf("%+v", s)
				}
			}},

		// Refunds.
		{name: "Refunds.List", method: "GET", path: "/refunds",
			query: "cursor=" + cursorA + "&held=true&includeMembers=false&limit=5&status=pending&status=rejected&txUuid=pay%40019eca82-5680-7b00-8000-0000000000d1",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Refunds.List(ctx, &RefundListParams{Limit: 5, Cursor: cursorA, IncludeMembers: &no, Held: &yes,
					Status: []RefundStatus{RefundStatusPending, RefundStatusRejected}, TxUUID: payID}, o...)
			},
			reply: page(fx("Refund")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Refund]](t, v); len(p.Items) != 1 || p.Items[0].Approval == nil || p.Items[0].Status != RefundStatusApproved {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Refunds.List member", method: "GET", path: "/refunds", query: "userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Refunds.List(ctx, &RefundListParams{UserUUID: memberUUID}, o...)
			},
			reply: bounded()},
		{name: "Refunds.Get", method: "GET", path: "/refunds/refund@" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Refunds.Get(ctx, "refund@"+uuidA, o...)
			},
			reply: fx("Refund"),
			check: func(t *testing.T, v any) {
				if r := is[*Refund](t, v); r.UUID != "refund@1" || r.Amount != "5.0021" {
					t.Errorf("%+v", r)
				}
			}},

		// Members.
		{name: "Members.List", method: "GET", path: "/members", query: "cursor=" + cursorA + "&limit=20",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.List(ctx, &MemberListParams{Limit: 20, Cursor: cursorA}, o...)
			},
			reply: page(fx("Member")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Member]](t, v); p.Items[0].UserUUID != "m-1" || len(p.Items[0].AcceptedCurrencies) != 1 {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Members.Get", method: "GET", path: "/members/" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Get(ctx, memberUUID, o...)
			},
			reply: fx("Member")},
		{name: "Members.Update", method: "PATCH", path: "/members/" + memberUUID, body: `{"organizationFeePercent":"0"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Update(ctx, memberUUID, &UpdateMemberParams{OrganizationFeePercent: "0"}, o...)
			},
			reply: fx("Member")},
		{name: "Members.Remove", method: "DELETE", path: "/members/" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Members.Remove(ctx, memberUUID, o...)
			},
			status: 204},
		{name: "Members.Trust", method: "POST", path: "/members/" + memberUUID + "/trust",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Trust(ctx, memberUUID, o...)
			},
			reply: `{"userUuid":"` + memberUUID + `","trustedAt":"2026-10-07T10:00:00Z","organizationFeePercent":"5","acceptedCurrencies":[]}`,
			check: func(t *testing.T, v any) {
				if m := is[*Member](t, v); m.TrustedAt == nil || m.OrganizationFeePercent != "5" {
					t.Errorf("%+v", m)
				}
			}},
		{name: "Members.ListHeldFunds", method: "GET", path: "/members/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.ListHeldFunds(ctx, o...)
			},
			reply: bounded(fx("MemberHeldFunds")),
			check: func(t *testing.T, v any) {
				if h := is[[]MemberHeldFunds](t, v); len(h) != 1 || h[0].Count != 2 || h[0].TotalUSD != "98.5" {
					t.Errorf("%+v", h)
				}
			}},
		{name: "Members.GetHeldFunds", method: "GET", path: "/members/" + memberUUID + "/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.GetHeldFunds(ctx, memberUUID, o...)
			},
			reply: fx("HeldFunds"),
			check: func(t *testing.T, v any) {
				if h := is[*HeldFunds](t, v); len(h.Ledgers) != 2 || h.TotalUSD != "-1.0933" {
					t.Errorf("%+v", h)
				}
			}},
		{name: "Members.GetOwnHeldFunds", method: "GET", path: "/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.GetOwnHeldFunds(ctx, o...)
			},
			reply: `{"ledgers":[],"totalUsd":"0"}`,
			check: func(t *testing.T, v any) {
				if h := is[*HeldFunds](t, v); len(h.Ledgers) != 0 || h.TotalUSD != "0" {
					t.Errorf("%+v", h)
				}
			}},

		// Invitations.
		{name: "Invitations.Create", method: "POST", path: "/invitations", idempotent: true, status: 201,
			body: `{"email":"seller@example.com","role":"user","holdFunds":true,"organizationFeePercent":"2.5","redirectUrl":"https://shop.example/welcome","sendEmail":false}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.Create(ctx, &CreateInvitationParams{Email: "seller@example.com", HoldFunds: true,
					OrganizationFeePercent: "2.5", RedirectURL: "https://shop.example/welcome", SendEmail: new(false)}, o...)
			},
			reply: fx("CreatedInvitation"),
			check: func(t *testing.T, v any) {
				if i := is[*CreatedInvitation](t, v); i.Link == "" || i.Status != InvitationStatusPending || i.UUID != "i-1" {
					t.Errorf("%+v", i)
				}
			}},
		{name: "Invitations.Create minimal", method: "POST", path: "/invitations", idempotent: true, status: 201,
			body: `{"email":"seller@example.com","role":"user","holdFunds":false}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.Create(ctx, &CreateInvitationParams{Email: "seller@example.com"}, o...)
			},
			reply: fx("CreatedInvitation")},
		{name: "Invitations.List", method: "GET", path: "/invitations", query: "status=pending",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.List(ctx, &InvitationListParams{Status: InvitationStatusPending}, o...)
			},
			reply: page(fx("Invitation"))},
		{name: "Invitations.Revoke", method: "DELETE", path: "/invitations/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Invitations.Revoke(ctx, uuidA, o...)
			},
			status: 204},

		// Wallets.
		{name: "Wallets.List", method: "GET", path: "/wallets", query: "withBalances=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.List(ctx, &WalletListParams{WithBalances: true}, o...)
			},
			reply: bounded(fx("Wallet")),
			check: func(t *testing.T, v any) {
				if w := is[[]Wallet](t, v); len(w) != 1 || w[0].TokenWallets[0].Balance == nil || w[0].Address != "0xpk" {
					t.Errorf("%+v", w)
				}
			}},
		{name: "Wallets.ListForMember", method: "GET", path: "/members/" + memberUUID + "/wallets",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.ListForMember(ctx, memberUUID, o...)
			},
			reply: bounded(fx("Wallet"))},
		{name: "Wallets.ListAcceptedCurrencies", method: "GET", path: "/wallets/accepted-currencies", query: "userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.ListAcceptedCurrencies(ctx, &AcceptedCurrenciesParams{UserUUID: memberUUID}, o...)
			},
			reply: bounded(fx("Currency"), fx("Currency/native")),
			check: func(t *testing.T, v any) {
				if cs := is[[]Currency](t, v); len(cs) != 2 || cs[0].Code != "BASE:USDC" || cs[1].MainCurrencyID != nil {
					t.Errorf("%+v", cs)
				}
			}},

		// Accounting.
		{name: "Accounting.ExportJSON", method: "GET", path: "/accounting/export", query: "format=json&from=2026-09-01&to=2026-09-30",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Accounting.ExportJSON(ctx, "2026-09-01", "2026-09-30", o...)
			},
			reply: bounded(fx("AccountingEvent")),
			check: func(t *testing.T, v any) {
				if a := is[[]AccountingEvent](t, v); len(a) != 1 || a[0].Type != AccountingEventTypeRefund || a[0].GrossAmount != "-10.0042" {
					t.Errorf("%+v", a)
				}
			}},
		{name: "Accounting.ExportCSV", method: "GET", path: "/accounting/export", query: "format=csv&from=2026-09-01&to=2026-09-30",
			accept: "text/csv, application/json",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Accounting.ExportCSV(ctx, "2026-09-01", "2026-09-30", o...)
			},
			reply: "txTimeUtc,type,reference\n2026-09-13T19:23:26Z,payment,order-1\n", contentType: "text/csv",
			check: func(t *testing.T, v any) {
				if s := is[string](t, v); s != "txTimeUtc,type,reference\n2026-09-13T19:23:26Z,payment,order-1\n" {
					t.Errorf("%q", s)
				}
			}},

		// Webhooks.
		{name: "Webhooks.VerifyRemote", method: "POST", path: "/webhooks/verify",
			body: `{"endpointUuid":"` + uuidA + `","body":"{\"id\":\"evt_1\"}","signature":"t=1790856000,v1=abc"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Webhooks.VerifyRemote(ctx, uuidA, []byte(`{"id":"evt_1"}`), "t=1790856000,v1=abc", o...)
			},
			reply: `{"message":"Signature valid"}`},
		{name: "Webhooks.Endpoints.List", method: "GET", path: "/webhooks/endpoints", query: "cursor=" + cursorA + "&limit=10",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.List(ctx, &EndpointListParams{Limit: 10, Cursor: cursorA}, o...)
			},
			reply: page(fx("WebhookEndpointCreated")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[WebhookEndpoint]](t, v); len(p.Items) != 1 || p.Items[0].URL != "https://x.io/hook" || !p.HasMore {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Webhooks.Endpoints.List nil", method: "GET", path: "/webhooks/endpoints",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.List(ctx, nil, o...)
			},
			reply: bounded()},
		{name: "Webhooks.Endpoints.Create", method: "POST", path: "/webhooks/endpoints", idempotent: true, status: 201,
			body: `{"url":"https://shop.example/hooks","events":["payment.completed","member.joined"],"includeMembers":false,"description":"Shop"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.Create(ctx, &CreateWebhookEndpointParams{URL: "https://shop.example/hooks",
					Events: []EventType{EventTypePaymentCompleted, EventTypeMemberJoined}, IncludeMembers: &no, Description: "Shop"}, o...)
			},
			reply: fx("WebhookEndpointCreated"),
			check: func(t *testing.T, v any) {
				if e := is[*WebhookEndpointCreated](t, v); e.Secret != "whsec_x" || e.UUID != "e-1" {
					t.Errorf("%+v", e)
				}
			}},
		{name: "Webhooks.Endpoints.Get", method: "GET", path: "/webhooks/endpoints/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.Get(ctx, uuidA, o...)
			},
			reply: fx("WebhookEndpointCreated")},
		{name: "Webhooks.Endpoints.Update", method: "PATCH", path: "/webhooks/endpoints/" + uuidA,
			body: `{"events":[],"description":"","payloadVersion":"v2","enabled":true}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.Update(ctx, uuidA, &UpdateWebhookEndpointParams{Events: []EventType{}, Description: ptr(""),
					PayloadVersion: WebhookPayloadVersionV2, Enabled: &yes}, o...)
			},
			reply: fx("WebhookEndpointCreated")},
		{name: "Webhooks.Endpoints.Update without events", method: "PATCH", path: "/webhooks/endpoints/" + uuidA, body: `{"url":"https://shop.example/v2"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.Update(ctx, uuidA, &UpdateWebhookEndpointParams{URL: "https://shop.example/v2"}, o...)
			},
			reply: fx("WebhookEndpointCreated")},
		{name: "Webhooks.Endpoints.Delete", method: "DELETE", path: "/webhooks/endpoints/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Webhooks.Endpoints.Delete(ctx, uuidA, o...)
			},
			status: 204},
		{name: "Webhooks.Events.List", method: "GET", path: "/webhooks/events", query: "cursor=" + cursorA + "&includeMembers=true&limit=2&type=webhook.test",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Events.List(ctx, &EventListParams{Limit: 2, Cursor: cursorA, Type: EventTypeWebhookTest, IncludeMembers: true}, o...)
			},
			reply: `{"items":[` + evt + `],"hasMore":false}`,
			check: func(t *testing.T, v any) {
				p := is[*Page[Event]](t, v)
				if len(p.Items) != 1 || p.HasMore {
					t.Fatalf("%+v", p)
				}
				if w, err := p.Items[0].WebhookTest(); err != nil || w.EndpointUUID == "" {
					t.Errorf("%+v %v", w, err)
				}
			}},
		{name: "Webhooks.Events.List newer type", method: "GET", path: "/webhooks/events", query: "type=invoice.paid",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				// A type newer than the SDK is sent as is (the API judges it).
				return c.Webhooks.Events.List(ctx, &EventListParams{Type: "invoice.paid"}, o...)
			},
			reply: bounded()},
		{name: "Webhooks.Events.Get", method: "GET", path: "/webhooks/events/" + eventID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Events.Get(ctx, eventID, o...)
			},
			reply: strings.TrimSuffix(strings.TrimSpace(evt), "}") + `,"deliveries":[{"endpointUuid":"` + uuidA + `","url":"https://x.io","delivered":true,
				"attempts":[{"uuid":"` + uuidB + `","eventId":"` + eventID + `","eventType":"webhook.test","endpointUuid":"` + uuidA + `","attempt":1,
				"delivered":true,"statusCode":200,"durationMs":12,"attemptedAt":"2026-10-01T12:00:01Z"}]}]}`,
			check: func(t *testing.T, v any) {
				d := is[*EventDetail](t, v)
				if d.ID != eventID || d.Type != EventTypeWebhookTest || len(d.Deliveries) != 1 || len(d.Deliveries[0].Attempts) != 1 || !d.Deliveries[0].Delivered {
					t.Errorf("%+v", d)
				}
			}},

		// Currencies.
		{name: "Currencies.List", method: "GET", path: "/currencies", query: "main=true&test=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.List(ctx, &CurrencyListParams{Test: true, Main: true}, o...)
			},
			reply: bounded(fx("Currency/native")),
			check: func(t *testing.T, v any) {
				if cs := is[[]Currency](t, v); len(cs) != 1 || cs[0].Code != "ETH" {
					t.Errorf("%+v", cs)
				}
			}},
		{name: "Currencies.List nil", method: "GET", path: "/currencies",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.List(ctx, nil, o...)
			},
			reply: bounded(fx("Currency"))},
		{name: "Currencies.Get", method: "GET", path: "/currencies/8",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.Get(ctx, 8, o...)
			},
			reply: fx("Currency"),
			check: func(t *testing.T, v any) {
				if cur := is[*Currency](t, v); cur.ID != 8 || cur.Symbol != "USDC" || cur.Chain != ChainBASE {
					t.Errorf("%+v", cur)
				}
			}},

		// The client.
		{name: "Me", method: "GET", path: "/me",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Me(ctx, o...)
			},
			reply: fx("Me")},
	}
}

// serve answers every request with the case's reply.
func (rc routeCase) serve(w http.ResponseWriter, _ *http.Request, _ int) {
	status := rc.status
	if status == 0 {
		status = http.StatusOK
	}
	contentType := rc.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(rc.reply))
}

// TestServiceRequests checks, for every method, the request it sends (verb, escaped path,
// query, body, headers) and the decoding of its answer; then again with request options.
func TestServiceRequests(t *testing.T) {
	for _, rc := range routeCases(t) {
		t.Run(rc.name, func(t *testing.T) {
			ts := newTestServer(t, rc.serve)
			c, _ := newTestClient(t, ts)
			ctx := context.Background()

			v, err := rc.call(ctx, c)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if rc.check != nil {
				rc.check(t, v)
			}
			// Request options: On-Behalf-Of, X-Request-Id and a caller's Idempotency-Key.
			if _, err := rc.call(ctx, c, WithOnBehalfOf(memberUUID), WithRequestID("req-42"), WithIdempotencyKey("order-42")); err != nil {
				t.Fatalf("call with options: %v", err)
			}

			reqs := ts.recorded()
			if len(reqs) != 2 {
				t.Fatalf("%d requests, want 2", len(reqs))
			}
			for i, r := range reqs {
				if r.method != rc.method || r.path != rc.path || r.query != rc.query {
					t.Errorf("request %d: %s %s?%s, want %s %s?%s", i, r.method, r.path, r.query, rc.method, rc.path, rc.query)
				}
				checkBody(t, r, rc.body)
				accept := rc.accept
				if accept == "" {
					accept = "application/json"
				}
				if got := r.header.Get("Accept"); got != accept {
					t.Errorf("request %d: Accept %q, want %q", i, got, accept)
				}
				if r.header.Get("X-API-Key") != testAPIKey {
					t.Errorf("request %d: X-API-Key missing", i)
				}
			}

			plain, opted := reqs[0], reqs[1]
			if _, set := plain.header["On-Behalf-Of"]; set {
				t.Error("On-Behalf-Of sent without the option")
			}
			if got := opted.header.Get("On-Behalf-Of"); got != memberUUID {
				t.Errorf("On-Behalf-Of = %q, want %q", got, memberUUID)
			}
			if got := opted.header.Get("X-Request-Id"); got != "req-42" {
				t.Errorf("X-Request-Id = %q", got)
			}
			key, opKey := plain.header.Get("Idempotency-Key"), opted.header.Get("Idempotency-Key")
			if rc.idempotent {
				if !isUUID(key) || opKey != "order-42" {
					t.Errorf("Idempotency-Key %q / %q, want a UUID / order-42", key, opKey)
				}
			} else if _, set := plain.header["Idempotency-Key"]; set || opKey != "" {
				t.Errorf("Idempotency-Key sent on a non-idempotent method: %q / %q", key, opKey)
			}
		})
	}
}

// checkBody compares a request's JSON body with the expected one (key by key, order aside).
func checkBody(t *testing.T, r recorded, want string) {
	t.Helper()
	if want == "" {
		if r.body != "" || r.header.Get("Content-Type") != "" {
			t.Errorf("unexpected body %q (Content-Type %q)", r.body, r.header.Get("Content-Type"))
		}
		return
	}
	if r.header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type %q", r.header.Get("Content-Type"))
	}
	var got, exp any
	if err := json.Unmarshal([]byte(r.body), &got); err != nil {
		t.Fatalf("body %q: %v", r.body, err)
	}
	if err := json.Unmarshal([]byte(want), &exp); err != nil {
		t.Fatalf("want %q: %v", want, err)
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("body\n got %s\nwant %s", r.body, want)
	}
}

// TestServiceRetries checks the retry policy per method: a 503 is retried by the reads and the 7
// idempotent creates (with the same Idempotency-Key), never by the other writes.
func TestServiceRetries(t *testing.T) {
	idempotent := map[string]bool{} // the methods; a case named "<Method> <variant>" is the same method
	for _, rc := range routeCases(t) {
		t.Run(rc.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, n int) {
				if n == 0 {
					jsonReply(w, 503, `{"error":"unavailable","code":"network_unavailable"}`)
					return
				}
				rc.serve(w, r, n)
			})
			c, _ := newTestClient(t, ts)
			_, err := rc.call(context.Background(), c)
			reqs := ts.recorded()

			retried := rc.method == http.MethodGet || rc.idempotent
			if !retried {
				var se *ServerError
				if !errors.As(err, &se) || se.StatusCode != 503 || len(reqs) != 1 {
					t.Fatalf("%s must not be retried: %d requests, err %v", rc.name, len(reqs), err)
				}
				return
			}
			if err != nil || len(reqs) != 2 {
				t.Fatalf("want a retried success: %d requests, err %v", len(reqs), err)
			}
			if rc.idempotent {
				k0, k1 := reqs[0].header.Get("Idempotency-Key"), reqs[1].header.Get("Idempotency-Key")
				if k0 == "" || k0 != k1 {
					t.Errorf("Idempotency-Key %q then %q: must be stable across retries", k0, k1)
				}
				if reqs[0].body != reqs[1].body {
					t.Error("the retry sent another body")
				}
			}
		})
		if rc.idempotent {
			idempotent[strings.Fields(rc.name)[0]] = true
		}
	}
	if len(idempotent) != 6 {
		t.Errorf("%d idempotent methods, want 6: %v", len(idempotent), slices.Sorted(maps.Keys(idempotent)))
	}
}

// TestServiceErrorsTyped checks an error answer reaches the caller typed, CSV export included.
func TestServiceErrorsTyped(t *testing.T) {
	ts := newTestServer(t, staticReply(409, `{"error":"Unknown checkout","code":"tx_already_sent","requestId":"r-1"}`))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	_, err := c.CheckoutSessions.Expire(ctx, payID)
	var ce *ConflictError
	if !errors.As(err, &ce) || ce.Code != "tx_already_sent" || ce.RequestID != "r-1" {
		t.Errorf("Expire: %v", err)
	}

	ts2 := newTestServer(t, staticReply(400, `{"error":"validation failed","code":"validation_failed",`+
		`"details":{"max":92,"errors":[{"field":"to","message":"to must be at most 92 days after from, both included"}]}}`))
	c2, _ := newTestClient(t, ts2)
	csv, err := c2.Accounting.ExportCSV(ctx, "2026-01-01", "2026-09-30")
	var ve *ValidationError
	if !errors.As(err, &ve) || csv != "" || ve.StatusCode != 400 || ve.Details["max"] != float64(92) || ve.FieldErrors[0].Field != "to" {
		t.Errorf("ExportCSV: %q %v", csv, err)
	}

	ts4 := newTestServer(t, staticReply(400, `{"error":"test mode only","code":"test_mode_only"}`))
	c4, _ := newTestClient(t, ts4)
	if _, err := c4.Subscriptions.TestBill(ctx, subID); !errors.As(err, new(*BadRequestError)) {
		t.Errorf("TestBill on a live subscription: %v", err)
	}

	ts3 := newTestServer(t, staticReply(404, `{"error":"not found","code":"not_found"}`))
	c3, _ := newTestClient(t, ts3)
	if _, err := c3.Subscriptions.Cancel(ctx, subID, nil); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("Cancel: %v", err)
	}
	if err := c3.Products.Delete(ctx, uuidA); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("Delete: %v", err)
	}
	if err := c3.Invitations.Revoke(ctx, uuidA); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("Revoke: %v", err)
	}
	n := 0
	for _, err := range c3.Customers.Iterate(ctx, nil) {
		n++
		if !errors.As(err, new(*NotFoundError)) {
			t.Errorf("Iterate: %v", err)
		}
	}
	if n != 1 {
		t.Errorf("Iterate yielded %d values, want the error only", n)
	}
}

// TestGetByReferenceNotFound checks the getByReference helpers: an empty page is a
// *NotFoundError like a 404 (status 404, code not_found) carrying the list's request id; the
// reference is checked before sending; a list error comes through as is.
func TestGetByReferenceNotFound(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		w.Header().Set("X-Request-Id", "req-list")
		jsonReply(w, 200, bounded())
	})
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	calls := map[string]func() error{
		"Products":      func() error { _, err := c.Products.GetByReference(ctx, "pro"); return err },
		"Customers":     func() error { _, err := c.Customers.GetByReference(ctx, "crm-1"); return err },
		"Payments":      func() error { _, err := c.Payments.GetByReference(ctx, "order-1"); return err },
		"Subscriptions": func() error { _, err := c.Subscriptions.GetByReference(ctx, "customer-7731"); return err },
	}
	for name, call := range calls {
		err := call()
		var nf *NotFoundError
		if !errors.As(err, &nf) || nf.StatusCode != 404 || nf.Code != "not_found" || nf.RequestID != "req-list" ||
			!strings.Contains(nf.Message, "reference") || IsRetryable(err) {
			t.Errorf("%s: %#v", name, err)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Details == nil {
			t.Errorf("%s: not an *APIError with details", name)
		}
	}
	if n := len(ts.recorded()); n != len(calls) {
		t.Errorf("%d requests, want one per call", n)
	}

	before := len(ts.recorded())
	for _, bad := range []string{"", "  ", "a b", "a/b", strings.Repeat("r", 101)} {
		if _, err := c.Products.GetByReference(ctx, bad); !errors.As(err, new(*ValidationError)) {
			t.Errorf("reference %q: %v", bad, err)
		}
	}
	if n := len(ts.recorded()) - before; n != 0 {
		t.Errorf("%d requests sent for invalid references", n)
	}

	unauthorized := newTestServer(t, staticReply(401, `{"error":"unauthorized","code":"unauthorized"}`))
	c2, _ := newTestClient(t, unauthorized)
	if _, err := c2.Customers.GetByReference(ctx, "crm-1"); !errors.As(err, new(*AuthenticationError)) {
		t.Errorf("list error: %v", err)
	}
	notAPage := newTestServer(t, staticReply(200, `[{"uuid":"p-1"}]`))
	c3, _ := newTestClient(t, notAPage)
	if _, err := c3.Products.GetByReference(ctx, "pro"); !errors.As(err, new(*ServerError)) {
		t.Errorf("a bare list: %v", err)
	}
}

// TestDeletesAnswer204 checks every DELETE treats 204 (no body) as success, and never reads a body.
func TestDeletesAnswer204(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) { w.WriteHeader(http.StatusNoContent) })
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"Products.Delete":           func() error { return c.Products.Delete(ctx, uuidA) },
		"Customers.Delete":          func() error { return c.Customers.Delete(ctx, uuidA) },
		"Members.Remove":            func() error { return c.Members.Remove(ctx, memberUUID) },
		"Invitations.Revoke":        func() error { return c.Invitations.Revoke(ctx, uuidA) },
		"Webhooks.Endpoints.Delete": func() error { return c.Webhooks.Endpoints.Delete(ctx, uuidA) },
	} {
		if err := call(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, r := range ts.recorded() {
		if r.method != http.MethodDelete || r.body != "" {
			t.Errorf("%s %s with body %q", r.method, r.path, r.body)
		}
	}
}

// TestCancelNoBody checks a 2xx without the subscription is a ServerError, not an empty one.
func TestCancelNoBody(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) { w.WriteHeader(202) })
	c, _ := newTestClient(t, ts)
	if _, err := c.Subscriptions.Cancel(context.Background(), subID, nil); !errors.As(err, new(*ServerError)) {
		t.Errorf("want *ServerError, got %v", err)
	}
}

// TestGetByReferenceChecksTheReference: a server that ignores the ?reference= filter answers an
// ordinary page; the helpers return the item with exactly that reference, or a *NotFoundError —
// never another resource.
func TestGetByReferenceChecksTheReference(t *testing.T) {
	ts := newTestServer(t, staticReply(200, bounded(
		`{"uuid":"pay@01a0f22f-9600-7000-8000-000000000001","reference":"order-9"}`,
		`{"uuid":"pay@01a0f22f-9600-7000-8000-000000000002","reference":"order-1042"}`,
	)))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	p, err := c.Payments.GetByReference(ctx, "order-1042")
	if err != nil || p.UUID != "pay@01a0f22f-9600-7000-8000-000000000002" {
		t.Errorf("matching item: %+v %v", p, err)
	}
	if _, err := c.Payments.GetByReference(ctx, "order-7"); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("no item with the reference: want *NotFoundError, got %v", err)
	}
	if _, err := c.Payments.GetByReference(ctx, "ORDER-1042"); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("the reference is compared exactly: %v", err)
	}
}

// TestBoundedListFollowsHasMore: a bounded list answering hasMore true (never expected) is
// followed with its nextCursor, the filters kept, so it is never cut short silently; the walk
// stops on the iterators' guards, and the result is never nil.
func TestBoundedListFollowsHasMore(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, n int) {
		switch n {
		case 0:
			jsonReply(w, 200, `{"items":[{"id":1}],"hasMore":true,"nextCursor":"c1"}`)
		case 1:
			jsonReply(w, 200, `{"items":[{"id":2}],"hasMore":false}`)
		default:
			t.Fatalf("request %d past hasMore false", n)
		}
	})
	c, _ := newTestClient(t, ts)
	cs, err := c.Currencies.List(context.Background(), &CurrencyListParams{Main: true})
	if err != nil || len(cs) != 2 || cs[0].ID != 1 || cs[1].ID != 2 {
		t.Fatalf("followed list: %+v %v", cs, err)
	}
	reqs := ts.recorded()
	if len(reqs) != 2 || strings.Contains(reqs[0].query, "cursor") ||
		!strings.Contains(reqs[1].query, "cursor=c1") || !strings.Contains(reqs[1].query, "main=true") {
		t.Errorf("requests: %+v", reqs)
	}

	for name, body := range map[string]string{
		"empty page with more": `{"items":[],"hasMore":true,"nextCursor":"x"}`,
		"more without cursor":  `{"items":[{"id":1}],"hasMore":true}`,
		"null items":           `{"items":null,"hasMore":false}`,
	} {
		one := newTestServer(t, staticReply(200, body))
		c1, _ := newTestClient(t, one)
		got, err := c1.Currencies.List(context.Background(), nil)
		if err != nil || got == nil || len(one.recorded()) != 1 {
			t.Errorf("%s: %#v %v after %d requests", name, got, err, len(one.recorded()))
		}
	}
}
