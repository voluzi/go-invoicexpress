package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
)

// ItemsService handles item/product operations.
type ItemsService struct {
	client *Client
}

// itemWrapper is used for JSON serialization of item requests.
type itemWrapper struct {
	Item interface{} `json:"item"`
}

// itemResponse is the JSON response for a single item.
type itemResponse struct {
	Item Item `json:"item"`
}

// itemListResponse is the JSON response for a list of items.
type itemListResponse struct {
	Items      []Item   `json:"items"`
	Pagination PageInfo `json:"pagination"`
}

// List returns a paginated list of items.
func (s *ItemsService) List(ctx context.Context, opts *ListOptions) ([]Item, *PageInfo, error) {
	var resp itemListResponse
	if err := s.client.do(ctx, http.MethodGet, "/items.json", paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: items.list: %w", err)
	}
	return resp.Items, &resp.Pagination, nil
}

// Get retrieves an item by ID.
func (s *ItemsService) Get(ctx context.Context, id int64) (*Item, error) {
	path := fmt.Sprintf("/items/%d.json", id)
	var resp itemResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: items.get: %w", err)
	}
	return &resp.Item, nil
}

// Create creates a new item.
func (s *ItemsService) Create(ctx context.Context, req *ItemCreateRequest) (*Item, error) {
	var resp itemResponse
	if err := s.client.do(ctx, http.MethodPost, "/items.json", nil, itemWrapper{Item: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: items.create: %w", err)
	}
	return &resp.Item, nil
}

// Update updates an existing item.
func (s *ItemsService) Update(ctx context.Context, id int64, req *ItemUpdateRequest) error {
	path := fmt.Sprintf("/items/%d.json", id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, itemWrapper{Item: req}, nil); err != nil {
		return fmt.Errorf("invoicexpress: items.update: %w", err)
	}
	return nil
}

// Delete deletes an item by ID.
func (s *ItemsService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/items/%d.json", id)
	if err := s.client.do(ctx, http.MethodDelete, path, nil, nil, nil); err != nil {
		return fmt.Errorf("invoicexpress: items.delete: %w", err)
	}
	return nil
}

// ListAll returns all items across all pages.
func (s *ItemsService) ListAll(ctx context.Context) ([]Item, error) {
	var all []Item
	page := 1
	for {
		items, pageInfo, err := s.List(ctx, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if page >= pageInfo.TotalPages || len(items) == 0 {
			break
		}
		page++
	}
	return all, nil
}
