package invoicexpress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
	const apiKey = "super+secret/key="
	cases := map[string]struct {
		method, path string
	}{
		// An invalid method triggers a request-creation error (not a transport
		// error). Its message omits the URL, but must still never leak the key.
		"invalid method": {"bad method with spaces", "/x.json"},
		// A control character in the path makes url.Parse fail with a message
		// that embeds the full URL — including ?api_key=… — which is itself
		// not a parseable URL, so a URL-only redactor would miss it.
		"unparseable url": {http.MethodGet, "/\x7f.json"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := NewClient("acct", apiKey, WithBaseURL("https://x.test"))
			err := c.do(context.Background(), tc.method, tc.path, nil, nil, nil)
			if err == nil {
				t.Fatal("expected a request-creation error")
			}
			if strings.Contains(err.Error(), apiKey) || strings.Contains(err.Error(), url.QueryEscape(apiKey)) {
				t.Errorf("api_key leaked in request-creation error: %v", err)
			}
		})
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

func TestEstimateAndGuideAliasesExposeInvoiceFields(t *testing.T) {
	// Estimate and Guide are aliases of Invoice for source compatibility while
	// keeping ergonomic, self-documenting return types.
	est := Estimate{ID: 1}
	gd := Guide{ID: 1}
	if est.ID != 1 || gd.ID != 1 {
		t.Fatalf("expected Estimate/Guide aliases to expose Invoice fields, got %+v %+v", est, gd)
	}
}

func TestGeneratePDFSurfacesErrorOn202(t *testing.T) {
	// A 202 carrying an undecodable body is a real error, not "still
	// generating" (which has an empty body). pollPDF must surface it instead of
	// retrying forever.
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{ not json`))
	})
	// A short deadline is a backstop: if the loop ever swallowed the error and
	// span forever, this would fail with a context error instead of "decode".
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.Invoices.GeneratePDF(ctx, 9, time.Millisecond)
	if err == nil {
		t.Fatal("expected a decode error to surface")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected the real decode error to surface, got %v", err)
	}
}

func TestTaxCreateValidatesBeforeNetwork(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"tax":{"id":1}}`))
	})
	ctx := context.Background()
	// Missing name and negative value must fail before any network call.
	if _, err := c.Taxes.Create(ctx, &TaxCreateRequest{Value: -1}); !IsValidation(err) {
		t.Errorf("taxes.Create: want ValidationError, got %v", err)
	}
	if hit {
		t.Error("invalid tax request must not reach the API")
	}
	// Zero-rate/exempt taxes are valid and must reach the server.
	if _, err := c.Taxes.Create(ctx, &TaxCreateRequest{Name: "IVA0", Value: 0}); err != nil {
		t.Errorf("valid tax create should succeed: %v", err)
	}
	if !hit {
		t.Error("valid tax request should reach the API")
	}
}

func TestItemValidationAcceptsZeroPrice(t *testing.T) {
	// A zero-priced catalog item (free item, 100%-discount line) is valid: only
	// an unset (empty) unit_price is rejected, matching validateItems.
	if err := (&ItemCreateRequest{Name: "Free sample", UnitPrice: NewDecimal("0")}).Validate(); err != nil {
		t.Errorf("zero-priced item should pass validation: %v", err)
	}
	if err := (&ItemCreateRequest{Name: "Free sample", UnitPrice: NewDecimal("0.00")}).Validate(); err != nil {
		t.Errorf("0.00-priced item should pass validation: %v", err)
	}
	// An unset unit_price is still rejected.
	if err := (&ItemCreateRequest{Name: "Widget"}).Validate(); !IsValidation(err) {
		t.Errorf("item without unit_price should fail validation, got %v", err)
	}
}

