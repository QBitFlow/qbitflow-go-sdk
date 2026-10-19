package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's products (GET /products; newest first, page size 20 by
// default, at most 100). Hidden products (IsActive false) are left out unless
// params.IncludeHidden; params.Subscription keeps only the subscription (true) or one-time
// (false) products; params.Reference the one with that reference. params may be nil. Iterate
// walks every page.
func (s *ProductsService) List(ctx context.Context, params *ProductListParams, opts ...RequestOption) (*Page[Product], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Product]](ctx, endpoint{method: http.MethodGet, path: "/products", query: params.values()}, opts)
}

// Iterate walks every product List returns, fetching the pages lazily from params.Cursor (or
// the first page) and keeping the filters and the page size. An error ends the sequence.
func (s *ProductsService) Iterate(ctx context.Context, params *ProductListParams, opts ...RequestOption) iter.Seq2[Product, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Product], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Create creates a product (POST /products, 201). It sends an Idempotency-Key and is retried
// on transient failures. A subscription product needs params.Subscription.Frequency.
//
// Errors: 400 validation_failed (e.g. a price above "5" in test mode, details.max "5"; a live
// subscription product at "0.25" or less, details.min); 409 unique_violation on reference
// (references are unique among the space's live products); a member's key needs the
// organization's members.products policy, and members.subscriptions to give the product
// subscription terms (403 policy_disabled, details.policy names it).
func (s *ProductsService) Create(ctx context.Context, params *CreateProductParams, opts ...RequestOption) (*Product, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodPost, path: "/products", body: params, idempotent: true}, opts)
}

// Get returns a product by its UUID (GET /products/:uuid), hidden ones included. An unknown,
// deleted or another space's product is a *NotFoundError.
func (s *ProductsService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*Product, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodGet, path: pathf("/products/%s", uuid)}, opts)
}

// GetByReference returns the space's product with your reference, hidden or not: it runs List's
// reference filter (GET /products?reference=) and returns its one product. No product with
// that reference is a *NotFoundError (status 404, code not_found), as an unknown uuid is. The
// reference is checked like a created one's (1 to 100 of A-Z a-z 0-9 . _ : @ -).
func (s *ProductsService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Product, error) {
	return s.client.getByReference[Product](ctx, "/products", "product", reference, func(v *Product) string { return v.Reference }, opts)
}

// Update changes a product in place (PATCH /products/:uuid): only the fields set in params
// change (an empty params is a no-op that returns the product); Metadata is merged. The
// reference cannot change. A new price or new terms apply to new checkouts and new subscribers
// only: existing subscriptions and open checkout sessions keep theirs. Not retried.
//
// A member's key gives a product subscription terms only with the organization's
// members.subscriptions policy (403 policy_disabled); RemoveSubscription and other edits never
// need it.
func (s *ProductsService) Update(ctx context.Context, uuid string, params *UpdateProductParams, opts ...RequestOption) (*Product, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Product](ctx, endpoint{method: http.MethodPatch, path: pathf("/products/%s", uuid), body: params}, opts)
}

// Delete deletes a product (DELETE /products/:uuid, 204; a soft delete): it leaves every read
// and frees its reference. It does not stop its subscriptions (they keep billing at their
// price): cancel them with Subscriptions.Cancel if the product is gone for good. Not retried.
func (s *ProductsService) Delete(ctx context.Context, uuid string, opts ...RequestOption) error {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/products/%s", uuid)}, opts)
}
