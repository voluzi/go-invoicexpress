package invoicexpress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCancelPartialPaymentEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	var body struct {
		Receipt struct {
			State   string `json:"state"`
			Message string `json:"message"`
		} `json:"receipt"`
	}
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte("{}"))
	})
	if err := c.Invoices.CancelPartialPayment(context.Background(), 42, "Error"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/receipts/42/change-state.json" {
		t.Errorf("path = %s, want /receipts/42/change-state.json", gotPath)
	}
	if body.Receipt.State != "canceled" || body.Receipt.Message != "Error" {
		t.Errorf("body.receipt = %+v", body.Receipt)
	}
}

func TestTransportErrorRedactsAPIKey(t *testing.T) {
	c := NewClient("acct", "supersecretkey",
		WithBaseURL("http://127.0.0.1:1"), // connection refused
		WithRetry(RetryConfig{MaxAttempts: 1}),
	)
	err := c.do(context.Background(), http.MethodGet, "/invoices.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected a transport error")
	}
	if strings.Contains(err.Error(), "supersecretkey") {
		t.Errorf("api_key leaked in error message: %v", err)
	}
	if !strings.Contains(err.Error(), "REDACTED") {
		t.Errorf("expected api_key to be REDACTED in error: %v", err)
	}
}

func TestDecimalMarshalEscapesSpecialChars(t *testing.T) {
	// A pathological value must still produce valid JSON (no injection).
	b, err := json.Marshal(NewDecimal(`1"2\3`))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("MarshalJSON produced invalid JSON %s: %v", b, err)
	}
	if s != `1"2\3` {
		t.Errorf("round-trip lost data: %q", s)
	}
}

func TestDecimalUnmarshalMalformedDoesNotPanic(t *testing.T) {
	// Previously a lone quote sliced out of range and panicked.
	for _, in := range []string{`"`, `"abc`, `"1\`, `x`, `[`} {
		var d Decimal
		_ = d.UnmarshalJSON([]byte(in)) // must not panic; error is fine
	}
}

func TestServiceCreateValidatesBeforeNetwork(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte("{}"))
	})
	ctx := context.Background()

	if _, err := c.Estimates.Create(ctx, DocumentTypeQuote, &InvoiceCreateRequest{}); !IsValidation(err) {
		t.Errorf("estimates.Create: want ValidationError, got %v", err)
	}
	if _, err := c.Guides.Create(ctx, DocumentTypeTransport, &GuideCreateRequest{}); !IsValidation(err) {
		t.Errorf("guides.Create: want ValidationError, got %v", err)
	}
	if _, err := c.Clients.Create(ctx, &ClientCreateRequest{}); !IsValidation(err) {
		t.Errorf("clients.Create: want ValidationError, got %v", err)
	}
	if _, err := c.Items.Create(ctx, &ItemCreateRequest{}); !IsValidation(err) {
		t.Errorf("items.Create: want ValidationError, got %v", err)
	}
	if hit {
		t.Error("invalid requests must not reach the API")
	}
}

func TestItemValidationRequiresBothPriceAndQuantity(t *testing.T) {
	withItems := func(items []ItemRef) *InvoiceCreateRequest {
		return &InvoiceCreateRequest{
			Date:   NewDate(time.Now()),
			Client: ClientRef{Name: "X"},
			Items:  items,
		}
	}
	if err := withItems([]ItemRef{{Name: "X", UnitPrice: NewDecimal("10")}}).Validate(); err == nil {
		t.Error("item with only unit_price should fail validation")
	}
	if err := withItems([]ItemRef{{Name: "X", Quantity: NewDecimal("1")}}).Validate(); err == nil {
		t.Error("item with only quantity should fail validation")
	}
	if err := withItems([]ItemRef{{Name: "X", UnitPrice: NewDecimal("10"), Quantity: NewDecimal("1")}}).Validate(); err != nil {
		t.Errorf("item with both should pass: %v", err)
	}
}
