package qbitflow

import (
	"strconv"
	"strings"
)

// maxAmountDecimals bounds the decimals FormatAmount and ParseAmount take (uint256's 78 digits).
const maxAmountDecimals = 77

// FormatAmount turns an amount in a token's smallest unit (an AmountMinUnits decimal string)
// into a decimal amount, exactly (string arithmetic, never a float): FormatAmount("1500000", 6)
// is "1.5", ("1", 6) is "0.000001", ("-10004200", 6) is "-10.0042". Leading zeros and the
// fraction's trailing zeros are trimmed, and there is never an exponent.
//
// minUnits must be an optional "-" and digits, decimals 0 to 77 (a currency's Decimals);
// anything else is a *ValidationError.
func FormatAmount(minUnits string, decimals int) (string, error) {
	if err := checkAmountDecimals(decimals); err != nil {
		return "", err
	}
	digits, negative := strings.CutPrefix(minUnits, "-")
	if !isASCIIDigits(digits) {
		return "", newFieldError("minUnits", "must be an integer (digits, optionally after a -)")
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return "0", nil
	}
	if len(digits) <= decimals {
		digits = strings.Repeat("0", decimals-len(digits)+1) + digits
	}
	whole, fraction := digits[:len(digits)-decimals], strings.TrimRight(digits[len(digits)-decimals:], "0")
	out := whole
	if fraction != "" {
		out += "." + fraction
	}
	if negative {
		out = "-" + out
	}
	return out, nil
}

// ParseAmount turns a decimal amount into the token's smallest unit, exactly: ParseAmount("1.5",
// 6) is "1500000", ("0.000001", 6) is "1", ("-0.5", 2) is "-50", "-0" is "0".
//
// amount must be an optional "-", digits, and optionally "." and digits (no exponent, no
// grouping, no "+"), with at most decimals fractional digits ("1.0000001" with 6 decimals is
// refused rather than rounded); decimals is 0 to 77. Anything else is a *ValidationError.
func ParseAmount(amount string, decimals int) (string, error) {
	if err := checkAmountDecimals(decimals); err != nil {
		return "", err
	}
	unsigned, negative := strings.CutPrefix(amount, "-")
	whole, fraction, hasPoint := strings.Cut(unsigned, ".")
	if !isASCIIDigits(whole) || (hasPoint && !isASCIIDigits(fraction)) {
		return "", newFieldError("amount", "must be a decimal number (digits, optionally a - and a . with digits after it)")
	}
	if len(fraction) > decimals {
		return "", newFieldError("amount", "has more than "+strconv.Itoa(decimals)+" decimals")
	}
	digits := strings.TrimLeft(whole+fraction+strings.Repeat("0", decimals-len(fraction)), "0")
	if digits == "" {
		return "0", nil
	}
	if negative {
		digits = "-" + digits
	}
	return digits, nil
}

// FormatAmount formats an amount in this currency's smallest unit with its Decimals (see
// FormatAmount): USDC's "1500000" is "1.5". A nil currency is a *ValidationError.
func (c *Currency) FormatAmount(minUnits string) (string, error) {
	if c == nil {
		return "", newFieldError("currency", "is required")
	}
	return FormatAmount(minUnits, int(c.Decimals))
}

func checkAmountDecimals(decimals int) error {
	if decimals < 0 || decimals > maxAmountDecimals {
		return newFieldError("decimals", "must be between 0 and 77")
	}
	return nil
}
