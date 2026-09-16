package invoicexpress

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func validInvoiceCreateRequest() *InvoiceCreateRequest {
	return &InvoiceCreateRequest{
		Date:           NewDate(time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)),
		Client:         ClientRef{Name: "ACME"},
		Items:          []ItemRef{{Name: "Service", UnitPrice: NewDecimal("10"), Quantity: NewDecimal("1")}},
		ProprietaryUID: "order-123",
	}
}

func validGuideCreateRequest() *GuideCreateRequest {
	return &GuideCreateRequest{
		Date:        NewDate(time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)),
		DueDate:     NewDate(time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)),
		LoadedAt:    NewDateTime(time.Date(2026, 9, 16, 19, 30, 45, 0, time.UTC)),
		Client:      ClientRef{Name: "ACME"},
		Items:       []ItemRef{{Name: "Service", Description: "Consulting", UnitPrice: NewDecimal("10"), Quantity: NewDecimal("1")}},
		AddressFrom: &AddressInfo{Detail: "Rua 5", City: "Lisboa", PostalCode: "1000-555", Country: "Portugal"},
		AddressTo:   &AddressInfo{Detail: "Avenida 10", City: "Porto", PostalCode: "2000-555", Country: "Portugal"},
	}
}

func TestInvoiceCreateSendsProprietaryUIDBesideInvoice(t *testing.T) {
	var body map[string]json.RawMessage
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/invoices.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"invoice":{"id":123}}`))
	})

	if _, err := c.Invoices.Create(context.Background(), DocumentTypeInvoice, validInvoiceCreateRequest()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := string(body["proprietary_uid"]); got != `"order-123"` {
		t.Fatalf("proprietary_uid = %s, want top-level string", got)
	}
	var invoice map[string]json.RawMessage
	if err := json.Unmarshal(body["invoice"], &invoice); err != nil {
		t.Fatalf("decode invoice: %v", err)
	}
	if _, ok := invoice["proprietary_uid"]; ok {
		t.Fatal("proprietary_uid must not be nested inside invoice")
	}
}

func TestInvoiceUpdateOmitsProprietaryUID(t *testing.T) {
	var body map[string]json.RawMessage
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/invoices/123.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := c.Invoices.Update(context.Background(), DocumentTypeInvoice, 123, validInvoiceCreateRequest()); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, ok := body["proprietary_uid"]; ok {
		t.Fatal("invoice updates must not send a top-level proprietary_uid")
	}
	var invoice map[string]json.RawMessage
	if err := json.Unmarshal(body["invoice"], &invoice); err != nil {
		t.Fatalf("decode invoice: %v", err)
	}
	if _, ok := invoice["proprietary_uid"]; ok {
		t.Fatal("invoice updates must not nest proprietary_uid")
	}
}

func TestInvoiceSequenceIDAcceptsStringIntegerAndNull(t *testing.T) {
	tests := []struct {
		name string
		wire string
		want string
	}{
		{name: "string", wire: `"42"`, want: "42"},
		{name: "integer", wire: `42`, want: "42"},
		{name: "null", wire: `null`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"invoice":{"id":1,"sequence_id":` + tt.wire + `}}`
			invoice, err := contractClient(t, "/invoices/1.json", body).Invoices.Get(context.Background(), DocumentTypeInvoice, 1)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if invoice.SequenceID != tt.want {
				t.Fatalf("sequence id = %q, want %q", invoice.SequenceID, tt.want)
			}
		})
	}
}

func TestInvoiceSequenceIDRejectsInvalidJSONShapes(t *testing.T) {
	for _, wire := range []string{`1.5`, `true`, `{}`, `[]`} {
		t.Run(wire, func(t *testing.T) {
			body := `{"invoice":{"id":1,"sequence_id":` + wire + `}}`
			if invoice, err := contractClient(t, "/invoices/1.json", body).Invoices.Get(context.Background(), DocumentTypeInvoice, 1); err == nil {
				t.Fatalf("get returned %+v, want sequence_id decode error", invoice)
			}
		})
	}
}

