package qbitflow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const routerSecret = "whsec_router_secret"

// routerNow is the clock every router test verifies against.
var routerNow = time.Unix(vectorT, 0)

func routerClock() VerifyOption { return WithClock(func() time.Time { return routerNow }) }

func readEventFixture(t *testing.T, eventType EventType) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "events", string(eventType)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// trace records the handlers that ran, in order.
type trace struct {
	mu  sync.Mutex
	ran []string
}

func (tr *trace) add(name string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.ran = append(tr.ran, name)
}

func (tr *trace) get() []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return slices.Clone(tr.ran)
}

func newTracedRouter(tr *trace, opts ...WebhookRouterOption) *WebhookRouter {
	return NewWebhookRouter(routerSecret, append([]WebhookRouterOption{routerClock()}, opts...)...).
		OnPaymentCompleted(func(_ context.Context, p *PaymentCompleted, e *Event) error {
			tr.add("payment.completed:" + p.Reference + ":" + e.ID)
			return nil
		}).
		OnUnknown(func(_ context.Context, e *Event) error {
			tr.add("unknown:" + string(e.Type))
			return nil
		}).
		OnAny(func(_ context.Context, e *Event) error {
			tr.add("any1:" + string(e.Type))
			return nil
		}).
		OnAny(func(_ context.Context, e *Event) error {
			tr.add("any2:" + string(e.Type))
			return nil
		})
}

func TestWebhookRouterStatusMatrix(t *testing.T) {
	payment := readEventFixture(t, EventTypePaymentCompleted)
	unknown := []byte(`{"id":"evt_1","type":"invoice.paid","version":"v2","createdAt":"2026-10-01T12:00:00Z","test":true,"data":{"x":1}}`)
	v1 := []byte(`{"id":"evt_1","type":"payment.completed","version":"v1","createdAt":"2026-10-01T12:00:00Z","data":{}}`)
	badData := []byte(`{"id":"evt_2","type":"payment.completed","version":"v2","createdAt":"2026-10-01T12:00:00Z","data":{"amount":"not a number"}}`)
	sig := func(body []byte) string { return SignWebhook(body, routerSecret, routerNow) }

	cases := []struct {
		name   string
		body   []byte
		header string
		status int
		ran    []string
		errAs  any
	}{
		{"valid", payment, sig(payment), 200, []string{
			"payment.completed:order-1042:evt_82c1e15d-c0d9-584f-a65a-fb4099925273", "any1:payment.completed", "any2:payment.completed",
		}, nil},
		{"bad signature", payment, SignWebhook(payment, "whsec_other", routerNow), 400, nil, new(*WebhookSignatureError)},
		{"missing header", payment, "", 400, nil, new(*WebhookSignatureError)},
		{"stale", payment, SignWebhook(payment, routerSecret, routerNow.Add(-301*time.Second)), 400, nil, new(*WebhookSignatureError)},
		{"not JSON", []byte("hello"), sig([]byte("hello")), 400, nil, new(*ValidationError)},
		{"v1 payload", v1, sig(v1), 400, nil, new(*ValidationError)},
		{"data not matching its type", badData, sig(badData), 400, nil, new(*ValidationError)},
		{"unknown type", unknown, sig(unknown), 200, []string{"unknown:invoice.paid", "any1:invoice.paid", "any2:invoice.paid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &trace{}
			res := newTracedRouter(tr).Handle(context.Background(), tc.body, tc.header)
			if res.Status != tc.status {
				t.Fatalf("status %d, want %d (err %v)", res.Status, tc.status, res.Err)
			}
			if got := tr.get(); !slices.Equal(got, tc.ran) {
				t.Errorf("ran %q, want %q", got, tc.ran)
			}
			if tc.errAs == nil {
				if res.Err != nil || res.Event == nil {
					t.Errorf("result %+v", res)
				}
				if string(res.ResponseBody()) != `{"received":true}` {
					t.Errorf("body %s", res.ResponseBody())
				}
				return
			}
			switch target := tc.errAs.(type) {
			case **WebhookSignatureError:
				if !errors.As(res.Err, target) {
					t.Errorf("err %T %v", res.Err, res.Err)
				}
				if string(res.ResponseBody()) != `{"error":"invalid signature"}` {
					t.Errorf("body %s", res.ResponseBody())
				}
			case **ValidationError:
				if !errors.As(res.Err, target) {
					t.Errorf("err %T %v", res.Err, res.Err)
				}
				if string(res.ResponseBody()) != `{"error":"invalid event"}` {
					t.Errorf("body %s", res.ResponseBody())
				}
			}
		})
	}
}

