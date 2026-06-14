package invoicexpress

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

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
// NewDecimal("29.99"). Whitespace is trimmed.
func NewDecimal(s string) Decimal { return Decimal{s: strings.TrimSpace(s)} }

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

// IsZero reports whether the value is unset or numerically zero.
func (d Decimal) IsZero() bool {
	if d.s == "" {
		return true
	}
	f, err := strconv.ParseFloat(d.s, 64)
	return err == nil && f == 0
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

// UnmarshalJSON accepts either a JSON string or a JSON number. The string case
// is decoded via json.Unmarshal so it handles escaping and rejects malformed
// input with an error rather than panicking.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		d.s = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		d.s = strings.TrimSpace(s)
		return nil
	}
	d.s = string(data)
	return nil
}
