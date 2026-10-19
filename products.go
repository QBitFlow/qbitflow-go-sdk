package qbitflow

import (
	"context"
	"net/http"
)

// List returns the space's products (GET /product), cheapest first; not paginated. Hidden
// products (IsActive false) are left out unless params.IncludeHidden; params.Subscription
// keeps only the subscription (true) or one-time (false) products. params may be nil.
func (s *ProductsService) List(ctx context.Context, params *ProductListParams, opts ...RequestOption) ([]Product, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.call[[]Product](ctx, endpoint{method: http.MethodGet, path: "/product", query: params.values()}, opts)
}

// Create creates a product (POST /product, 201). It sends an Idempotency-Key and is retried
// on transient failures. A subscription product needs params.Subscription.Frequency.
//
// Errors: 400 validation_failed (e.g. price above $5 in test mode, details.max); 409
// unique_violation on reference (references are unique among the space's live products); a
// member's key needs the organization's members.products policy, and members.subscriptions to
// give the product subscription terms (403 policy_disabled, details.policy names it).
func (s *ProductsService) Create(ctx context.Context, params *CreateProductParams, opts ...RequestOption) (*Product, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodPost, path: "/product", body: params, idempotent: true}, opts)
}

// Get returns a product by its UUID (GET /product/uuid/:uuid), hidden ones included. An
// unknown, deleted or another space's product is a *NotFoundError.
func (s *ProductsService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*Product, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodGet, path: pathf("/product/uuid/%s", uuid)}, opts)
}

// GetByReference returns a product by your reference (GET /product/reference/:reference).
// The reference is sent escaped (a "/" never splits the path).
func (s *ProductsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Product, error) {
	if err := checkPathRequired("reference", reference); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodGet, path: pathf("/product/reference/%s", reference)}, opts)
}

// Update changes a product in place (PUT /product/:uuid): only the fields set in params change
// (an empty params is a no-op that returns the product). The reference cannot change. A
// new price applies to new checkouts and new subscribers only: existing subscriptions and open
// checkout sessions keep theirs. Not retried.
//
// A member's key gives a product subscription terms only with the organization's
// members.subscriptions policy (403 policy_disabled); RemoveSubscription and other edits
// never need it.
func (s *ProductsService) Update(ctx context.Context, uuid string, params *UpdateProductParams, opts ...RequestOption) (*Product, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodPut, path: pathf("/product/%s", uuid), body: params}, opts)
}

// Delete deletes a product (DELETE /product/:uuid, a soft delete): it leaves every read and frees
// its reference. It does not stop its subscriptions (they keep billing at their price): cancel
// them with Subscriptions.Cancel if the product is gone for good. Not retried.
func (s *ProductsService) Delete(ctx context.Context, uuid string, opts ...RequestOption) error {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/product/%s", uuid)}, opts)
}
