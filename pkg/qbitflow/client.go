package qbitflow

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	qberrors "github.com/qbitflow/qbitflow-go-sdk/pkg/errors"
	qbmodels "github.com/qbitflow/qbitflow-go-sdk/pkg/models"
)

const (
	defaultBaseURL = "https://api.qbitflow.app/v1"
	defaultTimeout = 30 * time.Second
)

// Client represents the QBitFlow API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Config represents configuration for the QBitFlow client
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new QBitFlow client with the given API key
func NewClient(apiKey string) *Client {
	return NewClientWithConfig(Config{
		APIKey:  apiKey,
		BaseURL: defaultBaseURL,
		Timeout: defaultTimeout,
	})
}

// NewClientWithConfig creates a new QBitFlow client with custom configuration
func NewClientWithConfig(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}

	return &Client{
		apiKey:  config.APIKey,
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// makeRequest makes an HTTP request to the QBitFlow API
func (c *Client) makeRequest(method, endpoint string, body any, result any) error {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return qberrors.NewQBitFlowError("failed to marshal request body", 0, err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return qberrors.NewQBitFlowError("failed to create request", 0, err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return qberrors.NewQBitFlowError("request failed", 0, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return qberrors.NewQBitFlowError("failed to read response body", resp.StatusCode, err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp.StatusCode, respBody)
	}

	// Parse success response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return qberrors.NewQBitFlowError("failed to parse response", resp.StatusCode, err)
		}
	}

	return nil
}

// handleErrorResponse handles error responses from the API
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Handle bad request, can contain validation errors
	if statusCode == 400 {
		var validationErrors qberrors.ValidationErrors
		if err := json.Unmarshal(body, &validationErrors); err == nil {
			return qberrors.NewValidationErrorFromList(validationErrors)
		}

		// Try and parse unique validation error
		var singleError qberrors.ValidationError
		if err := json.Unmarshal(body, &singleError); err == nil {
			return &singleError
		}
	}

	var errResp qbmodels.ErrorResponse

	// Try to parse error response
	if err := json.Unmarshal(body, &errResp); err != nil {
		// If parsing fails, return generic error
		return qberrors.NewQBitFlowError(string(body), statusCode, nil)
	}

	message := errResp.Error
	if errResp.Message != "" {
		message = errResp.Message
	}

	// Handle 404 specifically
	if statusCode == 404 {
		return qberrors.NewNotFoundError(message)
	}

	return qberrors.NewQBitFlowError(message, statusCode, nil)
}

// SetBaseURL sets a custom base URL for the client
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = strings.TrimSuffix(baseURL, "/")
}

// SetTimeout sets a custom timeout for HTTP requests
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}
