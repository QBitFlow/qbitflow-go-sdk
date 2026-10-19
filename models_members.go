package qbitflow

import (
	"encoding/json"
	"net/url"
	"time"
)

// Me is what the API key is, and where the request acts (Client.Me).
type Me struct {
	// Credential is apiKey for an API key.
	Credential Credential `json:"credential"`
	// APIKeyUUID is the key's id.
	APIKeyUUID string `json:"apiKeyUuid,omitempty"`
	// UserUUID is who signed in, for a session; "" for an API key (a key acts for its space, not
	// for the person who created it).
	UserUUID string `json:"userUuid,omitempty"`
	// Role is what the request may do in the space: admin (an organization key) or user (a
	// member's key, or an organization key acting On-Behalf-Of a member).
	Role Role `json:"role,omitempty"`
	// OnBehalfOfUserUUID is the member an organization key acts as (On-Behalf-Of): the space is
	// that member's. "" otherwise.
	OnBehalfOfUserUUID string `json:"onBehalfOfUserUuid,omitempty"`
	// Space is the space the request reads and writes in (Space.Test says the mode).
	Space *MeSpace `json:"space,omitempty"`
}

// MeSpace is the space a request acts in: one organization's data in one mode, the
// organization's shared space or a member's personal space.
type MeSpace struct {
	// UUID is the space's id.
	UUID string `json:"uuid"`
	// OrganizationUUID is its organization.
	OrganizationUUID string `json:"organizationUuid"`
	// OrganizationName is its organization's name.
	OrganizationName string `json:"organizationName"`
	// UserUUID is the member whose space it is; "" for the organization's own space.
	UserUUID string `json:"userUuid,omitempty"`
	// Test is the space's mode: true for test, false for live.
	Test bool `json:"test"`
	// Role is the caller's own role in the organization: owner, admin, user or handle.
	Role Role `json:"role"`
	// Member names the member whose space it is; nil for the organization's own space.
	Member *MeMember `json:"member,omitempty"`
	// OrganizationFeePercent is the organization's fee on the member's payments in this space, a
	// percent string ("0": none); "" (absent) for the organization's own space.
	OrganizationFeePercent string `json:"organizationFeePercent,omitempty"`
	// TrustedAt is when the organization started paying the member directly; nil while it holds
	// their funds (always set on the organization's own space).
	TrustedAt *time.Time `json:"trustedAt,omitempty"`
	// CreatedAt is when the space was created.
	CreatedAt time.Time `json:"createdAt"`
}

// MeMember names the member whose space a request acts in.
type MeMember struct {
	// UserUUID is the member (as Members.List lists them).
	UserUUID string `json:"userUuid"`
	// FirstName is the first name.
	FirstName string `json:"firstName"`
	// LastName is the last name; "" (absent) when the account has none.
	LastName string `json:"lastName,omitempty"`
	// Email is the email.
	Email string `json:"email"`
}

// Member is a member of the organization (a seller of a marketplace), in one mode: their
// personal space there.
type Member struct {
	// UserUUID is the member's id: what On-Behalf-Of (WithOnBehalfOf, Client.OnBehalfOf) takes.
	UserUUID string `json:"userUuid"`
	// FirstName is the first name.
	FirstName string `json:"firstName"`
	// LastName is the last name; "" (absent) when the account has none.
	LastName string `json:"lastName,omitempty"`
	// Email is the email.
	Email string `json:"email"`
	// Test is the mode.
	Test bool `json:"test"`
	// OrganizationFeePercent is the organization's fee on the member's payments in this mode, a
	// percent string ("0": none).
	OrganizationFeePercent string `json:"organizationFeePercent"`
	// TrustedAt is when the organization started paying them directly; nil (absent) while it
	// holds their funds (held funds).
	TrustedAt *time.Time `json:"trustedAt,omitempty"`
	// JoinedAt is when they joined in this mode.
	JoinedAt time.Time `json:"joinedAt"`
	// AcceptedCurrencies are the currencies a checkout of the member can be paid in now: those
	// their wallets accept, always within the organization's (the organization's own while it
	// holds their funds). Empty: a checkout of theirs answers 409 merchant_not_ready.
	AcceptedCurrencies []Currency `json:"acceptedCurrencies"`
	// SpaceUUID is their personal space in this mode.
	SpaceUUID string `json:"spaceUuid"`
}

