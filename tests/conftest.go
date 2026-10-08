package tests

import (
	"fmt"
	"math/rand"

	qbf "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/qbitflow"
)

func getTestCustomerData() *qbf.CreateCustomer {
	randomEmail := fmt.Sprintf("john.doe.%d@example.com", rand.Intn(1000000))
	return &qbf.CreateCustomer{
		Name:     "John",
		LastName: "Doe",
		Email:    randomEmail,
	}
}

func getTestUserData() *qbf.CreateUser {
	randomEmail := fmt.Sprintf("test.user.%d@example.com", rand.Intn(1000000))
	return &qbf.CreateUser{
		Name:     "Test",
		LastName: "User",
		Email:    randomEmail,
		Role:     "user",
	}
}

func getTestProductData() *qbf.CreateProduct {
	ref := fmt.Sprintf("REF-%d", rand.Intn(1000000))
	return &qbf.CreateProduct{
		Name:        "Test Product",
		Description: "A product used for testing purposes",
		Price:       19.99,
		Reference:   &ref,
	}
}
