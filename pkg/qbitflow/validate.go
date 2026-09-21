package qbf

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
)

// productTextDisallowed mirrors the API's `producttext` validator: characters commonly
// used in XSS / injection payloads are rejected outright.
const productTextDisallowed = "<>{}[]`\\|;\"~^"

// validateProductText mirrors the API's `producttext` binding rule.
//
// It accepts letters (including accented and non-Latin), digits, spaces and ordinary
// punctuation, and rejects angle brackets and other markup characters. Text that merely
// looks script-like (for example the literal "javascript:alert(1)") is allowed — the
// server renders these fields as escaped text.
func validateProductText(field, value string, min, max int) error {
	if strings.TrimSpace(value) == "" {
		return qberrors.NewValidationError(fmt.Sprintf("%s must not be blank", field))
	}

	// Length is counted in runes, matching the server's min/max on multi-byte text.
	if n := utf8.RuneCountInString(value); n < min || n > max {
		return qberrors.NewValidationError(
			fmt.Sprintf("%s must be between %d and %d characters", field, min, max))
	}

	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return qberrors.NewValidationError(
				fmt.Sprintf("%s must not contain control characters", field))
		}
		if strings.ContainsRune(productTextDisallowed, r) {
			return qberrors.NewValidationError(
				fmt.Sprintf("%s must not contain the character %q", field, r))
		}
	}

	return nil
}

// validateRedirectURL checks that a success/cancel redirect target is an absolute HTTP(S)
// URL.
//
// Requiring an absolute URL with an http/https scheme keeps the checkout page from being
// pointed at a relative path or a non-web scheme such as javascript: — a redirect target
// is attacker-visible, so the SDK refuses obviously unsafe values before the round-trip.
func validateRedirectURL(field, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return qberrors.NewValidationError(fmt.Sprintf("%s must be a valid URL", field))
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return qberrors.NewValidationError(
			fmt.Sprintf("%s must be an absolute http:// or https:// URL", field))
	}

	if parsed.Host == "" {
		return qberrors.NewValidationError(fmt.Sprintf("%s must include a host", field))
	}

	return nil
}

// validateInlineProduct applies the API's rules for an inline ("ghost") product supplied
// on a session checkout, plus the redirect URLs.
//
// These fields are all optional on the API (`binding:"omitempty,..."`), and the server
// treats an empty string exactly like an absent field. The SDK matches that: a nil pointer
// and a pointer to "" are both skipped, so `SuccessURL: new(os.Getenv("SUCCESS_URL"))`
// behaves the same here as it does against the API.
func validateInlineProduct(productName, description *string, price *float64, successURL, cancelURL *string) error {
	if isSet(productName) {
		if err := validateProductText("productName", *productName, 2, 100); err != nil {
			return err
		}
	}
	if isSet(description) {
		if err := validateProductText("description", *description, 2, 500); err != nil {
			return err
		}
	}
	if price != nil && *price < 0 {
		return qberrors.NewValidationError("price must be a non-negative value")
	}
	if isSet(successURL) {
		if err := validateRedirectURL("successUrl", *successURL); err != nil {
			return err
		}
	}
	if isSet(cancelURL) {
		if err := validateRedirectURL("cancelUrl", *cancelURL); err != nil {
			return err
		}
	}
	return nil
}

// isSet reports whether an optional string field was actually supplied. A nil pointer and
// a pointer to the empty string both mean "not provided", matching the API's omitempty.
func isSet(value *string) bool {
	return value != nil && *value != ""
}

// ValidateProductTextForTest exposes the producttext rule to the external test package.
//
// The rule is deliberately unexported on the request path; this thin wrapper exists so the
// test suite can assert the accept-side of the rule (text that looks script-like but
// contains no markup characters must NOT be rejected) without a network round-trip.
func ValidateProductTextForTest(field, value string, min, max int) error {
	return validateProductText(field, value, min, max)
}
