package invoicexpress

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// muxHandler returns a minimal valid JSON body for whichever endpoint is hit,
// so we can exercise every CRUD wrapper without a bespoke server each.
func muxHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/api/pdf/"):
			w.Write([]byte(`{"output":{"pdf_url":"https://x/d.pdf"}}`))
		case strings.Contains(p, "/qr_codes/"):
			w.Write([]byte(`{"qr_code":{"url":"https://x/qr","data":"d"}}`))
		case strings.Contains(p, "/partial_payments"):
			w.Write([]byte(`{"partial_payment":{"id":1,"amount":10}}`))
		case strings.Contains(p, "/related-documents"):
			w.Write([]byte(`{"invoices":[{"id":1}]}`))
		case strings.HasPrefix(p, "/quotes"), strings.HasPrefix(p, "/proformas"), strings.HasPrefix(p, "/fees_notes"):
			if strings.HasSuffix(p, "s.json") {
				w.Write([]byte(`{"estimates":[{"id":1}],"pagination":{"current_page":1,"total_pages":1}}`))
			} else {
				w.Write([]byte(`{"estimate":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/transports"), strings.HasPrefix(p, "/shippings"), strings.HasPrefix(p, "/devolutions"):
			if strings.HasSuffix(p, "s.json") {
				w.Write([]byte(`{"guides":[{"id":1}],"pagination":{"current_page":1,"total_pages":1}}`))
			} else {
				w.Write([]byte(`{"guide":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/clients"):
			if strings.HasSuffix(p, "/clients.json") {
				w.Write([]byte(`{"clients":[{"id":1}],"pagination":{"current_page":1,"total_pages":1}}`))
			} else {
				w.Write([]byte(`{"client":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/items"):
			if strings.HasSuffix(p, "/items.json") {
				w.Write([]byte(`{"items":[{"id":1}],"pagination":{"current_page":1,"total_pages":1}}`))
			} else {
				w.Write([]byte(`{"item":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/taxes"):
			if strings.HasSuffix(p, "/taxes.json") {
				w.Write([]byte(`{"taxes":[{"id":1}]}`))
			} else {
				w.Write([]byte(`{"tax":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/sequences"):
			if strings.HasSuffix(p, "/sequences.json") {
				w.Write([]byte(`{"sequences":[{"id":1}]}`))
			} else {
				w.Write([]byte(`{"sequence":{"id":1}}`))
			}
		case strings.HasPrefix(p, "/users/accounts"):
			w.Write([]byte(`{"account":{"id":1}}`))
		default: // invoice family
			// A collection path has one slash (/invoices.json); a single
			// document has two (/invoices/1.json).
			if strings.Count(p, "/") == 1 && strings.HasSuffix(p, "s.json") {
				w.Write([]byte(`{"invoices":[{"id":1}],"pagination":{"current_page":1,"total_pages":1}}`))
			} else {
				w.Write([]byte(`{"invoice":{"id":1}}`))
			}
		}
	}
}

func TestInvoicesRemainingMethods(t *testing.T) {
	c := newTestServer(t, muxHandler(t))
	ctx := context.Background()
	dt := DocumentTypeInvoice

	if err := c.Invoices.Update(ctx, dt, 1, &InvoiceUpdateRequest{Client: ClientRef{Name: "X"}}); err != nil {
		t.Errorf("Update: %v", err)
	}
	if _, err := c.Invoices.RelatedDocuments(ctx, dt, 1); err != nil {
		t.Errorf("RelatedDocuments: %v", err)
	}
	if err := c.Invoices.SendByEmail(ctx, dt, 1, &EmailRequest{Client: EmailClientRef{Email: "a@b.c"}}); err != nil {
		t.Errorf("SendByEmail: %v", err)
	}
	if _, err := c.Invoices.CreatePartialPayment(ctx, 1, &PartialPaymentRequest{Amount: NewDecimal("10"), PaymentMechanism: PaymentMechanismMBWay, PaymentDate: NewDate(time.Now())}); err != nil {
		t.Errorf("CreatePartialPayment: %v", err)
	}
	if err := c.Invoices.CancelPartialPayment(ctx, 1, 2); err != nil {
		t.Errorf("CancelPartialPayment: %v", err)
	}
	if _, err := c.Invoices.GetQRCode(ctx, 1); err != nil {
		t.Errorf("GetQRCode: %v", err)
	}
}

func TestEstimatesAllMethods(t *testing.T) {
	c := newTestServer(t, muxHandler(t))
	ctx := context.Background()
	dt := DocumentTypeQuote
	if _, err := c.Estimates.Get(ctx, dt, 1); err != nil {
		t.Errorf("Get: %v", err)
	}
	if _, _, err := c.Estimates.List(ctx, dt, nil); err != nil {
		t.Errorf("List: %v", err)
	}
	if err := c.Estimates.Update(ctx, dt, 1, &InvoiceUpdateRequest{Client: ClientRef{Name: "X"}}); err != nil {
		t.Errorf("Update: %v", err)
	}
	if _, err := c.Estimates.ChangeState(ctx, dt, 1, StateFinalized, ""); err != nil {
		t.Errorf("ChangeState: %v", err)
	}
	if err := c.Estimates.SendByEmail(ctx, dt, 1, &EmailRequest{Client: EmailClientRef{Email: "a@b.c"}}); err != nil {
		t.Errorf("SendByEmail: %v", err)
	}
	if _, err := c.Estimates.GeneratePDF(ctx, 1, time.Millisecond); err != nil {
		t.Errorf("GeneratePDF: %v", err)
	}
}

func TestGuidesAllMethods(t *testing.T) {
	c := newTestServer(t, muxHandler(t))
	ctx := context.Background()
	dt := DocumentTypeTransport
	if _, err := c.Guides.Get(ctx, dt, 1); err != nil {
		t.Errorf("Get: %v", err)
	}
	if _, _, err := c.Guides.List(ctx, dt, nil); err != nil {
		t.Errorf("List: %v", err)
	}
	if err := c.Guides.Update(ctx, dt, 1, &GuideUpdateRequest{Client: ClientRef{Name: "X"}}); err != nil {
		t.Errorf("Update: %v", err)
	}
	if _, err := c.Guides.ChangeState(ctx, dt, 1, StateFinalized, ""); err != nil {
		t.Errorf("ChangeState: %v", err)
	}
	if err := c.Guides.SendByEmail(ctx, dt, 1, &EmailRequest{Client: EmailClientRef{Email: "a@b.c"}}); err != nil {
		t.Errorf("SendByEmail: %v", err)
	}
	if _, err := c.Guides.GeneratePDF(ctx, 1, time.Millisecond); err != nil {
		t.Errorf("GeneratePDF: %v", err)
	}
}

func TestClientsItemsTaxesSequencesAccountsRemaining(t *testing.T) {
	c := newTestServer(t, muxHandler(t))
	ctx := context.Background()

	if _, _, err := c.Clients.List(ctx, &ListOptions{Page: 1, PerPage: 10}); err != nil {
		t.Errorf("Clients.List: %v", err)
	}
	if err := c.Clients.Update(ctx, 1, &ClientUpdateRequest{Name: "X"}); err != nil {
		t.Errorf("Clients.Update: %v", err)
	}
	if _, err := c.Clients.ListAll(ctx); err != nil {
		t.Errorf("Clients.ListAll: %v", err)
	}

	if _, _, err := c.Items.List(ctx, nil); err != nil {
		t.Errorf("Items.List: %v", err)
	}
	if _, err := c.Items.Get(ctx, 1); err != nil {
		t.Errorf("Items.Get: %v", err)
	}
	if err := c.Items.Update(ctx, 1, &ItemUpdateRequest{Name: "X", UnitPrice: NewDecimal("1")}); err != nil {
		t.Errorf("Items.Update: %v", err)
	}
	if _, err := c.Items.ListAll(ctx); err != nil {
		t.Errorf("Items.ListAll: %v", err)
	}

	if _, err := c.Taxes.Get(ctx, 1); err != nil {
		t.Errorf("Taxes.Get: %v", err)
	}
	if _, err := c.Taxes.Create(ctx, &TaxCreateRequest{Name: "IVA23", Value: 23}); err != nil {
		t.Errorf("Taxes.Create: %v", err)
	}
	if err := c.Taxes.Update(ctx, 1, &TaxUpdateRequest{Name: "IVA23", Value: 23}); err != nil {
		t.Errorf("Taxes.Update: %v", err)
	}
	if err := c.Taxes.Delete(ctx, 1); err != nil {
		t.Errorf("Taxes.Delete: %v", err)
	}

	if _, err := c.Sequences.List(ctx); err != nil {
		t.Errorf("Sequences.List: %v", err)
	}
	if _, err := c.Sequences.Get(ctx, 1); err != nil {
		t.Errorf("Sequences.Get: %v", err)
	}
	if _, err := c.Sequences.Create(ctx, &SequenceCreateRequest{SerieNumber: "A"}); err != nil {
		t.Errorf("Sequences.Create: %v", err)
	}

	if _, err := c.Accounts.Get(ctx, 1); err != nil {
		t.Errorf("Accounts.Get: %v", err)
	}
}
