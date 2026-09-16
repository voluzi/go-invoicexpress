package invoicexpress

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The payloads below are the shapes the live API actually returns, captured
// from a real account. They are the contract these tests defend: the library
// was originally written against the published examples, which disagree with
// the API on three points — a rate is a string, a boolean is 1/0, and an unset
// region or code is null. Decoding the real payload into the documented types
// failed outright, so an account with a perfectly good tax table could not
// issue a single document.

func contractClient(t *testing.T, path, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return NewClient("acct", "test-key", WithBaseURL(srv.URL))
}

func TestTaxesListDecodesTheLiveWireShape(t *testing.T) {
	// value is a STRING, the default is "default_tax": 1 (not "is_default":
	// true), region and code are null, and there is no pagination object.
	const body = `{"taxes":[
		{"id":198036,"name":"IVA23","value":"23.0","region":"PT","code":null,"default_tax":1},
		{"id":198037,"name":"IVA18","value":"18.0","region":"PT-AC","code":null,"default_tax":0},
		{"id":198039,"name":"Isento","value":"0.0","region":null,"code":null,"default_tax":0}
	]}`

	taxes, err := contractClient(t, "/taxes.json", body).Taxes.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(taxes) != 3 {
		t.Fatalf("got %d taxes, want 3", len(taxes))
	}

	iva23 := taxes[0]
	if iva23.Name != "IVA23" {
		t.Errorf("Name = %q, want IVA23", iva23.Name)
	}
	if iva23.Value != 23 {
		t.Errorf("Value = %v, want 23 (decoded from the string \"23.0\")", iva23.Value)
	}
	if !iva23.IsDefault {
		t.Error("IsDefault = false, want true — the API marks it with \"default_tax\": 1")
	}
	if iva23.Region != "PT" {
		t.Errorf("Region = %q, want PT", iva23.Region)
	}

	if taxes[1].Value != 18 {
		t.Errorf("IVA18 Value = %v, want 18", taxes[1].Value)
	}
	if taxes[1].IsDefault {
		t.Error("IVA18 IsDefault = true, want false — \"default_tax\": 0")
	}
	// A null region must decode as empty, not fail the whole read.
	if taxes[2].Region != "" {
		t.Errorf("Isento Region = %q, want empty for a null region", taxes[2].Region)
	}
	if taxes[2].Value != 0 {
		t.Errorf("Isento Value = %v, want 0", taxes[2].Value)
	}
}

func TestDocumentTaxDecodesNumericRate(t *testing.T) {
	// The same rate is a JSON *number* when embedded in a document, which is
	// why Rate has to accept both shapes rather than simply becoming a string.
	const body = `{"invoice_receipt":{"id":42,"status":"settled","archived":false,
		"type":"InvoiceReceipt","sequence_number":"1/VER","total":"12.30",
		"items":[{"name":"Veridom Pro","unit_price":"10.00","quantity":"1.0",
		"tax":{"id":198036,"name":"IVA23","value":23.0}}]}}`

	inv, err := contractClient(t, "/invoice_receipts/42.json", body).
		Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(inv.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(inv.Items))
	}
	if got := inv.Items[0].Tax.Value; got != 23 {
		t.Errorf("item tax Value = %v, want 23 (decoded from the number 23.0)", got)
	}
	if inv.Archived {
		t.Error("Archived = true, want false")
	}
}

func TestInvoiceListDecodesTheDocumentTypedKey(t *testing.T) {
	// The regression that matters most: the list is keyed "invoice_receipts",
	// not "invoices". Decoding the documented key found nothing and returned an
	// empty slice with a nil error, so a caller scanning the list for an
	// already-issued document concluded "none exists" every time — and issued a
	// second legally-binding receipt on the next Stripe redelivery.
	const body = `{"invoice_receipts":[
		{"id":266232704,"status":"settled","sequence_number":"42/VER","total":"12.30",
		 "proprietary_uid":"in_1UGEZhJdLj4pHYGE8dBzXoh5"},
		{"id":266232705,"status":"draft","sequence_number":"","total":"1.23",
		 "proprietary_uid":"in_other"}
	],"pagination":{"total_entries":2,"current_page":1,"total_pages":1,"per_page":25}}`

	docs, err := contractClient(t, "/invoice_receipts.json", body).
		Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d documents, want 2 — an empty list here is the duplicate-issuance bug", len(docs))
	}

	var found bool
	for _, d := range docs {
		if d.ProprietaryUID == "in_1UGEZhJdLj4pHYGE8dBzXoh5" {
			found = true
			if d.SequenceNumber != "42/VER" {
				t.Errorf("SequenceNumber = %q, want 42/VER", d.SequenceNumber)
			}
			if d.Total.String() != "12.30" {
				t.Errorf("Total = %q, want 12.30", d.Total.String())
			}
		}
	}
	if !found {
		t.Error("the document was not found by its proprietary_uid — the idempotency lookup would issue a duplicate")
	}
}

