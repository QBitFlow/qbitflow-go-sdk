package qbitflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Ids used by the service tests.
const (
	uuidA    = "019eca82-5680-7b00-8000-0000000000c1"
	uuidB    = "019eca82-5680-7b00-8000-0000000000c2"
	payID    = "pay@019eca82-5680-7b00-8000-0000000000d1"
	subID    = "sub@019eca82-5680-7b00-8000-0000000000d2"
	billID   = "sub-hist@019eca82-5680-7b00-8000-0000000000d3"
	eventID  = "evt_bd1a913d-188e-5ac5-a2d6-5fd25cd67301"
	oddRef   = "a/b c@d"     // a reference that needs escaping
	oddRefEs = "a%2Fb%20c@d" // … as sent
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

func page(item string) string { return `{"items":[` + item + `],"nextCursor":"` + uuidB + `"}` }

func list(item string) string { return `[` + item + `]` }

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

// routeCases covers every service method.
func routeCases(t *testing.T) []routeCase {
	evt := eventFixture(t)
	no := false
	yes := true
	return []routeCase{
		// Products.
		{name: "Products.List", method: "GET", path: "/product", query: "includeHidden=true&subscription=false",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.List(ctx, &ProductListParams{IncludeHidden: true, Subscription: &no}, o...)
			},
			reply: list(fx("Product")),
			check: func(t *testing.T, v any) {
				if p := is[[]Product](t, v); len(p) != 1 || p[0].UUID != "p-1" || p[0].Subscription == nil {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Products.List nil", method: "GET", path: "/product",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.List(ctx, nil, o...)
			},
			reply: `[]`},
		{name: "Products.Create", method: "POST", path: "/product", idempotent: true, status: 201,
			body: `{"name":"Pro","price":9.99,"reference":"pro","subscription":{"frequency":{"value":1,"unit":"months"},"trialPeriod":{"value":0},"minPeriods":3}}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Create(ctx, &CreateProductParams{Name: "Pro", Price: 9.99, Reference: "pro",
					Subscription: &SubscriptionTermsParams{Frequency: &Duration{1, DurationUnitMonths}, TrialPeriod: &Duration{}, MinPeriods: ptr[uint32](3)}}, o...)
			},
			reply: fx("Product"),
			check: func(t *testing.T, v any) {
				if p := is[*Product](t, v); p.UUID != "p-1" || p.Price != 9.99 {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Products.Get", method: "GET", path: "/product/uuid/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Get(ctx, uuidA, o...)
			},
			reply: fx("Product")},
		{name: "Products.GetByReference", method: "GET", path: "/product/reference/" + oddRefEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.GetByReference(ctx, oddRef, o...)
			},
			reply: fx("Product")},
		{name: "Products.Update", method: "PUT", path: "/product/" + uuidA,
			body: `{"description":"","price":12,"isActive":false}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Products.Update(ctx, uuidA, &UpdateProductParams{Description: ptr(""), Price: ptr(12.0), IsActive: &no}, o...)
			},
			reply: fx("Product")},
		{name: "Products.Delete", method: "DELETE", path: "/product/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Products.Delete(ctx, uuidA, o...)
			},
			reply: `{"message":"Product deleted successfully"}`},

		// Customers.
		{name: "Customers.Create", method: "POST", path: "/customer", idempotent: true, status: 201,
			body: `{"name":"Ada","lastName":"Lovelace","email":"ada@example.com","phoneNumber":"+33 6 12 34 56 78","reference":"crm-1"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Create(ctx, &CreateCustomerParams{Name: "Ada", LastName: "Lovelace", Email: "ada@example.com",
					PhoneNumber: "+33 6 12 34 56 78", Reference: "crm-1"}, o...)
			},
			reply: fx("Customer"),
			check: func(t *testing.T, v any) {
				if c := is[*Customer](t, v); c.UUID != "c-1" || !c.Verified || c.UserUUID != "m-1" {
					t.Errorf("%+v", c)
				}
			}},
		{name: "Customers.Update", method: "PUT", path: "/customer/" + uuidA,
			body: `{"email":"new@example.com","phoneNumber":""}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Update(ctx, uuidA, &UpdateCustomerParams{Email: "new@example.com", PhoneNumber: ptr("")}, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.List", method: "GET", path: "/customer/all",
			query: "cursor=" + uuidA + "&email=ada%2B1%40example.com&limit=5&verified=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.List(ctx, &CustomerListParams{Limit: 5, Cursor: uuidA, Email: "ada+1@example.com", Verified: &yes}, o...)
			},
			reply: page(fx("Customer")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Customer]](t, v); len(p.Items) != 1 || !p.HasMore() || *p.NextCursor != uuidB {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Customers.Get", method: "GET", path: "/customer/uuid/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.Get(ctx, uuidA, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.GetByEmail", method: "GET", path: "/customer/email/ada+1@example.com",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.GetByEmail(ctx, "ada+1@example.com", o...)
			},
			reply: fx("Customer")},
		{name: "Customers.GetByReference", method: "GET", path: "/customer/reference/" + oddRefEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Customers.GetByReference(ctx, oddRef, o...)
			},
			reply: fx("Customer")},
		{name: "Customers.Delete", method: "DELETE", path: "/customer/uuid/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Customers.Delete(ctx, uuidA, o...)
			},
			reply: `{"message":"Customer deleted successfully"}`},

		// Checkout sessions.
		{name: "CheckoutSessions.CreatePayment", method: "POST", path: "/transaction/session-checkout/new/payment", idempotent: true, status: 201,
			body: `{"reference":"order-1","productName":"T-shirt","description":"Blue","price":4.5,"successUrl":"https://shop.example/ok?id={{UUID}}","cancelUrl":"https://shop.example/ko","customerReference":"crm-1","expiresInMinutes":30}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{Reference: "order-1", ProductName: "T-shirt",
					Description: "Blue", Price: 4.5, SuccessURL: "https://shop.example/ok?id={{UUID}}", CancelURL: "https://shop.example/ko",
					CustomerReference: "crm-1", ExpiresInMinutes: 30}, o...)
			},
			reply: fx("CheckoutSession"),
			check: func(t *testing.T, v any) {
				if s := is[*CheckoutSession](t, v); s.UUID != "pay@1" || s.Link == "" || s.ExpiresAt.IsZero() {
					t.Errorf("%+v", s)
				}
			}},
		{name: "CheckoutSessions.CreateSubscription", method: "POST", path: "/transaction/session-checkout/new/subscription", idempotent: true, status: 201,
			body: `{"productUuid":"` + uuidA + `","customerUuid":"` + uuidB + `","frequency":{"value":1,"unit":"weeks"},"trialPeriod":{"value":14,"unit":"days"},"minPeriods":0}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductUUID: uuidA, CustomerUUID: uuidB,
					Frequency: &Duration{1, DurationUnitWeeks}, TrialPeriod: &Duration{14, DurationUnitDays}, MinPeriods: ptr[uint32](0)}, o...)
			},
			reply: fx("CheckoutSession")},
		{name: "CheckoutSessions.GetStatus", method: "GET", path: "/transaction/session-checkout/" + payID + "/status",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.GetStatus(ctx, payID, o...)
			},
			reply: fx("CheckoutSessionStatus"),
			check: func(t *testing.T, v any) {
				if s := is[*CheckoutSessionStatus](t, v); s.Status != CheckoutSessionStatusCreated || s.LastAttempt == nil {
					t.Errorf("%+v", s)
				}
			}},
		{name: "CheckoutSessions.Expire", method: "POST", path: "/transaction/session-checkout/" + subID + "/expire",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.CheckoutSessions.Expire(ctx, subID, o...)
			},
			reply: `{"uuid":"` + subID + `","status":"expired"}`,
			check: func(t *testing.T, v any) {
				if s := is[*CheckoutSessionStatus](t, v); s.Status != CheckoutSessionStatusExpired {
					t.Errorf("%+v", s)
				}
			}},

		// Payments and failures.
		{name: "Payments.List", method: "GET", path: "/transaction/payments",
			query: "createdAfter=" + afterQ + "&createdBefore=" + beforeQ + "&cursor=" + uuidA + "&customerUuid=" + uuidB +
				"&limit=50&productUuid=" + uuidA + "&refunded=false&userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.List(ctx, &PaymentListParams{Limit: 50, Cursor: uuidA, CustomerUUID: uuidB, ProductUUID: uuidA,
					CreatedAfter: &after, CreatedBefore: &before, UserUUID: memberUUID, Refunded: &no}, o...)
			},
			reply: page(fx("Payment")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Payment]](t, v); len(p.Items) != 1 || p.Items[0].UUID != "pay@1" || p.Items[0].Currency == nil {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Payments.ListCombined", method: "GET", path: "/transaction/payments/combined",
			query: "includeMembers=true&refunded=true&source=subscriptionHistory&subscriptionUuid=sub%40019eca82-5680-7b00-8000-0000000000d2",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.ListCombined(ctx, &CombinedPaymentListParams{IncludeMembers: true, Refunded: &yes,
					Source: CombinedPaymentSourceSubscriptionHistory, SubscriptionUUID: subID}, o...)
			},
			reply: page(fx("CombinedPayment")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[CombinedPayment]](t, v); p.Items[0].Source != CombinedPaymentSourceSubscriptionHistory {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Payments.Get", method: "GET", path: "/transaction/payment/" + payID, query: "includeMembers=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.Get(ctx, payID, &ReadParams{IncludeMembers: true}, o...)
			},
			reply: fx("Payment"),
			check: func(t *testing.T, v any) {
				if p := is[*Payment](t, v); p.Reference != "order-1" || p.Metadata.OrganizationFee == nil {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Payments.Get nil params", method: "GET", path: "/transaction/payment/" + strings.TrimPrefix(payID, "pay@"),
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.Get(ctx, strings.TrimPrefix(payID, "pay@"), nil, o...)
			},
			reply: fx("Payment")},
		{name: "Payments.GetByReference", method: "GET", path: "/transaction/payment/reference/" + oddRefEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Payments.GetByReference(ctx, oddRef, o...)
			},
			reply: fx("Payment")},
		{name: "Failures.List", method: "GET", path: "/transaction/failures",
			query: "category=allowanceExhausted&kind=bill&limit=3&subscriptionUuid=sub%40019eca82-5680-7b00-8000-0000000000d2",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Failures.List(ctx, &FailureListParams{Limit: 3, Kind: FailureKindBill, Category: FailureCategoryAllowanceExhausted, SubscriptionUUID: subID}, o...)
			},
			reply: page(fx("Failure")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Failure]](t, v); p.Items[0].Kind != FailureKindBill {
					t.Errorf("%+v", p)
				}
			}},

		// Subscriptions.
		{name: "Subscriptions.List", method: "GET", path: "/transaction/subscriptions",
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
		{name: "Subscriptions.Get", method: "GET", path: "/transaction/subscription/" + subID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Get(ctx, subID, nil, o...)
			},
			reply: fx("Subscription")},
		{name: "Subscriptions.GetByReference", method: "GET", path: "/transaction/subscription/reference/subscription/" + oddRefEs,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.GetByReference(ctx, oddRef, o...)
			},
			reply: fx("Subscription")},
		{name: "Subscriptions.ListBills", method: "GET", path: "/transaction/subscription/" + subID + "/bills", query: "cursor=" + uuidA + "&limit=100",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.ListBills(ctx, subID, &BillListParams{Limit: 100, Cursor: uuidA}, o...)
			},
			reply: page(fx("Bill")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Bill]](t, v); p.Items[0].UUID != "sub-hist@1" || p.Items[0].SubscriptionUUID != "sub@1" {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Subscriptions.GetBill", method: "GET", path: "/transaction/subscription/bill/" + billID, query: "includeMembers=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.GetBill(ctx, billID, &ReadParams{IncludeMembers: true}, o...)
			},
			reply: fx("Bill")},
		{name: "Subscriptions.GetPublicHistory", method: "GET", path: "/transaction/subscription/history/" + subID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.GetPublicHistory(ctx, subID, o...)
			},
			reply: `[{"uuid":"sub-hist@1","createdAt":"2026-10-01T12:00:00+02:00","amount":9.99,"periodStart":"2026-10-01T12:00:00Z"}]`,
			check: func(t *testing.T, v any) {
				if b := is[[]Bill](t, v); len(b) != 1 || b[0].Amount != 9.99 || b[0].CustomerUUID != "" || b[0].Metadata.FeePercent != 0 {
					t.Errorf("%+v", b)
				}
			}},
		{name: "Subscriptions.Cancel 202", method: "POST", path: "/transaction/subscription/processing/force-cancel/" + subID, query: "immediate=false",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Cancel(ctx, subID, &CancelSubscriptionParams{Immediate: &no}, o...)
			},
			status: 202, reply: fx("Subscription"),
			check: func(t *testing.T, v any) {
				if s := is[*SubscriptionCancellation](t, v); !s.Pending || s.Subscription.UUID != "sub@1" {
					t.Errorf("%+v", s)
				}
			}},
		{name: "Subscriptions.Cancel 200", method: "POST", path: "/transaction/subscription/processing/force-cancel/" + subID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.Cancel(ctx, subID, nil, o...)
			},
			reply: `{"uuid":"sub@1","status":"cancelled","cancellationReason":"merchant"}`,
			check: func(t *testing.T, v any) {
				if s := is[*SubscriptionCancellation](t, v); s.Pending || s.Subscription.Status != SubscriptionStatusCancelled {
					t.Errorf("%+v", s)
				}
			}},
		{name: "Subscriptions.ExecuteTestBilling", method: "POST", path: "/transaction/subscription/processing/execute-billing/" + subID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Subscriptions.ExecuteTestBilling(ctx, subID, o...)
			},
			reply: fx("BillingState"),
			check: func(t *testing.T, v any) {
				if s := is[*BillingState](t, v); s.Stage != BillingStageDone || s.Outcome != BillingOutcomePaid {
					t.Errorf("%+v", s)
				}
			}},

		// Refunds.
		{name: "Refunds.List", method: "GET", path: "/transaction/refunds/all", query: "held=true&includeMembers=false",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				// Limit and Cursor are ListInactive's: never sent here.
				return c.Refunds.List(ctx, &RefundListParams{Limit: 5, Cursor: uuidA, IncludeMembers: &no, Held: &yes}, o...)
			},
			reply: list(fx("Refund")),
			check: func(t *testing.T, v any) {
				if r := is[[]Refund](t, v); len(r) != 1 || r[0].Approval == nil || r[0].Status != RefundStatusPending {
					t.Errorf("%+v", r)
				}
			}},
		{name: "Refunds.ListInactive", method: "GET", path: "/transaction/refunds/all/inactive", query: "cursor=" + uuidA + "&limit=5&userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Refunds.ListInactive(ctx, &RefundListParams{Limit: 5, Cursor: uuidA, UserUUID: memberUUID}, o...)
			},
			reply: page(fx("Refund"))},
		{name: "Refunds.Initiate", method: "POST", path: "/transaction/refunds/initiate", idempotent: true, status: 201,
			body: `{"txUuid":"` + billID + `","refundPercent":50.5,"reason":"Damaged","merchantMessage":"Sorry"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Refunds.Initiate(ctx, &InitiateRefundParams{TxUUID: billID, RefundPercent: ptr(50.5), Reason: "Damaged", MerchantMessage: "Sorry"}, o...)
			},
			reply: fx("Refund"),
			check: func(t *testing.T, v any) {
				if r := is[*Refund](t, v); r.UUID != "refund@1" || r.TxUUID != "pay@1" {
					t.Errorf("%+v", r)
				}
			}},

		// Members.
		{name: "Members.List", method: "GET", path: "/members", query: "cursor=" + memberUUID + "&limit=20",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.List(ctx, &MemberListParams{Limit: 20, Cursor: memberUUID}, o...)
			},
			reply: page(fx("Member")),
			check: func(t *testing.T, v any) {
				if p := is[*Page[Member]](t, v); p.Items[0].UserUUID != "m-1" || len(p.Items[0].AcceptedCurrencyIDs) != 3 {
					t.Errorf("%+v", p)
				}
			}},
		{name: "Members.Get", method: "GET", path: "/members/" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Get(ctx, memberUUID, o...)
			},
			reply: fx("Member")},
		{name: "Members.Update", method: "PUT", path: "/members/" + memberUUID, body: `{"organizationFeePercent":0}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Update(ctx, memberUUID, &UpdateMemberParams{OrganizationFeePercent: 0}, o...)
			},
			reply: fx("Member")},
		{name: "Members.Remove", method: "DELETE", path: "/members/" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Members.Remove(ctx, memberUUID, o...)
			},
			reply: `{"message":"Member removed"}`},
		{name: "Members.Trust", method: "POST", path: "/members/" + memberUUID + "/trust",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.Trust(ctx, memberUUID, o...)
			},
			reply: `{"userUuid":"` + memberUUID + `","trustedAt":"2026-10-07T10:00:00+02:00"}`,
			check: func(t *testing.T, v any) {
				if m := is[*Member](t, v); m.TrustedAt == nil {
					t.Errorf("%+v", m)
				}
			}},
		{name: "Members.ListHeldFunds", method: "GET", path: "/members/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.ListHeldFunds(ctx, o...)
			},
			reply: list(fx("MemberHeldFundsSummary")),
			check: func(t *testing.T, v any) {
				if h := is[[]MemberHeldFundsSummary](t, v); len(h) != 1 || h[0].Count != 2 {
					t.Errorf("%+v", h)
				}
			}},
		{name: "Members.GetHeldFunds", method: "GET", path: "/members/" + memberUUID + "/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.GetHeldFunds(ctx, memberUUID, o...)
			},
			reply: fx("HeldFunds"),
			check: func(t *testing.T, v any) {
				if h := is[*HeldFunds](t, v); len(h.Ledgers) != 1 || h.TotalAmount != -10.0042 {
					t.Errorf("%+v", h)
				}
			}},
		{name: "Members.GetOwnHeldFunds", method: "GET", path: "/user/held-funds",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Members.GetOwnHeldFunds(ctx, o...)
			},
			reply: `{"ledgers":[],"totalAmount":0}`},

		// Invitations.
		{name: "Invitations.Create", method: "POST", path: "/invitations", idempotent: true, status: 201,
			body: `{"email":"seller@example.com","role":"user","trustLayer":true,"organizationFeePercent":2.5,"redirectUrl":"https://shop.example/welcome"}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.Create(ctx, &CreateInvitationParams{Email: "seller@example.com", TrustLayer: true,
					OrganizationFeePercent: ptr(2.5), RedirectURL: "https://shop.example/welcome"}, o...)
			},
			reply: fx("InvitationCreated"),
			check: func(t *testing.T, v any) {
				if i := is[*InvitationCreated](t, v); i.Link == "" || i.Invitation.Status != InvitationStatusPending {
					t.Errorf("%+v", i)
				}
			}},
		{name: "Invitations.Create minimal", method: "POST", path: "/invitations", idempotent: true, status: 201,
			body: `{"email":"seller@example.com","role":"user","trustLayer":false}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.Create(ctx, &CreateInvitationParams{Email: "seller@example.com"}, o...)
			},
			reply: fx("InvitationCreated")},
		{name: "Invitations.List", method: "GET", path: "/invitations", query: "status=pending",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.List(ctx, &InvitationListParams{Status: InvitationStatusPending}, o...)
			},
			reply: page(fx("Invitation"))},
		{name: "Invitations.Revoke", method: "DELETE", path: "/invitations/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Invitations.Revoke(ctx, uuidA, o...)
			},
			reply: fx("Invitation"),
			check: func(t *testing.T, v any) {
				if i := is[*Invitation](t, v); i.UUID != "i-1" {
					t.Errorf("%+v", i)
				}
			}},

		// Wallets.
		{name: "Wallets.List", method: "GET", path: "/wallet/user", query: "withBalances=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.List(ctx, &WalletListParams{WithBalances: true}, o...)
			},
			reply: list(fx("Wallet")),
			check: func(t *testing.T, v any) {
				if w := is[[]Wallet](t, v); len(w) != 1 || w[0].TokenWallets[0].Balance == nil {
					t.Errorf("%+v", w)
				}
			}},
		{name: "Wallets.ListForMember", method: "GET", path: "/wallet/user/" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.ListForMember(ctx, memberUUID, o...)
			},
			reply: list(fx("Wallet"))},
		{name: "Wallets.ListSupportedCurrencies", method: "GET", path: "/wallet/supported-currencies", query: "userUuid=" + memberUUID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Wallets.ListSupportedCurrencies(ctx, &SupportedCurrenciesParams{UserUUID: memberUUID}, o...)
			},
			reply: list(fx("Currency")),
			check: func(t *testing.T, v any) {
				if cs := is[[]Currency](t, v); len(cs) != 1 || cs[0].MainCurrency == nil {
					t.Errorf("%+v", cs)
				}
			}},

		// Accounting.
		{name: "Accounting.ExportJSON", method: "GET", path: "/accounting/export", query: "format=json&from=2026-09-01&to=2026-09-30",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Accounting.ExportJSON(ctx, "2026-09-01", "2026-09-30", o...)
			},
			reply: list(fx("AccountingEvent")),
			check: func(t *testing.T, v any) {
				if a := is[[]AccountingEvent](t, v); len(a) != 1 || a[0].Type != AccountingEventTypeRefund {
					t.Errorf("%+v", a)
				}
			}},
		{name: "Accounting.ExportCSV", method: "GET", path: "/accounting/export", query: "format=csv&from=2026-09-01&to=2026-09-30",
			accept: "text/csv, application/json",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Accounting.ExportCSV(ctx, "2026-09-01", "2026-09-30", o...)
			},
			reply: "paymentUuid,type\npay@1,payment\n", contentType: "text/csv",
			check: func(t *testing.T, v any) {
				if s := is[string](t, v); s != "paymentUuid,type\npay@1,payment\n" {
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
		{name: "Webhooks.Endpoints.List", method: "GET", path: "/webhooks/endpoints",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.List(ctx, o...)
			},
			reply: list(fx("WebhookEndpointCreated")),
			check: func(t *testing.T, v any) {
				if e := is[[]WebhookEndpoint](t, v); len(e) != 1 || e[0].URL != "https://x.io/hook" {
					t.Errorf("%+v", e)
				}
			}},
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
		{name: "Webhooks.Endpoints.Update", method: "PUT", path: "/webhooks/endpoints/" + uuidA,
			body: `{"events":[],"description":"","payloadVersion":"v2","enabled":true}`,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Endpoints.Update(ctx, uuidA, &UpdateWebhookEndpointParams{Events: []EventType{}, Description: ptr(""),
					PayloadVersion: WebhookPayloadVersionV2, Enabled: &yes}, o...)
			},
			reply: fx("WebhookEndpointCreated")},
		{name: "Webhooks.Endpoints.Delete", method: "DELETE", path: "/webhooks/endpoints/" + uuidA,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return nil, c.Webhooks.Endpoints.Delete(ctx, uuidA, o...)
			},
			reply: `{"message":"Endpoint deleted"}`},
		{name: "Webhooks.Events.List", method: "GET", path: "/webhooks/events", query: "cursor=" + eventID + "&includeMembers=true&limit=2&type=webhook.test",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Events.List(ctx, &EventListParams{Limit: 2, Cursor: eventID, Type: EventTypeWebhookTest, IncludeMembers: true}, o...)
			},
			reply: `{"items":[` + evt + `],"nextCursor":null}`,
			check: func(t *testing.T, v any) {
				p := is[*Page[Event]](t, v)
				if len(p.Items) != 1 || p.HasMore() {
					t.Fatalf("%+v", p)
				}
				if w, err := p.Items[0].WebhookTest(); err != nil || w.EndpointUUID == "" {
					t.Errorf("%+v %v", w, err)
				}
			}},
		{name: "Webhooks.Events.Get", method: "GET", path: "/webhooks/events/" + eventID,
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Webhooks.Events.Get(ctx, eventID, o...)
			},
			reply: strings.TrimSuffix(strings.TrimSpace(evt), "}") + `,"deliveries":[{"endpointUuid":"` + uuidA + `","url":"https://x.io","delivered":true,
				"attempts":[{"uuid":"` + uuidB + `","eventId":"` + eventID + `","eventType":"webhook.test","endpointUuid":"` + uuidA + `","attempt":1,
				"delivered":true,"statusCode":200,"durationMs":12,"attemptedAt":"2026-10-01T12:00:01+02:00"}]}]}`,
			check: func(t *testing.T, v any) {
				d := is[*EventDetail](t, v)
				if d.ID != eventID || d.Type != EventTypeWebhookTest || len(d.Deliveries) != 1 || len(d.Deliveries[0].Attempts) != 1 || !d.Deliveries[0].Delivered {
					t.Errorf("%+v", d)
				}
			}},

		// Currencies.
		{name: "Currencies.ListAvailable", method: "GET", path: "/utils/all-available-currencies", query: "test=true",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.ListAvailable(ctx, &CurrencyListParams{Test: true}, o...)
			},
			reply: list(fx("Currency"))},
		{name: "Currencies.ListMain", method: "GET", path: "/utils/all-main-currencies",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.ListMain(ctx, nil, o...)
			},
			reply: list(fx("Currency"))},
		{name: "Currencies.Get", method: "GET", path: "/utils/currency/id/8",
			call: func(ctx context.Context, c *Client, o ...RequestOption) (any, error) {
				return c.Currencies.Get(ctx, 8, o...)
			},
			reply: fx("Currency"),
			check: func(t *testing.T, v any) {
				if cur := is[*Currency](t, v); cur.ID != 8 || cur.Symbol != "USDC" {
					t.Errorf("%+v", cur)
				}
			}},
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
	idempotent := 0
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
		if rc.idempotent && !strings.HasSuffix(rc.name, "minimal") {
			idempotent++
		}
	}
	if idempotent != 7 {
		t.Errorf("%d idempotent methods, want 7", idempotent)
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

	ts2 := newTestServer(t, staticReply(400, `{"error":"window too long","code":"bad_request"}`))
	c2, _ := newTestClient(t, ts2)
	csv, err := c2.Accounting.ExportCSV(ctx, "2026-01-01", "2026-09-30")
	var br *BadRequestError
	if !errors.As(err, &br) || csv != "" {
		t.Errorf("ExportCSV: %q %v", csv, err)
	}

	ts3 := newTestServer(t, staticReply(404, `{"error":"not found","code":"not_found"}`))
	c3, _ := newTestClient(t, ts3)
	if _, err := c3.Subscriptions.Cancel(ctx, subID, nil); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("Cancel: %v", err)
	}
	if err := c3.Products.Delete(ctx, uuidA); !errors.As(err, new(*NotFoundError)) {
		t.Errorf("Delete: %v", err)
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

// TestCancelNoBody checks a 2xx without the subscription is a ServerError, not an empty one.
func TestCancelNoBody(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, _ *http.Request, _ int) { w.WriteHeader(202) })
	c, _ := newTestClient(t, ts)
	if _, err := c.Subscriptions.Cancel(context.Background(), subID, nil); !errors.As(err, new(*ServerError)) {
		t.Errorf("want *ServerError, got %v", err)
	}
}
