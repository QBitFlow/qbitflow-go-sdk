package qbitflow

import (
	"context"
	"net/http"
)

// maxSettingKeyLength is the longest setting key (the API's binding rule).
const maxSettingKeyLength = 100

// checkSettingKey validates a setting key (a path segment): present, at most 100 characters,
// and not "." or "..". The API judges whether it exists (404).
func checkSettingKey(key string) error {
	if err := checkPathRequired("key", key); err != nil {
		return err
	}
	var v validator
	if !v.length("key", key, 0, maxSettingKeyLength) {
		return v.err()
	}
	return nil
}

// Definitions lists every setting and policy the API knows (GET /settings/definitions; a bounded
// list): its type, bounds, default and who sets a member's value. Read it to render or check a
// value before Set.
func (s *SettingsService) Definitions(ctx context.Context, opts ...RequestOption) ([]SettingDefinition, error) {
	return s.client.callItems[SettingDefinition](ctx, endpoint{method: http.MethodGet, path: "/settings/definitions"}, opts)
}

// List returns every setting and policy in the request's space (GET /settings; a bounded list):
// its value, where it comes from (the space's own, the organization's, or the default) and
// whether this key may change it (Editable). A member's key sees the policies that apply to them.
func (s *SettingsService) List(ctx context.Context, opts ...RequestOption) ([]SettingValue, error) {
	return s.client.callItems[SettingValue](ctx, endpoint{method: http.MethodGet, path: "/settings"}, opts)
}

// Set gives the space its own value of a setting (PUT /settings/:key, body {"value": …}) and
// returns the key's value there. value is typed by the key (Settings.Definitions): a bool, an
// integer, an enum string, or an object (a struct or map, whose unknown fields the API refuses);
// nil is refused. A wrong type, range or value is a *ValidationError (400) naming value or
// value.<field>; an unknown key is a *NotFoundError.
//
// Who may write depends on the key: an organization key writes every setting, a member's key its
// own (403 forbidden for the others; 403 policy_disabled, details.policy naming the policy, when
// a policy has turned the key off for members). A policy (members.*) is changed from the
// dashboard only: with an API key it answers 403 forbidden. A key writes its own mode only.
// Not retried.
func (s *SettingsService) Set(ctx context.Context, key string, value any, opts ...RequestOption) (*SettingValue, error) {
	if err := checkSettingKey(key); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newFieldError("value", "is required")
	}
	body := struct {
		Value any `json:"value"`
	}{value}
	return s.client.callRef[SettingValue](ctx, endpoint{method: http.MethodPut, path: pathf("/settings/%s", key), body: body}, opts)
}

// Reset removes the space's own value of a setting (DELETE /settings/:key, 204): it inherits the
// organization's value again, or the default (read it with List). Same access as Set (403
// forbidden, 403 policy_disabled; a policy is reset from the dashboard only); an unknown key is
// a *NotFoundError. Not retried.
func (s *SettingsService) Reset(ctx context.Context, key string, opts ...RequestOption) error {
	if err := checkSettingKey(key); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/settings/%s", key)}, opts)
}
