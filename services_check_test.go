package qbitflow

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"testing"
)

// TestServiceValidationBeforeRequest checks that invalid path ids and params fail with a
// *ValidationError naming the input, and that nothing is sent.
func TestServiceValidationBeforeRequest(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()
	yes := true
	type call = func() error
	drop := func(_ any, err error) error { return err }

	cases := map[string]struct {
		field string
		call  call
	}{
		// Path ids.
		"Products.Get empty":           {"uuid", func() error { return drop(c.Products.Get(ctx, "")) }},
		"Products.Get not uuid":        {"uuid", func() error { return drop(c.Products.Get(ctx, "42")) }},
		"Products.Update":              {"uuid", func() error { return drop(c.Products.Update(ctx, "../x", &UpdateProductParams{})) }},
		"Products.Delete":              {"uuid", func() error { return c.Products.Delete(ctx, "") }},
		"Products.GetByReference":      {"reference", func() error { return drop(c.Products.GetByReference(ctx, "  ")) }},
		"Customers.Get":                {"uuid", func() error { return drop(c.Customers.Get(ctx, "pay@"+uuidA)) }},
		"Customers.Update":             {"uuid", func() error { return drop(c.Customers.Update(ctx, "", &UpdateCustomerParams{})) }},
		"Customers.Delete":             {"uuid", func() error { return c.Customers.Delete(ctx, "x") }},
		"Customers.GetByEmail":         {"email", func() error { return drop(c.Customers.GetByEmail(ctx, "")) }},
		"Customers.GetByReference":     {"reference", func() error { return drop(c.Customers.GetByReference(ctx, "")) }},
		"CheckoutSessions.GetStatus":   {"uuid", func() error { return drop(c.CheckoutSessions.GetStatus(ctx, "")) }},
		"CheckoutSessions.Expire":      {"uuid", func() error { return drop(c.CheckoutSessions.Expire(ctx, "cs_123")) }},
		"Payments.Get":                 {"uuid", func() error { return drop(c.Payments.Get(ctx, "pay@123", nil)) }},
		"Payments.GetByReference":      {"reference", func() error { return drop(c.Payments.GetByReference(ctx, "")) }},
		"Subscriptions.Get":            {"uuid", func() error { return drop(c.Subscriptions.Get(ctx, "bad@"+uuidA, nil)) }},
		"Subscriptions.GetByReference": {"reference", func() error { return drop(c.Subscriptions.GetByReference(ctx, "")) }},
		"Subscriptions.ListBills":      {"uuid", func() error { return drop(c.Subscriptions.ListBills(ctx, "", nil)) }},
		"Subscriptions.GetBill":        {"billUuid", func() error { return drop(c.Subscriptions.GetBill(ctx, "1", nil)) }},
		"Subscriptions.GetPublicHistory": {"subscriptionUuid", func() error {
			return drop(c.Subscriptions.GetPublicHistory(ctx, ""))
		}},
		"Subscriptions.Cancel":             {"uuid", func() error { return drop(c.Subscriptions.Cancel(ctx, "", &CancelSubscriptionParams{Immediate: &yes})) }},
		"Subscriptions.ExecuteTestBilling": {"uuid", func() error { return drop(c.Subscriptions.ExecuteTestBilling(ctx, "sub")) }},
		"Members.Get":                      {"userUuid", func() error { return drop(c.Members.Get(ctx, "")) }},
		"Members.Update":                   {"userUuid", func() error { return drop(c.Members.Update(ctx, "1", &UpdateMemberParams{})) }},
		"Members.Remove":                   {"userUuid", func() error { return c.Members.Remove(ctx, "") }},
		"Members.Trust":                    {"userUuid", func() error { return drop(c.Members.Trust(ctx, "x")) }},
		"Members.GetHeldFunds":             {"userUuid", func() error { return drop(c.Members.GetHeldFunds(ctx, "")) }},
		"Invitations.Revoke":               {"uuid", func() error { return drop(c.Invitations.Revoke(ctx, "")) }},
		"Wallets.ListForMember":            {"userUuid", func() error { return drop(c.Wallets.ListForMember(ctx, "1")) }},
		"Webhooks.Endpoints.Get":           {"uuid", func() error { return drop(c.Webhooks.Endpoints.Get(ctx, "")) }},
		"Webhooks.Endpoints.Update": {"uuid", func() error {
			return drop(c.Webhooks.Endpoints.Update(ctx, "x", &UpdateWebhookEndpointParams{}))
		}},
		"Webhooks.Endpoints.Delete": {"uuid", func() error { return c.Webhooks.Endpoints.Delete(ctx, "") }},
		"Webhooks.Events.Get":       {"id", func() error { return drop(c.Webhooks.Events.Get(ctx, "")) }},
		"Webhooks.VerifyRemote uuid": {"endpointUuid", func() error {
			return c.Webhooks.VerifyRemote(ctx, "", []byte(`{}`), "t=1,v1=a")
		}},
		"Webhooks.VerifyRemote body": {"body", func() error { return c.Webhooks.VerifyRemote(ctx, uuidA, nil, "t=1,v1=a") }},
		"Webhooks.VerifyRemote utf8": {"body", func() error {
			return c.Webhooks.VerifyRemote(ctx, uuidA, []byte("\xff"), "t=1,v1=a")
		}},
		"Currencies.Get": {"id", func() error { return drop(c.Currencies.Get(ctx, 0)) }},

		// Required params.
		"Products.Create nil":                   {"", func() error { return drop(c.Products.Create(ctx, nil)) }},
		"Products.Update nil":                   {"", func() error { return drop(c.Products.Update(ctx, uuidA, nil)) }},
		"Customers.Create nil":                  {"", func() error { return drop(c.Customers.Create(ctx, nil)) }},
		"Customers.Update nil":                  {"", func() error { return drop(c.Customers.Update(ctx, uuidA, nil)) }},
		"CheckoutSessions.CreatePayment nil":    {"", func() error { return drop(c.CheckoutSessions.CreatePayment(ctx, nil)) }},
		"CheckoutSessions.CreateSubscription 0": {"", func() error { return drop(c.CheckoutSessions.CreateSubscription(ctx, nil)) }},
		"Refunds.Initiate nil":                  {"", func() error { return drop(c.Refunds.Initiate(ctx, nil)) }},
		"Members.Update nil":                    {"", func() error { return drop(c.Members.Update(ctx, memberUUID, nil)) }},
		"Invitations.Create nil":                {"", func() error { return drop(c.Invitations.Create(ctx, nil)) }},
		"Webhooks.Endpoints.Create nil":         {"", func() error { return drop(c.Webhooks.Endpoints.Create(ctx, nil)) }},
		"Webhooks.Endpoints.Update nil":         {"", func() error { return drop(c.Webhooks.Endpoints.Update(ctx, uuidA, nil)) }},

		// Invalid params.
		"Products.Create price": {"price", func() error { return drop(c.Products.Create(ctx, &CreateProductParams{Name: "Pro", Price: -1})) }},
		"Products.Update name":  {"name", func() error { return drop(c.Products.Update(ctx, uuidA, &UpdateProductParams{Name: "<b>"})) }},
		"Customers.Create email": {"email", func() error {
			return drop(c.Customers.Create(ctx, &CreateCustomerParams{Name: "Ada", Email: "ada"}))
		}},
		"Customers.Update phone": {"phoneNumber", func() error {
			return drop(c.Customers.Update(ctx, uuidA, &UpdateCustomerParams{PhoneNumber: ptr("call me")}))
		}},
		"Customers.List email": {"email", func() error { return drop(c.Customers.List(ctx, &CustomerListParams{Email: "x"})) }},
		"CheckoutSessions.CreatePayment fee amount": {"fees.items[0].amountUsd", func() error {
			return drop(c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductUUID: uuidA,
				Fees: &CheckoutFees{Items: []FeeItem{{Label: "Shipping", AmountUSD: 4.999}}}}))
		}},
		"CheckoutSessions.CreateSubscription fee label": {"fees.items[0].label", func() error {
			return drop(c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductUUID: uuidA,
				Fees: &CheckoutFees{Items: []FeeItem{{Label: "", AmountUSD: 2}}}}))
		}},
		"CheckoutSessions.CreatePayment mixed": {"productUuid", func() error {
			return drop(c.CheckoutSessions.CreatePayment(ctx, &CreatePaymentSessionParams{ProductUUID: uuidA, ProductReference: "pro"}))
		}},
		"CheckoutSessions.CreateSubscription frequency": {"frequency.unit", func() error {
			return drop(c.CheckoutSessions.CreateSubscription(ctx, &CreateSubscriptionSessionParams{ProductUUID: uuidA, Frequency: &Duration{Value: 1}}))
		}},
		"Payments.List exclusive": {"userUuid", func() error {
			return drop(c.Payments.List(ctx, &PaymentListParams{IncludeMembers: true, UserUUID: memberUUID}))
		}},
		"Payments.ListCombined source": {"source", func() error {
			return drop(c.Payments.ListCombined(ctx, &CombinedPaymentListParams{Source: "bills"}))
		}},
		"Failures.List kind": {"kind", func() error { return drop(c.Failures.List(ctx, &FailureListParams{Kind: "refund"})) }},
		"Subscriptions.List status": {"status", func() error {
			return drop(c.Subscriptions.List(ctx, &SubscriptionListParams{Status: "lowOnFunds"}))
		}},
		"Refunds.List userUuid": {"userUuid", func() error { return drop(c.Refunds.List(ctx, &RefundListParams{UserUUID: "1"})) }},
		"Refunds.ListInactive exclusive": {"userUuid", func() error {
			return drop(c.Refunds.ListInactive(ctx, &RefundListParams{IncludeMembers: &yes, UserUUID: memberUUID}))
		}},
		"Refunds.Initiate percent": {"refundPercent", func() error {
			return drop(c.Refunds.Initiate(ctx, &InitiateRefundParams{TxUUID: payID, RefundPercent: ptr(0.0)}))
		}},
		"Members.Update fee": {"organizationFeePercent", func() error {
			return drop(c.Members.Update(ctx, memberUUID, &UpdateMemberParams{OrganizationFeePercent: 51}))
		}},
		"Invitations.Create email": {"email", func() error { return drop(c.Invitations.Create(ctx, &CreateInvitationParams{})) }},
		"Invitations.List status": {"status", func() error {
			return drop(c.Invitations.List(ctx, &InvitationListParams{Status: "open"}))
		}},
		"Wallets.ListSupportedCurrencies": {"userUuid", func() error {
			return drop(c.Wallets.ListSupportedCurrencies(ctx, &SupportedCurrenciesParams{UserUUID: "x"}))
		}},
		"Accounting.ExportJSON order": {"to", func() error { return drop(c.Accounting.ExportJSON(ctx, "2026-09-30", "2026-09-01")) }},
		"Accounting.ExportCSV date":   {"from", func() error { return drop(c.Accounting.ExportCSV(ctx, "2026-02-30", "2026-03-01")) }},
		"Webhooks.Endpoints.Create url": {"url", func() error {
			return drop(c.Webhooks.Endpoints.Create(ctx, &CreateWebhookEndpointParams{URL: "ftp://x"}))
		}},
		"Webhooks.Endpoints.Update test event": {"events[0]", func() error {
			return drop(c.Webhooks.Endpoints.Update(ctx, uuidA, &UpdateWebhookEndpointParams{Events: []EventType{EventTypeWebhookTest}}))
		}},
		"Webhooks.Events.List type": {"type", func() error {
			return drop(c.Webhooks.Events.List(ctx, &EventListParams{Type: "payment.failed"}))
		}},
	}

	for name, tc := range cases {
		err := tc.call()
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want *ValidationError, got %v", name, err)
			continue
		}
		if ve.StatusCode != 0 {
			t.Errorf("%s: status %d on a client-side error", name, ve.StatusCode)
		}
		if tc.field != "" && !slices.ContainsFunc(ve.FieldErrors, func(f FieldError) bool { return f.Field == tc.field }) {
			t.Errorf("%s: field errors %+v, want one on %q", name, ve.FieldErrors, tc.field)
		}
	}
	if n := len(ts.recorded()); n != 0 {
		t.Errorf("%d requests sent", n)
	}
}

