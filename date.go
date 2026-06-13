package invoicexpress

import (
	"fmt"
	"time"
)

const dateFormat = "02/01/2006"

// Date is a wrapper around time.Time that serializes/deserializes as dd/mm/yyyy.
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
	if string(data) == "null" || string(data) == `""` {
		return nil
	}
	s := string(data)
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
