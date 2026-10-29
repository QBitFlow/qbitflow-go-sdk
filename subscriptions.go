package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's subscriptions (GET /subscriptions; newest first,
// cancelled ones included, page size 20 by default, at most 100). A checkout not completed yet
// is no subscription. Filters: Status, Reference, customer, product, creation window, and
// IncludeMembers or UserUUID (organization key). params may be nil.
//
// Grant access while now < AccessUntil, whatever the status (Subscription.HasAccess).
func (s *SubscriptionsService) List(ctx context.Context, params *SubscriptionListParams, opts ...RequestOption) (*Page[Subscription], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Subscription]](ctx, endpoint{method: http.MethodGet, path: "/subscriptions", query: params.values()}, opts)
}

// Iterate walks every subscription List returns, fetching the pages lazily and keeping the
// filters and the page size. An error ends the sequence.
func (s *SubscriptionsService) Iterate(ctx context.Context, params *SubscriptionListParams, opts ...RequestOption) iter.Seq2[Subscription, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Subscription], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.List(ctx, &page, opts...)
	})
}

// Get returns a subscription by its id (GET /subscriptions/:uuid; sub@… or the bare UUID),
// whatever its status (a cancelled one too). params.IncludeMembers reads a member's from the
// organization's space; params may be nil. Before its checkout completes it is a
// *NotFoundError: read the checkout session instead.
func (s *SubscriptionsService) Get(ctx context.Context, uuid string, params *ReadParams, opts ...RequestOption) (*Subscription, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Subscription](ctx, endpoint{method: http.MethodGet, path: pathf("/subscriptions/%s", uuid), query: params.values()}, opts)
}

// GetByReference returns the space's subscription with the reference given to its checkout
// session, whatever its status: it runs List's reference filter (GET /subscriptions?reference=)
// and returns its one subscription. No subscription with that reference is a *NotFoundError
// (status 404, code not_found).
func (s *SubscriptionsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Subscription, error) {
	return s.client.getByReference[Subscription](ctx, "/subscriptions", "subscription", reference, func(v *Subscription) string { return v.Reference }, opts)
}

// Update changes a subscription's metadata (PATCH /subscriptions/:uuid), merged into its own (a
// key set to "" deletes it; at most 50 keys once merged), cancelled ones too, and returns the
// subscription. Its price, terms and status don't change here. A member's key needs the organization's members.subscriptions policy (403
// policy_disabled). Not retried.
func (s *SubscriptionsService) Update(ctx context.Context, uuid string, params *UpdateSubscriptionParams, opts ...RequestOption) (*Subscription, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Subscription](ctx, endpoint{method: http.MethodPatch, path: pathf("/subscriptions/%s", uuid), body: params}, opts)
}

// ListBills returns one page of a subscription's bills (GET /subscriptions/:uuid/bills; newest
// first, page size 20 by default, at most 100). params may be nil.
func (s *SubscriptionsService) ListBills(ctx context.Context, uuid string, params *BillListParams, opts ...RequestOption) (*Page[Bill], error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Bill]](ctx, endpoint{method: http.MethodGet, path: pathf("/subscriptions/%s/bills", uuid), query: params.values()}, opts)
}

// IterateBills walks every bill of a subscription, fetching the pages lazily and keeping the
// page size. An error ends the sequence.
func (s *SubscriptionsService) IterateBills(ctx context.Context, uuid string, params *BillListParams, opts ...RequestOption) iter.Seq2[Bill, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Bill], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.ListBills(ctx, uuid, &page, opts...)
	})
}

// GetBill returns one bill by its id (GET /bills/:uuid; sub-hist@… or the bare UUID).
// params.IncludeMembers reads a member's from the organization's space; params may be nil.
func (s *SubscriptionsService) GetBill(ctx context.Context, billUUID string, params *ReadParams, opts ...RequestOption) (*Bill, error) {
	if err := checkPathTxID("billUuid", billUUID); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Bill](ctx, endpoint{method: http.MethodGet, path: pathf("/bills/%s", billUUID), query: params.values()}, opts)
}

// Cancel cancels a subscription without its customer signing (POST /subscriptions/:uuid/cancel).
// Not retried.
//
// By default (params nil, or Immediate nil or true) it is cancelled at once (status cancelled,
// cancellationReason merchant). Immediate false stops it now (status stopped, never billed
// again) and cancels it when its paid period ends. The answer is the subscription as the
// cancellation left it; Pending is true when the API answered 202: the on-chain cancel is still
// confirming, the status is not updated yet, and subscription.statusChanged tells the end (the
// cancellation goes on whatever happens).
//
// Errors: 404 for a cancelled or another space's subscription; 409
// subscription_already_stopped_or_inactive (Immediate false on a stopped one); 400
// validation_failed on immediate (Immediate false on a trial, which has no paid period to
// finish); 409 tx_already_sent while a cancellation runs.
func (s *SubscriptionsService) Cancel(ctx context.Context, uuid string, params *CancelSubscriptionParams, opts ...RequestOption) (*SubscriptionCancellation, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	sub, status, err := s.client.callStatus[Subscription](ctx, endpoint{
		method: http.MethodPost, path: pathf("/subscriptions/%s/cancel", uuid), query: params.values(),
	}, opts)
	if err != nil {
		return nil, err
	}
	return &SubscriptionCancellation{Subscription: sub, Pending: status == http.StatusAccepted}, nil
}

// TestBill bills a test subscription now (POST /subscriptions/:uuid/test-bill) and returns where
// the test bill stands once its attempt settled (done, waiting after a failure, or still running
// after 2 minutes). Its events are a live bill's. Test mode only: a live subscription is a
// *BadRequestError (400 test_mode_only); a bill not due yet is a 409 payment_not_due. Not
// retried.
func (s *SubscriptionsService) TestBill(ctx context.Context, uuid string, opts ...RequestOption) (*TestBill, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[TestBill](ctx, endpoint{
		method: http.MethodPost, path: pathf("/subscriptions/%s/test-bill", uuid),
	}, opts)
}
