package invoicexpress

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateMarshalJSON(t *testing.T) {
	d := NewDate(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC))
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `"09/03/2026"` {
		t.Errorf("marshal = %s, want \"09/03/2026\"", b)
	}
}

func TestDateMarshalZeroIsNull(t *testing.T) {
	var d Date
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("zero Date should marshal to null, got %s", b)
	}
}

func TestDateUnmarshalRoundTrip(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"25/12/2026"`), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.Year() != 2026 || d.Month() != time.December || d.Day() != 25 {
		t.Errorf("unmarshalled wrong date: %v", d.Time)
	}
	if d.String() != "25/12/2026" {
		t.Errorf("String() = %s", d.String())
	}
}

func TestDateUnmarshalNullAndEmpty(t *testing.T) {
	for _, in := range []string{`null`, `""`} {
		var d Date
		if err := json.Unmarshal([]byte(in), &d); err != nil {
			t.Errorf("unmarshal %s: %v", in, err)
		}
		if !d.IsZero() {
			t.Errorf("unmarshal %s should leave zero Date", in)
		}
	}
}

func TestDateUnmarshalInvalid(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`"2026-03-09"`), &d); err == nil {
		t.Error("expected error for wrong date format")
	}
}
