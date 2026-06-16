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

func TestDecimalUnmarshalRejectsNonNumeric(t *testing.T) {
	for _, in := range []string{`"not-a-number"`, `true`, `false`, `{}`, `[]`, `"abc"`, `"1e3"`} {
		var d Decimal
		if err := json.Unmarshal([]byte(in), &d); err == nil {
			t.Errorf("Decimal accepted invalid input %s (stored %q)", in, d.String())
		}
	}
	for _, in := range []string{`"50.00"`, `61.5`, `0`, `-1.5`, `null`, `""`} {
		var d Decimal
		if err := json.Unmarshal([]byte(in), &d); err != nil {
			t.Errorf("Decimal rejected valid input %s: %v", in, err)
		}
	}
}

func TestParseDecimalAndValid(t *testing.T) {
	for _, ok := range []string{"29.99", "0", "-1.50", "", "  12.30  "} {
		if _, err := ParseDecimal(ok); err != nil {
			t.Errorf("ParseDecimal(%q) unexpected error: %v", ok, err)
		}
	}
	for _, bad := range []string{"abc", "1e3", "1.2.3", "+5", "NaN", "{}"} {
		if _, err := ParseDecimal(bad); err == nil {
			t.Errorf("ParseDecimal(%q) should error", bad)
		}
	}
	if NewDecimal("oops").Valid() {
		t.Error("Valid() should be false for non-numeric")
	}
	if !NewDecimal("12.30").Valid() {
		t.Error("Valid() should be true for 12.30")
	}
}

func TestCancelPartialPaymentRequiresMessage(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte("{}"))
	})
	err := c.Invoices.CancelPartialPayment(context.Background(), 42, "   ")
	if !IsValidation(err) {
		t.Fatalf("want ValidationError for empty message, got %v", err)
	}
	if hit {
		t.Error("must not call the API to cancel without a message")
	}
}

func TestChangeStateCanceledRequiresMessage(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"invoice":{"id":1}}`))
	})
	ctx := context.Background()
	if _, err := c.Invoices.ChangeState(ctx, DocumentTypeInvoice, 1, StateCanceled, ""); !IsValidation(err) {
		t.Fatalf("cancel without message: want ValidationError, got %v", err)
	}
	if hit {
		t.Error("must not call the API to cancel without a message")
	}
	// A non-cancel transition without a message is fine.
	if _, err := c.Invoices.ChangeState(ctx, DocumentTypeInvoice, 1, StateFinalized, ""); err != nil {
		t.Errorf("finalize without message should succeed: %v", err)
	}
}

func TestErrorBodyRedactsAPIKey(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// A misbehaving proxy might echo the full request URL (incl. api_key).
		_, _ = w.Write([]byte(`{"error":"bad request to ` + r.URL.String() + `"}`))
	})
	err := c.do(context.Background(), http.MethodGet, "/x.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected an API error")
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Errorf("api_key leaked via error body: %v", err)
	}
}

func TestItemValidationRequiresUnitPrice(t *testing.T) {
	if err := (&ItemCreateRequest{Name: "X"}).Validate(); err == nil {
		t.Error("item without unit_price should fail validation")
	}
	if err := (&ItemCreateRequest{Name: "X", UnitPrice: NewDecimal("10")}).Validate(); err != nil {
		t.Errorf("item with name + unit_price should pass: %v", err)
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

func TestRequestCreationErrorRedactsAPIKey(t *testing.T) {
	c := NewClient("acct", "supersecretkey", WithBaseURL("https://x.test"))
	// An invalid method triggers a request-creation error (not a transport
	// error), and the URL containing the api_key would otherwise leak.
	err := c.do(context.Background(), "bad method with spaces", "/x.json", nil, nil, nil)
	if err == nil {
		t.Fatal("expected a request-creation error")
	}
	if strings.Contains(err.Error(), "supersecretkey") {
		t.Errorf("api_key leaked in request-creation error: %v", err)
	}
}

func TestRetryBackOffDoesNotOverflow(t *testing.T) {
	c := NewClient("acct", "k",
		WithBaseURL("https://x.test"),
		WithRetry(RetryConfig{MaxAttempts: 1000, BaseDelay: time.Millisecond, MaxDelay: time.Hour}),
	)
	// A huge attempt number must not panic or produce a negative duration.
	d := c.nextDelay(1000, nil)
	if d <= 0 {
		t.Errorf("nextDelay produced non-positive duration: %v", d)
	}
	if d > time.Hour {
		t.Errorf("nextDelay exceeded MaxDelay: %v", d)
	}
}

func TestEstimatesListAllPaginates(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			w.Write([]byte(`{"estimates":[{"id":1},{"id":2}],"pagination":{"current_page":1,"total_pages":2}}`))
		case "2":
			w.Write([]byte(`{"estimates":[{"id":3}],"pagination":{"current_page":2,"total_pages":2}}`))
		default:
			w.Write([]byte(`{"estimates":[],"pagination":{"current_page":3,"total_pages":2}}`))
		}
	})
	all, err := c.Estimates.ListAll(context.Background(), DocumentTypeQuote)
	if err != nil {
		t.Fatalf("list-all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d estimates across pages, want 3", len(all))
	}
}

func TestGuidesListAllPaginates(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			w.Write([]byte(`{"guides":[{"id":1},{"id":2}],"pagination":{"current_page":1,"total_pages":2}}`))
		case "2":
			w.Write([]byte(`{"guides":[{"id":3}],"pagination":{"current_page":2,"total_pages":2}}`))
		default:
			w.Write([]byte(`{"guides":[],"pagination":{"current_page":3,"total_pages":2}}`))
		}
	})
	all, err := c.Guides.ListAll(context.Background(), DocumentTypeTransport)
	if err != nil {
		t.Fatalf("list-all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d guides across pages, want 3", len(all))
	}
}

func TestDistinctEstimateAndGuideTypes(t *testing.T) {
	// Converting the aliases to distinct types means callers cannot pass an
	// Invoice where an Estimate is expected.
	var _ Estimate = Estimate{ID: 1}
	var _ Guide = Guide{ID: 1}
	// The compiler enforces the rest; this test documents the intent.
}

func TestUpdateRequestValidateDelegates(t *testing.T) {
	if err := (&InvoiceUpdateRequest{}).Validate(); !IsValidation(err) {
		t.Errorf("InvoiceUpdateRequest.Validate: want ValidationError, got %v", err)
	}
	if err := (&ClientUpdateRequest{}).Validate(); !IsValidation(err) {
		t.Errorf("ClientUpdateRequest.Validate: want ValidationError, got %v", err)
	}
	if err := (&ItemUpdateRequest{}).Validate(); !IsValidation(err) {
		t.Errorf("ItemUpdateRequest.Validate: want ValidationError, got %v", err)
	}
	if err := (&GuideUpdateRequest{}).Validate(); !IsValidation(err) {
		t.Errorf("GuideUpdateRequest.Validate: want ValidationError, got %v", err)
	}
	if err := (&TaxUpdateRequest{}).Validate(); !IsValidation(err) {
		t.Errorf("TaxUpdateRequest.Validate: want ValidationError, got %v", err)
	}
}
