package qbf

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	qberrors "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/errors"
	qbmodels "github.com/QBitFlow/qbitflow-go-sdk/v2/pkg/models"
)

// This file is the SDK's single set of client-side validation rules. They mirror the API's
// `binding` rules (and the behaviour verified against the live API) so invalid input fails
// before the round-trip with the same *qberrors.ValidationError type the API's own 400
// produces. The same rule set is implemented by the JavaScript, Python and PHP SDKs.

// productTextDisallowed mirrors the API's `producttext` validator: characters commonly
// used in XSS / injection payloads are rejected outright.
const productTextDisallowed = "<>{}[]`\\|;\"~^"

// maxUint32 is the largest value the API accepts for its uint32 fields (durations, minimum
// periods).
const maxUint32 = math.MaxUint32

// bareUUID matches a UUID in its canonical 8-4-4-4-12 hexadecimal form, without a prefix.
var bareUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// requireNonEmpty guards a required string argument.
//
// Without this an empty identifier builds a *different, valid* route — Get("") produces
// "/customer/uuid/", the collection endpoint — so the call quietly returns the wrong shape
// instead of failing fast.
//
// Label and wording match the other SDKs ("<Label> is required").
func requireNonEmpty(value, label string) error {
	if strings.TrimSpace(value) == "" {
		return qberrors.NewValidationError(label + " is required")
	}

	return nil
}

// requirePositive guards a required positive identifier.
//
// Zero is not a valid id on any resource, and interpolating it produces a request the
// server can only reject, so it is caught before the round-trip.
func requirePositive(value uint64, label string) error {
	if value == 0 {
		return qberrors.NewValidationError(label + " must be positive")
	}

	return nil
}

// requireArgument guards a required pointer argument (options, DTOs): a nil value would
// otherwise panic on the first field access.
func requireArgument(isNil bool, label string) error {
	if isNil {
		return qberrors.NewValidationError(label + " is required")
	}
	return nil
}

// validateLimit checks an optional cursor page size: when given it must be positive.
func validateLimit(limit *uint16) error {
	if limit != nil && *limit == 0 {
		return qberrors.NewValidationError("limit must be a positive integer")
	}
	return nil
}

// validateEmail is the SDK's structural email check: exactly one '@', a non-empty local
// part, a domain containing a '.' that neither starts nor ends with '.', and no whitespace.
// The address is never normalised (the API preserves case).
//
// The message never echoes the address, so user data does not end up in error strings
// or logs.
func validateEmail(field, value string) error {
	at := strings.Count(value, "@")
	if at != 1 {
		return qberrors.NewValidationError(field + " must be a valid email address")
	}
	local, domain, _ := strings.Cut(value, "@")
	if local == "" || domain == "" || !strings.Contains(domain, ".") ||
		strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") ||
		strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return qberrors.NewValidationError(field + " must be a valid email address")
	}
	return nil
}

// validateAlphanumSpace mirrors the API's `alphanumspace` binding rule with code-point
// length bounds: non-empty, and every character is a Unicode letter, a Unicode decimal
// digit (category Nd), a space, or one of - _ ' .  (names, last names).
//
// A whitespace-only value such as "   " is accepted, exactly as the API accepts it.
func validateAlphanumSpace(field, value string, min, max int) error {
	if value == "" {
		return qberrors.NewValidationError(fmt.Sprintf("%s is required", field))
	}

	if n := utf8.RuneCountInString(value); n < min || n > max {
		return qberrors.NewValidationError(
			fmt.Sprintf("%s must be between %d and %d characters", field, min, max))
	}

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.Is(unicode.Nd, r) {
			continue
		}
		switch r {
		case ' ', '-', '_', '\'', '.':
			continue
		}
		return qberrors.NewValidationError(
			fmt.Sprintf("%s may only contain letters, digits, spaces and - _ ' .", field))
	}

	return nil
}

// validateProductText mirrors the API's `producttext` binding rule.
//
// It rejects a value that is blank after trimming Unicode whitespace, and it accepts
// letters (including accented and non-Latin), digits, spaces and ordinary punctuation,
// rejecting the markup characters <>{}[]`\|;"~^ and control characters other than \n, \r
// and \t. Text that merely looks script-like (for example the literal
// "javascript:alert(1)") is allowed — the server renders these fields as escaped text.
// Length is counted in code points.
func validateProductText(field, value string, min, max int) error {
	if strings.TrimSpace(value) == "" {
		return qberrors.NewValidationError(fmt.Sprintf("%s must not be blank", field))
	}

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

// validatePrice checks a price: a finite number strictly greater than 0.
func validatePrice(field string, price float64) error {
	if math.IsNaN(price) || math.IsInf(price, 0) || price <= 0 {
		return qberrors.NewValidationError(field + " must be a finite number greater than 0")
	}
	return nil
}

// validateRedirectURL checks that a success/cancel redirect target is an absolute HTTP(S)
// URL with a host. The scheme is compared case-insensitively.
//
// Requiring an absolute URL with an http/https scheme keeps the checkout page from being
// pointed at a relative path or a non-web scheme such as javascript: — a redirect target
// is attacker-visible, so the SDK refuses obviously unsafe values before the round-trip.
func validateRedirectURL(field, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return qberrors.NewValidationError(fmt.Sprintf("%s must be a valid URL", field))
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return qberrors.NewValidationError(
			fmt.Sprintf("%s must be an absolute http:// or https:// URL", field))
	}

	if parsed.Host == "" {
		return qberrors.NewValidationError(fmt.Sprintf("%s must include a host", field))
	}

	return nil
}

