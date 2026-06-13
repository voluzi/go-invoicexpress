package invoicexpress

import (
	"fmt"
	"strings"
	"time"
)

const dateFormat = "02/01/2006"

// Date is a wrapper around time.Time that serializes/deserializes as dd/mm/yyyy.
//
// Note: because Date is a struct, a `json:"...,omitempty"` tag on a Date field
// does NOT omit a zero value — encoding/json's omitempty only applies to basic
// kinds. A zero Date therefore marshals to JSON null (see MarshalJSON), which
// InvoiceXpress treats the same as an absent optional date (verified against
// the live API). If you need a field to be omitted entirely, use a *Date.
type Date struct {
	time.Time
}

// NewDate creates a Date from a time.Time value.
func NewDate(t time.Time) Date {
	return Date{Time: t}
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format(dateFormat) + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" || s == `""` {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return fmt.Errorf("invoicexpress: parse date %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// String returns the date in dd/mm/yyyy format.
func (d Date) String() string {
	return d.Format(dateFormat)
}