// HeldFunds is what the organization holds for a member in one mode (held funds).
type HeldFunds struct {
	// Ledgers are the lines not settled yet, oldest first: the payments and bills held
	// (positive), and the refunds sent of them (negative, type refund). Empty when nothing is held.
	Ledgers []LedgerEntry `json:"ledgers"`
	// TotalUSD is what the organization owes the member, in USD, a decimal string: the sum of the
	// lines' OwedUSD. A release pays it, converted at the release's price. "0" or negative when
	// refunds cancel out the payments (nothing to send: the held funds are closed instead).
	TotalUSD string `json:"totalUsd"`
}

// LedgerEntry is a line of a member's held funds: a payment or a bill made to them while the
// organization held their funds (it adds what the organization owes them), or a refund of one
// of them (it takes off what the refund sent back to the customer).
type LedgerEntry struct {
	// TxUUID is the line's transaction: the payment (pay@…), the bill (sub-hist@…) or the refund
	// (refund@…).
	TxUUID string `json:"txUuid"`
	// Type is payment, bill or refund.
	Type LedgerEntryType `json:"type"`
	// Description says what the line is, in words ("Payment pay@…").
	Description string `json:"description"`
	// RefundedTxUUID is, on a refund line, the payment or bill it refunds; "" on the others.
	RefundedTxUUID string `json:"refundedTxUuid,omitempty"`
	// AmountUSD is what the customer paid (a payment or bill), or what a refund sent back
	// (positive: OwedUSD is its negative), in USD, a decimal string.
	AmountUSD string `json:"amountUsd"`
	// Settlement is a payment's or bill's fees and split (the member was owed
	// Settlement.Split.Merchant); nil on a refund line.
	Settlement *Settlement `json:"settlement,omitempty"`
	// Currency is the currency paid (and refunded).
	Currency Currency `json:"currency"`
	// Owed is what the line counts for in a release, in the currency's units, exact: a payment's
	// or bill's merchant share, or minus what a refund sent back (negative). In
	// heldFunds.released: what it counted for in that release.
	Owed string `json:"owed"`
	// OwedMinUnits is the same in the currency's smallest unit (negative for a refund).
	OwedMinUnits string `json:"owedMinUnits"`
	// OwedUSD is the same in USD, at the payment's time (negative for a refund).
	OwedUSD string `json:"owedUsd"`
	// CreatedAt is when the line was recorded.
	CreatedAt time.Time `json:"createdAt"`
}

// MemberHeldFunds is what the organization owes one member (Members.ListHeldFunds).
type MemberHeldFunds struct {
	// UserUUID is the member (a removed member still owed too).
	UserUUID string `json:"userUuid"`
	// TotalUSD is what the organization owes them, in USD, a decimal string ("0" or negative when
	// refunds cancel out the payments).
	TotalUSD string `json:"totalUsd"`
	// Count is their lines not settled: payments and bills held, refunds of them.
	Count int `json:"count"`
	// OldestAt is their oldest line not settled: how long the organization has held funds for them.
	OldestAt time.Time `json:"oldestAt"`
}

// UpdateMemberParams changes a member's terms in the key's mode (Members.Update).
type UpdateMemberParams struct {
	// OrganizationFeePercent is required: the organization's fee on the member's payments from
	// now on, a percent string from "0" to "50" with at most 2 decimals. A checkout already
	// created keeps its fee; a running subscription's next bills take the new one.
	OrganizationFeePercent string `json:"organizationFeePercent"`
}

// Validate checks the params (behaviour §8).
func (p *UpdateMemberParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("organizationFeePercent", p.OrganizationFeePercent) {
		v.percent("organizationFeePercent", p.OrganizationFeePercent, 50, false)
	}
	return v.err()
}

// MemberListParams pages Members.List.
type MemberListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
}

// Validate checks the params.
func (p *MemberListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	return v.err()
}

func (p *MemberListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	return q
}

