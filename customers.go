package qbitflow

import (
	"context"
	"iter"
	"net/http"
)

// Create creates a customer (POST /customer, 201). It sends an Idempotency-Key and is retried
// on transient failures. The email is stored lowercase.
//
// Errors: 400 validation_failed naming the field; 409 unique_violation when the email or the
// reference is taken by another live customer of the space (details.field "email" or
// "reference"; an email matches whatever its casing).
func (s *CustomersService) Create(ctx context.Context, params *CreateCustomerParams, opts ...RequestOption) (*Customer, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodPost, path: "/customer", body: params, idempotent: true}, opts)
}

// Update changes a customer (PUT /customer/:uuid): only the fields set in params change; a
// pointer to "" clears PhoneNumber or Address. The reference cannot change. A new email must
// stay unique in the space (409 unique_violation). Not retried.
func (s *CustomersService) Update(ctx context.Context, uuid string, params *UpdateCustomerParams, opts ...RequestOption) (*Customer, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodPut, path: pathf("/customer/%s", uuid), body: params}, opts)
}

// List returns one page of the space's customers (GET /customer/all; page size 10 by default,
// at most 100), filtered by email or by Verified. params may be nil. Iterate walks every page.
func (s *CustomersService) List(ctx context.Context, params *CustomerListParams, opts ...RequestOption) (*Page[Customer], error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return s.client.callRef[Page[Customer]](ctx, endpoint{method: http.MethodGet, path: "/customer/all", query: params.values()}, opts)
}

// Iterate walks every customer List returns, fetching the pages lazily from params.Cursor (or
// the first page) and keeping the filters and the page size. An error ends the sequence.
func (s *CustomersService) Iterate(ctx context.Context, params *CustomerListParams, opts ...RequestOption) iter.Seq2[Customer, error] {
	p := derefOrZero(params)
	return iteratePages(p.Cursor, func(cursor string) (*Page[Customer], error) {
		p.Cursor = cursor
		return s.List(ctx, &p, opts...)
	})
}

// Get returns a customer by its UUID (GET /customer/uuid/:uuid). An unknown, deleted or
// another space's customer is a *NotFoundError.
func (s *CustomersService) Get(ctx context.Context, uuid string, opts ...RequestOption) (*Customer, error) {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodGet, path: pathf("/customer/uuid/%s", uuid)}, opts)
}

// GetByEmail returns the space's customer with this email, whatever its casing
// (GET /customer/email/:email).
func (s *CustomersService) GetByEmail(ctx context.Context, email string, opts ...RequestOption) (*Customer, error) {
	if err := checkPathRequired("email", email); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodGet, path: pathf("/customer/email/%s", email)}, opts)
}

// GetByReference returns the space's customer with your reference
// (GET /customer/reference/:reference). The reference is sent escaped.
func (s *CustomersService) GetByReference(ctx context.Context, reference string, opts ...RequestOption) (*Customer, error) {
	if err := checkPathRequired("reference", reference); err != nil {
		return nil, err
	}
	return s.client.callRef[Customer](ctx, endpoint{method: http.MethodGet, path: pathf("/customer/reference/%s", reference)}, opts)
}

// Delete deletes a customer (DELETE /customer/uuid/:uuid, a soft delete): it leaves every
// read and frees its email and reference; its past payments keep naming it. Not retried.
func (s *CustomersService) Delete(ctx context.Context, uuid string, opts ...RequestOption) error {
	if err := checkPathUUID("uuid", uuid); err != nil {
		return err
	}
	return s.client.callVoid(ctx, endpoint{method: http.MethodDelete, path: pathf("/customer/uuid/%s", uuid)}, opts)
}
