package tests

import (
	"fmt"
	"math/rand"

	"github.com/qbitflow/qbitflow-go-sdk/pkg/qbitflow"
)

func getTestCustomerData() *qbitflow.CreateCustomer {
	// Generate a random email to avoid conflicts
	randomEmail := fmt.Sprintf("john.doe.%d@example.com", rand.Intn(1000))

	return &qbitflow.CreateCustomer{
		Name:     "John",
		LastName: "Doe",
		Email:    randomEmail,
	}
}

func getTestUserData() *qbitflow.CreateUser {
	// Generate a random email to avoid conflicts
	randomEmail := fmt.Sprintf("test.user.%d@example.com", rand.Intn(1000))

	return &qbitflow.CreateUser{
		Name:     "Test",
		LastName: "User",
		Email:    randomEmail,
		Password: "password123",
		Role:     "user",
	}
}

func getTestProductData() *qbitflow.CreateProduct {
	ref := fmt.Sprintf("REF-%d", rand.Intn(100000))
	return &qbitflow.CreateProduct{
		Name:        "Test Product",
		Description: "A product used for testing purposes",
		Price:       19.99,
		Reference:   &ref,
	}
}