// TestVerifyRemoteErrors checks how VerifyRemote maps the API's answers.
func TestVerifyRemoteErrors(t *testing.T) {
	ctx := context.Background()
	body := []byte(`{"id":"evt_1"}`)
	header := "t=1790856000,v1=abc"

	cases := []struct {
		name   string
		status int
		reply  string
		check  func(error) bool
	}{
		{"valid", 200, `{"message":"ok"}`, func(err error) bool { return err == nil }},
		{"invalid signature", 400, `{"error":"invalid signature","code":"invalid_signature","requestId":"r-9"}`, func(err error) bool {
			var se *WebhookSignatureError
			return errors.As(err, &se) && se.Reason == WebhookSignatureInvalidSignature && se.StatusCode == 400 &&
				se.Code == "invalid_signature" && se.RequestID == "r-9"
		}},
		{"validation", 400, `{"error":"invalid","code":"validation_failed","details":{"errors":[{"field":"signature","message":"required"}]}}`, func(err error) bool {
			var ve *ValidationError
			return errors.As(err, &ve) && ve.StatusCode == 400 && !errors.As(err, new(*WebhookSignatureError))
		}},
		{"other bad request", 400, `{"error":"bad","code":"bad_request"}`, func(err error) bool {
			return errors.As(err, new(*BadRequestError)) && !errors.As(err, new(*WebhookSignatureError))
		}},
		{"not found", 404, `{"error":"not found","code":"not_found"}`, func(err error) bool {
			return errors.As(err, new(*NotFoundError))
		}},
		{"server", 500, `{"error":"boom","code":"internal"}`, func(err error) bool {
			return errors.As(err, new(*ServerError))
		}},
	}
	for _, tc := range cases {
		ts := newTestServer(t, staticReply(tc.status, tc.reply))
		c, _ := newTestClient(t, ts)
		err := c.Webhooks.VerifyRemote(ctx, uuidA, body, header)
		if !tc.check(err) {
			t.Errorf("%s: unexpected %T %v", tc.name, err, err)
		}
		if n := len(ts.recorded()); n != 1 {
			t.Errorf("%s: %d requests (verify is never retried)", tc.name, n)
		}
	}

	// An empty header is a signature failure, found without a request.
	ts := newTestServer(t, staticReply(200, `{}`))
	c, _ := newTestClient(t, ts)
	err := c.Webhooks.VerifyRemote(ctx, uuidA, body, "")
	var se *WebhookSignatureError
	if !errors.As(err, &se) || se.Reason != WebhookSignatureMissingHeader {
		t.Errorf("empty header: %v", err)
	}
	big := make([]byte, maxWebhookBody+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := c.Webhooks.VerifyRemote(ctx, uuidA, big, header); !errors.As(err, new(*ValidationError)) {
		t.Errorf("1 MiB body: %v", err)
	}
	if n := len(ts.recorded()); n != 0 {
		t.Errorf("%d requests sent", n)
	}
}

// iterCase is one Iterate method over a two-page list.
type iterCase struct {
	name, path string
	// filters is the query every page carries (url-encoded, without cursor).
	filters string
	// item renders the list's i-th row with the id "id<i>".
	item func(i int) string
	// walk runs the iterator and returns the ids it yielded (stopping after max items, 0 = all).
	walk func(ctx context.Context, c *Client, stopAfter int) ([]string, error)
}

// collect runs seq, keeping id(item) for each item, until max items (0 = all) or an error.
func collect[T any](seq iter.Seq2[T, error], id func(T) string, stopAfter int) ([]string, error) {
	var ids []string
	for v, err := range seq {
		if err != nil {
			return ids, err
		}
		ids = append(ids, id(v))
		if stopAfter > 0 && len(ids) == stopAfter {
			break
		}
	}
	return ids, nil
}

func uuidItem(i int) string { return fmt.Sprintf(`{"uuid":"id%d"}`, i) }

func iterCases() []iterCase {
	no := false
	return []iterCase{
		{"Customers.Iterate", "/customer/all", "email=a%40b.co&limit=2&verified=false", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Customers.Iterate(ctx, &CustomerListParams{Limit: 2, Email: "a@b.co", Verified: &no}),
					func(v Customer) string { return v.UUID }, stopAfter)
			}},
		{"Payments.Iterate", "/transaction/payments", "createdAfter=" + afterQ + "&limit=2", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Payments.Iterate(ctx, &PaymentListParams{Limit: 2, CreatedAfter: &after}),
					func(v Payment) string { return v.UUID }, stopAfter)
			}},
		{"Payments.IterateCombined", "/transaction/payments/combined", "source=payment", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Payments.IterateCombined(ctx, &CombinedPaymentListParams{Source: CombinedPaymentSourcePayment}),
					func(v CombinedPayment) string { return v.UUID }, stopAfter)
			}},
		{"Failures.Iterate", "/transaction/failures", "category=reverted", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Failures.Iterate(ctx, &FailureListParams{Category: FailureCategoryReverted}),
					func(v Failure) string { return v.UUID }, stopAfter)
			}},
		{"Subscriptions.Iterate", "/transaction/subscriptions", "includeMembers=true&status=active", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Subscriptions.Iterate(ctx, &SubscriptionListParams{IncludeMembers: true, Status: SubscriptionStatusActive}),
					func(v Subscription) string { return v.UUID }, stopAfter)
			}},
		{"Subscriptions.IterateBills", "/transaction/subscription/" + subID + "/bills", "limit=100", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Subscriptions.IterateBills(ctx, subID, &BillListParams{Limit: 100}),
					func(v Bill) string { return v.UUID }, stopAfter)
			}},
		{"Refunds.IterateInactive", "/transaction/refunds/all/inactive", "held=false", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Refunds.IterateInactive(ctx, &RefundListParams{Held: &no}),
					func(v Refund) string { return v.UUID }, stopAfter)
			}},
		{"Members.Iterate", "/members", "limit=1", func(i int) string { return fmt.Sprintf(`{"userUuid":"id%d"}`, i) },
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Members.Iterate(ctx, &MemberListParams{Limit: 1}),
					func(v Member) string { return v.UserUUID }, stopAfter)
			}},
		{"Invitations.Iterate", "/invitations", "status=expired", uuidItem,
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Invitations.Iterate(ctx, &InvitationListParams{Status: InvitationStatusExpired}),
					func(v Invitation) string { return v.UUID }, stopAfter)
			}},
		{"Webhooks.Events.Iterate", "/webhooks/events", "type=payment.completed",
			func(i int) string {
				return fmt.Sprintf(`{"id":"id%d","type":"payment.completed","version":"v2","createdAt":"2026-10-01T12:00:00Z","data":{}}`, i)
			},
			func(ctx context.Context, c *Client, stopAfter int) ([]string, error) {
				return collect(c.Webhooks.Events.Iterate(ctx, &EventListParams{Type: EventTypePaymentCompleted}),
					func(v Event) string { return v.ID }, stopAfter)
			}},
	}
}

