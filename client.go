package invoicexpress

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// redactAPIKey strips the api_key value from any *url.Error in the chain. The
// api_key is sent as a query parameter, and net/http transport errors embed the
// full request URL in their message — without this, the key could leak into a
// caller's logs.
func redactAPIKey(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = redactAPIKeyInURL(ue.URL)
	}
	return err
}

// redactAPIKeyBytes removes the literal api_key value from a response body,
// in case the API or an intermediary proxy echoes the request URL.
func redactAPIKeyBytes(body []byte, key string) []byte {
	if key == "" {
		return body
	}
	body = bytes.ReplaceAll(body, []byte(key), []byte("REDACTED"))
	// buildURL passes the key through url.Values.Encode, so a key with reserved
	// characters appears query-escaped if the URL is echoed back; redact that
	// form too.
	if escaped := url.QueryEscape(key); escaped != key {
		body = bytes.ReplaceAll(body, []byte(escaped), []byte("REDACTED"))
	}
	return body
}

// redactAPIKeyString removes the api_key value from an arbitrary string. Unlike
// redactAPIKeyInURL it does not assume the whole string is a parseable URL, so
// it is safe for error messages that merely embed the URL (e.g. a url.Parse
// failure whose text is `parse "https://…?api_key=…": …`). Because buildURL
// passes the key through url.Values.Encode, a key containing reserved
// characters (+, /, =, …) appears query-escaped in the URL — so both the raw
// and the query-escaped forms are redacted.
func redactAPIKeyString(s, key string) string {
	if key == "" {
		return s
	}
	s = strings.ReplaceAll(s, key, "REDACTED")
	if escaped := url.QueryEscape(key); escaped != key {
		s = strings.ReplaceAll(s, escaped, "REDACTED")
	}
	return s
}

func redactAPIKeyInURL(raw string) string {
	u, parseErr := url.Parse(raw)
	if parseErr != nil {
		return raw
	}
	q := u.Query()
	if q.Get("api_key") == "" {
		return raw
	}
	q.Set("api_key", "REDACTED")
	u.RawQuery = q.Encode()
	return u.String()
}

const (
	// Version is the library version, surfaced in the default User-Agent.
	Version = "0.1.0"

	defaultBaseURLFormat = "https://%s.app.invoicexpress.com"
	defaultTimeout       = 30 * time.Second
	defaultUserAgent     = "go-invoicexpress/" + Version + " (+https://github.com/voluzi/go-invoicexpress)"

	// maxResponseBytes caps how much of a response body is read, protecting
	// against a misbehaving server streaming an unbounded body.
	maxResponseBytes = 10 << 20 // 10 MiB

	// maxPDFPolls bounds how many times pollPDF waits on a 202 before giving
	// up, so a document that never finishes generating cannot loop forever.
	maxPDFPolls = 60
)

// RetryConfig controls automatic retries for transient failures (HTTP 429 and
// 5xx, plus network errors on idempotent requests).
type RetryConfig struct {
	// MaxAttempts is the total number of attempts including the first. A value
	// <= 1 disables retries.
	MaxAttempts int
	// BaseDelay is the initial backoff; it doubles each attempt (with half
	// jitter) up to MaxDelay.
	BaseDelay time.Duration
	// MaxDelay caps the per-attempt backoff.
	MaxDelay time.Duration
}

// DefaultRetryConfig is applied unless overridden with WithRetry.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 4, BaseDelay: 500 * time.Millisecond, MaxDelay: 10 * time.Second}
}

