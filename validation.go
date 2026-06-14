package invoicexpress

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError is returned by request Validate() methods (and by Create/
// Update before any network call) when required fields are missing. It lets
// callers fail fast with a clear message instead of round-tripping to the API
// for a 422.
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

// Validate checks the minimum fields required to create an item.
func (r *ItemCreateRequest) Validate() error {
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	if strings.TrimSpace(r.Name) == "" {
		return validationError("name is required")
	}
	return nil
}