// TestServiceIterators walks each Iterate method over two pages: every item once, the filters
// on every page, the cursor passed back, an early break stopping the requests, and an error
// ending the sequence.
func TestServiceIterators(t *testing.T) {
	for _, ic := range iterCases() {
		t.Run(ic.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
				switch r.URL.Query().Get("cursor") {
				case "":
					jsonReply(w, 200, `{"items":[`+ic.item(0)+`,`+ic.item(1)+`],"nextCursor":"id1"}`)
				case "id1":
					jsonReply(w, 200, `{"items":[`+ic.item(2)+`],"nextCursor":null}`)
				default:
					jsonReply(w, 400, `{"error":"bad cursor","code":"validation_failed"}`)
				}
			})
			c, _ := newTestClient(t, ts)
			ctx := context.Background()

			ids, err := ic.walk(ctx, c, 0)
			if err != nil || !slices.Equal(ids, []string{"id0", "id1", "id2"}) {
				t.Fatalf("walk: %v %v", ids, err)
			}
			reqs := ts.recorded()
			if len(reqs) != 2 {
				t.Fatalf("%d requests, want 2", len(reqs))
			}
			for i, r := range reqs {
				if r.method != http.MethodGet || r.path != ic.path {
					t.Errorf("page %d: %s %s", i, r.method, r.path)
				}
				q, _ := url.ParseQuery(r.query)
				cursor := q.Get("cursor")
				q.Del("cursor")
				if q.Encode() != ic.filters {
					t.Errorf("page %d: filters %q, want %q", i, q.Encode(), ic.filters)
				}
				if want := []string{"", "id1"}[i]; cursor != want {
					t.Errorf("page %d: cursor %q, want %q", i, cursor, want)
				}
			}

			// An early break stops after the first page.
			ids, err = ic.walk(ctx, c, 1)
			if err != nil || len(ids) != 1 || len(ts.recorded()) != 3 {
				t.Errorf("early break: %v %v, %d requests", ids, err, len(ts.recorded())-2)
			}
		})
	}
}

