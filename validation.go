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

func validateClientRef(client ClientRef) string {
	if client.ID < 0 {
		return "client.id must be positive"
	}
	if client.ID > 0 || strings.TrimSpace(client.Code) != "" || strings.TrimSpace(client.Name) != "" {
		return ""
	}
	return "client.id, client.code, or client.name is required"
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
	issues = append(issues, validateClientRef(r.Client))
	if len(r.Items) == 0 {
		issues = append(issues, "at least one item is required")
	}
	issues = append(issues, validateItems(r.Items)...)
	return validationError(issues...)
}

// validateItems checks each document line item. It rejects a missing name or an
// unset (empty) unit_price/quantity, but accepts a zero value: the
// InvoiceXpress API allows zero-priced lines (free items, 100%-discount lines).
func validateItems(items []ItemRef) []string {
	var issues []string
	for i, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			issues = append(issues, fmt.Sprintf("items[%d].name is required", i))
		}
		if item.UnitPrice.s == "" {
			issues = append(issues, fmt.Sprintf("items[%d].unit_price is required", i))
		}
		if item.Quantity.s == "" {
			issues = append(issues, fmt.Sprintf("items[%d].quantity is required", i))
		}
	}
	return issues
}

func validateAddress(prefix string, address *AddressInfo) []string {
	if address == nil {
		return []string{prefix + " is required"}
	}
	var issues []string
	if strings.TrimSpace(address.Detail) == "" {
		issues = append(issues, prefix+".detail is required")
	}
	if strings.TrimSpace(address.City) == "" {
		issues = append(issues, prefix+".city is required")
	}
	if strings.TrimSpace(address.PostalCode) == "" {
		issues = append(issues, prefix+".postal_code is required")
	}
	if strings.TrimSpace(address.Country) == "" {
		issues = append(issues, prefix+".country is required")
	}
	return issues
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
	if r.DueDate.IsZero() {
		issues = append(issues, "due_date is required")
	}
	if r.LoadedAt.IsZero() {
		issues = append(issues, "loaded_at is required")
	}
	issues = append(issues, validateClientRef(r.Client))
	issues = append(issues, validateAddress("address_from", r.AddressFrom)...)
	issues = append(issues, validateAddress("address_to", r.AddressTo)...)
	if len(r.Items) == 0 {
		issues = append(issues, "at least one item is required")
	}
	issues = append(issues, validateItems(r.Items)...)
	for i, item := range r.Items {
		if strings.TrimSpace(item.Description) == "" {
			issues = append(issues, fmt.Sprintf("items[%d].description is required", i))
		}
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
	// Reject only an unset (empty) unit_price, not a zero value — the API allows
	// zero-priced catalog items (free items, 100%-discount lines), matching the
	// rule validateItems applies to document line items.
	if r.UnitPrice.s == "" {
		issues = append(issues, "unit_price is required")
	}
	return validationError(issues...)
}

// Validate checks the minimum fields required to create a sequence.
func (r *SequenceCreateRequest) Validate() error {
	if r == nil {
		return &ValidationError{Issues: []string{"request is nil"}}
	}
	if strings.TrimSpace(r.SerieNumber) == "" {
		return validationError("serie_number is required")
	}
	return nil
}
