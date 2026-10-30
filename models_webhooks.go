package qbitflow

import (
	"net/url"
	"time"
)

// WebhookEndpoint is a webhook endpoint of a space.
type WebhookEndpoint struct {
	// UUID is the endpoint's id.
	UUID string `json:"uuid"`
	// Test is true in test mode.
	Test bool `json:"test"`
	// URL is where the events are posted.
	URL string `json:"url"`
	// Events are the event types it receives (stored sorted, without duplicates); empty: every
	// type, including types added later.
	Events []EventType `json:"events"`
	// IncludeMembers is true for an organization endpoint that also receives its members' events.
	IncludeMembers bool `json:"includeMembers"`
	// Description is a note for the dashboard; "" (absent) when none.
	Description string `json:"description,omitempty"`
	// CreatedAt is when it was created.
	CreatedAt time.Time `json:"createdAt"`
	// DisabledAt is when it was disabled (it receives nothing): after 3 days without a successful
	// delivery (live mode only), or by its owner; nil while enabled. Update with Enabled true
	// enables it again.
	DisabledAt *time.Time `json:"disabledAt,omitempty"`
	// DisabledReason is why: failing (live mode only: a test-mode endpoint is never disabled for
	// failing), owner or closed.
	DisabledReason EndpointDisabledReason `json:"disabledReason,omitempty"`
	// FailingSince is when its deliveries started failing (live mode: failing for 3 days, it is
	// disabled).
	FailingSince *time.Time `json:"failingSince,omitempty"`
	// LastDeliveredAt is its last successful delivery.
	LastDeliveredAt *time.Time `json:"lastDeliveredAt,omitempty"`
}

// WebhookEndpointCreated is Webhooks.Endpoints.Create's answer: the endpoint and its secret.
type WebhookEndpointCreated struct {
	WebhookEndpoint

	// Secret (whsec_…) verifies the signature of its deliveries. Shown once: store it.
	Secret string `json:"secret"`
}

// EventDetail is an event of the log with its deliveries (Webhooks.Events.Get).
type EventDetail struct {
	Event

	// Deliveries are its deliveries to the space's endpoints.
	Deliveries []EndpointDelivery `json:"deliveries"`
}

// EndpointDelivery is an event's deliveries to one endpoint, and whether they go on.
//
// While Pending, QBitFlow still attempts the event on its own: NextAttemptAt is set when it is
// retried after a failed attempt, WaitingFor when it is queued behind an earlier event of the
// same publication, and neither when its delivery is about to start (the last two have no
// Attempts yet). Once Pending is false nothing more happens on its own, and Delivered tells how
// it ended (false: abandoned after its retries, or the endpoint was disabled). A deleted
// endpoint's deliveries are deleted with it and no longer listed; disable an endpoint to keep them.
type EndpointDelivery struct {
	// EndpointUUID is the endpoint.
	EndpointUUID string `json:"endpointUuid"`
	// URL is the endpoint's URL now.
	URL string `json:"url"`
	// Delivered is true when one attempt was answered with a 2xx.
	Delivered bool `json:"delivered"`
	// Attempts are its attempts, oldest first. Empty while the event waits for its first
	// attempt (Pending): check the length before reading the last one.
	Attempts []DeliveryAttempt `json:"attempts"`
	// Pending is true while QBitFlow still attempts the event on its own. False once delivered,
	// abandoned after its retries (3 days in live mode, 3 hours in test mode), or when the
	// endpoint was disabled (a deleted endpoint's deliveries are deleted with it: no longer
	// listed). When QBitFlow can't read a delivery's progress, Pending
	// comes from the attempts: true while the retry schedule still attempts the event again (a
	// delivery being retried never reads as abandoned), without NextAttemptAt; a delivery queued
	// with no attempt yet isn't listed until its progress can be read: read the event again a
	// moment later.
	Pending bool `json:"pending"`
	// NextAttemptAt is when the next attempt comes, while Pending after a failed attempt; nil
	// otherwise.
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	// WaitingFor is the id (evt_…) of the earlier event of the same publication this one is
	// queued behind, while Pending; "" otherwise. An endpoint gets a publication's events in
	// order, each once the previous one is delivered or abandoned.
	WaitingFor string `json:"waitingFor,omitempty"`
}

