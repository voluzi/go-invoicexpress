package invoicexpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// decimalRe matches a plain decimal number: optional leading minus, digits,
// optional fractional part. Scientific notation, NaN/Inf and a leading plus are
// rejected — none are valid for a monetary amount.
var decimalRe = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

func validDecimal(s string) bool { return decimalRe.MatchString(s) }

// Decimal represents a monetary or quantity value as an exact decimal string,
// avoiding float64 rounding error in a financial context (this library issues
// legally-binding invoices). It marshals to a JSON string — which the
// InvoiceXpress API expects for amounts — and unmarshals tolerantly from
// either a JSON string ("50.00") or a JSON number (50.0), since the API uses
// both shapes across endpoints.
type Decimal struct {
	s string
}

// NewDecimal builds a Decimal from its exact string representation, e.g.
// NewDecimal("29.99"). Whitespace is trimmed. It does NOT validate the format —
// use ParseDecimal for untrusted input, or check Valid afterwards.
func NewDecimal(s string) Decimal { return Decimal{s: strings.TrimSpace(s)} }

// ParseDecimal builds a Decimal from a string, returning an error if it is not
// a valid plain decimal number. An empty string yields the zero value.
func ParseDecimal(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, nil
	}
	if !validDecimal(s) {
		return Decimal{}, fmt.Errorf("invoicexpress: invalid decimal %q", s)
	}
	return Decimal{s: s}, nil
}

// Valid reports whether the Decimal holds a syntactically valid decimal number.
func (d Decimal) Valid() bool { return validDecimal(d.s) }

// DecimalFromFloat builds a Decimal from a float64 fixed to places decimals
// (use 2 for currency). Prefer NewDecimal when you already have an exact
// string (e.g. an amount from Stripe) to avoid any float round-trip.
func DecimalFromFloat(f float64, places int) Decimal {
	return Decimal{s: strconv.FormatFloat(f, 'f', places, 64)}
}

// String returns the exact decimal text ("0" when unset).
func (d Decimal) String() string {
	if d.s == "" {
		return "0"
	}
	return d.s
}

// IsZero reports whether the value is unset or numerically zero. It works
// purely on the string so extreme magnitudes never lose precision through a
// float round-trip: a value is zero iff, after an optional sign and decimal
// point, every digit is '0'.
func (d Decimal) IsZero() bool {
	s := strings.TrimSpace(d.s)
	if s == "" {
		return true
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	sawDigit := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '0':
			sawDigit = true
		case '.':
		default:
			return false
		}
	}
	return sawDigit
}

// Float64 parses the decimal into a float64. Use only for display or
// non-authoritative aggregation — never as the canonical stored value.
func (d Decimal) Float64() (float64, error) {
	if d.s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(d.s, 64)
}

// MarshalJSON emits the value as a JSON string. It uses json.Marshal so any
// quotes/backslashes/control characters are escaped — a raw concatenation would
// produce invalid JSON for an unusual value.
func (d Decimal) MarshalJSON() ([]byte, error) {
	s := d.s
	if s == "" {
		s = "0"
	}
	return json.Marshal(s)
}

// UnmarshalJSON accepts a JSON string ("50.00") or a JSON number (50.0). Any
// other token (object, array, boolean) and any non-numeric content is rejected
// with an error rather than silently stored.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		d.s = ""
		return nil
	}

	var s string
	switch data[0] {
	case '"':
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		s = string(data)
	default:
		return fmt.Errorf("invoicexpress: cannot unmarshal %s into Decimal", data)
	}

	s = strings.TrimSpace(s)
	if s == "" {
		d.s = ""
		return nil
	}
	if !validDecimal(s) {
		return fmt.Errorf("invoicexpress: invalid decimal %q", s)
	}
	d.s = s
	return nil
}
