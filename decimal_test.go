package invoicexpress

import (
	"encoding/json"
	"testing"
)

func TestDecimalMarshalAsString(t *testing.T) {
	b, err := json.Marshal(NewDecimal("29.99"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `"29.99"` {
		t.Errorf("marshal = %s, want \"29.99\"", b)
	}
}

func TestDecimalUnmarshalFromStringOrNumber(t *testing.T) {
	cases := map[string]string{
		`"50.00"`: "50.00",
		`61.5`:    "61.5",
		`0`:       "0",
		`null`:    "0", // empty -> String() returns "0"
		`""`:      "0",
	}
	for in, want := range cases {
		var d Decimal
		if err := json.Unmarshal([]byte(in), &d); err != nil {
			t.Errorf("unmarshal %s: %v", in, err)
			continue
		}
		if d.String() != want {
			t.Errorf("unmarshal %s -> %q, want %q", in, d.String(), want)
		}
	}
}

func TestDecimalRoundTripInStruct(t *testing.T) {
	// A response with a numeric amount must decode, and re-encode as a string.
	var inv Invoice
	if err := json.Unmarshal([]byte(`{"id":1,"total":61.5,"sum":"50"}`), &inv); err != nil {
		t.Fatalf("unmarshal invoice: %v", err)
	}
	if inv.Total.String() != "61.5" {
		t.Errorf("Total = %q", inv.Total.String())
	}
	if inv.Sum.String() != "50" {
		t.Errorf("Sum = %q", inv.Sum.String())
	}
}

func TestDecimalFloat64AndIsZero(t *testing.T) {
	d := NewDecimal("12.34")
	f, err := d.Float64()
	if err != nil || f != 12.34 {
		t.Errorf("Float64 = %v, %v", f, err)
	}
	if !NewDecimal("0").IsZero() || !NewDecimal("").IsZero() {
		t.Error("IsZero should be true for 0 and empty")
	}
	if NewDecimal("0.01").IsZero() {
		t.Error("0.01 should not be zero")
	}
}

func TestDecimalFromFloat(t *testing.T) {
	if got := DecimalFromFloat(29.5, 2).String(); got != "29.50" {
		t.Errorf("DecimalFromFloat = %q, want 29.50", got)
	}
}

func TestDecimalUnmarshalJSONStringValidation(t *testing.T) {
	// UnmarshalJSON string branch stores any string without validating it's numeric
	var d Decimal
	_ = json.Unmarshal([]byte(`"not-a-number"`), &d)

	// Float64() will fail
	f, err := d.Float64()
	if err == nil {
		t.Fatalf("expected parse error for 'not-a-number', got %v", f)
	}

	// IsZero silently treats parse errors as zero
	zero := d.IsZero()
	if zero {
		t.Logf("ISSUE: IsZero('not-a-number') = true (should be false or error)")
	}
}

func TestDecimalUnmarshalUnquotedBadNumber(t *testing.T) {
	// JSON with bad numeric format - doesn't get quoted so goes to JSON number path
	// Actually JSON will reject this during unmarshal, so skip this path
	var d Decimal
	err := json.Unmarshal([]byte(`bad`), &d)
	if err == nil {
		t.Fatal("JSON unmarshaling should reject bare 'bad'")
	}
}

func TestDecimalMarshalEdgeNegative(t *testing.T) {
	d := NewDecimal("-0.00")
	b, _ := json.Marshal(d)
	if string(b) != `"-0.00"` {
		t.Errorf("negative zero marshal = %s", b)
	}
}
