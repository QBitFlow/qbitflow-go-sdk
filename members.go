package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the organization's members in the key's mode (GET /members; by
// user UUID, page size 20 by default, at most 100). Organization key only (403 otherwise, also
// with On-Behalf-Of). params may be nil.
func (s *MembersService) List(ctx context.Context, params *MemberListParams, opts ...RequestOption) (*Page[Member], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Member]](ctx, endpoint{method: http.MethodGet, path: "/members", query: params.values()}, opts)
}

// Iterate walks every member List returns, fetching the pages lazily and keeping the page size.
// An error ends the sequence. Organization key only.
func (s *MembersService) Iterate(ctx context.Context, params *MemberListParams, opts ...RequestOption) iter.Seq2[Member, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Member], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Get returns a member by their user UUID (GET /members/:userUuid; Member.UserUUID, the value
// WithOnBehalfOf takes). Organization key only. Someone who is not a member in the key's mode
// is a *NotFoundError.
func (s *MembersService) Get(ctx context.Context, userUUID string, opts ...RequestOption) (*Member, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	return s.client.callRef[Member](ctx, endpoint{method: http.MethodGet, path: pathf("/members/%s", userUUID)}, opts)
}

// Update changes a member's organization fee in the key's mode (PUT /members/:userUuid). A
// checkout already created keeps the fee it was created with. Organization key only. Not
// retried.
func (s *MembersService) Update(ctx context.Context, userUUID string, params *UpdateMemberParams, opts ...RequestOption) (*Member, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Member](ctx, endpoint{method: http.MethodPut, path: pathf("/members/%s", userUUID), body: params}, opts)
}

// Remove removes a member from the key's mode (DELETE /members/:userUuid): their API keys stop
// working at once and their checkouts stop taking payments; their data stays with the
// organization. Organization key only. Not retried.
//
// Errors: 409 held_funds_pending while the organization holds live funds for them (release
// them in the dashboard first).
func (s *MembersService) Remove(ctx context.Context, userUUID string, opts ...RequestOption) error {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/members/%s", userUUID)}, opts)
}

// Trust makes a member's new payments in the key's mode go to their own wallets
// (POST /members/:userUuid/trust) and returns the member with TrustedAt set. What the
// organization already holds for them stays held until released (in the dashboard).
// Organization key only. Not retried.
func (s *MembersService) Trust(ctx context.Context, userUUID string, opts ...RequestOption) (*Member, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	return s.client.callRef[Member](ctx, endpoint{method: http.MethodPost, path: pathf("/members/%s/trust", userUUID)}, opts)
}

// ListHeldFunds returns what the organization holds for each member in the key's mode
// (GET /members/held-funds). Organization key only.
func (s *MembersService) ListHeldFunds(ctx context.Context, opts ...RequestOption) ([]MemberHeldFundsSummary, error) {
	return s.client.call[[]MemberHeldFundsSummary](ctx, endpoint{method: http.MethodGet, path: "/members/held-funds"}, opts)
}

// GetHeldFunds returns what the organization holds for one member in the key's mode, a
// removed member's too (GET /members/:userUuid/held-funds): the lines not paid out yet, oldest
// first, and their total in USD. Organization key only.
func (s *MembersService) GetHeldFunds(ctx context.Context, userUUID string, opts ...RequestOption) (*HeldFunds, error) {
	if err := checkPathUUID("userUuid", userUUID); err != nil {
		return nil, err
	}
	return s.client.callRef[HeldFunds](ctx, endpoint{method: http.MethodGet, path: pathf("/members/%s/held-funds", userUUID)}, opts)
}

// GetOwnHeldFunds returns what the organization holds for the request's space
// (GET /user/held-funds): call it with a member's key, or OnBehalfOf a member. Empty for the
// organization's own space.
func (s *MembersService) GetOwnHeldFunds(ctx context.Context, opts ...RequestOption) (*HeldFunds, error) {
	return s.client.callRef[HeldFunds](ctx, endpoint{method: http.MethodGet, path: "/user/held-funds"}, opts)
}

// Create invites someone to join the organization as a member in the key's mode
// (POST /invitations, 201) and returns the invitation and its link (also emailed). The SDK
// always sends role "user": the team is invited from the dashboard. It sends an
// Idempotency-Key and is retried on transient failures. Organization key only; rate limited
// (50 an hour per organization).
//
// The person exists in the organization once they accepted: act on the member.joined webhook
// (MemberJoined.InvitationUUID). Errors: 409 already_joined for someone already a member in
// that mode.
func (s *InvitationsService) Create(ctx context.Context, params *CreateInvitationParams, opts ...RequestOption) (*InvitationCreated, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[InvitationCreated](ctx, endpoint{method: http.MethodPost, path: "/invitations", body: params, idempotent: true}, opts)
}

// List returns one page of the organization's invitations (GET /invitations; page size 20 by
// default, at most 100), optionally filtered by Status. Organization key only. params may be
// nil.
func (s *InvitationsService) List(ctx context.Context, params *InvitationListParams, opts ...RequestOption) (*Page[Invitation], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Invitation]](ctx, endpoint{method: http.MethodGet, path: "/invitations", query: params.values()}, opts)
}

// Iterate walks every invitation List returns, fetching the pages lazily and keeping the filter
// and the page size. An error ends the sequence. Organization key only.
func (s *InvitationsService) Iterate(ctx context.Context, params *InvitationListParams, opts ...RequestOption) iter.Seq2[Invitation, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Invitation], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Revoke revokes an invitation (DELETE /invitations/:uuid) and returns it. Organization key
// only. Not retried.
func (s *InvitationsService) Revoke(ctx context.Context, uuid string, opts ...RequestOption) (*Invitation, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Invitation](ctx, endpoint{method: http.MethodDelete, path: pathf("/invitations/%s", uuid)}, opts)
}
