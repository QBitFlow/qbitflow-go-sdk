package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// List returns one page of the space's customers (GET /customers; newest first, page size 20
// by default, at most 100), filtered by Email (several customers may share one) or Reference.
// params may be nil. Iterate walks every page.
func (s *CustomersService) List(ctx context.Context, params *CustomerListParams, opts ...RequestOption) (*Page[Customer], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Customer]](ctx, endpoint{method: http.MethodGet, path: "/customers", query: params.values()}, opts)
}

// Iterate walks every customer List returns, fetching the pages lazily from params.Cursor (or
// the first page) and keeping the filters and the page size. An error ends the sequence.
func (s *CustomersService) Iterate(ctx context.Context, params *CustomerListParams, opts ...RequestOption) iter.Seq2[Customer, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Customer], error) {
		page := p // a copy per request: ranges never share params
		page.Cursor = cursor
		return s.List(ctx, &page, opts...)
	})
}

// Create creates a customer (POST /customers, 201). It sends an Idempotency-Key and
// is retried on transient failures. The email is stored lowercase and is not unique: several
// customers of a space may share one.
//
// Errors: 400 validation_failed naming the field; 409 unique_violation when the reference is
// taken by another live customer of the space (details.field "reference").
func (s *CustomersService) Create(ctx context.Context, params *CreateCustomerParams, opts ...RequestOption) (*Customer, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodPost, path: "/customers", body: params, idempotent: true}, opts)
}

// Get returns a customer by its UUID (GET /customers/:uuid). An unknown, deleted or another
// space's customer is a *NotFoundError.
func (s *CustomersService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*Customer, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodGet, path: pathf("/customers/%s", uuid)}, opts)
}

// GetByReference returns the space's customer with your reference: it runs List's reference
// filter (GET /customers?reference=) and returns its one customer. No customer with that
// reference is a *NotFoundError (status 404, code not_found). To find customers by email, use
// List with params.Email (an email may match several).
func (s *CustomersService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Customer, error) {
	return s.client.getByReference[Customer](ctx, "/customers", "customer", reference, func(v *Customer) string { return v.Reference }, opts)
}

// Update changes a customer (PATCH /customers/:uuid): only the fields set in params change; a
// pointer to "" clears LastName, PhoneNumber or Address; Metadata is merged. The reference
// cannot change. Not retried.
func (s *CustomersService) Update(ctx context.Context, uuid string, params *UpdateCustomerParams, opts ...RequestOption) (*Customer, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodPatch, path: pathf("/customers/%s", uuid), body: params}, opts)
}

// Delete deletes a customer (DELETE /customers/:uuid, 204; a soft delete): it leaves every read
// and frees its reference; its past payments keep naming it (CustomerSummary.Deleted). Not
// retried.
func (s *CustomersService) Delete(ctx context.Context, uuid string, opts ...RequestOption) error {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/customers/%s", uuid)}, opts)
}
