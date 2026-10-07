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

// NewApiKeyService creates the API-key service bound to client.
func NewApiKeyService(client *Client) *ApiKeyService {
	return &ApiKeyService{client: client}
}

// OnBehalfOf returns an ApiKeyService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
func (s *ApiKeyService) OnBehalfOf(userID uint64) *ApiKeyService {
	return NewApiKeyService(s.client.withOnBehalfOf(userID))
}

// GetAll returns the API keys visible to the caller: a user-level key sees only that
// user's keys, while an organization-level (admin/owner) key sees every key in the
// organization. Through OnBehalfOf it returns the acted-for user's keys.
func (s *ApiKeyService) GetAll() ([]qbmodels.ApiKey, error) {
	return getList[qbmodels.ApiKey](s.client, "/api-key/")
}

// GetForUser returns the API keys for a specific user (admin only).
func (s *ApiKeyService) GetForUser(userID uint64) ([]qbmodels.ApiKey, error) {
	if err := requirePositive(userID, "User ID"); err != nil {
		return nil, err
	}

	return getList[qbmodels.ApiKey](s.client, "/api-key/user/"+fmt.Sprint(userID))
}