// DeliveryAttempt is one attempt to deliver an event to an endpoint.
type DeliveryAttempt struct {
	// UUID is the attempt's id.
	UUID string `json:"uuid"`
	// EventID is the event (evt_…).
	EventID string `json:"eventId"`
	// EventType is the event's type.
	EventType EventType `json:"eventType"`
	// EndpointUUID is the endpoint.
	EndpointUUID string `json:"endpointUuid"`
	// Attempt is 1 for the first, then one more per retry or resend.
	Attempt int `json:"attempt"`
	// Delivered is true when the endpoint answered with a 2xx.
	Delivered bool `json:"delivered"`
	// Skipped is true when it was not posted: a member's endpoint while the organization's
	// members.webhooks policy was off for them (never retried; not counted against the endpoint's
	// health). Absent when false.
	Skipped bool `json:"skipped,omitempty"`
	// StatusCode is the endpoint's answer; nil when it didn't answer or the attempt was Skipped.
	StatusCode *int `json:"statusCode,omitempty"`
	// Error is why it wasn't delivered.
	Error string `json:"error,omitempty"`
	// DurationMs is how long it took, in milliseconds.
	DurationMs int `json:"durationMs"`
	// AttemptedAt is when it was made.
	AttemptedAt time.Time `json:"attemptedAt"`
}

// CreateWebhookEndpointParams creates a webhook endpoint (Webhooks.Endpoints.Create).
type CreateWebhookEndpointParams struct {
	// URL is required: where the events are posted (https and a public host in live mode).
	URL string `json:"url"`
	// Events are the event types to receive (at most 20, not webhook.test; an unknown type is the
	// API's 400); empty: every type, including types added later.
	Events []EventType `json:"events,omitempty"`
	// IncludeMembers makes an organization endpoint receive its members' events too (nil: true
	// for an organization endpoint, false for a member's).
	IncludeMembers *bool `json:"includeMembers,omitempty"`
	// Description is a note for the dashboard (at most 200 characters).
	Description string `json:"description,omitempty"`
}

// Validate checks the params.
func (p *CreateWebhookEndpointParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	if v.required("url", p.URL) {
		v.url("url", p.URL)
	}
	v.endpointEvents("events", p.Events)
	v.text("description", p.Description, 0, 200)
	return v.err()
}

// UpdateWebhookEndpointParams updates a webhook endpoint (Webhooks.Endpoints.Update): unset
// fields are unchanged.
type UpdateWebhookEndpointParams struct {
	// URL is the new URL ("" = unchanged).
	URL string `json:"url,omitempty"`
	// Events replaces the event types (at most 20, not webhook.test): nil leaves them unchanged,
	// an empty non-nil slice ([]EventType{}) means every type.
	Events []EventType `json:"events,omitzero"`
	// IncludeMembers changes whether an organization endpoint receives its members' events.
	IncludeMembers *bool `json:"includeMembers,omitempty"`
	// Description is the new note; a pointer to "" clears it, nil leaves it unchanged.
	Description *string `json:"description,omitempty"`
	// Enabled false pauses the endpoint (it receives nothing, deliveries in progress stop); true
	// enables it again, also after it was disabled for failing.
	Enabled *bool `json:"enabled,omitempty"`
}

// Validate checks the params.
func (p *UpdateWebhookEndpointParams) Validate() error {
	if p == nil {
		return newValidationError("params are required")
	}
	var v validator
	v.url("url", p.URL)
	v.endpointEvents("events", p.Events)
	if p.Description != nil {
		v.text("description", *p.Description, 0, 200)
	}
	return v.err()
}

// EndpointListParams pages Webhooks.Endpoints.List.
type EndpointListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim; "" for the first page.
	Cursor string
}

// Validate checks the params.
func (p *EndpointListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	return v.err()
}

func (p *EndpointListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	return q
}

// EventListParams filters Webhooks.Events.List.
type EventListParams struct {
	// Limit is the page size (server default 20, max 100; 0 = default).
	Limit int
	// Cursor is the previous page's NextCursor, verbatim (opaque: never an event id); "" for the
	// first page.
	Cursor string
	// Type keeps only the events of this type. Not checked against the SDK's known types (newer
	// ones appear over time); an unknown type is the API's 400.
	Type EventType
	// IncludeMembers adds the members' events (organization space only; 400 from a member's).
	IncludeMembers bool
}

// Validate checks the filters.
func (p *EventListParams) Validate() error {
	if p == nil {
		return nil
	}
	var v validator
	v.cursor(p.Cursor)
	return v.err()
}

func (p *EventListParams) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	setPage(q, p.Limit, p.Cursor)
	setString(q, "type", string(p.Type))
	setBool(q, "includeMembers", p.IncludeMembers)
	return q
}

// verifyWebhookBody is POST /webhooks/verify's body (Webhooks.VerifyRemote).
type verifyWebhookBody struct {
	EndpointUUID string `json:"endpointUuid"`
	Body         string `json:"body"`
	Signature    string `json:"signature"`
}
