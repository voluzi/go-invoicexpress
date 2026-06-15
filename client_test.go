package invoicexpress

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// newTestServer spins up an httptest server and returns a Client pointed at it
// with near-instant retry backoff so retry paths are fast.
func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("acct", "test-key",
		WithBaseURL(srv.URL),
		WithRetry(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}),
	)
}

func TestBuildURLAppendsAPIKeyWithoutMutating(t *testing.T) {
	c := NewClient("acct", "secret", WithBaseURL("https://example.test"))
	params := url.Values{"page": []string{"2"}}
	got := c.buildURL("/invoices.json", params)

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if u.Query().Get("api_key") != "secret" {
		t.Errorf("api_key not appended: %s", got)
	}
	if u.Query().Get("page") != "2" {
		t.Errorf("page param lost: %s", got)
	}
	// Caller's map must be untouched.
	if params.Get("api_key") != "" {
		t.Errorf("buildURL mutated caller params: %v", params)
	}
}

func TestUserAgentAndAcceptHeaders(t *testing.T) {
	var gotUA, gotAccept string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.Write([]byte("{}"))
	})
	if err := c.do(context.Background(), http.MethodGet, "/ping.json", nil, nil, nil); err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotUA != defaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, defaultUserAgent)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
}

func TestWithUserAgentOverride(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := NewClient("acct", "k", WithBaseURL(srv.URL), WithUserAgent("myapp/1.0"))
	_ = c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil)
	if gotUA != "myapp/1.0" {
		t.Errorf("User-Agent = %q, want myapp/1.0", gotUA)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var attempts int32
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"errors":["rate limited"]}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	})
	if err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestRetryExhaustedReturnsAPIError(t *testing.T) {
	var attempts int32
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errors":["nope"]}`))
	})
	err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil)
	if !IsRateLimited(err) {
		t.Fatalf("expected rate-limited error, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3 (MaxAttempts)", got)
	}
}

func TestRetryOn500ForIdempotentOnly(t *testing.T) {
	t.Run("GET is retried", func(t *testing.T) {
		var attempts int32
		c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			n := atomic.AddInt32(&attempts, 1)
			if n < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write([]byte("{}"))
		})
		if err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil); err != nil {
			t.Fatalf("GET should recover: %v", err)
		}
		if got := atomic.LoadInt32(&attempts); got != 2 {
			t.Errorf("attempts = %d, want 2", got)
		}
	})

	t.Run("POST is not retried on 500", func(t *testing.T) {
		var attempts int32
		c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusInternalServerError)
		})
		err := c.do(context.Background(), http.MethodPost, "/x.json", nil, map[string]string{"a": "b"}, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if got := atomic.LoadInt32(&attempts); got != 1 {
			t.Errorf("attempts = %d, want 1 (POST not retried on 5xx)", got)
		}
	})
}

func TestNoRetryOn4xxClientError(t *testing.T) {
	var attempts int32
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":["missing"]}`))
	})
	err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil)
	if !IsNotFound(err) {
		t.Fatalf("expected not-found, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (404 not retried)", got)
	}
}

func TestContextCancelStopsRetry(t *testing.T) {
	c := NewClient("acct", "k",
		WithBaseURL("http://127.0.0.1:0"), // unroutable; Do will error
		WithRetry(RetryConfig{MaxAttempts: 5, BaseDelay: time.Hour, MaxDelay: time.Hour}),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := c.do(ctx, http.MethodGet, "/x.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"5", 5 * time.Second, true},
		{"0", 0, true},
		{"-3", 0, true},
		{"garbage", 0, false},
		{now.Add(10 * time.Second).Format(http.TimeFormat), 10 * time.Second, true},
		{now.Add(-10 * time.Second).Format(http.TimeFormat), 0, true},
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.in, now)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseRetryAfter(%q) = (%v,%v), want (%v,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestOptionsApply(t *testing.T) {
	custom := &http.Client{Timeout: time.Second}
	c := NewClient("acct", "k",
		WithBaseURL("https://x.test/"),
		WithHTTPClient(custom),
		WithTimeout(5*time.Second),
		WithRetry(RetryConfig{MaxAttempts: 1}),
		WithRateLimit(50, 5),
	)
	if c.baseURL != "https://x.test" {
		t.Errorf("baseURL trailing slash not trimmed: %q", c.baseURL)
	}
	if c.httpClient != custom {
		t.Error("WithHTTPClient not applied")
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("WithTimeout not applied: %v", c.httpClient.Timeout)
	}
	if c.retry.MaxAttempts != 1 {
		t.Errorf("WithRetry not applied: %d", c.retry.MaxAttempts)
	}
	if c.limiter == nil {
		t.Error("WithRateLimit not applied")
	}
}

func TestNetworkErrorIsReturned(t *testing.T) {
	c := NewClient("acct", "k",
		WithBaseURL("http://127.0.0.1:1"), // connection refused
		WithRetry(RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}),
	)
	err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected a network error")
	}
}

func TestRateLimiterPacesRequests(t *testing.T) {
	rl := newRateLimiter(100, 1) // 1 burst, 100/s refill
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := rl.wait(ctx); err != nil {
			t.Fatalf("wait: %v", err)
		}
	}
	// 1 immediate + 2 waits of ~10ms each.
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Errorf("rate limiter did not pace requests: %v", elapsed)
	}
}
