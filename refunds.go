package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's refunds (GET /refunds; newest first, page size 20 by
// default, at most 100). params.Status keeps the given statuses (RefundStatusPending: the ones
// awaiting an answer; empty: every status); params.TxUUID the refund of one transaction. From
// the organization's space the members' refunds are included by default (IncludeMembers false
// leaves them out); UserUUID reads one member's; Held filters on held transactions. params may
// be nil. Iterate walks every page.
func (s *RefundsService) List(ctx context.Context, params *RefundListParams, opts ...RequestOption) (*Page[Refund], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Refund]](ctx, endpoint{method: http.MethodGet, path: "/refunds", query: params.values()}, opts)
}

// Iterate walks every refund List returns, fetching the pages lazily and keeping the filters
// and the page size. An error ends the sequence.
func (s *RefundsService) Iterate(ctx context.Context, params *RefundListParams, opts ...RequestOption) iter.Seq2[Refund, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Refund], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Get returns a refund by its id (GET /refunds/:uuid; refund@… or the bare UUID). Another
// space's refund is a *NotFoundError.
func (s *RefundsService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*Refund, error) {
	if err := checkPathTxID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Refund](ctx, endpoint{method: http.MethodGet, path: pathf("/refunds/%s", uuid)}, opts)
}

// Create starts a refund of a payment (pay@…) or a bill (sub-hist@…) of the space
// (POST /refunds, 201). It sends an Idempotency-Key and is retried on transient failures. It
// creates a pending refund, a draft that blocks nothing: no money moves until the merchant signs
// the transfer in the dashboard, and a new refund of the same transaction replaces it.
// RefundPercent (default "100") applies to everything the customer paid, network fee included,
// rounded down.
//
// Errors: 404 for a transaction outside the space; 400 validation_failed (refundPercent, or a
// refund rounding to nothing); 409 refund_already_exists (details.refundUuid: one refund per
// transaction, whatever it became; only an unsent draft gives way) or held_funds_released (a
// held payment already released to its member); a member's key needs the members.refunds policy
// (403 policy_disabled).
func (s *RefundsService) Create(ctx context.Context, params *CreateRefundParams, opts ...RequestOption) (*Refund, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Refund](ctx, endpoint{method: http.MethodPost, path: "/refunds", body: params, idempotent: true}, opts)
}
