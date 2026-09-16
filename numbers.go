package invoicexpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Rate is a percentage — a tax rate, not an amount — decoded tolerantly.
//
// The API is inconsistent about how it writes one: GET /taxes.json returns
// "23.0" (a JSON string) while the tax embedded in a document returns 23.0 (a
// JSON number). A plain float64 field decodes the second and fails the first
// with "cannot unmarshal string into Go value of type float64", which is how a
// tax table read can fail for an account that is perfectly well configured.
//
// Amounts keep using Decimal: they are money, and float64 rounding is not
// acceptable on a legally-binding document. A rate is matched against a
// known percentage with a tolerance, never summed, so float64 is safe here.
type Rate float64

// UnmarshalJSON accepts a JSON number (23.0), a JSON string ("23.0") or null
// (zero). Any other token, and any non-numeric string, is rejected rather than
// silently read as zero — a tax silently at 0% would issue a wrong invoice.
func (r *Rate) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*r = 0
		return nil
	}

	s := string(data)
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*r = 0
			return nil
		}
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invoicexpress: cannot unmarshal %s into Rate", data)
	}
	*r = Rate(f)
	return nil
}

// MarshalJSON emits a JSON number, which is what the API expects when a rate
// is sent (tax create/update).
func (r Rate) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(r))
}

// Float64 returns the rate as a float64, for arithmetic and comparisons.
func (r Rate) Float64() float64 { return float64(r) }

// Flag is a boolean decoded tolerantly.
//
// The API writes booleans as 1/0 in some payloads ("default_tax": 1,
// "default_sequence": 1) and as true/false in others ("archived": false). A
// plain bool field fails the first with "cannot unmarshal number into Go value
// of type bool". Its underlying type is bool, so a Flag still reads as one at
// the call site: `if tax.IsDefault`.
type Flag bool

// UnmarshalJSON accepts true/false, 1/0, "1"/"0", "true"/"false" and null
// (false). Any other value is rejected rather than guessed at.
func (f *Flag) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*f = false
		return nil
	}

	s := string(data)
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = false
			return nil
		}
	}

	switch strings.ToLower(s) {
	case "true", "1", "1.0":
		*f = true
	case "false", "0", "0.0":
		*f = false
	default:
		return fmt.Errorf("invoicexpress: cannot unmarshal %s into Flag", data)
	}
	return nil
}

// MarshalJSON emits a JSON boolean.
func (f Flag) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(f))
}

// Bool returns the flag as a plain bool.
func (f Flag) Bool() bool { return bool(f) }
