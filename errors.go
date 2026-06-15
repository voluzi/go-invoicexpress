package invoicexpress

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents an error returned by the InvoiceXpress API.
type APIError struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Status is the HTTP status line (e.g. "422 Unprocessable Entity").
	Status string
	// Body is the raw response body.
	Body string
	// Errors holds parsed validation messages, when the body could be parsed
	// (typically on a 422 response). Nil/empty otherwise.
	Errors []string
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("invoicexpress: API error %d %s: %s",
			e.StatusCode, e.Status, strings.Join(e.Errors, "; "))
	}
	return fmt.Sprintf("invoicexpress: API error %d %s: %s", e.StatusCode, e.Status, e.Body)
}

// newAPIError builds an APIError, best-effort parsing validation messages from
// the body.
func newAPIError(statusCode int, status string, body []byte) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Status:     status,
		Body:       string(body),
		Errors:     parseValidationErrors(body),
	}
}

// parseValidationErrors extracts human-readable messages from an InvoiceXpress
// error body. The API has used a few shapes over time, so this is defensive:
//   - {"errors": {"field": ["msg", ...], ...}}
//   - {"errors": [{"error": "msg"}, ...]}  /  {"errors": ["msg", ...]}
//   - {"error": "msg"}  /  {"message": "msg"}
//
// Returns nil when nothing parseable is found.
func parseValidationErrors(body []byte) []string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed[0] != '{' {
		return nil
	}

	var generic map[string]json.RawMessage
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil
	}

	var out []string

	if raw, ok := generic["errors"]; ok {
		out = append(out, extractMessages(raw)...)
	}
	for _, key := range []string{"error", "message"} {
		if raw, ok := generic[key]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				out = append(out, s)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractMessages(raw json.RawMessage) []string {
	// Try map[string][]string or map[string]string.
	var asMap map[string]json.RawMessage
	if json.Unmarshal(raw, &asMap) == nil {
		var out []string
		for field, v := range asMap {
			for _, msg := range stringsFrom(v) {
				out = append(out, fmt.Sprintf("%s: %s", field, msg))
			}
		}
		return out
	}
	// Try []string or []object.
	var asSlice []json.RawMessage
	if json.Unmarshal(raw, &asSlice) == nil {
		var out []string
		for _, v := range asSlice {
			out = append(out, stringsFrom(v)...)
		}
		return out
	}
	return stringsFrom(raw)
}

// stringsFrom coerces a JSON value (string, []string, or {"error"/"message": ...})
// into a slice of message strings.
func stringsFrom(raw json.RawMessage) []string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	var ss []string
	if json.Unmarshal(raw, &ss) == nil {
		return ss
	}
	var obj map[string]string
	if json.Unmarshal(raw, &obj) == nil {
		for _, key := range []string{"error", "message"} {
			if v := obj[key]; v != "" {
				return []string{v}
			}
		}
	}
	return nil
}

// AsAPIError extracts an *APIError from anywhere in err's chain.
func AsAPIError(err error) (*APIError, bool) {
	var e *APIError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func hasStatus(err error, status int) bool {
	e, ok := AsAPIError(err)
	return ok && e.StatusCode == status
}

// IsNotFound reports whether err is an API 404 Not Found.
func IsNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// IsUnprocessable reports whether err is an API 422 Unprocessable Entity
// (validation failure). Inspect APIError.Errors for the field messages.
func IsUnprocessable(err error) bool { return hasStatus(err, http.StatusUnprocessableEntity) }

// IsRateLimited reports whether err is an API 429 Too Many Requests. Note that
// the client retries these automatically by default; this matches the error
// returned after retries are exhausted.
func IsRateLimited(err error) bool { return hasStatus(err, http.StatusTooManyRequests) }

// IsUnauthorized reports whether err is an API 401 Unauthorized (bad API key).
func IsUnauthorized(err error) bool { return hasStatus(err, http.StatusUnauthorized) }

// IsConflict reports whether err is an API 409 Conflict.
func IsConflict(err error) bool { return hasStatus(err, http.StatusConflict) }