// Client is the InvoiceXpress API client. It is safe for concurrent use.
type Client struct {
	accountName string
	apiKey      string
	baseURL     string
	userAgent   string
	httpClient  *http.Client
	retry       RetryConfig
	limiter     *rateLimiter // optional; nil disables client-side rate limiting
	randFloat   func() float64

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

// Option configures a Client at construction time.
type Option func(*Client)

// WithBaseURL overrides the API base URL. Primarily useful for tests
// (pointing at an httptest server) or a proxy.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient sets a custom *http.Client (e.g. with custom transport,
// proxy, or TLS config).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithUserAgent overrides the User-Agent header sent on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithTimeout sets the per-request timeout on the underlying HTTP client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithRetry overrides the retry policy. Pass RetryConfig{MaxAttempts: 1} to
// disable retries entirely.
func WithRetry(cfg RetryConfig) Option {
	return func(c *Client) { c.retry = cfg }
}

// WithRateLimit enables a client-side token-bucket limiter that paces requests
// to at most requestsPerSecond, allowing short bursts up to burst. This helps
// stay under InvoiceXpress's server-side rate limits proactively.
func WithRateLimit(requestsPerSecond float64, burst int) Option {
	return func(c *Client) {
		if requestsPerSecond > 0 && burst > 0 {
			c.limiter = newRateLimiter(requestsPerSecond, burst)
		}
	}
}

// NewClient creates a new InvoiceXpress API client for the given account name
// (the subdomain of your InvoiceXpress account) and API key.
func NewClient(accountName, apiKey string, opts ...Option) *Client {
	c := &Client{
		accountName: accountName,
		apiKey:      apiKey,
		baseURL:     fmt.Sprintf(defaultBaseURLFormat, accountName),
		userAgent:   defaultUserAgent,
		httpClient:  &http.Client{Timeout: defaultTimeout},
		retry:       DefaultRetryConfig(),
		randFloat:   rand.Float64,
	}
	for _, opt := range opts {
		opt(c)
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

// buildURL constructs the full request URL with the api_key appended. It does
// not mutate the caller's params.
func cloneURLValues(v url.Values) url.Values {
	q := url.Values{}
	for k, vals := range v {
		q[k] = append([]string(nil), vals...)
	}
	return q
}

func (c *Client) buildURL(path string, params url.Values) string {
	q := cloneURLValues(params)
	q.Set("api_key", c.apiKey)
	return c.baseURL + path + "?" + q.Encode()
}

// do executes an HTTP request (with retries) and decodes the JSON response
// into v. If v is nil, the response body is discarded.
func (c *Client) do(ctx context.Context, method, path string, params url.Values, body, v interface{}) error {
	_, err := c.doWithStatus(ctx, method, path, params, body, v)
	return err
}

// doWithStatus is like do but also returns the final HTTP status code. Used by
// async operations that return 202 Accepted.
func (c *Client) doWithStatus(ctx context.Context, method, path string, params url.Values, body, v interface{}) (int, error) {
	var reqBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("invoicexpress: marshal request: %w", err)
		}
		reqBytes = b
	}

	fullURL := c.buildURL(path, params)
	idempotent := isIdempotent(method)

	var lastErr error
	for attempt := 1; ; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.wait(ctx); err != nil {
				return 0, err
			}
		}

		var reqBody io.Reader
		if reqBytes != nil {
			reqBody = bytes.NewReader(reqBytes)
		}
		// fullURL embeds the api_key in its query string. If request creation
		// fails (malformed method/path), the error message would otherwise leak
		// the literal key because it is not wrapped in a *url.Error.
		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			// NewRequestWithContext errors are plain errors, not *url.Error, so
			// redactAPIKey would not catch them. The error text may embed the
			// full URL (e.g. a url.Parse failure), which is not itself a
			// parseable URL — so strip the literal key rather than parsing.
			return 0, fmt.Errorf("invoicexpress: create request: %s", redactAPIKeyString(err.Error(), c.apiKey))
		}
		if reqBytes != nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("invoicexpress: do request: %w", redactAPIKey(err))
			if idempotent && c.shouldRetry(attempt) {
				if werr := c.backoff(ctx, attempt, nil); werr != nil {
					return 0, werr
				}
				continue
			}
			return 0, lastErr
		}

		status := resp.StatusCode
		// When the server advertises a Content-Length over the cap, reject
		// before allocating to read the body. ContentLength is -1 when unknown,
		// so this only fires on an explicit, oversized header.
		if resp.ContentLength > maxResponseBytes {
			_ = resp.Body.Close()
			return status, fmt.Errorf("invoicexpress: response exceeds %d bytes", maxResponseBytes)
		}
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("invoicexpress: read response: %w", readErr)
			if idempotent && c.shouldRetry(attempt) {
				if werr := c.backoff(ctx, attempt, nil); werr != nil {
					return status, werr
				}
				continue
			}
			return status, lastErr
		}
		if len(respBody) > maxResponseBytes {
			return status, fmt.Errorf("invoicexpress: response exceeds %d bytes", maxResponseBytes)
		}

		if status >= 400 {
			apiErr := newAPIError(status, resp.Status, redactAPIKeyBytes(respBody, c.apiKey))
			if c.retryableStatus(status, idempotent) && c.shouldRetry(attempt) {
				if werr := c.backoff(ctx, attempt, resp); werr != nil {
					return status, werr
				}
				continue
			}
			return status, apiErr
		}

		if v != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, v); err != nil {
				return status, fmt.Errorf("invoicexpress: decode response: %w", err)
			}
		}
		return status, nil
	}
}

