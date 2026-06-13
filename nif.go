package invoicexpress

// ValidPortugueseNIF reports whether s is a structurally valid Portuguese tax
// number (NIF / NIPC): exactly 9 digits with a correct mod-11 check digit.
//
// It validates form, not registration — it cannot tell you the number belongs
// to a real entity. It's useful before issuing a B2B document because
// InvoiceXpress silently falls back to "Consumidor Final" for an unrecognized
// NIF rather than returning an error, so a typo would otherwise produce a
// wrong (B2C) invoice without any signal.
//
// The official "Consumidor Final" number 999999990 is considered valid.
func ValidPortugueseNIF(s string) bool {
	if len(s) != 9 {
		return false
	}
	var digits [9]int
	for i := 0; i < 9; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		digits[i] = int(c - '0')
	}
	if digits[0] == 0 {
		return false
	}
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * (9 - i)
	}
	control := 11 - (sum % 11)
	if control >= 10 {
		control = 0
	}
	return control == digits[8]
}
