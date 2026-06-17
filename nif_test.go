package invoicexpress

import "testing"

func TestValidPortugueseNIF(t *testing.T) {
	valid := []string{
		"999999990", // official Consumidor Final (public, documented placeholder)
		"123456789", // synthetic, sequential — structurally valid, not a real entity
	}
	for _, n := range valid {
		if !ValidPortugueseNIF(n) {
			t.Errorf("ValidPortugueseNIF(%q) = false, want true", n)
		}
	}

	invalid := []string{
		"",
		"12345678",   // too short
		"1234567890", // too long
		"12345678a",  // non-digit
		"000000000",  // leading zero
		"123456780",  // wrong check digit
		"NOPE",
	}
	for _, n := range invalid {
		if ValidPortugueseNIF(n) {
			t.Errorf("ValidPortugueseNIF(%q) = true, want false", n)
		}
	}
}
