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
		{"id":101,"name":"IVA23","value":"23.0","region":"PT","code":null,"default_tax":1},
		{"id":102,"name":"IVA18","value":"18.0","region":"PT-AC","code":null,"default_tax":0},
		{"id":103,"name":"Isento","value":"0.0","region":null,"code":null,"default_tax":0}
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
		"type":"InvoiceReceipt","sequence_number":"1/AA","total":"12.30",
		"items":[{"name":"Pro plan","unit_price":"10.00","quantity":"1.0",
		"tax":{"id":101,"name":"IVA23","value":23.0}}]}}`

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
		{"id":501,"status":"settled","sequence_number":"42/AA","total":"12.30",
		 "proprietary_uid":"in_test_0001"},
		{"id":502,"status":"draft","sequence_number":"","total":"1.23",
		 "proprietary_uid":"in_test_0002"}
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
		if d.ProprietaryUID == "in_test_0001" {
			found = true
			if d.SequenceNumber != "42/AA" {
				t.Errorf("SequenceNumber = %q, want 42/AA", d.SequenceNumber)
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

func TestDocumentEnvelopeRefusesABodyWithNoDocument(t *testing.T) {
	// Every caller of this envelope is owed a document. Returning a zero one
	// with a nil error is how a finalized receipt gets issued and then not
	// recorded — and re-issued on the next redelivery.
	for _, body := range []string{
		`{}`,
		`{"invoice_receipt":null}`,
		// A 200 that carries an error payload instead of a document.
		`{"error":{"message":"boom"}}`,
		// A document of another type entirely.
		`{"credit_note":{"id":9,"type":"CreditNote"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			doc, err := contractClient(t, "/invoice_receipts/7.json", body).
				Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7)
			if err == nil {
				t.Fatalf("Get returned %+v with no error, want an error", doc)
			}
		})
	}
}

func TestDocumentEnvelopeRefusesAnEmptyResponseBody(t *testing.T) {
	// A 200 with no body at all never reaches the envelope — the client skipped
	// decoding when there was nothing to decode, and handed back a zero
	// document with a nil error. CreateAndFinalize then asked the API to
	// finalize document id 0.
	_, err := contractClient(t, "/invoice_receipts/7.json", "").
		Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7)
	if err == nil {
		t.Fatal("Get succeeded on an empty body, want an error")
	}
}

func TestSynchronousCallRefusesAnEmptyAcceptedBody(t *testing.T) {
	// 202 with no body is meaningful only to the async pollers. On an ordinary
	// document call it must not pass: it would hand back a document with no id,
	// and CreateAndFinalize would then finalize document 0.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	c := NewClient("acct", "test-key", WithBaseURL(srv.URL))
	if doc, err := c.Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7); err == nil {
		t.Fatalf("Get returned %+v on an empty 202, want an error", doc)
	}
	if doc, err := c.Invoices.Create(context.Background(), DocumentTypeInvoiceReceipt, &InvoiceCreateRequest{
		Client: ClientRef{Name: "ACME"},
		Items:  []ItemRef{{Name: "Pro", UnitPrice: NewDecimal("10.00"), Quantity: NewDecimal("1")}},
	}); err == nil {
		t.Fatalf("Create returned %+v on an empty 202, want an error", doc)
	}
}