// Invitation is an invitation to the organization: to be a member in one mode (role user), or
// to its team (role admin, from the dashboard: mode-free, so Test, HoldFunds and
// OrganizationFeePercent are absent).
type Invitation struct {
	// UUID is the invitation's id.
	UUID string `json:"uuid"`
	// Test is a member invitation's mode; nil (absent) for a team invitation.
	Test *bool `json:"test,omitempty"`
	// Email is the invited address.
	Email string `json:"email"`
	// Role is admin (the team) or user (a member).
	Role Role `json:"role"`
	// HoldFunds is, for a member, true when the organization holds their payments until it trusts
	// them (Members.Trust), false when they are paid directly; nil (absent) for the team.
	HoldFunds *bool `json:"holdFunds,omitempty"`
	// OrganizationFeePercent is, for a member, the organization's fee on their payments, a percent
	// string ("0": none); "" (absent) for the team.
	OrganizationFeePercent string `json:"organizationFeePercent,omitempty"`
	// RedirectURL is where the person goes once they accepted (with ?invitationUuid=<uuid>); ""
	// when none.
	RedirectURL string `json:"redirectUrl,omitempty"`
	// ExpiresAt is when it expires.
	ExpiresAt time.Time `json:"expiresAt"`
	// AcceptedByUserUUID is the account that accepted it; "" until accepted.
	AcceptedByUserUUID string `json:"acceptedByUserUuid,omitempty"`
	// AcceptedAt is when it was accepted; nil until then.
	AcceptedAt *time.Time `json:"acceptedAt,omitempty"`
	// RevokedAt is when it was revoked (or replaced by a newer one); nil unless revoked.
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// Status is pending, accepted, revoked or expired (computed when read).
	Status InvitationStatus `json:"status"`
}

// CreatedInvitation is an invitation just created (Invitations.Create): the invitation's fields
// and its link, only returned here.
type CreatedInvitation struct {
	Invitation

	// Link is the link to accept it, also emailed to the person: share it in your own onboarding
	// if you prefer.
	Link string `json:"link"`
}

// CreateInvitationParams invites a member (Invitations.Create). The SDK always sends role
// "user": team invitations are made from the dashboard.
type CreateInvitationParams struct {
	// Email is required: the person to invite, at most 254 characters.
	Email string
	// HoldFunds is required (always sent): true holds the member's payments until the
	// organization trusts them (Members.Trust); false pays them directly.
	HoldFunds bool
	// OrganizationFeePercent is the organization's fee on their payments, a percent string from
	// "0" to "50" with at most 2 decimals ("" = "0").
	OrganizationFeePercent string
	// RedirectURL is where the person goes once they accepted (with ?invitationUuid=<uuid>): a
	// convenience, confirm with the member.joined event.
	RedirectURL string
}

// MarshalJSON writes the invitation's body, with role "user".
func (p CreateInvitationParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Email                  string `json:"email"`
		Role                   Role   `json:"role"`
		HoldFunds              bool   `json:"holdFunds"`
		OrganizationFeePercent string `json:"organizationFeePercent,omitempty"`
		RedirectURL            string `json:"redirectUrl,omitempty"`
	}{p.Email, RoleUser, p.HoldFunds, p.OrganizationFeePercent, p.RedirectURL})
}

// Validate checks the params (behaviour §8).
func (p *CreateInvitationParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("email", p.Email) {
		v.email("email", p.Email)
	}
	if p.OrganizationFeePercent != "" {
		v.percent("organizationFeePercent", p.OrganizationFeePercent, 50, false)
	}
	v.url("redirectUrl", p.RedirectURL)
	return v.err()
}

// InvitationListParams filters Invitations.List.
type InvitationListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
	// Status keeps only the invitations with this status (pending, accepted, revoked, expired).
	Status InvitationStatus
}

// Validate checks the filters.
func (p *InvitationListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	oneOf(&v, "status", p.Status, InvitationStatusPending, InvitationStatusAccepted, InvitationStatusRevoked, InvitationStatusExpired)
	return v.err()
}

func (p *InvitationListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	setString(q, "status", string(p.Status))
	return q
}
