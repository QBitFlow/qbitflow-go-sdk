package qbitflow

import (
	"fmt"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

type ProductService struct {
	client *Client
}

type CreateProduct struct {
	Name        string  `json:"name" binding:"required"`        // Product name
	Description string  `json:"description" binding:"required"` // Product description
	Price       float64 `json:"price" binding:"required"`       // Product price
	Reference   *string `json:"reference,omitempty"`            // Optional reference for the product (e.g., SKU, product ID, etc.)
}

func (p *CreateProduct) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if p.Price <= 0 {
		return fmt.Errorf("product price cannot be negative")
	}
	return nil
}

type UpdateProduct struct {
	Name        string  `json:"name" binding:"required"`        // Product name
	Description string  `json:"description" binding:"required"` // Product description
	Price       float64 `json:"price" binding:"required"`       // Product price
}

func (p *UpdateProduct) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if p.Price <= 0 {
		return fmt.Errorf("product price cannot be negative")
	}
	return nil
}

// Create creates a new product with the provided information
func (s *ProductService) Create(product *CreateProduct) (*models.Product, error) {
	if err := product.Validate(); err != nil {
		return nil, err
	}

	var p models.Product
	err := s.client.makeRequest("POST", "/product/", product, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Get retrieves a product by its ID
func (s *ProductService) Get(productID uint64) (*models.Product, error) {
	var p models.Product
	endpoint := fmt.Sprintf("/product/id/%d", productID)
	err := s.client.makeRequest("GET", endpoint, nil, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAll retrieves all products
func (s *ProductService) GetAll() ([]models.Product, error) {
	var products []models.Product
	err := s.client.makeRequest("GET", "/product/", nil, &products)
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetByReference retrieves a product by its reference
func (s *ProductService) GetByReference(reference string) (*models.Product, error) {
	if reference == "" {
		return nil, fmt.Errorf("product reference cannot be empty")
	}

	var p models.Product
	endpoint := fmt.Sprintf("/product/reference/%s", reference)
	err := s.client.makeRequest("GET", endpoint, nil, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Update updates an existing product with the provided information
func (s *ProductService) Update(productID uint64, product *UpdateProduct) (*models.Product, error) {
	if err := product.Validate(); err != nil {
		return nil, err
	}

	var p models.Product
	endpoint := fmt.Sprintf("/product/%d", productID)
	err := s.client.makeRequest("PUT", endpoint, product, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete deletes a product by its ID
func (s *ProductService) Delete(productID uint64) error {
	endpoint := fmt.Sprintf("/product/%d", productID)
	return s.client.makeRequest("DELETE", endpoint, nil, nil)
}
