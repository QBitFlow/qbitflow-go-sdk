package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// ApiKeyService exposes read access to API keys.
//
// NOTE: creating and deleting API keys is a JWT-only operation on the API
// (auth.RequiresTokenType(JWT)) — it cannot be performed with an API key, which is
// the only credential this SDK uses. Manage keys from the QBitFlow dashboard instead.
type ApiKeyService struct {
	client *Client
}

func NewApiKeyService(client *Client) *ApiKeyService {
	return &ApiKeyService{client: client}
}

// OnBehalfOf returns an ApiKeyService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
func (s *ApiKeyService) OnBehalfOf(userID uint64) *ApiKeyService {
	return NewApiKeyService(s.client.withOnBehalfOf(userID))
}

// GetAll returns the API keys associated with the caller (the key used for the request).
func (s *ApiKeyService) GetAll() ([]qbmodels.ApiKey, error) {
	var result []qbmodels.ApiKey
	err := s.client.makeRequest("GET", "/api-key/", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetForUser returns the API keys for a specific user (admin only).
func (s *ApiKeyService) GetForUser(userID uint64) ([]qbmodels.ApiKey, error) {
	var result []qbmodels.ApiKey
	endpoint := "/api-key/user/" + fmt.Sprint(userID)
	err := s.client.makeRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
