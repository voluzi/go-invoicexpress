package invoicexpress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestInvoicesCreate(t *testing.T) {
	var gotMethod, gotPath, gotAPIKey string
	var gotBody map[string]json.RawMessage
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAPIKey = r.URL.Query().Get("api_key")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"invoice":{"id":123,"status":"draft","total":61.5,"sequence_number":"FT 2026/1"}}`))
	})

	inv, err := c.Invoices.Create(context.Background(), DocumentTypeInvoiceReceipt, &InvoiceCreateRequest{
		Date:   NewDate(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)),
		Client: ClientRef{Name: "ACME", FiscalID: "500000000"},
		Items: []ItemRef{
			{Name: "Plano Pro", UnitPrice: NewDecimal("50"), Quantity: NewDecimal("1"), Tax: &TaxRef{Name: "IVA23"}},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/invoice_receipts.json" {
		t.Errorf("path = %s, want /invoice_receipts.json", gotPath)
	}
	if gotAPIKey != "test-key" {
		t.Errorf("api_key = %s", gotAPIKey)
	}
	if _, ok := gotBody["invoice"]; !ok {
		t.Errorf("request body not wrapped in {invoice:...}: %v", gotBody)
	}
	if inv.ID != 123 || inv.SequenceNumber != "FT 2026/1" {
		t.Errorf("decoded invoice wrong: %+v", inv)
	}
}

func TestInvoicesGet(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/invoices/55.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"invoice":{"id":55,"status":"finalized"}}`))
	})
	inv, err := c.Invoices.Get(context.Background(), DocumentTypeInvoice, 55)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if inv.ID != 55 || inv.Status != "finalized" {
		t.Errorf("got %+v", inv)
	}
}

func TestInvoicesChangeState(t *testing.T) {
	var body struct {
		Invoice ChangeStateRequest `json:"invoice"`
	}
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/7/change-state.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Write([]byte(`{"invoice":{"id":7,"status":"finalized"}}`))
	})
	_, err := c.Invoices.ChangeState(context.Background(), DocumentTypeInvoice, 7, StateFinalized, "")
	if err != nil {
		t.Fatalf("change-state: %v", err)
	}
	if body.Invoice.State != StateFinalized {
		t.Errorf("state sent = %q, want finalized", body.Invoice.State)
	}
}

func TestInvoicesGeneratePDFPolls(t *testing.T) {
	var attempts int32
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pdf/9.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusAccepted) // 202: still generating
			return
		}
		w.Write([]byte(`{"output":{"pdf_url":"https://x/inv.pdf"}}`))
	})
	url, err := c.Invoices.GeneratePDF(context.Background(), 9, time.Millisecond)
	if err != nil {
		t.Fatalf("generate-pdf: %v", err)
	}
	if url != "https://x/inv.pdf" {
		t.Errorf("pdf url = %s", url)
	}
	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected polling, attempts = %d", attempts)
	}
}

func TestInvoicesListAllPaginates(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			w.Write([]byte(`{"invoices":[{"id":1},{"id":2}],"pagination":{"current_page":1,"total_pages":2}}`))
		default:
			w.Write([]byte(`{"invoices":[{"id":3}],"pagination":{"current_page":2,"total_pages":2}}`))
		}
	})
	all, err := c.Invoices.ListAll(context.Background(), DocumentTypeInvoice)
	if err != nil {
		t.Fatalf("list-all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d invoices across pages, want 3", len(all))
	}
}

func TestInvoicesCreateValidationError(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"date":["is required"]}}`))
	})
	// Valid client-side so the request reaches the server's 422.
	_, err := c.Invoices.Create(context.Background(), DocumentTypeInvoice, &InvoiceCreateRequest{
		Date:   NewDate(time.Now()),
		Client: ClientRef{Name: "ACME"},
		Items:  []ItemRef{{Name: "X", UnitPrice: NewDecimal("1"), Quantity: NewDecimal("1")}},
	})
	if !IsUnprocessable(err) {
		t.Fatalf("expected 422, got %v", err)
	}
	apiErr, _ := AsAPIError(err)
	if len(apiErr.Errors) == 0 {
		t.Error("expected parsed validation messages")
	}
}