// TestIteratorErrors checks an error ends the sequence and invalid params send nothing.
func TestIteratorErrors(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		if r.URL.Query().Get("cursor") == "" {
			jsonReply(w, 200, `{"items":[{"uuid":"id0"}],"nextCursor":"id0"}`)
			return
		}
		jsonReply(w, 401, `{"error":"unauthorized","code":"unauthorized"}`)
	})
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	ids, err := collect(c.Payments.Iterate(ctx, nil), func(p Payment) string { return p.UUID }, 0)
	if !errors.As(err, new(*AuthenticationError)) || !slices.Equal(ids, []string{"id0"}) {
		t.Errorf("want id0 then an AuthenticationError, got %v %v", ids, err)
	}

	before := len(ts.recorded())
	_, err = collect(c.Subscriptions.IterateBills(ctx, "nope", nil), func(b Bill) string { return b.UUID }, 0)
	if !errors.As(err, new(*ValidationError)) {
		t.Errorf("bad subscription id: %v", err)
	}
	_, err = collect(c.Members.Iterate(ctx, nil, WithOnBehalfOf("x")), func(m Member) string { return m.UserUUID }, 0)
	if !errors.As(err, new(*ValidationError)) {
		t.Errorf("bad On-Behalf-Of: %v", err)
	}
	if n := len(ts.recorded()) - before; n != 0 {
		t.Errorf("%d requests sent for invalid input", n)
	}

	// Resuming from a cursor starts there (the first page is never re-requested).
	ts2 := pagedServer(t, "")
	c2, _ := newTestClient(t, ts2)
	ids, err = collect(c2.Customers.Iterate(ctx, &CustomerListParams{Cursor: "c3"}), func(cu Customer) string { return cu.UUID }, 0)
	if err != nil || !slices.Equal(ids, []string{"c4"}) || len(ts2.recorded()) != 1 {
		t.Errorf("resume: %v %v", ids, err)
	}
}
