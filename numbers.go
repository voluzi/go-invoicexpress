package invoicexpress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// NullableString is a write field with three states, because two are not
// enough for a field whose "default" is expressed as null.
//
// Unset is omitted from the request entirely, which leaves whatever is stored
// alone — the API's PUT merges, so an absent field is "don't touch". Null is
// sent as JSON null, which is how the API restores a value to the account
// default. A value sets it.
//
// A plain string cannot express the middle state: the empty string either
// marshals as "" — a literal empty value, not a default — or, with omitempty,
// vanishes into "don't touch". Neither clears anything. A *string is no better:
// omitempty on a nil pointer omits the field rather than writing null.
//
// Pair it with the `omitzero` tag, which honours IsZero; `omitempty` does not
// and would emit null for the unset state.
type NullableString struct {
	set   bool
	value *string
}

// String is a NullableString carrying v.
func String(v string) NullableString { return NullableString{set: true, value: &v} }

// Null is a NullableString that writes JSON null, restoring the stored value to
// the account's own default.
func Null() NullableString { return NullableString{set: true} }

// IsZero reports whether the field is unset, so `omitzero` drops it from the
// request and the stored value is left as it is.
func (n NullableString) IsZero() bool { return !n.set }

// Value returns the string and whether one was set. A set-but-null field
// returns ("", false), the same as an unset one: neither names a value.
func (n NullableString) Value() (string, bool) {
	if n.value == nil {
		return "", false
	}
	return *n.value, true
}

func (n NullableString) MarshalJSON() ([]byte, error) {
	if n.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.value)
}

// UnmarshalJSON accepts a string or null. Both are "set": the API answering
// null is telling us the value is the account default, which is a fact, not an
// absence.
func (n *NullableString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	n.set = true
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invoicexpress: cannot unmarshal %s into NullableString", data)
	}
	n.value = &s
	return nil
}

// Rate is a percentage — a tax rate, not an amount — decoded tolerantly.
//
// The API is inconsistent about how it writes one, for the very same field:
// GET /taxes.json and GET /items.json answer "23.0" (a JSON string), while an
// embedded document tax has been observed as 23.0 (a JSON number). A plain
// float64 field decodes the second and fails the first with "cannot unmarshal
// string into Go value of type float64", which is how a tax table read can fail
// for an account that is perfectly well configured.
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
	if len(data) == 0 {
		*r = 0
		return nil
	}
	if string(data) == "null" {
		// An explicit null rate is an unknown rate, not 0%. The API sends null
		// for a tax's region and code, never for its value, so this is
		// something unexpected — and reading it as zero would let a caller
		// matching on the rate stamp this tax on an untaxed line. An absent
		// field is different: this method is never called for one.
		return errors.New("invoicexpress: cannot unmarshal null into Rate")
	}

	s := string(data)
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			// A blank rate is not 0%. The API writes an explicit "0.0" for a
			// zero-rated tax, so a blank is something unexpected — and reading
			// it as zero would let a caller matching on the rate pick this tax
			// for an untaxed line.
			return errors.New("invoicexpress: cannot unmarshal an empty string into Rate")
		}
		// Gate the string form exactly as Decimal does. ParseFloat alone would
		// accept "NaN", "Inf", "0x17p0", "1_0" and "+23" — and a NaN rate
		// compares unequal to every rate, so a caller matching the rate Stripe
		// charged would silently find no tax and issue nothing, or stamp a
		// nonsense percentage on a legally-binding document.
		if !validDecimal(s) {
			return fmt.Errorf("invoicexpress: invalid rate %q", s)
		}
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invoicexpress: cannot unmarshal %s into Rate", data)
	}
	// ParseFloat reports overflow but underflows quietly to zero: 1e-350, or a
	// decimal with enough leading zeros, would arrive as a 0% rate that nothing
	// wrote. Reject anything that is not actually zero but reads as zero.
	if f == 0 && !writesZero(s) {
		return fmt.Errorf("invoicexpress: rate %q underflows to zero", s)
	}
	*r = Rate(f)
	return nil
}

// writesZero reports whether s spells the number zero ("0", "0.00", "-0").
func writesZero(s string) bool {
	for _, r := range s {
		if r >= '1' && r <= '9' {
			return false
		}
	}
	return true
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