// validateCustomerUUID checks an optional customer UUID: "" (or nil) means not provided;
// anything else must be a bare UUID — the API answers 400 to an empty, prefixed or
// malformed value.
func validateCustomerUUID(value *string) error {
	if !isSet(value) {
		return nil
	}
	if !bareUUID.MatchString(*value) {
		return qberrors.NewValidationError(
			"CustomerUUID must be a UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx, no prefix)")
	}
	return nil
}

// validateSessionProduct applies the API's product-resolution and inline ("ghost")
// product rules for a session checkout, plus the redirect URLs and the customer UUID.
//
// The caller must select a product by ID (> 0), by a non-empty reference, or inline with
// the full (productName + description + price) triplet. A price, when given, must be a
// finite number greater than 0 (the API rejects an inline price of 0). The inline fields,
// the URLs and the customer UUID are optional; a nil pointer and a pointer to "" both mean
// "not provided" (the empty value is left out of the request), so
// `SuccessURL: new(os.Getenv("SUCCESS_URL"))` works when the variable is unset.
func validateSessionProduct(productID *uint64, productReference, productName, description *string,
	price *float64, successURL, cancelURL, customerUUID *string) error {
	if productID != nil && *productID == 0 {
		return qberrors.NewValidationError("ProductID must be positive")
	}

	hasProduct := productID != nil || isSet(productReference)
	hasInline := isSet(productName) && isSet(description) && price != nil
	if !hasProduct && !hasInline {
		return qberrors.NewValidationError(
			"either ProductID, ProductReference, or ProductName, Description, and Price must be provided")
	}

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
	if price != nil {
		if err := validatePrice("price", *price); err != nil {
			return err
		}
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
	return validateCustomerUUID(customerUUID)
}

// validateDuration checks a subscription duration: an integer value within
// minValue..4294967295 (the API stores it as uint32) and one of the seven units the API
// accepts. A billing Frequency needs minValue 1; a TrialPeriod may be 0.
func validateDuration(field string, d qbmodels.Duration, minValue int) error {
	if d.Value < minValue || int64(d.Value) > maxUint32 {
		return qberrors.NewValidationError(
			fmt.Sprintf("%s.Value must be between %d and %d", field, minValue, uint64(maxUint32)))
	}
	switch d.Unit {
	case qbmodels.DurationUnitSeconds, qbmodels.DurationUnitMinutes, qbmodels.DurationUnitHours,
		qbmodels.DurationUnitDays, qbmodels.DurationUnitWeeks, qbmodels.DurationUnitMonths,
		qbmodels.DurationUnitYears:
		return nil
	}
	return qberrors.NewValidationError(
		field + ".Unit must be one of seconds, minutes, hours, days, weeks, months, years")
}

// validateRole mirrors the API's `oneof=admin user` rule on user creation. The other two
// tiers (handle, owner) exist on responses but cannot be assigned.
func validateRole(role qbmodels.UserRole) error {
	if role != qbmodels.RoleAdmin && role != qbmodels.RoleUser {
		return qberrors.NewValidationError("Role must be \"admin\" or \"user\"")
	}
	return nil
}

// maxOrganizationFeeBps is the API's upper bound for organizationFeeBps (50%).
const maxOrganizationFeeBps = 5000

// validateFeeBps mirrors the API's `min=0,max=5000` rule on organizationFeeBps.
func validateFeeBps(bps uint16) error {
	if bps > maxOrganizationFeeBps {
		return qberrors.NewValidationError(
			fmt.Sprintf("OrganizationFeeBps must be between 0 and %d", maxOrganizationFeeBps))
	}
	return nil
}

// accountingDateLayout is the API's `dateformat` rule (YYYY-MM-DD).
const accountingDateLayout = "2006-01-02"

// validateAccountingRange mirrors the API's rules for an accounting export: both dates
// are required, formatted YYYY-MM-DD and real calendar dates, and `from` is not after `to`
// (the API rejects an inverted range). The maximum span is left to the API.
func validateAccountingRange(from, to string) error {
	if from == "" || to == "" {
		return qberrors.NewValidationError("From and To dates are required")
	}
	fromDate, err := time.Parse(accountingDateLayout, from)
	if err != nil {
		return qberrors.NewValidationError("From must be a date formatted YYYY-MM-DD")
	}
	toDate, err := time.Parse(accountingDateLayout, to)
	if err != nil {
		return qberrors.NewValidationError("To must be a date formatted YYYY-MM-DD")
	}
	if toDate.Before(fromDate) {
		return qberrors.NewValidationError("To must not be before From")
	}
	return nil
}

// isSet reports whether an optional string field was actually supplied. A nil pointer and
// a pointer to the empty string both mean "not provided", matching the API's omitempty.
func isSet(value *string) bool {
	return value != nil && *value != ""
}

// omitEmpty returns value unless it points to "", in which case it returns nil so the field
// is left out of the request body entirely.
func omitEmpty(value *string) *string {
	if !isSet(value) {
		return nil
	}
	return value
}

// ValidateProductTextForTest exposes the producttext rule to the external test package.
//
// The rule is deliberately unexported on the request path; this thin wrapper exists so the
// test suite can assert the accept-side of the rule (text that looks script-like but
// contains no markup characters must NOT be rejected) without a network round-trip.
func ValidateProductTextForTest(field, value string, min, max int) error {
	return validateProductText(field, value, min, max)
}
