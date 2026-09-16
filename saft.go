package invoicexpress

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrNoSAFTDocuments reports that the selected period contains no documents
// to export.
var ErrNoSAFTDocuments = errors.New("invoicexpress: no SAF-T documents for selected period")

// SAFTService handles SAF-T export operations.
type SAFTService struct {
	client *Client
}

// saftResponse is the JSON response for a SAF-T export.
type saftResponse struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

// Export starts a SAF-T export and polls until it is ready.
// month is 1-12, year is a 4-digit year.
// It polls with the given interval until done or context is cancelled.
func (s *SAFTService) Export(ctx context.Context, month, year int, pollInterval time.Duration) (*SAFTExportResult, error) {
	if month < 1 || month > 12 {
		return nil, &ValidationError{Issues: []string{"month must be between 1 and 12"}}
	}
	if year < 2000 || year > 2100 {
		return nil, &ValidationError{Issues: []string{"year must be between 2000 and 2100"}}
	}
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	params := url.Values{
		"month": []string{strconv.Itoa(month)},
		"year":  []string{strconv.Itoa(year)},
	}
	for poll := 0; poll < maxPDFPolls; poll++ {
		var resp saftResponse
		statusCode, err := s.client.doWithStatus(ctx, http.MethodGet, "/api/export_saft.json", params, nil, &resp)
		if err != nil {
			// A real failure (transport, read, or decode error) must surface
			// immediately. A genuine "still generating" reply is a 202 with an
			// empty body, which returns no error — so we only keep polling on a
			// clean 202 below, never swallow an error here.
			return nil, fmt.Errorf("invoicexpress: saft.export: %w", err)
		}
		if statusCode == http.StatusAccepted {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollInterval):
				continue
			}
		}
		if strings.TrimSpace(resp.URL) != "" {
			return &SAFTExportResult{URL: resp.URL}, nil
		}
		if strings.TrimSpace(resp.Message) != "" {
			return nil, fmt.Errorf("%w: %s", ErrNoSAFTDocuments, resp.Message)
		}
		return nil, fmt.Errorf("invoicexpress: saft.export: completed response carries no URL")
	}
	return nil, fmt.Errorf("invoicexpress: saft.export: not ready after %d polls", maxPDFPolls)
}
