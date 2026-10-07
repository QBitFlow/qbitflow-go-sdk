package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// ProductService manages the products you sell. Products are scoped to the
// (organization, user) pair of the credential; use OnBehalfOf to reach a specific user's
// products with an organization key.
type ProductService struct {
	client *Client
}

// NewProductService creates the product service bound to client.
func NewProductService(client *Client) *ProductService {
	return &ProductService{client: client}
}

// OnBehalfOf returns a ProductService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
// A userID of 0 returns a copy that acts at the organization level (no header sent).
//
//	userProducts, err := client.Products.OnBehalfOf(123).GetAll()
func (s *ProductService) OnBehalfOf(userID uint64) *ProductService {
	return NewProductService(s.client.withOnBehalfOf(userID))
}

// CreateProduct holds the fields required to create a product.
type CreateProduct struct {
	Name        string  `json:"name"`        // 2-100 chars, producttext (no markup characters)
	Description string  `json:"description"` // 2-500 chars, producttext
	Price       float64 `json:"price"`       // USD, a finite number greater than 0
	// Reference is an optional reference for the product (e.g. a SKU); auto-generated when
	// nil or "".
	Reference *string `json:"reference,omitempty"`
}

// Validate mirrors the API's binding rules for a new product: name and description are
// required `producttext` (2-100 and 2-500 characters), and price must be a finite number
// greater than 0.
func (p *CreateProduct) Validate() error {
	if err := requireArgument(p == nil, "Product"); err != nil {
		return err
	}
	if err := validateProductText("Product name", p.Name, 2, 100); err != nil {
		return err
	}
	if err := validateProductText("Product description", p.Description, 2, 500); err != nil {
		return err
	}
	return validatePrice("Product price", p.Price)
}

// UpdateProduct holds the fields that can be changed on an existing product.
//
// Every field is optional and the update is PARTIAL: a nil field is omitted from the
// request and the stored value is left unchanged — there is no side-effect reset. An
// entirely empty UpdateProduct is a valid no-op.
//
// Fields that are not updatable (reference, isActive, test, organizationId, userId, id,
// createdAt) are silently ignored by the API and are deliberately absent here.
type UpdateProduct struct {
	Name        *string  `json:"name,omitempty"`        // 2-100 chars, producttext (no angle brackets)
	Description *string  `json:"description,omitempty"` // 2-500 chars, producttext (no angle brackets)
	Price       *float64 `json:"price,omitempty"`       // USD, a finite number greater than 0
}

// Validate checks the fields that are present. Absent (nil) fields are not validated,
// since they are not sent and therefore leave the stored value untouched. A present field
// is validated exactly as on create — including a pointer to "", which the API rejects.
//
// Length is counted in code points (runes), as the server counts it.
func (p *UpdateProduct) Validate() error {
	if err := requireArgument(p == nil, "Product update"); err != nil {
		return err
	}
	if p.Name != nil {
		if err := validateProductText("Product name", *p.Name, 2, 100); err != nil {
			return err
		}
	}
	if p.Description != nil {
		if err := validateProductText("Product description", *p.Description, 2, 500); err != nil {
			return err
		}
	}
	if p.Price != nil {
		return validatePrice("Product price", *p.Price)
	}
	return nil
}

// Create creates a new product with the provided information
func (s *ProductService) Create(product *CreateProduct) (*qbmodels.Product, error) {
	if err := product.Validate(); err != nil {
		return nil, err
	}

	body := &CreateProduct{
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Reference:   omitEmpty(product.Reference),
	}

	var p qbmodels.Product
	err := s.client.makeRequest("POST", "/product/", body, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Get retrieves a product by its ID
func (s *ProductService) Get(productID uint64) (*qbmodels.Product, error) {
	if err := requirePositive(productID, "Product ID"); err != nil {
		return nil, err
	}

	var p qbmodels.Product
	endpoint := fmt.Sprintf("/product/id/%d", productID)
	err := s.client.makeRequest("GET", endpoint, nil, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAll retrieves all active products (hidden "ghost" products created by a session
// checkout are excluded; they stay reachable by Get and GetByReference).
func (s *ProductService) GetAll() ([]qbmodels.Product, error) {
	return getList[qbmodels.Product](s.client, "/product/")
}

// GetByReference retrieves a product by its reference.
//
// The reference is path-escaped correctly, but the API currently cannot route a reference
// containing "/" (it answers 404).
func (s *ProductService) GetByReference(reference string) (*qbmodels.Product, error) {
	if err := requireNonEmpty(reference, "Product reference"); err != nil {
		return nil, err
	}

	var p qbmodels.Product
	endpoint := fmt.Sprintf("/product/reference/%s", escapePath(reference))
	err := s.client.makeRequest("GET", endpoint, nil, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Update partially updates an existing product (see UpdateProduct for the field rules).
func (s *ProductService) Update(productID uint64, product *UpdateProduct) (*qbmodels.Product, error) {
	if err := requirePositive(productID, "Product ID"); err != nil {
		return nil, err
	}
	if err := product.Validate(); err != nil {
		return nil, err
	}

	var p qbmodels.Product
	endpoint := fmt.Sprintf("/product/%d", productID)
	err := s.client.makeRequest("PUT", endpoint, product, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete soft-deletes a product by its ID.
//
// It returns only an error, the idiomatic Go shape; the API's confirmation message carries
// no information beyond success.
func (s *ProductService) Delete(productID uint64) error {
	if err := requirePositive(productID, "Product ID"); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("/product/%d", productID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