func TestDocumentEnvelopeAcceptsTheDocumentedKey(t *testing.T) {
	// The published examples wrap every type as {"invoice": ...}. Whatever the
	// API does today, a response in the documented shape must still decode.
	const body = `{"invoice":{"id":7,"status":"draft","total":"5.00"}}`

	inv, err := contractClient(t, "/invoice_receipts/7.json", body).
		Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if inv.ID != 7 {
		t.Errorf("ID = %d, want 7 — the sole-key fallback should have found it", inv.ID)
	}
}

func TestDocumentEnvelopeRefusesAnAmbiguousBody(t *testing.T) {
	// Two plausible keys and neither is the document type: guessing would be
	// how a wrong document gets adopted. Fail loudly instead.
	const body = `{"invoice":{"id":7},"credit_note":{"id":9}}`

	_, err := contractClient(t, "/invoice_receipts/7.json", body).
		Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7)
	if err == nil {
		t.Fatal("Get succeeded on an ambiguous body, want an error")
	}
}

func TestSequencesListDecodesSerieAndNumericDefault(t *testing.T) {
	// The API returns "serie", while the create request takes "serie_number".
	const body = `{"sequences":[
		{"id":1184817,"serie":"A","default_sequence":1,"current_invoice_number":70},
		{"id":1184900,"serie":"VER","default_sequence":0,"current_invoice_number":0}
	]}`

	seqs, err := contractClient(t, "/sequences.json", body).Sequences.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(seqs) != 2 {
		t.Fatalf("got %d sequences, want 2", len(seqs))
	}
	if seqs[0].SerieNumber != "A" {
		t.Errorf("SerieNumber = %q, want A — filled from the \"serie\" key", seqs[0].SerieNumber)
	}
	if !seqs[0].DefaultSequence {
		t.Error("DefaultSequence = false, want true — \"default_sequence\": 1")
	}
	if seqs[1].SerieNumber != "VER" {
		t.Errorf("SerieNumber = %q, want VER", seqs[1].SerieNumber)
	}
	if seqs[1].DefaultSequence {
		t.Error("DefaultSequence = true, want false")
	}
}

func TestSequenceAcceptsTheDocumentedKeyToo(t *testing.T) {
	var s Sequence
	if err := json.Unmarshal([]byte(`{"id":7,"serie_number":"B","default_sequence":false}`), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.SerieNumber != "B" {
		t.Errorf("SerieNumber = %q, want B", s.SerieNumber)
	}
	if s.DefaultSequence {
		t.Error("DefaultSequence = true, want false")
	}
}

func TestRateUnmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    Rate
		wantErr bool
	}{
		{in: `23.0`, want: 23},
		{in: `"23.0"`, want: 23},
		{in: `"23"`, want: 23},
		{in: `0`, want: 0},
		{in: `"0.0"`, want: 0},
		{in: `null`, want: 0},
		{in: `""`, want: 0},
		{in: `"6"`, want: 6},
		{in: `"abc"`, wantErr: true},
		{in: `"23%"`, wantErr: true},
		{in: `{}`, wantErr: true},
		{in: `[]`, wantErr: true},
		{in: `true`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var r Rate
			err := json.Unmarshal([]byte(tc.in), &r)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("unmarshal(%s) = %v, want an error", tc.in, r)
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal(%s): %v", tc.in, err)
			}
			if r != tc.want {
				t.Errorf("unmarshal(%s) = %v, want %v", tc.in, r, tc.want)
			}
		})
	}
}

func TestRateMarshalsAsNumber(t *testing.T) {
	b, err := json.Marshal(TaxCreateRequest{Name: "IVA23", Value: 23})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["value"].(float64); !ok {
		t.Errorf("value marshalled as %T, want a JSON number: %s", got["value"], b)
	}
}

func TestFlagUnmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    Flag
		wantErr bool
	}{
		{in: `true`, want: true},
		{in: `false`, want: false},
		{in: `1`, want: true},
		{in: `0`, want: false},
		{in: `"1"`, want: true},
		{in: `"0"`, want: false},
		{in: `"true"`, want: true},
		{in: `"false"`, want: false},
		{in: `null`, want: false},
		{in: `""`, want: false},
		{in: `2`, wantErr: true},
		{in: `"maybe"`, wantErr: true},
		{in: `{}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var f Flag
			err := json.Unmarshal([]byte(tc.in), &f)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("unmarshal(%s) = %v, want an error", tc.in, f)
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal(%s): %v", tc.in, err)
			}
			if f != tc.want {
				t.Errorf("unmarshal(%s) = %v, want %v", tc.in, f, tc.want)
			}
		})
	}
}

func TestFlagMarshalsAsBoolean(t *testing.T) {
	b, err := json.Marshal(struct {
		Archived Flag `json:"archived"`
	}{Archived: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `{"archived":true}` {
		t.Errorf("marshalled %s, want {\"archived\":true}", b)
	}
}
