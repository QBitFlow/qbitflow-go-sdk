package qbitflow

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/pkg/models"
)

type UserService struct {
	client *Client
}

type CreateUser struct {
	Name               string `json:"name" binding:"required"`
	LastName           string `json:"lastName" binding:"required"`
	Email              string `json:"email" binding:"required,email"`
	Password           string `json:"password" binding:"required"`
	Role               string `json:"role" binding:"required,oneof=admin user"`              // Role can be 'admin' or 'user'
	OrganizationFeeBps uint16 `json:"organizationFeeBps" binding:"omitempty,min=0,max=1000"` // Organization fee in bps (0 to 1000). For example, 100 bps = 1%
}

type UpdateUser struct {
	Name               string  `json:"name" binding:"required"`
	LastName           string  `json:"lastName" binding:"required"`
	Email              string  `json:"email" binding:"required,email"`
	Password           *string `json:"password,omitempty"`                          // Optional: User's password
	OrganizationFeeBps uint16  `json:"organizationFeeBps" binding:"min=0,max=1000"` // Organization fee in bps (0 to 1000). For example, 100 bps = 1%
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
