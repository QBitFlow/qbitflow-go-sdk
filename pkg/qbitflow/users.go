package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// UserService manages the users of your organization. Most operations require an
// admin-level credential; Get (self) and Update (self) work for any user.
type UserService struct {
	client *Client
}

// NewUserService creates the user service bound to client.
func NewUserService(client *Client) *UserService {
	return &UserService{client: client}
}

// OnBehalfOf returns a UserService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *UserService) OnBehalfOf(userID uint64) *UserService {
	return NewUserService(s.client.withOnBehalfOf(userID))
}

// CreateUser holds the fields required to provision a user in your organization
// (admin only). The created user is unclaimed until they follow a claim link; any
// password in the body would be ignored by the API, so none is accepted here.
type CreateUser struct {
	Name     string `json:"name"`     // 2-100 chars, letters/digits/spaces and - _ ' .
	LastName string `json:"lastName"` // 2-100 chars, same character set
	Email    string `json:"email"`    // must be a valid email, unique within the organization
	// Role must be RoleAdmin or RoleUser (untyped string constants "admin"/"user" are
	// accepted). RoleHandle and RoleOwner exist on responses but cannot be assigned.
	Role qbmodels.UserRole `json:"role"`
	// OrganizationFeeBps is the fee the organization keeps on this user's payments, in
	// basis points (100 bps = 1%), 0-5000. Optional; defaults to 0.
	OrganizationFeeBps uint16 `json:"organizationFeeBps"`
}

// Validate mirrors the API's binding rules for a new user: name and last name are
// `alphanumspace` (2-100 characters), the email must be well-formed, the role must be
// admin or user, and the organization fee must not exceed 5000 bps. It is called by
// Create, so invalid input is rejected before the round-trip.
func (u *CreateUser) Validate() error {
	if err := requireArgument(u == nil, "User"); err != nil {
		return err
	}
	if err := validateAlphanumSpace("Name", u.Name, 2, 100); err != nil {
		return err
	}
	if err := validateAlphanumSpace("LastName", u.LastName, 2, 100); err != nil {
		return err
	}
	if err := requireNonEmpty(u.Email, "Email"); err != nil {
		return err
	}
	if err := validateEmail("Email", u.Email); err != nil {
		return err
	}
	if err := validateRole(u.Role); err != nil {
		return err
	}
	return validateFeeBps(u.OrganizationFeeBps)
}

// UpdateUser holds the fields that can be changed on an existing user.
//
// Every field is optional and the update is PARTIAL: a nil field — or a pointer to "" for
// the string fields — is left out of the request and the stored value is left unchanged.
// An entirely empty UpdateUser is a valid no-op.
//
// NOTE: password is deliberately absent. Changing a password is a JWT-only, self-service
// operation on the API — it cannot be done with an API key, which is the only credential
// this SDK uses. A password sent with an API key is silently ignored by the API (the
// request still succeeds), so exposing it here would be misleading. Change passwords
// from the QBitFlow dashboard instead.
type UpdateUser struct {
	Name     *string `json:"name,omitempty"`     // 2-100 chars, alphanumspace
	LastName *string `json:"lastName,omitempty"` // 2-100 chars, alphanumspace
	Email    *string `json:"email,omitempty"`    // must be a valid email, unique within the organization

	// OrganizationFeeBps requires admin authority: an admin/owner key acting directly, or
	// an organization-level key acting via OnBehalfOf. A non-admin caller that sends this
	// field is rejected with 403. Nil omits it entirely; a pointer to 0 explicitly sets 0.
	OrganizationFeeBps *uint16 `json:"organizationFeeBps,omitempty"` // 0-5000 bps (100 bps = 1%)
}

// Validate checks the fields that are present. Absent (nil) or empty fields are not
// validated, since they are not sent and therefore leave the stored value untouched.
func (u *UpdateUser) Validate() error {
	if err := requireArgument(u == nil, "User update"); err != nil {
		return err
	}
	if isSet(u.Name) {
		if err := validateAlphanumSpace("Name", *u.Name, 2, 100); err != nil {
			return err
		}
	}
	if isSet(u.LastName) {
		if err := validateAlphanumSpace("LastName", *u.LastName, 2, 100); err != nil {
			return err
		}
	}
	if isSet(u.Email) {
		if err := validateEmail("Email", *u.Email); err != nil {
			return err
		}
	}
	if u.OrganizationFeeBps != nil {
		if err := validateFeeBps(*u.OrganizationFeeBps); err != nil {
			return err
		}
	}
	return nil
}

// Create provisions a new user in the organization (admin only). The user is created
// unclaimed; see ClaimService to send them a claim link.
func (s *UserService) Create(user *CreateUser) (*qbmodels.User, error) {
	if err := user.Validate(); err != nil {
		return nil, err
	}

	var result qbmodels.User
	err := s.client.makeRequest("POST", "/user/", user, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get returns the current user — the one the API key belongs to, or the acted-for user
// when called through OnBehalfOf.
func (s *UserService) Get() (*qbmodels.User, error) {
	var result qbmodels.User
	err := s.client.makeRequest("GET", "/user/", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByID retrieves a user by their ID (must be an admin to access other users within the organization)
func (s *UserService) GetByID(userID uint64) (*qbmodels.User, error) {
	if err := requirePositive(userID, "User ID"); err != nil {
		return nil, err
	}

	var result qbmodels.User
	endpoint := "/user/id/" + fmt.Sprint(userID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByEmail retrieves a user by their email address (must be an admin to access other users within the organization)
func (s *UserService) GetByEmail(email string) (*qbmodels.User, error) {
	if err := requireNonEmpty(email, "User email"); err != nil {
		return nil, err
	}
	if err := validateEmail("User email", email); err != nil {
		return nil, err
	}

	var result qbmodels.User
	endpoint := "/user/email/" + escapePath(email)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves all users in the organization (must be an admin)
func (s *UserService) GetAll() ([]qbmodels.User, error) {
	return getList[qbmodels.User](s.client, "/user/all")
}

// Update partially updates a user. A regular user may update only themselves; an admin
// may update any non-owner user in the organization (see UpdateUser for field rules).
func (s *UserService) Update(userID uint64, user *UpdateUser) (*qbmodels.User, error) {
	if err := requirePositive(userID, "User ID"); err != nil {
		return nil, err
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}

	body := &UpdateUser{
		Name:               omitEmpty(user.Name),
		LastName:           omitEmpty(user.LastName),
		Email:              omitEmpty(user.Email),
		OrganizationFeeBps: user.OrganizationFeeBps,
	}

	var result qbmodels.User
	endpoint := "/user/" + fmt.Sprint(userID)
	err := s.client.makeRequest("PUT", endpoint, body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete soft-deletes a user (admin only). The organization owner cannot be deleted.
// Deleting a user also removes their API keys and wallets.
//
// It returns only an error, the idiomatic Go shape; the API's confirmation message carries
// no information beyond success.
func (s *UserService) Delete(userID uint64) error {
	if err := requirePositive(userID, "User ID"); err != nil {
		return err
	}

	endpoint := "/user/" + fmt.Sprint(userID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
