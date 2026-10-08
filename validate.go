package qbitflow

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// This file is the SDK's one set of client-side checks (behaviour §8, docs validators.md).
// They mirror the API's binding rules so a bad input fails before the round trip, with a
// *ValidationError naming each failing input by its wire name. Lengths count code points.
// Rules that depend on the key's mode or on server state (test-mode price cap, https-only
// URLs in live mode, the frequency minimum, the 95-day export window, uniqueness) are left to
// the API.

// validator collects the field errors of one params value.
type validator struct {
	fields []FieldError
}

// add records a failing input.
func (v *validator) add(field, format string, args ...any) {
	v.fields = append(v.fields, FieldError{Field: field, Message: field + " " + fmt.Sprintf(format, args...)})
}

// err returns the collected failures as a *ValidationError, or nil.
func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return newValidationError("validation failed", v.fields...)
}

// required checks that a required string is set (not blank). It reports whether it is.
func (v *validator) required(field, value string) bool {
	if strings.TrimSpace(value) == "" {
		v.add(field, "is required")
		return false
	}
	return true
}

// length checks a code-point length (maxLen 0 = no maximum). It reports whether it passed.
func (v *validator) length(field, value string, minLen, maxLen int) bool {
	n := utf8.RuneCountInString(value)
	switch {
	case maxLen > 0 && minLen > 0 && (n < minLen || n > maxLen):
		v.add(field, "must be %d to %d characters", minLen, maxLen)
	case minLen > 0 && n < minLen:
		v.add(field, "must be at least %d characters", minLen)
	case maxLen > 0 && n > maxLen:
		v.add(field, "must be at most %d characters", maxLen)
	default:
		return true
	}
	return false
}

// textDisallowed are the characters names and free text refuse (validators.md).
const textDisallowed = "<>{}[]`\\|;\"~^"

// bidiControls reorder the text around them (U+202A–U+202E, U+2066–U+2069).
var bidiControls = &unicode.RangeTable{R16: []unicode.Range16{{Lo: 0x202a, Hi: 0x202e, Stride: 1}, {Lo: 0x2066, Hi: 0x2069, Stride: 1}}}

// isText is the API's text rule: not blank, no control character (line breaks and tabs only
// when multiline), none of textDisallowed.
func isText(value string, multiline bool) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if multiline && (r == '\n' || r == '\r' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) || strings.ContainsRune(textDisallowed, r) {
			return false
		}
	}
	return true
}

// isName is the API's name rule: the text rule on one line, without bidirectional controls.
func isName(value string) bool {
	return isText(value, false) && !strings.ContainsFunc(value, func(r rune) bool { return unicode.Is(bidiControls, r) })
}

// name checks a name (product, customer) when set: the name rule and minLen..maxLen.
func (v *validator) name(field, value string, minLen, maxLen int) {
	if value == "" || !v.length(field, value, minLen, maxLen) {
		return
	}
	if !isName(value) {
		v.add(field, "must be one line, not blank, without < > { } [ ] ` \\ | ; \" ~ ^")
	}
}

// text checks free text (descriptions, addresses, messages) when set: the text rule and
// minLen..maxLen.
func (v *validator) text(field, value string, minLen, maxLen int) {
	if value == "" || !v.length(field, value, minLen, maxLen) {
		return
	}
	if !isText(value, true) {
		v.add(field, "must not be blank nor contain control characters or < > { } [ ] ` \\ | ; \" ~ ^")
	}
}

var referencePattern = regexp.MustCompile(`^[A-Za-z0-9._:@-]+$`)

// reference checks a merchant reference when set: 1 to 100 of A-Z a-z 0-9 . _ : @ -.
func (v *validator) reference(field, value string) {
	if value == "" || !v.length(field, value, 1, 100) {
		return
	}
	if !referencePattern.MatchString(value) {
		v.add(field, "must contain only letters, digits and . _ : @ -")
	}
}

var phonePattern = regexp.MustCompile(`^\+?[0-9][0-9 ().-]{4,}[0-9]$`)

// phone checks a phone number when set: at most 32 characters, an optional +, digits and
// the separators space . - ( ), starting and ending with a digit.
func (v *validator) phone(field, value string) {
	if value == "" || !v.length(field, value, 0, 32) {
		return
	}
	if !phonePattern.MatchString(value) {
		v.add(field, "must be a valid phone number")
	}
}

