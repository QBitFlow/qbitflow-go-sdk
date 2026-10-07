package qbitflow

import (
	"encoding/json"
	"net/url"
	"time"
)

// Me is what the API key is (Client.Me).
type Me struct {
	// Credential is apiKey for an API key.
	Credential Credential `json:"credential"`
	// APIKeyUUID is the key's id.
	APIKeyUUID string `json:"apiKeyUuid,omitempty"`
	// UserUUID is who signed in, for a session ("" for an API key).
	UserUUID string `json:"userUuid,omitempty"`
	// Role is admin (organization key) or user (a member's key, or On-Behalf-Of).
	Role Role `json:"role,omitempty"`
	// OnBehalfOf is the member an organization key acts as (On-Behalf-Of); "" otherwise.
	OnBehalfOf string `json:"onBehalfOf,omitempty"`
	// Space is the space the request reads and writes in.
	Space *MeSpace `json:"space,omitempty"`
}

// MeSpace is the space a request acts in.
type MeSpace struct {
	// UUID is the space's id.
	UUID string `json:"uuid"`
	// OrganizationUUID is its organization.
	OrganizationUUID string `json:"organizationUuid"`
	// OrganizationName is its organization's name.
	OrganizationName string `json:"organizationName"`
	// UserUUID is the member whose space it is; "" for the organization's own space.
	UserUUID string `json:"userUuid,omitempty"`
	// Member names that member; nil for the organization's own space.
	Member *MeMember `json:"member,omitempty"`
	// Test is the space's mode: true for test, false for live.
	Test bool `json:"test"`
}

// MeMember names the member whose space a request acts in.
type MeMember struct {
	// UserUUID is the member (as Members.List lists them).
	UserUUID string `json:"userUuid"`
	// Name is the first name.
	Name string `json:"name"`
	// LastName is the last name.
	LastName string `json:"lastName"`
	// Email is the email.
	Email string `json:"email"`
}

// Member is a member of the organization (a seller of a marketplace), in one mode.
type Member struct {
	// UserUUID is the member's id: what On-Behalf-Of takes.
	UserUUID string `json:"userUuid"`
	// Name is the first name.
	Name string `json:"name"`
	// LastName is the last name.
	LastName string `json:"lastName"`
	// Email is the email.
	Email string `json:"email"`
	// Test is the mode.
	Test bool `json:"test"`
	// OrganizationFeePercent is the organization's fee on the member's payments, in percent.
	OrganizationFeePercent float64 `json:"organizationFeePercent"`
	// TrustedAt is when the organization started paying them directly; nil while it holds their
	// funds (trust layer).
	TrustedAt *time.Time `json:"trustedAt"`
	// JoinedAt is when they joined in this mode.
	JoinedAt time.Time `json:"joinedAt"`
	// AcceptedCurrencyIDs are the currencies their wallets accept.
	AcceptedCurrencyIDs []uint64 `json:"acceptedCurrencyIds"`
	// SpaceUUID is their personal space in this mode.
	SpaceUUID string `json:"spaceUuid"`
}

// HeldFunds is what the organization holds for a member (trust layer).
type HeldFunds struct {
	// Ledgers are the lines not settled yet, oldest first: payments and bills held (positive),
	// and the refunds sent of them (negative).
	Ledgers []LedgerEntry `json:"ledgers"`
	// TotalAmount is what the organization owes the member, in USD (may be 0 or less).
	TotalAmount float64 `json:"totalAmount"`
}

// LedgerEntry is one held-funds line.
type LedgerEntry struct {
	// TxUUID is the line's transaction: pay@…, sub-hist@… or refund@….
	TxUUID string `json:"txUuid"`
	// Type is payment, subscriptionHistory or refund.
	Type LedgerEntryType `json:"type"`
	// Description says what the line is, in words.
	Description string `json:"description"`
	// RefundedTxUUID is, on a refund line, the payment or bill it refunds.
	RefundedTxUUID string `json:"refundedTxUuid,omitempty"`
	// Amount is what the customer paid (a payment or bill), or what a refund sent back, in USD.
	Amount float64 `json:"amount"`
	// Metadata is a payment's or bill's fees and split; nil on refund lines.
	Metadata *PaymentMetadata `json:"metadata"`
	// CurrencyID is the currency paid (and refunded).
	CurrencyID uint64 `json:"currencyId"`
	// OwedMinUnits is what the line counts for in a release, in min units (a decimal string,
	// negative on refunds).
	OwedMinUnits string `json:"owedMinUnits"`
	// OwedUSD is the same in USD.
	OwedUSD float64 `json:"owedUsd"`
	// CreatedAt is when the line was recorded.
	CreatedAt time.Time `json:"createdAt"`
}

