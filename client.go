package invoicexpress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://%s.app.invoicexpress.com"
	defaultTimeout = 30 * time.Second
)

// Client is the InvoiceXpress API client.
type Client struct {
	accountName string
	apiKey      string
	baseURL     string
	httpClient  *http.Client

	// Service fields.
	Invoices  *InvoicesService
	Estimates *EstimatesService
	Guides    *GuidesService
	Clients   *ClientsService
	Items     *ItemsService
	Sequences *SequencesService
	Taxes     *TaxesService
	SAFT      *SAFTService
	Accounts  *AccountsService
}

// NewClient creates a new InvoiceXpress API client.
func NewClient(accountName, apiKey string) *Client {
	c := &Client{
		accountName: accountName,
		apiKey:      apiKey,
		baseURL:     fmt.Sprintf(defaultBaseURL, accountName),
		httpClient:  &http.Client{Timeout: defaultTimeout},
	}
	c.Invoices = &InvoicesService{client: c}
	c.Estimates = &EstimatesService{client: c}
	c.Guides = &GuidesService{client: c}
	c.Clients = &ClientsService{client: c}
	c.Items = &ItemsService{client: c}
	c.Sequences = &SequencesService{client: c}
	c.Taxes = &TaxesService{client: c}
	c.SAFT = &SAFTService{client: c}
	c.Accounts = &AccountsService{client: c}
	return c
}

// WithHTTPClient sets a custom HTTP client.
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// buildURL constructs the full request URL with the api_key appended.
func (c *Client) buildURL(path string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", c.apiKey)
	return c.baseURL + path + "?" + params.Encode()
}

// do executes an HTTP request and decodes the JSON response into v.
// If v is nil, the response body is discarded.
func (c *Client) do(ctx context.Context, method, path string, params url.Values, body, v interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("invoicexpress: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, params), reqBody)
	if err != nil {
		return fmt.Errorf("invoicexpress: create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("invoicexpress: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("invoicexpress: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if v != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, v); err != nil {
			return fmt.Errorf("invoicexpress: decode response: %w", err)
		}
	}
	return nil
}

// doWithStatus is like do but also returns the HTTP status code.
// Used for async operations that return 202 Accepted.
func (c *Client) doWithStatus(ctx context.Context, method, path string, params url.Values, body, v interface{}) (int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("invoicexpress: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, params), reqBody)
	if err != nil {
		return 0, fmt.Errorf("invoicexpress: create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("invoicexpress: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("invoicexpress: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if v != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, v); err != nil {
			return resp.StatusCode, fmt.Errorf("invoicexpress: decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// paginationParams builds query params from ListOptions.
func paginationParams(opts *ListOptions) url.Values {
	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
	}
	return params
}
