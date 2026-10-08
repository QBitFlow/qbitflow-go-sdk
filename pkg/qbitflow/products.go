package qbf

import (
	"fmt"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

type ProductService struct {
	client *Client
}

func NewProductService(client *Client) *ProductService {
	return &ProductService{client: client}
}

// OnBehalfOf returns a ProductService that acts on behalf of the given user
// within the same organization. Requires an admin-level (organization) API key.
//
//	userProducts, err := client.Products.OnBehalfOf(123).GetAll()
func (s *ProductService) OnBehalfOf(userID uint64) *ProductService {
	return NewProductService(s.client.withOnBehalfOf(userID))
}

type CreateProduct struct {
	Name        string  `json:"name" binding:"required"`        // Product name
	Description string  `json:"description" binding:"required"` // Product description
	Price       float64 `json:"price" binding:"required"`       // Product price
	Reference   *string `json:"reference,omitempty"`            // Optional reference for the product (e.g., SKU, product ID, etc.)
}

// Validate mirrors the API's binding rules for a new product: name and description are
// required `producttext` (2-100 and 2-500 characters), and price must be greater than 0.
func (p *CreateProduct) Validate() error {
	if err := validateProductText("Product name", p.Name, 2, 100); err != nil {
		return err
	}
	if err := validateProductText("Product description", p.Description, 2, 500); err != nil {
		return err
	}
	if p.Price <= 0 {
		return qberrors.NewValidationError("Product price must be greater than 0")
	}
	return nil
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
	Price       *float64 `json:"price,omitempty"`       // USD, must be strictly greater than 0
}

// Validate checks the fields that are present. Absent (nil) fields are not validated,
// since they are not sent and therefore leave the stored value untouched.
//
// Length is counted in runes, matching the server. The previous byte-based check rejected
// valid accented or CJK names locally — a 60-character accented name is 120 bytes.
func (p *UpdateProduct) Validate() error {
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
	if p.Price != nil && *p.Price <= 0 {
		return qberrors.NewValidationError("Product price must be greater than 0")
	}
	return nil
}

// Create creates a new product with the provided information
func (s *ProductService) Create(product *CreateProduct) (*qbmodels.Product, error) {
	if err := product.Validate(); err != nil {
		return nil, err
	}

	var p qbmodels.Product
	err := s.client.makeRequest("POST", "/product/", product, &p)
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

// GetAll retrieves all products
func (s *ProductService) GetAll() ([]qbmodels.Product, error) {
	var products []qbmodels.Product
	err := s.client.makeRequest("GET", "/product/", nil, &products)
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetByReference retrieves a product by its reference
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

// Update updates an existing product with the provided information
func (s *ProductService) Update(productID uint64, product *UpdateProduct) (*qbmodels.Product, error) {
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

// Delete deletes a product by its ID
func (s *ProductService) Delete(productID uint64) error {
	if err := requirePositive(productID, "Product ID"); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("/product/%d", productID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