func TestWebhookRouterIgnoredEventIs200(t *testing.T) {
	body := readEventFixture(t, EventTypeWebhookTest)
	ran := false
	r := NewWebhookRouter(routerSecret, routerClock()).OnPaymentCompleted(func(context.Context, *PaymentCompleted, *Event) error {
		ran = true
		return nil
	})
	res := r.Handle(context.Background(), body, SignWebhook(body, routerSecret, routerNow))
	if res.Status != 200 || ran || res.Event == nil || res.Event.Type != EventTypeWebhookTest {
		t.Fatalf("%+v ran=%v", res, ran)
	}
	// Known types never reach OnUnknown.
	r.OnUnknown(func(context.Context, *Event) error { ran = true; return nil })
	if res := r.Handle(context.Background(), body, SignWebhook(body, routerSecret, routerNow)); res.Status != 200 || ran {
		t.Fatalf("%+v ran=%v", res, ran)
	}
}

func TestWebhookRouterHandlerError(t *testing.T) {
	body := readEventFixture(t, EventTypePaymentCompleted)
	boom := errors.New("database down")
	tr := &trace{}
	var reported []error
	r := NewWebhookRouter(routerSecret, routerClock(), WithWebhookErrorHandler(func(e *Event, err error) {
		if e == nil || e.Type != EventTypePaymentCompleted {
			t.Errorf("error handler event %+v", e)
		}
		reported = append(reported, err)
	})).
		OnPaymentCompleted(func(context.Context, *PaymentCompleted, *Event) error { tr.add("first"); return nil }).
		OnPaymentCompleted(func(context.Context, *PaymentCompleted, *Event) error { tr.add("second"); return boom }).
		OnPaymentCompleted(func(context.Context, *PaymentCompleted, *Event) error { tr.add("third"); return nil }).
		OnAny(func(context.Context, *Event) error { tr.add("any"); return nil })

	res := r.Handle(context.Background(), body, SignWebhook(body, routerSecret, routerNow))
	if res.Status != 500 || !errors.Is(res.Err, boom) || res.Event == nil {
		t.Fatalf("%+v", res)
	}
	if got := tr.get(); !slices.Equal(got, []string{"first", "second"}) {
		t.Errorf("ran %q: the handlers after the failing one must not run", got)
	}
	if len(reported) != 1 || !errors.Is(reported[0], boom) {
		t.Errorf("reported %v", reported)
	}
	if string(res.ResponseBody()) != `{"error":"internal error"}` || strings.Contains(string(res.ResponseBody()), "database") {
		t.Errorf("body %s", res.ResponseBody())
	}

	// An OnAny failing is a 500 too.
	r2 := NewWebhookRouter(routerSecret, routerClock()).OnAny(func(context.Context, *Event) error { return boom })
	if res := r2.Handle(context.Background(), body, SignWebhook(body, routerSecret, routerNow)); res.Status != 500 {
		t.Errorf("OnAny error: %+v", res)
	}
}

func TestWebhookRouterErrorHandlerSeesRefusals(t *testing.T) {
	var got []error
	r := NewWebhookRouter(routerSecret, routerClock(), WithWebhookErrorHandler(func(e *Event, err error) {
		if e != nil {
			t.Errorf("event %+v on a refused signature", e)
		}
		got = append(got, err)
	}))
	if res := r.Handle(context.Background(), []byte("{}"), "t=1,v1=00"); res.Status != 400 {
		t.Fatalf("%+v", res)
	}
	if len(got) != 1 || !errors.As(got[0], new(*WebhookSignatureError)) {
		t.Errorf("reported %v", got)
	}
}

func TestWebhookRouterWithoutSecret(t *testing.T) {
	body := readEventFixture(t, EventTypeWebhookTest)
	res := NewWebhookRouter("").Handle(context.Background(), body, SignWebhook(body, "", routerNow))
	if res.Status != 500 || !errors.As(res.Err, new(*ValidationError)) {
		t.Fatalf("%+v", res)
	}
}

