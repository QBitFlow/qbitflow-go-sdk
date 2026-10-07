package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns the refunds awaiting the merchant's answer (respondedAt null)
// (GET /transaction/refunds/all; not paginated). From the
// organization's space the members' refunds are included by default (IncludeMembers false
// leaves them out); UserUUID reads one member's; Held filters on held transactions. Limit and
// Cursor are not sent. params may be nil.
func (s *RefundsService) List(ctx context.Context, params *RefundListParams, opts ...RequestOption) ([]Refund, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Refund](ctx, endpoint{method: http.MethodGet, path: "/transaction/refunds/all", query: params.values(false)}, opts)
}

// ListInactive returns one page of the answered refunds, approved or rejected (respondedAt set)
// (GET /transaction/refunds/all/inactive; page size 10 by default, at most 50), with List's
// filters. params may be nil.
func (s *RefundsService) ListInactive(ctx context.Context, params *RefundListParams, opts ...RequestOption) (*Page[Refund], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Refund]](ctx, endpoint{method: http.MethodGet, path: "/transaction/refunds/all/inactive", query: params.values(true)}, opts)
}

// IterateInactive walks every refund ListInactive returns, fetching the pages lazily and
// keeping the filters and the page size. An error ends the sequence.
func (s *RefundsService) IterateInactive(ctx context.Context, params *RefundListParams, opts ...RequestOption) iter.Seq2[Refund, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Refund], error) {
		p.Cursor = cursor
		return s.ListInactive(ctx, &p, opts...)
	})
}

// Initiate starts a refund of a payment (pay@…) or a bill (sub-hist@…) of the space
// (POST /transaction/refunds/initiate, 201). It sends an Idempotency-Key and is retried on
// transient failures. It creates a pending refund: no money moves until the merchant signs the
// transfer in the dashboard. RefundPercent (default 100) applies to everything the customer
// paid, network fee included, rounded down.
//
// Errors: 404 for a transaction outside the space; 400 validation_failed (refundPercent, or a
// refund rounding to nothing); 409 refund_already_exists (details.refundUuid: one refund per
// transaction) or held_funds_released; a member's key needs the members.refunds policy (403
// policy_disabled).
func (s *RefundsService) Initiate(ctx context.Context, params *InitiateRefundParams, opts ...RequestOption) (*Refund, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Refund](ctx, endpoint{method: http.MethodPost, path: "/transaction/refunds/initiate", body: params, idempotent: true}, opts)
}