// MemberHeldFundsSummary is one member's held funds, in the organization's overview.
type MemberHeldFundsSummary struct {
	// UserUUID is the member (a removed member still owed too).
	UserUUID string `json:"userUuid"`
	// TotalAmount is what the organization owes them, in USD.
	TotalAmount float64 `json:"totalAmount"`
	// Count is their lines not settled.
	Count int `json:"count"`
	// OldestAt is their oldest line not settled.
	OldestAt time.Time `json:"oldestAt"`
}

// UpdateMemberParams changes a member's terms (Members.Update).
type UpdateMemberParams struct {
	// OrganizationFeePercent is the organization's fee on the member's payments from now on:
	// 0 to 50, at most 2 decimals. Always sent.
	OrganizationFeePercent float64 `json:"organizationFeePercent"`
}

// Validate checks the params (behaviour §8).
func (p *UpdateMemberParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.percent("organizationFeePercent", p.OrganizationFeePercent, 50, false)
	return v.err()
}

// MemberListParams pages Members.List.
type MemberListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor (a member's userUuid); "" for the first page.
	Cursor string
}

// Validate checks the params (nothing to check: present for uniformity).
func (p *MemberListParams) Validate() error { return nil }

func (p *MemberListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	return q
}

// Invitation is an invitation to join the organization.
type Invitation struct {
	// UUID is the invitation's id.
	UUID string `json:"uuid"`
	// Test is a member invitation's mode; nil for a team invitation.
	Test *bool `json:"test"`
	// Email is the invited address.
	Email string `json:"email"`
	// Role is admin (the team) or user (a member).
	Role Role `json:"role"`
	// TrustLayer is true when the organization holds the member's payments until it trusts
	// them; nil for the team.
	TrustLayer *bool `json:"trustLayer"`
	// OrganizationFeePercent is the organization's fee on the member's payments, in percent.
	OrganizationFeePercent float64 `json:"organizationFeePercent"`
	// RedirectURL is where the person goes once they accepted (with ?invitationUuid=<uuid>).
	RedirectURL string `json:"redirectUrl,omitempty"`
	// ExpiresAt is when it expires.
	ExpiresAt time.Time `json:"expiresAt"`
	// AcceptedByUserUUID is the account that accepted it.
	AcceptedByUserUUID string `json:"acceptedByUserUuid,omitempty"`
	// AcceptedAt is when it was accepted; nil until then.
	AcceptedAt *time.Time `json:"acceptedAt"`
	// RevokedAt is when it was revoked; nil unless revoked.
	RevokedAt *time.Time `json:"revokedAt"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// Status is pending, accepted, revoked or expired.
	Status InvitationStatus `json:"status"`
}

// InvitationCreated is Invitations.Create's answer.
type InvitationCreated struct {
	// Invitation is the invitation.
	Invitation Invitation `json:"invitation"`
	// Link is the link to accept it, also emailed to the person.
	Link string `json:"link"`
}

// CreateInvitationParams invites a member (Invitations.Create). The SDK always sends role
// "user": team invitations are made from the dashboard.
type CreateInvitationParams struct {
	// Email is required: the person to invite.
	Email string
	// TrustLayer is required (always sent): true holds the member's payments until the
	// organization trusts them (Members.Trust); false pays them directly.
	TrustLayer bool
	// OrganizationFeePercent is the organization's fee on their payments: 0 to 50, at most 2
	// decimals (nil = 0).
	OrganizationFeePercent *float64
	// RedirectURL is where the person goes once they accepted (with ?invitationUuid=<uuid>).
	RedirectURL string
}

// MarshalJSON writes the invitation's body, with role "user".
func (p CreateInvitationParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Email                  string   `json:"email"`
		Role                   Role     `json:"role"`
		TrustLayer             bool     `json:"trustLayer"`
		OrganizationFeePercent *float64 `json:"organizationFeePercent,omitempty"`
		RedirectURL            string   `json:"redirectUrl,omitempty"`
	}{p.Email, RoleUser, p.TrustLayer, p.OrganizationFeePercent, p.RedirectURL})
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
	if p.OrganizationFeePercent != nil {
		v.percent("organizationFeePercent", *p.OrganizationFeePercent, 50, false)
	}
	v.url("redirectUrl", p.RedirectURL)
	return v.err()
}

// InvitationListParams filters Invitations.List.
type InvitationListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor; "" for the first page.
	Cursor string
	// Status keeps only the invitations with this status.
	Status InvitationStatus
}

// Validate checks the filters.
func (p *InvitationListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
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
