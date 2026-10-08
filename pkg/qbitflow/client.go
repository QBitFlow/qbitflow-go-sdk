package qbf

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v3/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v3/pkg/models"
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

	// onBehalfOfUser, when non-zero, causes requests to be executed on behalf
	// of the given user via the On-Behalf-Of header. Requires an admin-level
	// (organization) API key.
	onBehalfOfUser uint64
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

// withOnBehalfOf returns a shallow copy of the client configured to act on
// behalf of the given user. The underlying http.Client is shared so that
// connection pooling is preserved; only the on-behalf-of target differs.
func (c *Client) withOnBehalfOf(userID uint64) *Client {
	clone := *c
	clone.onBehalfOfUser = userID
	return &clone
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
	if c.onBehalfOfUser != 0 {
		req.Header.Set("On-Behalf-Of", strconv.FormatUint(c.onBehalfOfUser, 10))
	}

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

	// Handle error responses. Anything outside 2xx is an error: a 3xx reaching this point
	// means the request was not fulfilled (http.Client already followed the redirects it
	// is willing to follow), so parsing its body as a result would hand the caller a
	// zero-valued struct instead of surfacing the problem.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
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

// handleErrorResponse converts a non-2xx response into a typed SDK error.
//
// The API uses two error envelopes: a field-level list for validation failures,
//
//	{"errors":[{"field":"Price","message":"Price is too short"}]}
//
// and a single message for everything else,
//
//	{"error":"resource not found"}
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	isValidationStatus := statusCode == 400 || statusCode == 422

	// Choose the envelope by attempting the decode, not by substring-matching the body:
	// "errors" occurs inside plenty of plain messages, and a message that merely contains
	// the word "error" is not an envelope.
	if isValidationStatus {
		var validationErrors qberrors.ValidationErrors
		if err := json.Unmarshal(body, &validationErrors); err == nil &&
			len(validationErrors.Errors) > 0 {
			return qberrors.NewValidationErrorFromList(validationErrors)
		}
	}

	var errResp qbmodels.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Not JSON at all — keep the raw body, which is usually a gateway or proxy page
		// and is the only clue the caller has.
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(statusCode)
		}
		if isValidationStatus {
			return qberrors.NewValidationError(message)
		}
		return qberrors.NewQBitFlowError(message, statusCode, nil)
	}

	message := errResp.Error
	if message == "" {
		message = errResp.Message
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch {
	case statusCode == 404:
		return qberrors.NewNotFoundError(message)
	case isValidationStatus:
		return qberrors.NewValidationError(message)
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
