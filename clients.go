package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ClientsService handles client (customer) operations.
type ClientsService struct {
	client *Client
}

// clientWrapper is used for JSON serialization of client requests.
type clientWrapper struct {
	Client interface{} `json:"client"`
}

// clientResponse is the JSON response for a single client.
type clientResponse struct {
	Client Customer `json:"client"`
}

// clientListResponse is the JSON response for a list of clients.
type clientListResponse struct {
	Clients    []Customer `json:"clients"`
	Pagination PageInfo   `json:"pagination"`
}

// List returns a paginated list of clients.
func (s *ClientsService) List(ctx context.Context, opts *ListOptions) ([]Customer, *PageInfo, error) {
	var resp clientListResponse
	if err := s.client.do(ctx, http.MethodGet, "/clients.json", paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: clients.list: %w", err)
	}
	return resp.Clients, &resp.Pagination, nil
}

// Get retrieves a client by ID.
func (s *ClientsService) Get(ctx context.Context, id int64) (*Customer, error) {
	path := fmt.Sprintf("/clients/%d.json", id)
	var resp clientResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: clients.get: %w", err)
	}
	return &resp.Client, nil
}

// Create creates a new client. The request is validated client-side before any
// network call.
func (s *ClientsService) Create(ctx context.Context, req *ClientCreateRequest) (*Customer, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var resp clientResponse
	if err := s.client.do(ctx, http.MethodPost, "/clients.json", nil, clientWrapper{Client: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: clients.create: %w", err)
	}
	return &resp.Client, nil
}

// Update updates an existing client.
func (s *ClientsService) Update(ctx context.Context, id int64, req *ClientUpdateRequest) error {
	path := fmt.Sprintf("/clients/%d.json", id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, clientWrapper{Client: req}, nil); err != nil {
		return fmt.Errorf("invoicexpress: clients.update: %w", err)
	}
	return nil
}

// FindByName searches for clients by name.
func (s *ClientsService) FindByName(ctx context.Context, name string) ([]Customer, error) {
	params := url.Values{"client_name": []string{name}}
	var resp clientListResponse
	if err := s.client.do(ctx, http.MethodGet, "/clients/find-by-name.json", params, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: clients.find-by-name: %w", err)
	}
	return resp.Clients, nil
}

// FindByCode searches for clients by code.
func (s *ClientsService) FindByCode(ctx context.Context, code string) (*Customer, error) {
	params := url.Values{"client_code": []string{code}}
	var resp clientResponse
	if err := s.client.do(ctx, http.MethodGet, "/clients/find-by-code.json", params, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: clients.find-by-code: %w", err)
	}
	return &resp.Client, nil
}

// ListInvoices returns the invoices for a specific client.
func (s *ClientsService) ListInvoices(ctx context.Context, clientID int64, opts *ListOptions) ([]Invoice, *PageInfo, error) {
	path := fmt.Sprintf("/clients/%d/invoices.json", clientID)
	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodGet, path, paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: clients.list-invoices: %w", err)
	}
	return resp.Invoices, &resp.Pagination, nil
}

// ListAll returns all clients across all pages.
func (s *ClientsService) ListAll(ctx context.Context) ([]Customer, error) {
	var all []Customer
	page := 1
	for {
		clients, pageInfo, err := s.List(ctx, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, clients...)
		if page >= pageInfo.TotalPages || len(clients) == 0 {
			break
		}
		page++
	}
	return all, nil
}
