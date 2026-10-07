package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// VerifyRemote has the API check a webhook delivery's signature (POST /webhooks/verify):
// endpointUUID is the endpoint that received it, rawBody the body exactly as received,
// signatureHeader its QBitFlow-Signature header. It is the same check as VerifyWebhook, made
// server side with the endpoint's current secret (useful when you do not store it). Not retried.
//
// It returns nil when the signature is valid. A mismatch, a malformed header, or a timestamp
// more than 5 minutes from the server's clock (400 invalid_signature) is a
// *WebhookSignatureError with Reason invalidSignature; an empty header is one with Reason
// missingHeader (nothing is sent). Other errors are returned as usual (another space's
// endpoint: *NotFoundError). Parse the verified body with ParseEvent.
func (s *WebhooksService) VerifyRemote(ctx context.Context, endpointUUID string, rawBody []byte, signatureHeader string, opts ...RequestOption) error {
	if err := checkPathUUID("endpointUuid", endpointUUID); err != nil {
		return err
	}
	if len(rawBody) == 0 {
		return newFieldError("body", "is required")
	}
	if len(rawBody) > maxWebhookBody {
		return newFieldError("body", "must be at most 1 MiB")
	}
	if signatureHeader == "" {
		return newSignatureError(WebhookSignatureMissingHeader, "missing "+SignatureHeader+" header")
	}
	err := s.client.callVoid(ctx, endpoint{
		method: http.MethodPost, path: "/webhooks/verify",
		body: verifyWebhookBody{EndpointUUID: endpointUUID, Body: string(rawBody), Signature: signatureHeader},
	}, opts)
	return asWebhookSignatureError(err)
}

// List returns the space's webhook endpoints (GET /webhooks/endpoints; at most 10, not
// paginated). A member's key needs the organization's members.webhooks policy (403
// policy_disabled), as every Endpoints and Events method.
func (s *WebhookEndpointsService) List(ctx context.Context, opts ...RequestOption) ([]WebhookEndpoint, error) {
	return s.client.call[[]WebhookEndpoint](ctx, endpoint{method: http.MethodGet, path: "/webhooks/endpoints"}, opts)
}

// Create adds a webhook endpoint to the space (POST /webhooks/endpoints, 201). The answer
// carries the endpoint's Secret (whsec_…), shown only this once: store it to verify the
// deliveries. It sends an Idempotency-Key and is retried on transient failures.
//
// Errors: 400 validation_failed (live mode needs https and a public host); 409 conflict beyond
// 10 endpoints per space and mode.
func (s *WebhookEndpointsService) Create(ctx context.Context, params *CreateWebhookEndpointParams, opts ...RequestOption) (*WebhookEndpointCreated, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[WebhookEndpointCreated](ctx, endpoint{method: http.MethodPost, path: "/webhooks/endpoints", body: params, idempotent: true}, opts)
}

// Get returns a webhook endpoint (GET /webhooks/endpoints/:uuid). Its secret is never
// returned again (rotate it in the dashboard).
func (s *WebhookEndpointsService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*WebhookEndpoint, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[WebhookEndpoint](ctx, endpoint{method: http.MethodGet, path: pathf("/webhooks/endpoints/%s", uuid)}, opts)
}

// Update changes a webhook endpoint (PUT /webhooks/endpoints/:uuid): only the fields set in
// params change. Enabled false pauses it, true enables it again; PayloadVersion v2 moves an
// endpoint migrated from v1 to the event envelope. Not retried.
func (s *WebhookEndpointsService) Update(ctx context.Context, uuid string, params *UpdateWebhookEndpointParams, opts ...RequestOption) (*WebhookEndpoint, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[WebhookEndpoint](ctx, endpoint{method: http.MethodPut, path: pathf("/webhooks/endpoints/%s", uuid), body: params}, opts)
}

// Delete deletes a webhook endpoint (DELETE /webhooks/endpoints/:uuid): it receives nothing
// more. Not retried.
func (s *WebhookEndpointsService) Delete(ctx context.Context, uuid string, opts ...RequestOption) error {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/webhooks/endpoints/%s", uuid)}, opts)
}

// List returns one page of the space's event log (GET /webhooks/events; newest first, page
// size 20 by default, at most 100), optionally filtered by Type; IncludeMembers adds the
// members' events (organization key). The cursor is an event id (evt_…). Decode each event's
// data with its typed accessor. params may be nil.
func (s *WebhookEventsService) List(ctx context.Context, params *EventListParams, opts ...RequestOption) (*Page[Event], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Event]](ctx, endpoint{method: http.MethodGet, path: "/webhooks/events", query: params.values()}, opts)
}

// Iterate walks every event List returns, fetching the pages lazily and keeping the filters and
// the page size. An error ends the sequence.
func (s *WebhookEventsService) Iterate(ctx context.Context, params *EventListParams, opts ...RequestOption) iter.Seq2[Event, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Event], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Get returns an event of the log with its deliveries to the space's endpoints
// (GET /webhooks/events/:id; id is evt_…).
func (s *WebhookEventsService) Get(ctx context.Context, id string, opts ...RequestOption) (*EventDetail, error) {
	if err := checkPathRequired("id", id); err != nil {
		return nil, err
	}
	return s.client.callRef[EventDetail](ctx, endpoint{method: http.MethodGet, path: pathf("/webhooks/events/%s", id)}, opts)
}