func TestDocumentEnvelopeRefusesAWrapperWithoutAnID(t *testing.T) {
	for _, body := range []string{
		`{"invoice_receipt":{}}`,
		`{"invoice_receipt":{"error":"failed"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			doc, err := contractClient(t, "/invoice_receipts/7.json", body).
				Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 7)
			if err == nil {
				t.Fatalf("Get returned %+v with no error, want an error", doc)
			}
		})
	}
}

func TestDocumentEnvelopeRefusesAnotherFamilysWrapper(t *testing.T) {
	// "estimate" is a documented generic wrapper, but for quotes — not for a
	// legal invoice-receipt. Adopting it would file a quote as a receipt.
	_, err := contractClient(t, "/invoice_receipts/9.json", `{"estimate":{"id":9,"type":"Quote"}}`).
		Invoices.Get(context.Background(), DocumentTypeInvoiceReceipt, 9)
	if err == nil {
		t.Fatal("Get accepted an estimate wrapper for an invoice-receipt, want an error")
	}

	_, listErr := contractClient(t, "/invoice_receipts.json", `{"guides":[{"id":9}]}`).
		Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
	if listErr == nil {
		t.Fatal("ListAll accepted a guides list for invoice-receipts, want an error")
	}
}

func TestDocumentListRefusesAMissingCollection(t *testing.T) {
	// The read an idempotency check depends on. "No documents, no error" is how
	// a caller concludes a receipt it already issued does not exist — and
	// issues a second legally-binding one.
	for _, body := range []string{
		`{}`,
		`{"pagination":{"total_entries":10,"total_pages":1,"current_page":1,"per_page":25}}`,
		`{"invoice_receipts":null}`,
	} {
		t.Run(body, func(t *testing.T) {
			docs, err := contractClient(t, "/invoice_receipts.json", body).
				Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
			if err == nil {
				t.Fatalf("ListAll returned %d documents with no error, want an error", len(docs))
			}
		})
	}
}

func TestDocumentListRefusesEntriesWithoutAnID(t *testing.T) {
	// A collection whose entries are unusable is not a collection of documents.
	// Scanning these for a proprietary_uid finds no match, which reads as "not
	// issued yet" — and issues a duplicate.
	for _, body := range []string{
		`{"invoice_receipts":[null]}`,
		`{"invoice_receipts":[{}]}`,
		`{"invoice_receipts":[{"id":1},{"error":"failed"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			docs, err := contractClient(t, "/invoice_receipts.json", body).
				Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
			if err == nil {
				t.Fatalf("ListAll returned %d documents with no error, want an error", len(docs))
			}
		})
	}
}

func TestTaxesListRefusesANullRate(t *testing.T) {
	// A tax whose rate is unknown must not read as 0%: a caller matching the
	// rate Stripe charged could otherwise stamp IVA23 on an untaxed line.
	const body = `{"taxes":[{"id":101,"name":"IVA23","value":null,"region":"PT","default_tax":1}]}`

	taxes, err := contractClient(t, "/taxes.json", body).Taxes.ListAll(context.Background())
	if err == nil {
		t.Fatalf("ListAll returned %+v with no error, want an error", taxes)
	}
}

func TestDocumentListAcceptsAGenuinelyEmptyCollection(t *testing.T) {
	// The flip side: an account with none must still be an empty list, not an
	// error, or a first issuance could never happen.
	docs, err := contractClient(t, "/invoice_receipts.json",
		`{"invoice_receipts":[],"pagination":{"total_entries":0,"total_pages":0,"current_page":1,"per_page":25}}`).
		Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
	if err != nil {
		t.Fatalf("ListAll on an empty account: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("got %d documents, want 0", len(docs))
	}
}

func TestDocumentListEnvelopeRefusesANonDocumentKey(t *testing.T) {
	// {"errors":[…]} would otherwise decode as a list of zero-valued documents,
	// which reads as "this account has no such document".
	const body = `{"errors":[{"error":"nope"}],"pagination":{"total_pages":1}}`

	_, err := contractClient(t, "/invoice_receipts.json", body).
		Invoices.ListAll(context.Background(), DocumentTypeInvoiceReceipt)
	if err == nil {
		t.Fatal("ListAll succeeded on an error body, want an error")
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
		{"id":201,"serie":"A","default_sequence":1,"current_invoice_number":70},
		{"id":202,"serie":"BB","default_sequence":0,"current_invoice_number":0}
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
	if seqs[1].SerieNumber != "BB" {
		t.Errorf("SerieNumber = %q, want BB", seqs[1].SerieNumber)
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
		{in: `"6"`, want: 6},
		// An explicit null rate is unknown, not zero: the API nulls a tax's
		// region and code, never its value.
		{in: `null`, wantErr: true},
		// A blank rate is not a zero rate: the API writes "0.0" for zero-rated
		// tax, so blank is something unexpected and must not be read as 0%.
		{in: `""`, wantErr: true},
		{in: `"   "`, wantErr: true},
		{in: `"abc"`, wantErr: true},
		// ParseFloat would take all of these. A NaN rate is the worst of them:
		// it compares unequal to every rate, so a caller looking for the rate
		// that was charged silently finds no tax at all.
		{in: `"NaN"`, wantErr: true},
		{in: `"Inf"`, wantErr: true},
		{in: `"+Inf"`, wantErr: true},
		{in: `"+23"`, wantErr: true},
		{in: `"0x17p0"`, wantErr: true},
		{in: `"1_0"`, wantErr: true},
		{in: `"1e1"`, wantErr: true},
		{in: `"23."`, wantErr: true},
		{in: `".5"`, wantErr: true},
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
