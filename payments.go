package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's one-time payments (GET /payments; newest first, page
// size 20 by default, at most 100). A payment exists once its transaction is confirmed.
// Filters: customer, product, creation window (both bounds excluded), Reference, Refunded, and
// IncludeMembers or UserUUID (organization key without On-Behalf-Of). params may be nil.
func (s *PaymentsService) List(ctx context.Context, params *PaymentListParams, opts ...RequestOption) (*Page[Payment], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Payment]](ctx, endpoint{method: http.MethodGet, path: "/payments", query: params.values()}, opts)
}

// Iterate walks every payment List returns, fetching the pages lazily from params.Cursor (or
// the first page) and keeping the filters and the page size. An error ends the sequence.
func (s *PaymentsService) Iterate(ctx context.Context, params *PaymentListParams, opts ...RequestOption) iter.Seq2[Payment, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Payment], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.List(ctx, &page, opts...)
	})
}

// Get returns a one-time payment by its id (GET /payments/:uuid; pay@… or the bare UUID).
// params.IncludeMembers reads a member's payment from the organization's space; params may be
// nil. A payment not confirmed yet (read its checkout session), or another space's, is a
// *NotFoundError.
func (s *PaymentsService) Get(ctx context.Context, uuid string, params *ReadParams, opts ...RequestOption) (*Payment, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Payment](ctx, endpoint{method: http.MethodGet, path: pathf("/payments/%s", uuid), query: params.values()}, opts)
}

// GetByReference returns the space's one-time payment with the reference given to its checkout
// session: it runs List's reference filter (GET /payments?reference=) and returns its one
// payment. No payment with that reference (none, or not confirmed yet) is a *NotFoundError
// (status 404, code not_found).
func (s *PaymentsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Payment, error) {
	return s.client.getByReference[Payment](ctx, "/payments", "payment", reference, func(v *Payment) string { return v.Reference }, opts)
}

// List returns one page of the transactions feed (GET /transactions; newest first, page size
// 20 by default, at most 100): the one-time payments and the subscriptions' bills together,
// each row with its Type (payment or bill). It takes the payment lists' filters plus Type,
// SubscriptionUUID (one subscription's bills) and Refunded. params may be nil.
func (s *TransactionsService) List(ctx context.Context, params *TransactionListParams, opts ...RequestOption) (*Page[Transaction], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Transaction]](ctx, endpoint{method: http.MethodGet, path: "/transactions", query: params.values()}, opts)
}

// Iterate walks every row List returns, fetching the pages lazily and keeping the filters and
// the page size. An error ends the sequence.
func (s *TransactionsService) Iterate(ctx context.Context, params *TransactionListParams, opts ...RequestOption) iter.Seq2[Transaction, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Transaction], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.List(ctx, &page, opts...)
	})
}

// List returns one page of the failed payment attempts (GET /payment-failures; newest first,
// page size 20 by default, at most 100): failed attempts never moved money. It takes the
// payment lists' filters plus Kind, Category and SubscriptionUUID. params may be nil.
func (s *FailuresService) List(ctx context.Context, params *FailureListParams, opts ...RequestOption) (*Page[Failure], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Failure]](ctx, endpoint{method: http.MethodGet, path: "/payment-failures", query: params.values()}, opts)
}

// Iterate walks every failure List returns, fetching the pages lazily and keeping the filters
// and the page size. An error ends the sequence.
func (s *FailuresService) Iterate(ctx context.Context, params *FailureListParams, opts ...RequestOption) iter.Seq2[Failure, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Failure], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.List(ctx, &page, opts...)
	})
}