// isEmail is the SDK's structural email check: one @, a non-empty local part, a dotted domain
// that neither starts nor ends with a dot, no whitespace. Never normalised.
func isEmail(value string) bool {
	if strings.Count(value, "@") != 1 || strings.ContainsFunc(value, unicode.IsSpace) {
		return false
	}
	local, domain, _ := strings.Cut(value, "@")
	return local != "" && strings.Contains(domain, ".") &&
		!strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// email checks an email address when set: at most 254 characters, structurally valid.
func (v *validator) email(field, value string) {
	if value == "" || !v.length(field, value, 0, 254) {
		return
	}
	if !isEmail(value) {
		v.add(field, "must be a valid email address")
	}
}

// isHTTPURL reports whether value is an absolute http(s) URL with a host.
func isHTTPURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

// url checks a URL when set: at most 2048 characters, absolute http or https with a host.
func (v *validator) url(field, value string) {
	if value == "" || !v.length(field, value, 0, 2048) {
		return
	}
	if !isHTTPURL(value) {
		v.add(field, "must be an absolute http or https URL")
	}
}

// price checks a price in USD: finite and above 0.
func (v *validator) price(field string, value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		v.add(field, "must be a number above 0")
	}
}

// percent checks a rate in percent: finite, from 0 (above 0 when positive) to maxPercent,
// with at most 2 decimals (1 bps).
func (v *validator) percent(field string, value, maxPercent float64, positive bool) {
	low := "0"
	if positive {
		low = "above 0"
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || (positive && value == 0) || value > maxPercent || !hasAtMostTwoDecimals(value) {
		v.add(field, "must be a percentage from %s to %s, with at most 2 decimals", low, strconv.FormatFloat(maxPercent, 'f', -1, 64))
	}
}

// usd checks an amount in USD: finite, above 0, at most maxUSD, with at most 2 decimals (the
// cent) in its shortest decimal form.
func (v *validator) usd(field string, value, maxUSD float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > maxUSD || !hasAtMostTwoDecimals(value) {
		v.add(field, "must be an amount in USD above 0 and at most %s, with at most 2 decimals", strconv.FormatFloat(maxUSD, 'f', -1, 64))
	}
}

// hasAtMostTwoDecimals reports whether value's shortest decimal form has at most 2 decimals
// (the form encoding/json sends).
func hasAtMostTwoDecimals(value float64) bool {
	s := strconv.FormatFloat(value, 'f', -1, 64)
	_, frac, found := strings.Cut(s, ".")
	return !found || len(frac) <= 2
}

// unitSeconds is each DurationUnit's length (months = 30 days, years = 365 days).
var unitSeconds = map[DurationUnit]uint64{
	DurationUnitSeconds: 1,
	DurationUnitMinutes: 60,
	DurationUnitHours:   3600,
	DurationUnitDays:    86400,
	DurationUnitWeeks:   7 * 86400,
	DurationUnitMonths:  30 * 86400,
	DurationUnitYears:   365 * 86400,
}

// duration checks a Duration: a unit (one of the 7) is required when value > 0, and a unit
// given with 0 must be one of them too. A frequency must be at least 1 unit and at most 1 year.
func (v *validator) duration(field string, d Duration, frequency bool) {
	seconds, known := unitSeconds[d.Unit]
	switch {
	case d.Value > 0 && d.Unit == "":
		v.add(field+".unit", "is required when value is above 0")
		return
	case d.Unit != "" && !known:
		v.add(field+".unit", "must be one of seconds, minutes, hours, days, weeks, months, years")
		return
	}
	if !frequency {
		return
	}
	switch {
	case d.Value < 1:
		v.add(field+".value", "must be at least 1")
	case uint64(d.Value)*seconds > 365*86400:
		v.add(field, "must be at most 1 year")
	}
}

// intRange checks an integer bound.
func (v *validator) intRange(field string, value, minValue, maxValue int64) {
	if value < minValue || value > maxValue {
		v.add(field, "must be from %d to %d", minValue, maxValue)
	}
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID reports whether value is a UUID in its 8-4-4-4-12 hex form (any case).
func isUUID(value string) bool { return uuidPattern.MatchString(value) }

// isNilUUID reports whether value is the nil UUID.
func isNilUUID(value string) bool { return value == "00000000-0000-0000-0000-000000000000" }

// uuid checks a UUID when set.
func (v *validator) uuid(field, value string) {
	if value != "" && !isUUID(value) {
		v.add(field, "must be a UUID")
	}
}

// txPrefixes are the prefixes of the API's transaction ids (pay@…, sub-hist@…).
var txPrefixes = []string{"pay", "sub", "payg", "sub-hist", "refund", "transfer"}

// isTxID reports whether value is a transaction id: <prefix>@<uuid>, or a bare UUID.
func isTxID(value string) bool {
	prefix, id, found := strings.Cut(value, "@")
	if !found {
		return isUUID(value)
	}
	for _, p := range txPrefixes {
		if prefix == p {
			return isUUID(id)
		}
	}
	return false
}

// txID checks a transaction id (pay@…, sub@…, sub-hist@…, refund@…, or a bare UUID) when set.
func (v *validator) txID(field, value string) {
	if value != "" && !isTxID(value) {
		v.add(field, "must be a transaction id (pay@…, sub@…, sub-hist@… or a UUID)")
	}
}

// date checks a YYYY-MM-DD calendar date. It returns the date and whether it is valid.
func (v *validator) date(field, value string) (time.Time, bool) {
	if !v.required(field, value) {
		return time.Time{}, false
	}
	d, err := time.Parse(time.DateOnly, value)
	if err != nil {
		v.add(field, "must be a date (YYYY-MM-DD)")
		return time.Time{}, false
	}
	return d, true
}

// dateRange checks an export window: two YYYY-MM-DD dates, from <= to. The API enforces the
// window's maximum (95 days).
func (v *validator) dateRange(fromField, from, toField, to string) {
	f, okFrom := v.date(fromField, from)
	t, okTo := v.date(toField, to)
	if okFrom && okTo && f.After(t) {
		v.add(toField, "must not be before %s", fromField)
	}
}

// exclusive checks that two filters are not both set.
func (v *validator) exclusive(fieldA string, a bool, fieldB string, b bool) {
	if a && b {
		v.add(fieldB, "cannot be combined with %s", fieldA)
	}
}

// oneOf checks an enum value when set.
func oneOf[T ~string](v *validator, field string, value T, allowed ...T) {
	if value == "" {
		return
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	names := make([]string, len(allowed))
	for i, a := range allowed {
		names[i] = string(a)
	}
	v.add(field, "must be one of %s", strings.Join(names, ", "))
}

// endpointEvents checks a webhook endpoint's event list: at most 20, never webhook.test.
func (v *validator) endpointEvents(field string, events []EventType) {
	if len(events) > 20 {
		v.add(field, "must list at most 20 event types")
	}
	for i, e := range events {
		switch {
		case e == "":
			v.add(fmt.Sprintf("%s[%d]", field, i), "is required")
		case e == EventTypeWebhookTest:
			v.add(fmt.Sprintf("%s[%d]", field, i), "cannot be webhook.test (sent by the endpoint test only)")
		}
	}
}

// isIdempotencyKey reports whether key is a valid Idempotency-Key: 1 to 255 printable ASCII
// characters without spaces (0x21–0x7E).
func isIdempotencyKey(key string) bool {
	if len(key) < 1 || len(key) > 255 {
		return false
	}
	for i := range len(key) {
		if key[i] < 0x21 || key[i] > 0x7e {
			return false
		}
	}
	return true
}

// isRequestID reports whether id is a valid X-Request-Id: 1 to 128 of A-Z a-z 0-9 - _ . :.
func isRequestID(id string) bool {
	if len(id) < 1 || len(id) > 128 {
		return false
	}
	for i := range len(id) {
		c := id[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}

// checkPathUUID validates a UUID path parameter (members, invitations, endpoints, customers,
// products): an empty one would address another route.
func checkPathUUID(field, value string) error {
	if value == "" {
		return newFieldError(field, "is required")
	}
	if !isUUID(value) {
		return newFieldError(field, "must be a UUID")
	}
	return nil
}

// checkPathTxID validates a transaction id path parameter (payments, subscriptions, bills,
// sessions).
func checkPathTxID(field, value string) error {
	if value == "" {
		return newFieldError(field, "is required")
	}
	if !isTxID(value) {
		return newFieldError(field, "must be a transaction id (pay@…, sub@…, sub-hist@… or a UUID)")
	}
	return nil
}

// checkPathRequired validates a free-form path parameter (a reference, an email, an event id):
// only its presence, the API judges the rest.
func checkPathRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return newFieldError(field, "is required")
	}
	// "." and ".." would be resolved as relative path segments (by URL parsers and by the
	// server's path cleaning) and reach another route: refused, identically in every SDK.
	if value == "." || value == ".." {
		return newFieldError(field, `cannot be "." or ".."`)
	}
	return nil
}
