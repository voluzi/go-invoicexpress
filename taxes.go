package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
)

// TaxesService handles tax operations.
type TaxesService struct {
	client *Client
}

// taxWrapper is used for JSON serialization of tax requests.
type taxWrapper struct {
	Tax interface{} `json:"tax"`
}

// taxResponse is the JSON response for a single tax.
type taxResponse struct {
	Tax Tax `json:"tax"`
}

// taxListResponse is the JSON response for a list of taxes.
type taxListResponse struct {
	Taxes      []Tax    `json:"taxes"`
	Pagination PageInfo `json:"pagination"`
}

// List returns all taxes.
func (s *TaxesService) List(ctx context.Context) ([]Tax, error) {
	var resp taxListResponse
	if err := s.client.do(ctx, http.MethodGet, "/taxes.json", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: taxes.list: %w", err)
	}
	return resp.Taxes, nil
}

// Get retrieves a tax by ID.
func (s *TaxesService) Get(ctx context.Context, id int64) (*Tax, error) {
	path := fmt.Sprintf("/taxes/%d.json", id)
	var resp taxResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: taxes.get: %w", err)
	}
	return &resp.Tax, nil
}

// Create creates a new tax.
func (s *TaxesService) Create(ctx context.Context, req *TaxCreateRequest) (*Tax, error) {
	var resp taxResponse
	if err := s.client.do(ctx, http.MethodPost, "/taxes.json", nil, taxWrapper{Tax: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: taxes.create: %w", err)
	}
	return &resp.Tax, nil
}

// Update updates an existing tax.
func (s *TaxesService) Update(ctx context.Context, id int64, req *TaxUpdateRequest) error {
	path := fmt.Sprintf("/taxes/%d.json", id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, taxWrapper{Tax: req}, nil); err != nil {
		return fmt.Errorf("invoicexpress: taxes.update: %w", err)
	}
	return nil
}

// Delete deletes a tax by ID.
func (s *TaxesService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/taxes/%d.json", id)
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil, nil); err != nil {
		return fmt.Errorf("invoicexpress: taxes.delete: %w", err)
	}
	return nil
}
