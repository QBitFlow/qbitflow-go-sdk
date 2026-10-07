package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's one-time payments (GET /transaction/payments; newest
// first, page size 10 by default, at most 50). A payment exists once its transaction is
// confirmed. Filters: customer, product, creation window (both bounds excluded), Refunded, and
// IncludeMembers or UserUUID (organization key without On-Behalf-Of). params may be nil.
func (s *PaymentsService) List(ctx context.Context, params *PaymentListParams, opts ...RequestOption) (*Page[Payment], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Payment]](ctx, endpoint{method: http.MethodGet, path: "/transaction/payments", query: params.values()}, opts)
}

// Iterate walks every payment List returns, fetching the pages lazily from params.Cursor (or
// the first page) and keeping the filters and the page size. An error ends the sequence.
func (s *PaymentsService) Iterate(ctx context.Context, params *PaymentListParams, opts ...RequestOption) iter.Seq2[Payment, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Payment], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// ListCombined returns one page of the combined feed of one-time payments and subscription
// bills (GET /transaction/payments/combined; newest first, page size 10 by default, at most
// 50). It takes List's filters plus Source (payment or subscriptionHistory) and
// SubscriptionUUID (one subscription's bills). params may be nil.
func (s *PaymentsService) ListCombined(ctx context.Context, params *CombinedPaymentListParams, opts ...RequestOption) (*Page[CombinedPayment], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[CombinedPayment]](ctx, endpoint{method: http.MethodGet, path: "/transaction/payments/combined", query: params.values()}, opts)
}

// IterateCombined walks every row ListCombined returns, fetching the pages lazily and keeping
// the filters and the page size. An error ends the sequence.
func (s *PaymentsService) IterateCombined(ctx context.Context, params *CombinedPaymentListParams, opts ...RequestOption) iter.Seq2[CombinedPayment, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[CombinedPayment], error) {
		p.Cursor = cursor
		return s.ListCombined(ctx, &p, opts...)
	})
}

// Get returns a one-time payment by its id (GET /transaction/payment/:uuid; pay@… or the bare
// UUID). params.IncludeMembers reads a member's payment from the organization's space; params
// may be nil. A payment not confirmed yet, or another space's, is a *NotFoundError.
func (s *PaymentsService) Get(ctx context.Context, uuid string, params *ReadParams, opts ...RequestOption) (*Payment, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Payment](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/payment/%s", uuid), query: params.values()}, opts)
}

// GetByReference returns a one-time payment by the reference given to its checkout session
// (GET /transaction/payment/reference/:reference). The reference is sent escaped.
func (s *PaymentsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Payment, error) {
	if err := checkPathRequired("reference", reference); err != nil {
		return nil, err
	}
	return s.client.callRef[Payment](ctx, endpoint{method: http.MethodGet, path: pathf("/transaction/payment/reference/%s", reference)}, opts)
}

// List returns one page of the failed payment attempts (GET /transaction/failures; newest
// first, page size 10 by default, at most 50): failed attempts never moved money. It takes the
// payment lists' filters plus Kind, Category and SubscriptionUUID. params may be nil.
func (s *FailuresService) List(ctx context.Context, params *FailureListParams, opts ...RequestOption) (*Page[Failure], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Failure]](ctx, endpoint{method: http.MethodGet, path: "/transaction/failures", query: params.values()}, opts)
}

// Iterate walks every failure List returns, fetching the pages lazily and keeping the filters
// and the page size. An error ends the sequence.
func (s *FailuresService) Iterate(ctx context.Context, params *FailureListParams, opts ...RequestOption) iter.Seq2[Failure, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Failure], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}
