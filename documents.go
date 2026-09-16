package invoicexpress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// family is the documented generic wrapper for this document type's family:
// "invoice" for invoices, simplified invoices, receipts and notes; "estimate"
// for quotes, proformas and fees notes; "guide" for transport documents.
func (d DocumentType) family() string {
	switch d {
	case DocumentTypeQuote, DocumentTypeProforma, DocumentTypeFeesNote:
		return "estimate"
	case DocumentTypeShipping, DocumentTypeTransport, DocumentTypeDevolution:
		return "guide"
	default:
		return "invoice"
	}
}

// fallbackKeys are the wrapper names accepted when the document type's own key
// is absent: the documented generic wrapper for ITS OWN family, plus the
// "document" pair that related-documents uses.
//
// Family matters. A flat list of generic wrappers would let {"estimate": …}
// answer a request for an invoice-receipt, adopting a quote as a legal
// document — and refusing is always better than adopting the wrong one.
func (d DocumentType) fallbackKeys() (single, plural map[string]bool) {
	f := d.family()
	return map[string]bool{f: true, "document": true},
		map[string]bool{f + "s": true, "documents": true}
}

// The document endpoints wrap their payload in a key named after the document
// type: GET /invoice_receipts/42.json answers {"invoice_receipt": {...}} and
// GET /invoice_receipts.json answers {"invoice_receipts": [...]}. The published
// examples show a fixed {"invoice": ...} / {"invoices": [...]} for every type,
// which is what this library was written to.
//
// Decoding the real payload into the documented key does not fail — JSON
// simply finds no such key and leaves the value zero. So Get returned an empty
// document and List returned an empty slice, both with a nil error, for every
// type except plain invoices. That is the dangerous shape of this bug: a
// duplicate-check that scans the returned list concludes "nothing exists yet"
// every single time, and issues a second legally-binding document.

// singularKey is the wrapper key for a single document of this type. The path
// segment is the plural, and the wrapper is its singular.
func (d DocumentType) singularKey() string {
	return strings.TrimSuffix(string(d), "s")
}

// sortedKeys lists the keys of a decoded object, for error messages that say
// what the server actually sent.
func sortedKeys(raw map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Only responses are decoded this way. Request bodies keep the fixed key the
// API documents for every document type ({"invoice": ...}, {"estimate": ...},
// {"guide": ...}): the docs are the only evidence about what the endpoints
// accept, and inventing a doc-typed request key from the response shape would
// risk breaking document creation — the one call that must work — to fix a
// problem nothing has observed.

// documentEnvelope decodes a single-document response, taking the wrapper key
// from the document type. When the key is absent but the object holds exactly
// one key, that one is used: endpoints like related-documents wrap their
// payload under a name of their own, and guessing right is better than
// silently returning a zero document.
type documentEnvelope struct {
	docType DocumentType
	doc     Invoice
}

func (e *documentEnvelope) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// Every caller of this envelope — Create, Get, ChangeState — is owed a
	// document. An empty body is therefore a failure, not a tolerable answer:
	// returning nil here would hand back a zero document with a nil error,
	// which is the silent shape this file exists to eliminate. A finalized
	// receipt whose id never reached the caller is a legal document nothing
	// records, and the next redelivery issues a second one.
	if len(raw) == 0 {
		return errors.New("invoicexpress: response carries no document")
	}

	key := e.docType.singularKey()
	payload, ok := raw[key]
	if !ok && len(raw) == 1 {
		// One key, and it has to be a wrapper for this document's own family.
		// Adopting whatever single object is present would let {"credit_note":
		// …} answer a request for an invoice-receipt, or a 200 error body
		// decode to a zero document.
		accepted, _ := e.docType.fallbackKeys()
		for k, v := range raw {
			if accepted[k] {
				key, payload, ok = k, v, true
			}
		}
	}
	if !ok {
		return fmt.Errorf("invoicexpress: response has no %q document (keys: %s)",
			e.docType.singularKey(), strings.Join(sortedKeys(raw), ", "))
	}
	if string(bytes.TrimSpace(payload)) == "null" {
		return fmt.Errorf("invoicexpress: response has a null %q document", key)
	}
	if err := json.Unmarshal(payload, &e.doc); err != nil {
		return fmt.Errorf("invoicexpress: decode %q: %w", key, err)
	}
	// A wrapper with nothing usable in it — {"invoice_receipt":{}} or a payload
	// carrying an error instead of a document — is the same silent zero as an
	// absent key, and loses the identity of a document that may already exist.
	if e.doc.ID == 0 {
		return fmt.Errorf("invoicexpress: %q in the response has no id", key)
	}
	return nil
}

// documentListEnvelope decodes a list response, taking the wrapper key from
// the document type and the pagination block from its own fixed key. A list
// endpoint that omits pagination (as /taxes.json does) simply leaves it zero.
type documentListEnvelope struct {
	docType DocumentType
	docs    []Invoice
	page    PageInfo
}

func (e *documentListEnvelope) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// An absent collection is NOT an empty one. This is the read an idempotency
	// check depends on: "no documents, no error" is how a caller concludes that
	// a receipt it already issued does not exist, and issues it a second time.
	// A genuinely empty account answers {"invoice_receipts":[]}, which decodes
	// to an empty slice here — the difference is the whole point.
	if len(raw) == 0 {
		return errors.New("invoicexpress: response carries no document list")
	}

	if p, ok := raw["pagination"]; ok {
		if err := json.Unmarshal(p, &e.page); err != nil {
			return fmt.Errorf("invoicexpress: decode pagination: %w", err)
		}
	}

	key := string(e.docType)
	payload, ok := raw[key]
	if !ok {
		// The only other key, ignoring pagination — and only if it names
		// documents of this document's own family. An error body like
		// {"errors":[…]} would otherwise decode to a list of zero documents.
		_, accepted := e.docType.fallbackKeys()
		var candidates []string
		for k := range raw {
			if k != "pagination" {
				candidates = append(candidates, k)
			}
		}
		if len(candidates) == 1 && accepted[candidates[0]] {
			key, payload, ok = candidates[0], raw[candidates[0]], true
		}
	}
	if !ok {
		return fmt.Errorf("invoicexpress: response has no %q list (keys: %s)",
			string(e.docType), strings.Join(sortedKeys(raw), ", "))
	}
	if string(bytes.TrimSpace(payload)) == "null" {
		return fmt.Errorf("invoicexpress: response has a null %q list", key)
	}
	if err := json.Unmarshal(payload, &e.docs); err != nil {
		return fmt.Errorf("invoicexpress: decode %q: %w", key, err)
	}
	// Entries have to be documents too. [null, {}, {"error":"failed"}] decodes
	// happily into zero-valued documents, and a duplicate check scanning their
	// ids or proprietary uids would find no match — the same false "nothing
	// exists here" the collection-level guard above refuses.
	for i := range e.docs {
		if e.docs[i].ID == 0 {
			return fmt.Errorf("invoicexpress: %q entry %d has no id", key, i)
		}
	}
	return nil
}
