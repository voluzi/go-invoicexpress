package invoicexpress

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrTaxNotFound is returned by TaxesService.FindByName when no tax matches.
var ErrTaxNotFound = errors.New("invoicexpress: tax not found")

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

// List returns the first page of taxes. Use ListPage to control pagination or
// ListAll to fetch every page.
func (s *TaxesService) List(ctx context.Context) ([]Tax, error) {
	taxes, _, err := s.ListPage(ctx, nil)
	return taxes, err
}

// ListPage returns a single page of taxes along with the pagination metadata.
func (s *TaxesService) ListPage(ctx context.Context, opts *ListOptions) ([]Tax, *PageInfo, error) {
	var resp taxListResponse
	if err := s.client.do(ctx, http.MethodGet, "/taxes.json", paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: taxes.list: %w", err)
	}
	return resp.Taxes, &resp.Pagination, nil
}

// ListAll returns all taxes across all pages.
func (s *TaxesService) ListAll(ctx context.Context) ([]Tax, error) {
	var all []Tax
	page := 1
	for {
		taxes, pageInfo, err := s.ListPage(ctx, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, taxes...)
		if page >= pageInfo.TotalPages || len(taxes) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// FindByName returns the tax with the given name (exact match), or
// ErrTaxNotFound if none matches. Use it to confirm a tax exists before
// issuing a document — InvoiceXpress silently applies the default tax for an
// unknown name rather than erroring, which would produce a wrong invoice.
func (s *TaxesService) FindByName(ctx context.Context, name string) (*Tax, error) {
	taxes, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range taxes {
		if taxes[i].Name == name {
			return &taxes[i], nil
		}
	}
	return nil, ErrTaxNotFound
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

// Create creates a new tax. The request is validated client-side before any
// network call, consistent with the other Create methods.
func (s *TaxesService) Create(ctx context.Context, req *TaxCreateRequest) (*Tax, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
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
