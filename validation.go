package invoicexpress

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError is returned by request Validate() methods and by the Create
// methods (which call Validate before any network call) when required fields
// are missing. It lets callers fail fast with a clear message instead of
// round-tripping to the API for a 422.
//
// Update methods do NOT validate — they are a pass-through, since update
// payloads may legitimately be partial. Call Validate yourself if you need it.
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return "invoicexpress: invalid request: " + strings.Join(e.Issues, "; ")
}

// IsValidation reports whether err is a client-side ValidationError.
func IsValidation(err error) bool {
	var e *ValidationError
	return errors.As(err, &e)
}

// requireCancelMessage returns a ValidationError when a cancellation is
// requested without a message. InvoiceXpress requires a reason to cancel a
// document or a partial-payment receipt.
func requireCancelMessage(state DocumentState, message string) error {
	if state == StateCanceled && strings.TrimSpace(message) == "" {
		return &ValidationError{Issues: []string{"a message is required to cancel"}}
	}
	return nil
}

func validationError(issues ...string) error {
	cleaned := make([]string, 0, len(issues))
	for _, i := range issues {
		if i != "" {
			cleaned = append(cleaned, i)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return &ValidationError{Issues: cleaned}
}

// Validate checks the minimum fields InvoiceXpress requires to create a
// document, so callers don't have to round-trip to the API for a 422.
func (r *InvoiceCreateRequest) Validate() error {
	var issues []string
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	if r.Date.IsZero() {
		issues = append(issues, "date is required")
	}
	if strings.TrimSpace(r.Client.Name) == "" {
		issues = append(issues, "client.name is required")
	}
	if len(r.Items) == 0 {
		issues = append(issues, "at least one item is required")
	}
	for i, item := range r.Items {
		if strings.TrimSpace(item.Name) == "" {
			issues = append(issues, fmt.Sprintf("items[%d].name is required", i))
		}
		if item.UnitPrice.IsZero() || item.Quantity.IsZero() {
			issues = append(issues, fmt.Sprintf("items[%d] needs a unit_price and quantity", i))
		}
	}
	return validationError(issues...)
}

// Validate checks the minimum fields required to create a guide.
func (r *GuideCreateRequest) Validate() error {
	var issues []string
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	if r.Date.IsZero() {
		issues = append(issues, "date is required")
	}
	if strings.TrimSpace(r.Client.Name) == "" {
		issues = append(issues, "client.name is required")
	}
	if len(r.Items) == 0 {
		issues = append(issues, "at least one item is required")
	}
	return validationError(issues...)
}

// Validate checks the minimum fields required to create a tax.
func (r *TaxCreateRequest) Validate() error {
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	var issues []string
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, "name is required")
	}
	if r.Value < 0 {
		issues = append(issues, "value must not be negative")
	}
	return validationError(issues...)
}

// Validate checks the minimum fields required to create a client.
func (r *ClientCreateRequest) Validate() error {
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	if strings.TrimSpace(r.Name) == "" {
		return validationError("name is required")
	}
	return nil
}

// Validate checks the minimum fields required to create an item. The API
// documents name, description and unit_price as required; we enforce name and
// unit_price (description is left to the server, which is lenient in practice).
func (r *ItemCreateRequest) Validate() error {
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	var issues []string
	if strings.TrimSpace(r.Name) == "" {
		issues = append(issues, "name is required")
	}
	if r.UnitPrice.IsZero() {
		issues = append(issues, "unit_price is required")
	}
	return validationError(issues...)
}
