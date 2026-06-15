package invoicexpress

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestClientsCRUDAndFinders(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/clients.json":
			w.Write([]byte(`{"client":{"id":1,"name":"ACME"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/clients/1.json":
			w.Write([]byte(`{"client":{"id":1,"name":"ACME"}}`))
		case r.URL.Path == "/clients/find-by-name.json":
			if r.URL.Query().Get("client_name") != "ACME" {
				t.Errorf("missing client_name param")
			}
			w.Write([]byte(`{"clients":[{"id":1,"name":"ACME"}]}`))
		case r.URL.Path == "/clients/find-by-code.json":
			w.Write([]byte(`{"client":{"id":1,"code":"AC"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/clients/1/invoices.json":
			w.Write([]byte(`{"invoices":[{"id":9}],"pagination":{"current_page":1,"total_pages":1}}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	ctx := context.Background()

	cust, err := c.Clients.Create(ctx, &ClientCreateRequest{Name: "ACME"})
	if err != nil || cust.ID != 1 {
		t.Fatalf("create: %v %+v", err, cust)
	}
	if _, err := c.Clients.Get(ctx, 1); err != nil {
		t.Fatalf("get: %v", err)
	}
	found, err := c.Clients.FindByName(ctx, "ACME")
	if err != nil || len(found) != 1 {
		t.Fatalf("find-by-name: %v %v", err, found)
	}
	if _, err := c.Clients.FindByCode(ctx, "AC"); err != nil {
		t.Fatalf("find-by-code: %v", err)
	}
	invs, _, err := c.Clients.ListInvoices(ctx, 1, nil)
	if err != nil || len(invs) != 1 {
		t.Fatalf("list-invoices: %v %v", err, invs)
	}
}

func TestItemsCRUD(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/items/5.json":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost:
			w.Write([]byte(`{"item":{"id":5,"name":"Widget"}}`))
		default:
			w.Write([]byte(`{"item":{"id":5,"name":"Widget"}}`))
		}
	})
	ctx := context.Background()
	it, err := c.Items.Create(ctx, &ItemCreateRequest{Name: "Widget", UnitPrice: NewDecimal("10")})
	if err != nil || it.ID != 5 {
		t.Fatalf("create: %v %+v", err, it)
	}
	if err := c.Items.Delete(ctx, 5); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestTaxesList(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taxes.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"taxes":[{"id":1,"name":"IVA23","value":23}]}`))
	})
	taxes, err := c.Taxes.List(context.Background())
	if err != nil || len(taxes) != 1 || taxes[0].Value != 23 {
		t.Fatalf("taxes.list: %v %v", err, taxes)
	}
}

func TestSequencesSetCurrent(t *testing.T) {
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/sequences/3/set_current.json" {
			hit = true
		}
		w.Write([]byte(`{}`))
	})
	if err := c.Sequences.SetCurrent(context.Background(), 3); err != nil {
		t.Fatalf("set-current: %v", err)
	}
	if !hit {
		t.Error("set_current endpoint not called")
	}
}

func TestSAFTExportPolls(t *testing.T) {
	first := true
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/export_saft.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("month") != "6" || r.URL.Query().Get("year") != "2026" {
			t.Errorf("bad params: %s", r.URL.RawQuery)
		}
		if first {
			first = false
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Write([]byte(`{"output":{"xml_url":"https://x/saft.xml","pdf_url":"https://x/saft.pdf"}}`))
	})
	res, err := c.SAFT.Export(context.Background(), 6, 2026, time.Millisecond)
	if err != nil {
		t.Fatalf("saft.export: %v", err)
	}
	if res.XMLURL == "" {
		t.Error("missing xml url")
	}
}

func TestAccountsList(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/accounts.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"accounts":[{"id":1,"name":"My Co"}]}`))
	})
	accts, err := c.Accounts.List(context.Background())
	if err != nil || len(accts) != 1 {
		t.Fatalf("accounts.list: %v %v", err, accts)
	}
}

func TestEstimatesCreateWrapsAsEstimate(t *testing.T) {
	var raw string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quotes.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		raw = string(b)
		w.Write([]byte(`{"estimate":{"id":2}}`))
	})
	est, err := c.Estimates.Create(context.Background(), DocumentTypeQuote, &InvoiceCreateRequest{
		Client: ClientRef{Name: "X"},
	})
	if err != nil || est.ID != 2 {
		t.Fatalf("estimates.create: %v %+v", err, est)
	}
	if !contains(raw, "estimate") {
		t.Errorf("estimate body not wrapped: %s", raw)
	}
}

func TestGuidesCreateWrapsAsGuide(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transports.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"guide":{"id":4}}`))
	})
	g, err := c.Guides.Create(context.Background(), DocumentTypeTransport, &GuideCreateRequest{
		Client: ClientRef{Name: "X"},
	})
	if err != nil || g.ID != 4 {
		t.Fatalf("guides.create: %v %+v", err, g)
	}
}
