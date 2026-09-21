package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

type UserService struct {
	client *Client
}

func NewUserService(client *Client) *UserService {
	return &UserService{client: client}
}

// OnBehalfOf returns a UserService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *UserService) OnBehalfOf(userID uint64) *UserService {
	return NewUserService(s.client.withOnBehalfOf(userID))
}

type CreateUser struct {
	Name               string `json:"name" binding:"required"`
	LastName           string `json:"lastName" binding:"required"`
	Email              string `json:"email" binding:"required,email"`
	Role               string `json:"role" binding:"required,oneof=admin user"`              // Role can be 'admin' or 'user'
	OrganizationFeeBps uint16 `json:"organizationFeeBps" binding:"omitempty,min=0,max=5000"` // Organization fee in bps (0 to 5000). For example, 100 bps = 1%
}

// UpdateUser holds the fields that can be changed on an existing user.
//
// Every field is optional and the update is PARTIAL: a nil field is omitted from the
// request and the stored value is left unchanged. An entirely empty UpdateUser is a
// valid no-op.
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

func (s *UserService) Create(user *CreateUser) (*qbmodels.User, error) {
	var result qbmodels.User
	err := s.client.makeRequest("POST", "/user/", user, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get the current user (based on API key)
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
	var result qbmodels.User
	endpoint := "/user/email/" + email
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves all users in the organization (must be an admin)
func (s *UserService) GetAll() ([]qbmodels.User, error) {
	var result []qbmodels.User
	err := s.client.makeRequest("GET", "/user/all", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *UserService) Update(userID uint64, user *UpdateUser) (*qbmodels.User, error) {
	var result qbmodels.User
	endpoint := "/user/" + fmt.Sprint(userID)
	err := s.client.makeRequest("PUT", endpoint, user, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *UserService) Delete(userID uint64) error {
	endpoint := "/user/" + fmt.Sprint(userID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