func TestCreatePartialPaymentDecodesReceipt(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"receipt":{"id":987,"status":"finalized","total":"10.00","date":"16/09/2026"}}`))
	})

	request := &PartialPaymentRequest{
		PaymentMechanism: PaymentMechanismTransfer,
		Note:             "Bank transfer",
		Serie:            "A",
		Amount:           NewDecimal("10.00"),
		PaymentDate:      NewDate(time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)),
	}
	payment, err := c.Invoices.CreatePartialPayment(context.Background(), 123, request)
	if err != nil {
		t.Fatalf("create partial payment: %v", err)
	}
	if payment.ID != 987 {
		t.Fatalf("legacy ID = %d, want 987", payment.ID)
	}
	if payment.Receipt.ID != 987 || payment.Receipt.Total.String() != "10.00" {
		t.Fatalf("receipt = %+v, want full receipt document", payment.Receipt)
	}
	if payment.PaymentMechanism != request.PaymentMechanism || payment.Note != request.Note || payment.Serie != request.Serie {
		t.Fatalf("legacy request fields = %+v", payment)
	}
	if payment.Amount.String() != "10.00" || payment.PaymentDate.String() != "16/09/2026" {
		t.Fatalf("receipt-derived fields = %+v", payment)
	}
}

func TestCreatePartialPaymentRejectsMissingReceipt(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`{"receipt":null}`,
		`{"receipt":{}}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			payment, err := c.Invoices.CreatePartialPayment(context.Background(), 123, &PartialPaymentRequest{})
			if err == nil {
				t.Fatalf("returned %+v, want an error", payment)
			}
		})
	}
}

func TestCreatePartialPaymentRejectsNilRequest(t *testing.T) {
	c := NewClient("acct", "key")
	if payment, err := c.Invoices.CreatePartialPayment(context.Background(), 123, nil); !IsValidation(err) {
		t.Fatalf("payment = %+v, error = %v, want validation error", payment, err)
	}
}

func TestGeneratePDFDecodesOfficialURL(t *testing.T) {
	responses := []struct {
		status int
		body   string
	}{
		{status: http.StatusAccepted},
		{status: http.StatusOK, body: `{"output":{"pdfUrl":"https://example.test/invoice.pdf"}}`},
	}
	index := 0
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := responses[index]
		index++
		w.WriteHeader(response.status)
		_, _ = w.Write([]byte(response.body))
	})

	url, err := c.Invoices.GeneratePDF(context.Background(), 42, time.Millisecond)
	if err != nil {
		t.Fatalf("generate pdf: %v", err)
	}
	if url != "https://example.test/invoice.pdf" {
		t.Fatalf("url = %q, want official pdfUrl", url)
	}
}

func TestGeneratePDFRejectsCompletedResponseWithoutURL(t *testing.T) {
	for _, body := range []string{`{}`, `{"output":{}}`, `{"output":{"pdfUrl":"   "}}`} {
		t.Run(body, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			if url, err := c.Invoices.GeneratePDF(context.Background(), 42, time.Millisecond); err == nil {
				t.Fatalf("url = %q, want an error", url)
			}
		})
	}
}

func TestSAFTExportDecodesTopLevelURLVariants(t *testing.T) {
	for _, body := range []string{
		`{"url":"https://example.test/saft.zip"}`,
		`{"Url":"https://example.test/saft.zip"}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			result, err := c.SAFT.Export(context.Background(), 9, 2026, time.Millisecond)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if result.URL != "https://example.test/saft.zip" {
				t.Fatalf("url = %q", result.URL)
			}
			if result.PDFURL != "" || result.XMLURL != "" {
				t.Fatalf("deprecated compatibility fields must remain empty: %+v", result)
			}
		})
	}
}

func TestSAFTExportDistinguishesNoDocuments(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":"No documents for selected period"}`))
	})
	result, err := c.SAFT.Export(context.Background(), 9, 2026, time.Millisecond)
	if !errors.Is(err, ErrNoSAFTDocuments) {
		t.Fatalf("result = %+v, error = %v, want ErrNoSAFTDocuments", result, err)
	}
}

func TestSAFTExportRejectsCompletedResponseWithoutURL(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if result, err := c.SAFT.Export(context.Background(), 9, 2026, time.Millisecond); err == nil {
		t.Fatalf("result = %+v, want an error", result)
	}
}

