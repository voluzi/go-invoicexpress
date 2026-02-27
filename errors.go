package invoicexpress

import (
	"fmt"
	"net/http"
)

// APIError represents an error returned by the InvoiceXpress API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("invoicexpress: API error %d %s: %s", e.StatusCode, e.Status, e.Body)
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	var e *APIError
	if apiErr, ok := err.(*APIError); ok {
		e = apiErr
	}
	return e != nil && e.StatusCode == http.StatusNotFound
}

// IsUnprocessable returns true if the error is a 422 Unprocessable Entity.
func IsUnprocessable(err error) bool {
	var e *APIError
	if apiErr, ok := err.(*APIError); ok {
		e = apiErr
	}
	return e != nil && e.StatusCode == http.StatusUnprocessableEntity
}