func (c *Client) shouldRetry(attempt int) bool {
	return attempt < c.retry.MaxAttempts
}

// retryableStatus reports whether an HTTP status should be retried. 429 is
// always retryable (the request was throttled, not processed); 5xx is retried
// only for idempotent methods.
func (c *Client) retryableStatus(status int, idempotent bool) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && idempotent
}

// backoff sleeps before the next attempt, honoring a Retry-After header when
// present, otherwise using exponential backoff with half jitter.
func (c *Client) backoff(ctx context.Context, attempt int, resp *http.Response) error {
	delay := c.nextDelay(attempt, resp)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) nextDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			if maxD := c.retry.MaxDelay; maxD > 0 && d > maxD {
				d = maxD
			}
			return d
		}
	}
	base := c.retry.BaseDelay
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	// Exponential: base * 2^(attempt-1), capped at MaxDelay. Guard against
	// shift overflow when MaxAttempts is configured with a huge value.
	shift := attempt - 1
	const maxShift = 62 // keep room for signed time.Duration multiplication.
	if shift > maxShift {
		shift = maxShift
	}
	d := base << shift
	if d <= 0 || (c.retry.MaxDelay > 0 && d > c.retry.MaxDelay) {
		d = c.retry.MaxDelay
	}
	if d <= 0 {
		d = base
	}
	// Half jitter: guarantee at least d/2 of backoff, then add a random
	// component up to d/2 so a near-zero sleep can never happen.
	half := d / 2
	return half + time.Duration(c.randFloat()*float64(half))
}

// parseRetryAfter parses a Retry-After header value, which may be an integer
// number of seconds or an HTTP date.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
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

// pollPDF polls the shared PDF endpoint until the document is ready or the
// context is cancelled. It is used by InvoicesService, EstimatesService, and
// GuidesService so the polling loop is implemented once.
func (c *Client) pollPDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	path := fmt.Sprintf("/api/pdf/%d.json", id)
	for poll := 0; poll < maxPDFPolls; poll++ {
		var resp pdfResponse
		statusCode, err := c.doWithStatus(ctx, http.MethodGet, path, nil, nil, &resp)
		if err != nil {
			// A real failure (transport, read, or decode error) must surface
			// immediately. A genuine "still generating" reply is a 202 with an
			// empty body, which returns no error — so we only keep polling on a
			// clean 202 below, never swallow an error here.
			return "", err
		}
		if statusCode == http.StatusAccepted {
			// Still generating, wait and retry.
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(pollInterval):
				continue
			}
		}
		return resp.Output.PDFURL, nil
	}
	return "", fmt.Errorf("invoicexpress: PDF not ready after %d polls", maxPDFPolls)
}

// rateLimiter is a minimal token-bucket limiter (no external dependencies).
type rateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	max          float64
	refillPerSec float64
	last         time.Time
}

func newRateLimiter(perSec float64, burst int) *rateLimiter {
	return &rateLimiter{
		tokens:       float64(burst),
		max:          float64(burst),
		refillPerSec: perSec,
		last:         time.Now(),
	}
}

func (r *rateLimiter) wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		r.tokens = min(r.max, r.tokens+now.Sub(r.last).Seconds()*r.refillPerSec)
		r.last = now
		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}
		deficit := 1 - r.tokens
		wait := time.Duration(deficit / r.refillPerSec * float64(time.Second))
		r.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