func TestRelatedDocumentsUsesDocumentRoute(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/document/42/related_documents.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"documents":[{"id":43}]}`))
	})
	documents, err := c.Invoices.RelatedDocuments(context.Background(), DocumentTypeInvoice, 42)
	if err != nil {
		t.Fatalf("related documents: %v", err)
	}
	if len(documents) != 1 || documents[0].ID != 43 {
		t.Fatalf("documents = %+v", documents)
	}
}

func TestClientsFindByNameDecodesSingularClient(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/clients/find-by-name.json" || r.URL.Query().Get("client_name") != "ACME" {
			t.Fatalf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"client":{"id":321,"name":"ACME"}}`))
	})
	clients, err := c.Clients.FindByName(context.Background(), "ACME")
	if err != nil {
		t.Fatalf("find by name: %v", err)
	}
	if len(clients) != 1 || clients[0].ID != 321 {
		t.Fatalf("clients = %+v", clients)
	}
}

func TestClientsFindByNameRejectsMissingClientIdentity(t *testing.T) {
	for _, body := range []string{`{}`, `{"client":null}`, `{"client":{}}`} {
		t.Run(body, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			if clients, err := c.Clients.FindByName(context.Background(), "ACME"); err == nil {
				t.Fatalf("clients = %+v, want an error", clients)
			}
		})
	}
}

func TestClientsListInvoicesUsesPOSTWithPagination(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/clients/77/invoices.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "15" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"invoices":[{"id":9}],"pagination":{"current_page":2,"total_pages":2}}`))
	})
	invoices, _, err := c.Clients.ListInvoices(context.Background(), 77, &ListOptions{Page: 2, PerPage: 15})
	if err != nil {
		t.Fatalf("list invoices: %v", err)
	}
	if len(invoices) != 1 || invoices[0].ID != 9 {
		t.Fatalf("invoices = %+v", invoices)
	}
}

func TestClientsListInvoicesRetriesServerFailureAsReadOnlyOperation(t *testing.T) {
	var attempts atomic.Int32
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/clients/77/invoices.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"invoices":[{"id":9}],"pagination":{"current_page":1,"total_pages":1}}`))
	})
	c.retry = RetryConfig{MaxAttempts: 2, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}

	invoices, _, err := c.Clients.ListInvoices(context.Background(), 77, nil)
	if err != nil {
		t.Fatalf("list invoices: %v", err)
	}
	if attempts.Load() != 2 || len(invoices) != 1 || invoices[0].ID != 9 {
		t.Fatalf("attempts = %d, invoices = %+v", attempts.Load(), invoices)
	}
}

