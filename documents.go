package invoicexpress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// The sole-key fallback accepts only the GENERIC wrappers: the fixed keys the
// documentation uses for every type in a family, plus the "document" pair for
// endpoints (related-documents) keyed by neither the requested type nor one of
// its own.
//
// It deliberately does NOT accept another specific type. Doing so would let
// {"credit_note": …} answer a request for an invoice-receipt — adopting the
// wrong legal document, which is worse than failing.
var knownDocumentKeys = map[string]bool{
	"invoice": true, "estimate": true, "guide": true, "document": true,
}

var knownDocumentListKeys = map[string]bool{
	"invoices": true, "estimates": true, "guides": true, "documents": true,
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
		// One key, and it has to be a document key we recognise. Adopting
		// whatever single object is present would let {"credit_note": …} answer
		// a request for an invoice-receipt, or a 200 error body decode to a
		// zero document.
		for k, v := range raw {
			if knownDocumentKeys[k] {
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
	if len(raw) == 0 {
		return nil
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
		// documents. An error body like {"errors":[…]} would otherwise decode
		// to a list of zero-valued documents.
		var candidates []string
		for k := range raw {
			if k != "pagination" {
				candidates = append(candidates, k)
			}
		}
		if len(candidates) == 1 && knownDocumentListKeys[candidates[0]] {
			key, payload, ok = candidates[0], raw[candidates[0]], true
		}
	}
	if !ok {
		if _, paginationOnly := raw["pagination"]; paginationOnly && len(raw) == 1 {
			return nil
		}
		return fmt.Errorf("invoicexpress: response has no %q list (keys: %s)",
			string(e.docType), strings.Join(sortedKeys(raw), ", "))
	}
	if err := json.Unmarshal(payload, &e.docs); err != nil {
		return fmt.Errorf("invoicexpress: decode %q: %w", key, err)
	}
	return nil
}
