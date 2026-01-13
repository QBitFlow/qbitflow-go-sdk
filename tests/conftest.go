package tests

import (
	"fmt"
	"math/rand"

	qbf "github.com/QBitFlow/qbitflow-go-sdk/pkg/qbitflow"
)

func getTestCustomerData() *qbf.CreateCustomer {
	// Generate a random email to avoid conflicts
	randomEmail := fmt.Sprintf("john.doe.%d@example.com", rand.Intn(1000))

	return &qbf.CreateCustomer{
		Name:     "John",
		LastName: "Doe",
		Email:    randomEmail,
	}
}

func getTestUserData() *qbf.CreateUser {
	// Generate a random email to avoid conflicts
	randomEmail := fmt.Sprintf("test.user.%d@example.com", rand.Intn(1000))

	return &qbf.CreateUser{
		Name:     "Test",
		LastName: "User",
		Email:    randomEmail,
		Password: "password123",
		Role:     "user",
	}
}

func getTestProductData() *qbf.CreateProduct {
	ref := fmt.Sprintf("REF-%d", rand.Intn(100000))
	return &qbf.CreateProduct{
		Name:        "Test Product",
		Description: "A product used for testing purposes",
		Price:       19.99,
		Reference:   &ref,
	}
}