func TestDocumentClientReferenceValidation(t *testing.T) {
	references := []struct {
		name    string
		client  ClientRef
		wantErr bool
	}{
		{name: "id only", client: ClientRef{ID: 12}},
		{name: "code only", client: ClientRef{Code: "ACME"}},
		{name: "name only", client: ClientRef{Name: "ACME"}},
		{name: "empty", client: ClientRef{}, wantErr: true},
		{name: "non-positive id", client: ClientRef{ID: -1}, wantErr: true},
		{name: "negative id masks code", client: ClientRef{ID: -1, Code: "ACME"}, wantErr: true},
	}
	for _, tt := range references {
		t.Run(tt.name, func(t *testing.T) {
			invoice := validInvoiceCreateRequest()
			invoice.Client = tt.client
			if err := invoice.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("invoice validation error = %v, wantErr %v", err, tt.wantErr)
			}

			estimate := *invoice
			if err := estimate.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("estimate validation error = %v, wantErr %v", err, tt.wantErr)
			}

			guide := validGuideCreateRequest()
			guide.Client = tt.client
			if err := guide.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("guide validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientReferenceSerializesIDCodeAndName(t *testing.T) {
	data, err := json.Marshal(ClientRef{ID: 7, Code: "CODE", Name: "Name"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != float64(7) || body["code"] != "CODE" || body["name"] != "Name" {
		t.Fatalf("client reference = %s", data)
	}
}

func TestDateTimeJSONRoundTrip(t *testing.T) {
	want := time.Date(2026, 9, 16, 19, 30, 45, 0, time.UTC)
	data, err := json.Marshal(NewDateTime(want))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"16/09/2026 19:30:45"` {
		t.Fatalf("marshal = %s", data)
	}
	var got DateTime
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("round trip = %v, want %v", got.Time, want)
	}
	if err := json.Unmarshal([]byte(`"2026-09-16T19:30:45Z"`), &got); err == nil {
		t.Fatal("accepted an invalid date-time format")
	}
}

func TestGuideCreateRequestSerializesDocumentedFields(t *testing.T) {
	req := validGuideCreateRequest()
	req.LicensePlate = "11-AA-22"
	req.ManualSequenceNumber = "1"
	req.TaxExemptionReason = "M00"
	req.LoadSite = "Lisbon, Portugal"
	req.DeliverySite = "Madrid, Spain"
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := map[string]string{
		"loaded_at":              "16/09/2026 19:30:45",
		"license_plate":          "11-AA-22",
		"manual_sequence_number": "1",
		"tax_exemption_reason":   "M00",
		"load_site":              "Lisbon, Portugal",
		"delivery_site":          "Madrid, Spain",
	}
	for field, value := range want {
		if body[field] != value {
			t.Errorf("%s = %#v, want %q", field, body[field], value)
		}
	}
}

func TestGuideCreateValidationRequiresDocumentedFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GuideCreateRequest)
	}{
		{name: "due date", mutate: func(r *GuideCreateRequest) { r.DueDate = Date{} }},
		{name: "loaded at", mutate: func(r *GuideCreateRequest) { r.LoadedAt = DateTime{} }},
		{name: "source address", mutate: func(r *GuideCreateRequest) { r.AddressFrom = nil }},
		{name: "source detail", mutate: func(r *GuideCreateRequest) { r.AddressFrom.Detail = "" }},
		{name: "source city", mutate: func(r *GuideCreateRequest) { r.AddressFrom.City = "" }},
		{name: "source postal code", mutate: func(r *GuideCreateRequest) { r.AddressFrom.PostalCode = "" }},
		{name: "source country", mutate: func(r *GuideCreateRequest) { r.AddressFrom.Country = "" }},
		{name: "destination address", mutate: func(r *GuideCreateRequest) { r.AddressTo = nil }},
		{name: "destination detail", mutate: func(r *GuideCreateRequest) { r.AddressTo.Detail = "" }},
		{name: "destination city", mutate: func(r *GuideCreateRequest) { r.AddressTo.City = "" }},
		{name: "destination postal code", mutate: func(r *GuideCreateRequest) { r.AddressTo.PostalCode = "" }},
		{name: "destination country", mutate: func(r *GuideCreateRequest) { r.AddressTo.Country = "" }},
		{name: "item description", mutate: func(r *GuideCreateRequest) { r.Items[0].Description = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validGuideCreateRequest()
			tt.mutate(req)
			if err := req.Validate(); !IsValidation(err) {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
	if err := validGuideCreateRequest().Validate(); err != nil {
		t.Fatalf("complete guide rejected: %v", err)
	}
}

func TestGuideUpdateOmitsUnsetCreateOnlyFields(t *testing.T) {
	var body string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body = string(data)
		w.WriteHeader(http.StatusOK)
	})

	err := c.Guides.Update(context.Background(), DocumentTypeShipping, 7, &GuideUpdateRequest{Observations: "updated"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if body != `{"shipping":{"observations":"updated"}}` {
		t.Fatalf("body = %s", body)
	}

	body = ""
	err = c.Guides.Update(context.Background(), DocumentTypeShipping, 7, &GuideUpdateRequest{
		Client:      ClientRef{ID: 42},
		Items:       []ItemRef{},
		AddressFrom: &AddressInfo{City: "Lisboa"},
	})
	if err != nil {
		t.Fatalf("update explicit fields: %v", err)
	}
	if body != `{"shipping":{"address_from":{"city":"Lisboa"},"client":{"id":42},"items":[]}}` {
		t.Fatalf("body = %s", body)
	}
}

func TestEstimateWritesUseQuoteEnvelopeForEveryType(t *testing.T) {
	types := []DocumentType{DocumentTypeQuote, DocumentTypeProforma, DocumentTypeFeesNote}
	operations := []string{"create", "update", "change-state"}
	for _, docType := range types {
		for _, operation := range operations {
			t.Run(string(docType)+"/"+operation, func(t *testing.T) {
				var body map[string]json.RawMessage
				c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
					data, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("read body: %v", err)
					}
					if err := json.Unmarshal(data, &body); err != nil {
						t.Fatalf("decode body: %v", err)
					}
					if operation == "create" || operation == "change-state" {
						key := docType.singularKey()
						_, _ = w.Write([]byte(`{"` + key + `":{"id":1}}`))
						return
					}
					w.WriteHeader(http.StatusOK)
				})

				switch operation {
				case "create":
					_, _ = c.Estimates.Create(context.Background(), docType, validInvoiceCreateRequest())
				case "update":
					_ = c.Estimates.Update(context.Background(), docType, 1, validInvoiceCreateRequest())
				case "change-state":
					_, _ = c.Estimates.ChangeState(context.Background(), docType, 1, StateFinalized, "")
				}
				if len(body) != 1 || body["quote"] == nil {
					t.Fatalf("body keys = %v, want only quote", body)
				}
			})
		}
	}
}

func TestGuideWritesUseShippingEnvelopeForEveryType(t *testing.T) {
	types := []DocumentType{DocumentTypeShipping, DocumentTypeTransport, DocumentTypeDevolution}
	operations := []string{"create", "update", "change-state"}
	for _, docType := range types {
		for _, operation := range operations {
			t.Run(string(docType)+"/"+operation, func(t *testing.T) {
				var body map[string]json.RawMessage
				c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
					data, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("read body: %v", err)
					}
					if err := json.Unmarshal(data, &body); err != nil {
						t.Fatalf("decode body: %v", err)
					}
					if operation == "create" || operation == "change-state" {
						key := docType.singularKey()
						_, _ = w.Write([]byte(`{"` + key + `":{"id":1}}`))
						return
					}
					w.WriteHeader(http.StatusOK)
				})

				switch operation {
				case "create":
					_, _ = c.Guides.Create(context.Background(), docType, validGuideCreateRequest())
				case "update":
					_ = c.Guides.Update(context.Background(), docType, 1, validGuideCreateRequest())
				case "change-state":
					_, _ = c.Guides.ChangeState(context.Background(), docType, 1, StateFinalized, "")
				}
				if len(body) != 1 || body["shipping"] == nil {
					t.Fatalf("body keys = %v, want only shipping", body)
				}
			})
		}
	}
}

func TestPaymentMechanismConstants(t *testing.T) {
	tests := []struct {
		name string
		got  PaymentMechanism
		want string
	}{
		{name: "transfer", got: PaymentMechanismTransfer, want: "TB"},
		{name: "multibanco", got: PaymentMechanismMultiBanco, want: "MB"},
		{name: "cash", got: PaymentMechanismCash, want: "NU"},
		{name: "debit card", got: PaymentMechanismDebitCard, want: "CD"},
		{name: "credit card", got: PaymentMechanismCreditCard, want: "CC"},
		{name: "bank check", got: PaymentMechanismCheck, want: "CH"},
		{name: "check or voucher", got: PaymentMechanismCheckOrVoucher, want: "CO"},
		{name: "compensation", got: PaymentMechanismCompensation, want: "CS"},
		{name: "other", got: PaymentMechanismOther, want: "OU"},
		{name: "mb way compatibility", got: PaymentMechanismMBWay, want: "MW"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(struct {
				Mechanism PaymentMechanism `json:"payment_mechanism"`
			}{Mechanism: tt.got})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			want := `{"payment_mechanism":"` + tt.want + `"}`
			if string(data) != want {
				t.Fatalf("marshal = %s, want %s", data, want)
			}
		})
	}
}

func TestSequenceCreateUsesDocumentedRequestAndPluralResponse(t *testing.T) {
	var body map[string]map[string]any
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sequences.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"sequences":{"id":146090,"serie":"2026","default_sequence":1}}`))
	})

	sequence, err := c.Sequences.Create(context.Background(), &SequenceCreateRequest{
		SerieNumber:     "2026",
		DefaultSequence: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sequence.ID != 146090 || sequence.SerieNumber != "2026" {
		t.Fatalf("sequence = %+v", sequence)
	}
	request := body["sequence"]
	if request["serie"] != "2026" || request["default_sequence"] != "1" {
		t.Fatalf("request = %+v", body)
	}
	if _, ok := request["serie_number"]; ok {
		t.Fatal("request must not contain serie_number")
	}
}

func TestSequenceGetAcceptsPluralAndLegacyWrappers(t *testing.T) {
	for _, body := range []string{
		`{"sequences":{"id":7,"serie":"A"}}`,
		`{"sequence":{"id":7,"serie_number":"A"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/sequences/7.json" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(body))
			})
			sequence, err := c.Sequences.Get(context.Background(), 7)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if sequence.ID != 7 || sequence.SerieNumber != "A" {
				t.Fatalf("sequence = %+v", sequence)
			}
		})
	}
}

func TestSequencesRegisterUsesExactRouteAndDecodesCollection(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/sequences/7/register.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"sequences":[{"id":7,"serie":"A","current_invoice_number":3,"current_invoice_sequence_id":7,"current_invoice_validation_code":"ABCD1234"}]}`))
	})
	sequences, err := c.Sequences.Register(context.Background(), 7)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(sequences) != 1 {
		t.Fatalf("sequences = %+v", sequences)
	}
	sequence := sequences[0]
	if sequence.CurrentInvoiceNumber != 3 || sequence.CurrentInvoiceSequenceID != 7 || sequence.CurrentInvoiceValidationCode != "ABCD1234" {
		t.Fatalf("sequence registration fields = %+v", sequence)
	}
}

