package qbitflow

import (
	"fmt"
	"time"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

type ApiKeyService struct {
	client *Client
}

type CreateApiKeyDto struct {
	Name      string     `json:"name" binding:"required"`   // Name of the API key
	UserID    uint64     `json:"userId" binding:"required"` // ID of the user for whom the API key is created (if UserID != the one making the request, the requester must be an admin)
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`       // Timestamp when the API key expires, optional (nil means no expiration)
	Test      bool       `json:"test"`                      // Whether the API key is for test environment or production
}

type CreatedKeyResponse struct {
	Data models.ApiKey `json:"data"` // The created API key details
	Key  string        `json:"key"`  // The actual API key string (only returned upon creation)
}

// Create a new API key for the specified user
func (s *ApiKeyService) Create(apiKey *CreateApiKeyDto) (*CreatedKeyResponse, error) {
	var result CreatedKeyResponse
	err := s.client.makeRequest("POST", "/api-key/", apiKey, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get all the API keys for this user (associated with the API key used to make the request)
func (s *ApiKeyService) GetAll() ([]models.ApiKey, error) {
	var result []models.ApiKey
	err := s.client.makeRequest("GET", "/api-key/", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get the API keys for a specific user (must be an admin to access other users' API keys)
func (s *ApiKeyService) GetForUser(userID uint64) ([]models.ApiKey, error) {
	var result []models.ApiKey
	endpoint := "/api-key/user/" + fmt.Sprint(userID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Delete an API key by its ID
func (s *ApiKeyService) Delete(apiKeyID uint64) error {
	endpoint := "/api-key/" + fmt.Sprint(apiKeyID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
