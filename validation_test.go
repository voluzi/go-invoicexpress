package invoicexpress

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestInvoiceCreateRequestValidate(t *testing.T) {
	valid := &InvoiceCreateRequest{
		Date:   NewDate(time.Now()),
		Client: ClientRef{Name: "ACME"},
		Items:  []ItemRef{{Name: "X", UnitPrice: NewDecimal("10"), Quantity: NewDecimal("1")}},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid request rejected: %v", err)
	}

	bad := &InvoiceCreateRequest{}
	err := bad.Validate()
	if !IsValidation(err) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestCreateValidatesBeforeNetwork(t *testing.T) {
	// The handler must never be hit when the request is invalid.
	hit := false
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{"invoice":{"id":1}}`))
	})
	_, err := c.Invoices.Create(context.Background(), DocumentTypeInvoice, &InvoiceCreateRequest{})
	if !IsValidation(err) {
		t.Fatalf("expected client-side validation error, got %v", err)
	}
	if hit {
		t.Error("invalid request should not reach the API")
	}
}

func TestCreateAndFinalize(t *testing.T) {
	var states []string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Write([]byte(`{"invoice":{"id":42,"status":"draft"}}`))
			return
		}
		states = append(states, r.URL.Path)
		w.Write([]byte(`{"invoice":{"id":42,"status":"finalized"}}`))
	})
	inv, err := c.Invoices.CreateAndFinalize(context.Background(), DocumentTypeInvoiceReceipt, &InvoiceCreateRequest{
		Date:   NewDate(time.Now()),
		Client: ClientRef{Name: "ACME"},
		Items:  []ItemRef{{Name: "Pro", UnitPrice: NewDecimal("29"), Quantity: NewDecimal("1")}},
	})
	if err != nil {
		t.Fatalf("create-and-finalize: %v", err)
	}
	if inv.Status != "finalized" {
		t.Errorf("status = %q, want finalized", inv.Status)
	}
	if len(states) != 1 || states[0] != "/invoice_receipts/42/change-state.json" {
		t.Errorf("change-state not called as expected: %v", states)
	}
}

func TestValidateOtherRequests(t *testing.T) {
	if (&ClientCreateRequest{}).Validate() == nil {
		t.Error("empty client should fail validation")
	}
	if (&ItemCreateRequest{}).Validate() == nil {
		t.Error("empty item should fail validation")
	}
	if (&GuideCreateRequest{}).Validate() == nil {
		t.Error("empty guide should fail validation")
	}
	if err := (&ClientCreateRequest{Name: "OK"}).Validate(); err != nil {
		t.Errorf("valid client rejected: %v", err)
	}
}