func TestSequenceRegistrationPreservesStructuredErrorCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   string
	}{
		{name: "conflict", statusCode: http.StatusConflict, body: `{"errors":{"code":"007","message":"All Sequences are registered in AT."}}`, wantCode: "007"},
		{name: "unprocessable", statusCode: http.StatusUnprocessableEntity, body: `{"errors":{"code":"001","message":"Sequence name not allowed."}}`, wantCode: "001"},
		{name: "legacy top level", statusCode: http.StatusUnprocessableEntity, body: `{"code":"000","message":"Unknown error."}`, wantCode: "000"},
		{name: "nested null falls back", statusCode: http.StatusConflict, body: `{"code":"009","errors":{"code":null,"message":"Conflict."}}`, wantCode: "009"},
		{name: "nested blank falls back", statusCode: http.StatusUnprocessableEntity, body: `{"code":"008","errors":{"code":"  ","message":"Invalid."}}`, wantCode: "008"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/sequences/7/register.json" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			})
			_, err := c.Sequences.Register(context.Background(), 7)
			apiErr, ok := AsAPIError(err)
			if !ok {
				t.Fatalf("error = %v, want APIError", err)
			}
			if apiErr.StatusCode != tc.statusCode || apiErr.Code != tc.wantCode {
				t.Fatalf("API error = %+v, want status %d and code %q", apiErr, tc.statusCode, tc.wantCode)
			}
		})
	}
}

