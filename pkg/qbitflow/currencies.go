package qbf

import (
	"fmt"

	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// CurrencyService exposes the supported-currency lookups. These endpoints are public
// (no authentication required); use them to resolve the currency IDs returned in
// SessionCheckout.AvailableCurrencies and on payment/subscription records.
type CurrencyService struct {
	client *Client
}

// NewCurrencyService creates the currency service bound to client.
func NewCurrencyService(client *Client) *CurrencyService {
	return &CurrencyService{client: client}
}

// GetAllAvailable returns all supported currencies, native currencies and tokens alike.
// Pass test=true to list test-network currencies.
func (s *CurrencyService) GetAllAvailable(test bool) ([]qbmodels.Currency, error) {
	return getList[qbmodels.Currency](s.client, fmt.Sprintf("/utils/all-available-currencies?test=%t", test))
}

// GetAllMain returns only the main (native/blockchain) currencies, excluding tokens.
// Pass test=true to list test-network currencies.
func (s *CurrencyService) GetAllMain(test bool) ([]qbmodels.Currency, error) {
	return getList[qbmodels.Currency](s.client, fmt.Sprintf("/utils/all-main-currencies?test=%t", test))
}