func TestWebhookRouterTypedRegistrations(t *testing.T) {
	// Every typed method decodes its fixture and gets it.
	tr := &trace{}
	r := NewWebhookRouter(routerSecret, routerClock())
	h := func(name string) func() { return func() { tr.add(name) } }
	register := map[EventType]func(func()){
		EventTypePaymentCompleted: func(f func()) {
			r.OnPaymentCompleted(func(_ context.Context, d *PaymentCompleted, _ *Event) error {
				decoded(t, d.UUID != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionCreated: func(f func()) {
			r.OnSubscriptionCreated(func(_ context.Context, d *SubscriptionCreated, _ *Event) error {
				decoded(t, d.UUID != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionBilled: func(f func()) {
			r.OnSubscriptionBilled(func(_ context.Context, d *SubscriptionBilled, _ *Event) error {
				decoded(t, d.UUID != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionStatusChanged: func(f func()) {
			r.OnSubscriptionStatusChanged(func(_ context.Context, d *SubscriptionStatusChanged, _ *Event) error {
				decoded(t, d.PreviousStatus != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionActionRequiredChanged: func(f func()) {
			r.OnSubscriptionActionRequiredChanged(func(_ context.Context, d *SubscriptionActionRequiredChanged, _ *Event) error {
				decoded(t, d.UUID != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionBillingFailed: func(f func()) {
			r.OnSubscriptionBillingFailed(func(_ context.Context, d *SubscriptionBillingFailed, _ *Event) error {
				decoded(t, d.Reason != "")
				f()
				return nil
			})
		},
		EventTypeSubscriptionUpcomingBill: func(f func()) {
			r.OnSubscriptionUpcomingBill(func(_ context.Context, d *SubscriptionUpcomingBill, _ *Event) error {
				decoded(t, !d.BillingDate.IsZero())
				f()
				return nil
			})
		},
		EventTypeRefundRequested: func(f func()) {
			r.OnRefundRequested(func(_ context.Context, d *Refund, _ *Event) error { decoded(t, d.UUID != ""); f(); return nil })
		},
		EventTypeRefundCompleted: func(f func()) {
			r.OnRefundCompleted(func(_ context.Context, d *Refund, _ *Event) error { decoded(t, d.UUID != ""); f(); return nil })
		},
		EventTypeRefundDenied: func(f func()) {
			r.OnRefundDenied(func(_ context.Context, d *Refund, _ *Event) error { decoded(t, d.UUID != ""); f(); return nil })
		},
		EventTypeMemberJoined: func(f func()) {
			r.OnMemberJoined(func(_ context.Context, d *MemberJoined, _ *Event) error {
				decoded(t, d.InvitationUUID != "")
				f()
				return nil
			})
		},
		EventTypeMemberRemoved: func(f func()) {
			r.OnMemberRemoved(func(_ context.Context, d *Member, _ *Event) error { decoded(t, d.UserUUID != ""); f(); return nil })
		},
		EventTypeHeldFundsReleased: func(f func()) {
			r.OnHeldFundsReleased(func(_ context.Context, d *HeldFundsReleased, _ *Event) error {
				decoded(t, d.UUID != "")
				f()
				return nil
			})
		},
		EventTypeCheckoutExpired: func(f func()) {
			r.OnCheckoutExpired(func(_ context.Context, d *CheckoutExpired, _ *Event) error {
				decoded(t, d.Payment != nil || d.Subscription != nil)
				f()
				return nil
			})
		},
		EventTypeWebhookTest: func(f func()) {
			r.OnWebhookTest(func(_ context.Context, d *WebhookTest, _ *Event) error {
				decoded(t, d.EndpointUUID != "")
				f()
				return nil
			})
		},
	}
	if len(register) != len(knownEventTypes) {
		t.Fatalf("%d registrations for %d types", len(register), len(knownEventTypes))
	}
	for eventType, reg := range register {
		reg(h(string(eventType)))
	}
	for _, eventType := range knownEventTypes {
		body := readEventFixture(t, eventType)
		res := r.Handle(context.Background(), body, SignWebhook(body, routerSecret, routerNow))
		if res.Status != 200 {
			t.Errorf("%s: %+v", eventType, res)
		}
	}
	if got := tr.get(); !slices.Equal(got, func() []string {
		names := make([]string, len(knownEventTypes))
		for i, e := range knownEventTypes {
			names[i] = string(e)
		}
		return names
	}()) {
		t.Errorf("ran %q", got)
	}
}

func decoded(t *testing.T, ok bool) {
	t.Helper()
	if !ok {
		t.Error("typed data not decoded")
	}
}

func TestWebhookRouterNilHandlerPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	NewWebhookRouter(routerSecret).OnPaymentCompleted(nil)
}

func TestWebhookRouterHTTPHandler(t *testing.T) {
	payment := readEventFixture(t, EventTypePaymentCompleted)
	tr := &trace{}
	var ctxSeen context.Context
	type ctxKey struct{}
	router := newTracedRouter(tr).OnAny(func(ctx context.Context, _ *Event) error {
		ctxSeen = ctx
		return nil
	})
	mux := http.NewServeMux()
	mux.Handle("/webhooks/qbitflow", router)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, "request")))
	}))
	defer srv.Close()

	post := func(body []byte, headerKey, header string) (*http.Response, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhooks/qbitflow", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if headerKey != "" {
			req.Header[headerKey] = []string{header} // sent with this exact case
		}
		res, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		got, _ := io.ReadAll(res.Body)
		return res, string(got)
	}

	for _, key := range []string{"QBitFlow-Signature", "qbitflow-signature", "QBITFLOW-SIGNATURE"} {
		res, body := post(payment, key, SignWebhook(payment, routerSecret, routerNow))
		if res.StatusCode != 200 || body != `{"received":true}` || res.Header.Get("Content-Type") != "application/json" {
			t.Errorf("%s: %d %s", key, res.StatusCode, body)
		}
	}
	if ctxSeen == nil || ctxSeen.Value(ctxKey{}) != "request" {
		t.Error("handlers did not get the request's context")
	}

	if res, body := post(payment, "QBitFlow-Signature", SignWebhook(payment, "whsec_other", routerNow)); res.StatusCode != 400 || body != `{"error":"invalid signature"}` {
		t.Errorf("bad signature: %d %s", res.StatusCode, body)
	}
	if res, body := post([]byte("[]"), "QBitFlow-Signature", SignWebhook([]byte("[]"), routerSecret, routerNow)); res.StatusCode != 400 || body != `{"error":"invalid event"}` {
		t.Errorf("not an event: %d %s", res.StatusCode, body)
	}

	// Over 1 MiB: 413, whether the length is announced or not.
	big := bytes.Repeat([]byte("a"), maxWebhookBody+1)
	if res, body := post(big, "QBitFlow-Signature", "t=1,v1=00"); res.StatusCode != 413 || body != `{"error":"body too large"}` {
		t.Errorf("big body: %d %s", res.StatusCode, body)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/qbitflow", io.MultiReader(bytes.NewReader(big)))
	req.ContentLength = -1
	router.ServeHTTP(rec, req)
	if rec.Code != 413 {
		t.Errorf("big body (unknown length): %d", rec.Code)
	}
	// Exactly 1 MiB is read (and refused for its signature, not its size).
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(big[:maxWebhookBody])))
	if rec.Code != 400 {
		t.Errorf("1 MiB body: %d", rec.Code)
	}

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodHead} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(method, "/webhooks/qbitflow", nil))
		if rec.Code != 405 || rec.Header().Get("Allow") != http.MethodPost {
			t.Errorf("%s: %d %v", method, rec.Code, rec.Header())
		}
	}

	// A hand-built request: nil body, a non-canonical header key.
	rec = httptest.NewRecorder()
	hand := &http.Request{Method: http.MethodPost, Header: http.Header{"qbitflow-signature": {SignWebhook(nil, routerSecret, routerNow)}}}
	router.ServeHTTP(rec, hand.WithContext(context.Background()))
	if rec.Code != 400 || rec.Body.String() != `{"error":"invalid event"}` {
		t.Errorf("hand-built request: %d %s", rec.Code, rec.Body)
	}
}

func TestWebhookRouterConcurrent(t *testing.T) {
	body := readEventFixture(t, EventTypePaymentCompleted)
	tr := &trace{}
	router := newTracedRouter(tr)
	header := SignWebhook(body, routerSecret, routerNow)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if res := router.Handle(context.Background(), body, header); res.Status != 200 {
				t.Errorf("%+v", res)
			}
		})
	}
	wg.Wait()
	if n := len(tr.get()); n != 16*3 {
		t.Errorf("%d handler runs", n)
	}
}

func TestSignWebhook(t *testing.T) {
	body := []byte(vectorBody)
	want := "t=1790856000,v1=" + vectorNew
	if got := SignWebhook(body, "whsec_new_secret", time.Unix(vectorT, 0)); got != want {
		t.Fatalf("SignWebhook = %s, want %s", got, want)
	}
	// Round trip, at now (the zero time) too.
	if err := VerifyWebhook(body, SignWebhook(body, "whsec_x", time.Time{}), "whsec_x"); err != nil {
		t.Fatal(err)
	}
	if e, err := ConstructEvent(body, SignWebhook(body, "whsec_x", time.Now()), "whsec_x"); err != nil || e.Type != EventTypeWebhookTest {
		t.Fatalf("%+v %v", e, err)
	}
	if err := VerifyWebhook(body, SignWebhook(body, "whsec_x", time.Now()), "whsec_y"); reasonOf(err) != WebhookSignatureNoMatchingSignature {
		t.Fatalf("other secret: %v", err)
	}
}
