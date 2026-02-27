package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
)

// AccountsService handles account operations.
type AccountsService struct {
	client *Client
}

// accountResponse is the JSON response for a single account.
type accountResponse struct {
	Account Account `json:"account"`
}

// accountListResponse is the JSON response for a list of accounts.
type accountListResponse struct {
	Accounts []Account `json:"accounts"`
}

// List returns all accounts accessible to the authenticated user.
func (s *AccountsService) List(ctx context.Context) ([]Account, error) {
	var resp accountListResponse
	if err := s.client.do(ctx, http.MethodGet, "/users/accounts.json", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: accounts.list: %w", err)
	}
	return resp.Accounts, nil
}

// Get retrieves an account by ID.
func (s *AccountsService) Get(ctx context.Context, id int64) (*Account, error) {
	path := fmt.Sprintf("/users/accounts/%d.json", id)
	var resp accountResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: accounts.get: %w", err)
	}
	return &resp.Account, nil
}
