package invoicexpress

import (
	"fmt"
	"testing"
)

func TestErrorHelpersMatchWrappedErrors(t *testing.T) {
	// The bug this guards against: service methods wrap the APIError with
	// fmt.Errorf("%w"), so a plain type assertion fails. errors.As must work.
	base := &APIError{StatusCode: 404, Status: "404 Not Found"}
	wrapped := fmt.Errorf("invoicexpress: invoices.get: %w", base)

	if !IsNotFound(wrapped) {
		t.Error("IsNotFound should match a wrapped APIError")
	}
	if IsUnprocessable(wrapped) {
		t.Error("IsUnprocessable should not match a 404")
	}

	got, ok := AsAPIError(wrapped)
	if !ok || got.StatusCode != 404 {
		t.Errorf("AsAPIError = %+v, %v", got, ok)
	}
}

func TestErrorHelpersByStatus(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
		name   string
	}{
		{404, IsNotFound, "IsNotFound"},
		{422, IsUnprocessable, "IsUnprocessable"},
		{429, IsRateLimited, "IsRateLimited"},
		{401, IsUnauthorized, "IsUnauthorized"},
		{409, IsConflict, "IsConflict"},
	}
	for _, tc := range cases {
		err := fmt.Errorf("wrap: %w", &APIError{StatusCode: tc.status})
		if !tc.check(err) {
			t.Errorf("%s did not match status %d", tc.name, tc.status)
		}
		// A different status must not match.
		other := fmt.Errorf("wrap: %w", &APIError{StatusCode: 418})
		if tc.check(other) {
			t.Errorf("%s wrongly matched status 418", tc.name)
		}
	}
}

func TestErrorHelpersIgnoreNonAPIErrors(t *testing.T) {
	if IsNotFound(fmt.Errorf("some other error")) {
		t.Error("IsNotFound matched a non-API error")
	}
	if IsNotFound(nil) {
		t.Error("IsNotFound matched nil")
	}
}

func TestParseValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int // number of messages expected (>0 means non-empty)
	}{
		{"field map", `{"errors":{"date":["is required"],"client":["is invalid"]}}`, 2},
		{"string slice", `{"errors":["a","b","c"]}`, 3},
		{"object slice", `{"errors":[{"error":"bad"}]}`, 1},
		{"single error", `{"error":"boom"}`, 1},
		{"message", `{"message":"nope"}`, 1},
		{"empty", ``, 0},
		{"not json", `<html>500</html>`, 0},
		{"unrelated json", `{"foo":"bar"}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValidationErrors([]byte(tt.body))
			if (len(got) > 0) != (tt.want > 0) {
				t.Fatalf("parseValidationErrors(%q) = %v, want %d msgs", tt.body, got, tt.want)
			}
			if tt.want > 0 && len(got) != tt.want {
				t.Errorf("got %d messages %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestAPIErrorMessageIncludesParsedErrors(t *testing.T) {
	e := newAPIError(422, "422 Unprocessable Entity", []byte(`{"errors":["date is required"]}`))
	if len(e.Errors) != 1 {
		t.Fatalf("expected 1 parsed error, got %v", e.Errors)
	}
	msg := e.Error()
	if msg == "" || !contains(msg, "date is required") {
		t.Errorf("error message %q should include parsed validation message", msg)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
