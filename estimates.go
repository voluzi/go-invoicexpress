package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// EstimatesService handles estimate document operations (quotes, proformas, fees_notes).
type EstimatesService struct {
	client *Client
}

// estimateWrapper is used for JSON serialization of estimate requests.
type estimateWrapper struct {
	Estimate interface{} `json:"estimate"`
}

// estimateResponse is the JSON response for a single estimate.
type estimateResponse struct {
	Estimate Estimate `json:"estimate"`
}

// estimateListResponse is the JSON response for a list of estimates.
type estimateListResponse struct {
	Estimates  []Estimate `json:"estimates"`
	Pagination PageInfo   `json:"pagination"`
}

// Create creates a new estimate document.
func (s *EstimatesService) Create(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Estimate, error) {
	path := fmt.Sprintf("/%s.json", docType)
	var resp estimateResponse
	if err := s.client.do(ctx, http.MethodPost, path, nil, estimateWrapper{Estimate: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: estimates.create: %w", err)
	}
	return &resp.Estimate, nil
}

// Get retrieves an estimate document by ID.
func (s *EstimatesService) Get(ctx context.Context, docType DocumentType, id int64) (*Estimate, error) {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	var resp estimateResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: estimates.get: %w", err)
	}
	return &resp.Estimate, nil
}

// List returns a paginated list of estimate documents.
func (s *EstimatesService) List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Estimate, *PageInfo, error) {
	path := fmt.Sprintf("/%s.json", docType)
	var resp estimateListResponse
	if err := s.client.do(ctx, http.MethodGet, path, paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: estimates.list: %w", err)
	}
	return resp.Estimates, &resp.Pagination, nil
}

// Update updates an existing estimate document.
func (s *EstimatesService) Update(ctx context.Context, docType DocumentType, id int64, req *InvoiceUpdateRequest) error {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, estimateWrapper{Estimate: req}, nil); err != nil {
		return fmt.Errorf("invoicexpress: estimates.update: %w", err)
	}
	return nil
}

// ChangeState transitions an estimate document to a new state.
func (s *EstimatesService) ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Estimate, error) {
	path := fmt.Sprintf("/%s/%d/change-state.json", docType, id)
	body := struct {
		Estimate ChangeStateRequest `json:"estimate"`
	}{Estimate: ChangeStateRequest{State: state, Message: message}}
	var resp estimateResponse
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: estimates.change-state: %w", err)
	}
	return &resp.Estimate, nil
}

// SendByEmail sends an estimate document by email.
func (s *EstimatesService) SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error {
	path := fmt.Sprintf("/%s/%d/email-document.json", docType, id)
	body := struct {
		Message *EmailRequest `json:"message"`
	}{Message: req}
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, nil); err != nil {
		return fmt.Errorf("invoicexpress: estimates.send-by-email: %w", err)
	}
	return nil
}

// GeneratePDF starts PDF generation and polls until the PDF is ready.
func (s *EstimatesService) GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	path := fmt.Sprintf("/api/pdf/%d.json", id)
	for {
		var resp pdfResponse
		statusCode, err := s.client.doWithStatus(ctx, http.MethodGet, path, nil, nil, &resp)
		if err != nil {
			if statusCode == http.StatusAccepted {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(pollInterval):
					continue
				}
			}
			return "", fmt.Errorf("invoicexpress: estimates.generate-pdf: %w", err)
		}
		if statusCode == http.StatusAccepted {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(pollInterval):
				continue
			}
		}
		return resp.Output.PDFURL, nil
	}
}