func TestSequencesRegisterRejectsMissingNullAndIDLessCollections(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "missing", body: `{}`, want: "no sequences"},
		{name: "null", body: `{"sequences":null}`, want: "null sequences"},
		{name: "idless", body: `{"sequences":[{"serie":"A"}]}`, want: "has no id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			})
			_, err := c.Sequences.Register(context.Background(), 7)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Register error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSequencesRegisterDoesNotRetry(t *testing.T) {
	for _, statusCode := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			var attempts atomic.Int32
			c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.WriteHeader(statusCode)
			})
			c.retry = RetryConfig{MaxAttempts: 3, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}

			if _, err := c.Sequences.Register(context.Background(), 7); err == nil {
				t.Fatal("Register returned no error")
			}
			if attempts.Load() != 1 {
				t.Fatalf("attempts = %d, want 1", attempts.Load())
			}
		})
	}
}

func TestAccountsGetUsesDocumentedRouteAndFields(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/accounts/42/get.json" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"account":{"organization_name":"Company","fiscal_id":"508000111","email":null,"state":"active","at_configured":true,"trial":true}}`))
	})
	account, err := c.Accounts.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if account.Organization != "Company" || account.FiscalID != "508000111" || account.Email != "" {
		t.Fatalf("account identity = %+v", account)
	}
	if account.State != "active" || !account.ATConfigured || !account.Trial {
		t.Fatalf("account state = %+v", account)
	}
}

func TestAccountDecodesLegacyOrganizationField(t *testing.T) {
	var account Account
	if err := json.Unmarshal([]byte(`{"id":7,"organization":"Legacy Company"}`), &account); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if account.Organization != "Legacy Company" {
		t.Fatalf("Organization = %q, want legacy value", account.Organization)
	}
}