func TestSequenceCreateValidatesBeforeNetwork(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"sequence":{"id":1}}`))
	})
	ctx := context.Background()

	// A nil request and an empty serie_number must fail before any network call.
	if _, err := c.Sequences.Create(ctx, nil); !IsValidation(err) {
		t.Errorf("sequences.Create(nil): want ValidationError, got %v", err)
	}
	if _, err := c.Sequences.Create(ctx, &SequenceCreateRequest{}); !IsValidation(err) {
		t.Errorf("sequences.Create(empty): want ValidationError, got %v", err)
	}
	if _, err := c.Sequences.Create(ctx, &SequenceCreateRequest{SerieNumber: "   "}); !IsValidation(err) {
		t.Errorf("sequences.Create(blank): want ValidationError, got %v", err)
	}
	if hit {
		t.Error("invalid sequence request must not reach the API")
	}

	// A valid request must reach the server.
	if _, err := c.Sequences.Create(ctx, &SequenceCreateRequest{SerieNumber: "2026"}); err != nil {
		t.Errorf("valid sequence create should succeed: %v", err)
	}
	if !hit {
		t.Error("valid sequence request should reach the API")
	}
}

func TestTaxesListPageAndListAll(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taxes.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Write([]byte(`{"taxes":[{"id":1,"name":"IVA23"},{"id":2,"name":"IVA6"}],"pagination":{"current_page":1,"total_pages":2}}`))
		case "2":
			w.Write([]byte(`{"taxes":[{"id":3,"name":"Isento"}],"pagination":{"current_page":2,"total_pages":2}}`))
		default:
			w.Write([]byte(`{"taxes":[],"pagination":{"current_page":3,"total_pages":2}}`))
		}
	})
	ctx := context.Background()

	// List returns the first page only, preserving the existing signature.
	first, err := c.Taxes.List(ctx)
	if err != nil || len(first) != 2 {
		t.Fatalf("taxes.List: %v %v", err, first)
	}

	// ListPage exposes pagination metadata.
	page2, info, err := c.Taxes.ListPage(ctx, &ListOptions{Page: 2})
	if err != nil || len(page2) != 1 || info.CurrentPage != 2 || info.TotalPages != 2 {
		t.Fatalf("taxes.ListPage: %v %v %+v", err, page2, info)
	}

	// ListAll walks every page.
	all, err := c.Taxes.ListAll(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("taxes.ListAll: got %d taxes (%v), want 3", len(all), err)
	}
}

func TestSequencesListPageAndListAll(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sequences.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Write([]byte(`{"sequences":[{"id":1,"serie_number":"A"},{"id":2,"serie_number":"B"}],"pagination":{"current_page":1,"total_pages":2}}`))
		case "2":
			w.Write([]byte(`{"sequences":[{"id":3,"serie_number":"C"}],"pagination":{"current_page":2,"total_pages":2}}`))
		default:
			w.Write([]byte(`{"sequences":[],"pagination":{"current_page":3,"total_pages":2}}`))
		}
	})
	ctx := context.Background()

	first, err := c.Sequences.List(ctx)
	if err != nil || len(first) != 2 {
		t.Fatalf("sequences.List: %v %v", err, first)
	}
	page2, info, err := c.Sequences.ListPage(ctx, &ListOptions{Page: 2})
	if err != nil || len(page2) != 1 || info.CurrentPage != 2 {
		t.Fatalf("sequences.ListPage: %v %v %+v", err, page2, info)
	}
	all, err := c.Sequences.ListAll(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("sequences.ListAll: got %d (%v), want 3", len(all), err)
	}
}

func TestWithTimeoutIsOrderIndependent(t *testing.T) {
	const want = 7 * time.Second
	custom := &http.Client{}

	// WithTimeout after WithHTTPClient: the timeout must still apply.
	c1 := NewClient("acct", "k", WithHTTPClient(custom), WithTimeout(want))
	if c1.httpClient.Timeout != want {
		t.Errorf("timeout lost when WithTimeout follows WithHTTPClient: got %v, want %v", c1.httpClient.Timeout, want)
	}

	// WithTimeout before WithHTTPClient: the supplied client must not discard it.
	c2 := NewClient("acct", "k", WithTimeout(want), WithHTTPClient(&http.Client{}))
	if c2.httpClient.Timeout != want {
		t.Errorf("timeout lost when WithHTTPClient follows WithTimeout: got %v, want %v", c2.httpClient.Timeout, want)
	}

	// Without WithTimeout, a supplied client's own timeout is preserved.
	c3 := NewClient("acct", "k", WithHTTPClient(&http.Client{Timeout: 3 * time.Second}))
	if c3.httpClient.Timeout != 3*time.Second {
		t.Errorf("custom client timeout overwritten: got %v, want 3s", c3.httpClient.Timeout)
	}
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
