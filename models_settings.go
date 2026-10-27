package qbitflow

import "encoding/json"

// SettingValue is a setting's (or policy's) value in the request's space, as Settings.List and
// Settings.Set answer it. Value is typed by the key (Settings.Definitions): a JSON boolean,
// integer, enum string or object; decode it with encoding/json.
//
//	{"key": "refunds.windowDays", "kind": "setting", "value": 30, "source": "default", "editable": true}
type SettingValue struct {
	// Key names the setting (refunds.windowDays, checkout.defaults, branding, …).
	Key string `json:"key"`
	// Kind is whether it is a setting or a policy.
	Kind SettingKind `json:"kind"`
	// Value is the key's value, as JSON; its type is the definition's Type.
	Value json.RawMessage `json:"value"`
	// Source is where the value comes from: the space's own, its organization's, or the default.
	Source SettingSource `json:"source"`
	// Editable is whether this key may change it here (Settings.Set). A policy is never editable
	// with an API key.
	Editable bool `json:"editable"`
}

// SettingDefinition describes one key, as Settings.Definitions lists it: render and check a value
// from it.
type SettingDefinition struct {
	// Key names the setting.
	Key string `json:"key"`
	// Category is the key's prefix (members, checkout, emails, …), to group the keys.
	Category string `json:"category"`
	// Kind is whether it is a setting or a policy.
	Kind SettingKind `json:"kind"`
	// Description says what the key does.
	Description string `json:"description"`
	// Type is the value's type: boolean, integer, enum or object.
	Type string `json:"type"`
	// Enum is the values of an enum; empty for another type.
	Enum []string `json:"enum,omitempty"`
	// Min is the lower bound of an integer; nil when unbounded.
	Min *int `json:"min,omitempty"`
	// Max is the upper bound of an integer; nil when unbounded.
	Max *int `json:"max,omitempty"`
	// Default is the value the key has when nobody set it, as JSON.
	Default json.RawMessage `json:"default"`
	// MembersEdit is who sets the key's value in a member's space: always, never, or the name of
	// the policy that lets members do it.
	MembersEdit string `json:"membersEdit"`
}
