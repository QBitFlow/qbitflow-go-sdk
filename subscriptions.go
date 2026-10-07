package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's subscriptions (GET /transaction/subscriptions; newest
// first, cancelled ones included, page size 20 by default, at most 100). A checkout not
// completed yet is no subscription. Filters: Status, Reference, customer, product, creation
// window, and IncludeMembers or UserUUID (organization key). params may be nil.
//
// Grant access while now < CurrentPeriodEnd, whatever the status.
func (s *SubscriptionsService) List(ctx context.Context, params *SubscriptionListParams, opts ...RequestOption) (*Page[Subscription], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Subscription]](ctx, endpoint{method: http.MethodGet, path: "/transaction/subscriptions", query: params.values()}, opts)
}

// Iterate walks every subscription List returns, fetching the pages lazily and keeping the
// filters and the page size. An error ends the sequence.
func (s *SubscriptionsService) Iterate(ctx context.Context, params *SubscriptionListParams, opts ...RequestOption) iter.Seq2[Subscription, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Subscription], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Get returns a subscription by its id (GET /transaction/subscription/:uuid; sub@… or the bare
// UUID), whatever its status (a cancelled one too). params.IncludeMembers reads a member's from
// the organization's space; params may be nil. Before its checkout completes it is a
// *NotFoundError: read the checkout session instead.
func (s *SubscriptionsService) Get(ctx context.Context, uuid string, params *ReadParams, opts ...RequestOption) (*Subscription, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Subscription](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/subscription/%s", uuid), query: params.values()}, opts)
}

// GetByReference returns a subscription by the reference given to its checkout session
// (GET /transaction/subscription/reference/subscription/:reference), whatever its status. The
// reference is sent escaped.
func (s *SubscriptionsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Subscription, error) {
	if err := checkPathRequired("reference", reference); err != nil {
		return nil, err
	}
	return s.client.callRef[Subscription](ctx, endpoint{
		method: http.MethodGet, path: pathf("/transaction/subscription/reference/subscription/%s", reference),
	}, opts)
}

// ListBills returns one page of a subscription's bills (GET /transaction/subscription/:uuid/bills;
// newest first, page size 20 by default, at most 100). params may be nil.
func (s *SubscriptionsService) ListBills(ctx context.Context, uuid string, params *BillListParams, opts ...RequestOption) (*Page[Bill], error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Bill]](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/subscription/%s/bills", uuid), query: params.values()}, opts)
}

// IterateBills walks every bill of a subscription, fetching the pages lazily and keeping the
// page size. An error ends the sequence.
func (s *SubscriptionsService) IterateBills(ctx context.Context, uuid string, params *BillListParams, opts ...RequestOption) iter.Seq2[Bill, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Bill], error) {
		p.Cursor = cursor
		return s.ListBills(ctx, uuid, &p, opts...)
	})
}

// GetBill returns one bill by its id (GET /transaction/subscription/bill/:uuid; sub-hist@… or
// the bare UUID). params.IncludeMembers reads a member's from the organization's space; params
// may be nil.
func (s *SubscriptionsService) GetBill(ctx context.Context, billUUID string, params *ReadParams, opts ...RequestOption) (*Bill, error) {
	if err := checkPathTxID("billUuid", billUUID); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Bill](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/subscription/bill/%s", billUUID), query: params.values()}, opts)
}

// GetPublicHistory returns a subscription's 10 most recent bills, newest first, as its
// customer's page shows them (GET /transaction/subscription/history/:subscriptionUuid, a
// public route; not paginated). The fields the API only returns to the subscription's owner
// (Metadata, CustomerUUID, CustomerReference, UserUUID, …) are empty here: use ListBills for
// the full bills.
func (s *SubscriptionsService) GetPublicHistory(ctx context.Context, subscriptionUUID string, opts ...RequestOption) ([]Bill, error) {
	if err := checkPathTxID("subscriptionUuid", subscriptionUUID); err != nil {
		return nil, err
	}
	return s.client.call[[]Bill](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/subscription/history/%s", subscriptionUUID)}, opts)
}

// Cancel cancels a subscription without its customer signing
// (POST /transaction/subscription/processing/force-cancel/:uuid). Not retried.
//
// By default (params nil, or Immediate nil or true) it is cancelled at once (status cancelled,
// cancellationReason merchant). Immediate false stops it now (status stopped, never billed
// again) and cancels it at the end of its paid period. The answer is the subscription as the
// cancellation left it; Pending is true when the API answered 202: the on-chain cancel is still
// confirming, the status is not updated yet, and subscription.statusChanged tells the end (the
// cancellation goes on whatever happens).
//
// Errors: 404 for a cancelled or another space's subscription; 409
// subscription_already_stopped_or_inactive (Immediate false on a stopped one); 400 on
// immediate for a trial with Immediate false.
func (s *SubscriptionsService) Cancel(ctx context.Context, uuid string, params *CancelSubscriptionParams, opts ...RequestOption) (*SubscriptionCancellation, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	sub, status, err := s.client.callStatus[Subscription](ctx, endpoint{
		method: http.MethodPost, path: pathf("/transaction/subscription/processing/force-cancel/%s", uuid), query: params.values(),
	}, opts)
	if err != nil {
		return nil, err
	}
	return &SubscriptionCancellation{Subscription: sub, Pending: status == http.StatusAccepted}, nil
}

// ExecuteTestBilling bills a test subscription now
// (POST /transaction/subscription/processing/execute-billing/:uuid) and returns the test bill's
// billing state. Test mode only (a live subscription is refused with a 400). Not retried.
func (s *SubscriptionsService) ExecuteTestBilling(ctx context.Context, uuid string, opts ...RequestOption) (*BillingState, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[BillingState](ctx, endpoint{
		method: http.MethodPost, path: pathf("/transaction/subscription/processing/execute-billing/%s", uuid),
	}, opts)
}
