package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// InvoicesService handles invoice document operations.
type InvoicesService struct {
	client *Client
}

// invoiceWrapper is used for JSON serialization of invoice requests.
type invoiceWrapper struct {
	Invoice interface{} `json:"invoice"`
}

// invoiceResponse is the JSON response for a single invoice.
type invoiceResponse struct {
	Invoice Invoice `json:"invoice"`
}

// invoiceListResponse is the JSON response for a list of invoices.
type invoiceListResponse struct {
	Invoices   []Invoice `json:"invoices"`
	Pagination PageInfo  `json:"pagination"`
}

// Create creates a new invoice document of the given type.
func (s *InvoicesService) Create(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Invoice, error) {
	path := fmt.Sprintf("/%s.json", docType)
	var resp invoiceResponse
	if err := s.client.do(ctx, http.MethodPost, path, nil, invoiceWrapper{Invoice: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.create: %w", err)
	}
	return &resp.Invoice, nil
}

// Get retrieves an invoice document by ID.
func (s *InvoicesService) Get(ctx context.Context, docType DocumentType, id int64) (*Invoice, error) {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	var resp invoiceResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.get: %w", err)
	}
	return &resp.Invoice, nil
}

// List returns a paginated list of invoice documents.
func (s *InvoicesService) List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Invoice, *PageInfo, error) {
	path := fmt.Sprintf("/%s.json", docType)
	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodGet, path, paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: invoices.list: %w", err)
	}
	return resp.Invoices, &resp.Pagination, nil
}

// Update updates an existing invoice document.
func (s *InvoicesService) Update(ctx context.Context, docType DocumentType, id int64, req *InvoiceUpdateRequest) error {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, invoiceWrapper{Invoice: req}, nil); err != nil {
		return fmt.Errorf("invoicexpress: invoices.update: %w", err)
	}
	return nil
}

// ChangeState transitions a document to a new state.
// Message is required for canceled state.
func (s *InvoicesService) ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Invoice, error) {
	path := fmt.Sprintf("/%s/%d/change-state.json", docType, id)
	body := struct {
		Invoice ChangeStateRequest `json:"invoice"`
	}{Invoice: ChangeStateRequest{State: state, Message: message}}
	var resp invoiceResponse
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.change-state: %w", err)
	}
	return &resp.Invoice, nil
}

// RelatedDocuments returns documents related to the given invoice.
func (s *InvoicesService) RelatedDocuments(ctx context.Context, docType DocumentType, id int64) ([]Invoice, error) {
	path := fmt.Sprintf("/%s/%d/related-documents.json", docType, id)
	var resp invoiceListResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.related-documents: %w", err)
	}
	return resp.Invoices, nil
}

// SendByEmail sends a document by email.
func (s *InvoicesService) SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error {
	path := fmt.Sprintf("/%s/%d/email-document.json", docType, id)
	body := struct {
		Message *EmailRequest `json:"message"`
	}{Message: req}
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, nil); err != nil {
		return fmt.Errorf("invoicexpress: invoices.send-by-email: %w", err)
	}
	return nil
}

// pdfResponse is the JSON response for a PDF request.
type pdfResponse struct {
	Output struct {
		PDFURL string `json:"pdf_url"`
	} `json:"output"`
}

// GeneratePDF starts PDF generation and returns a PDF URL (async, polls until ready).
// It polls with the given interval until the PDF is ready or context is cancelled.
func (s *InvoicesService) GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	path := fmt.Sprintf("/api/pdf/%d.json", id)
	for {
		var resp pdfResponse
		statusCode, err := s.client.doWithStatus(ctx, http.MethodGet, path, nil, nil, &resp)
		if err != nil {
			if statusCode == http.StatusAccepted {
				// Still generating, wait and retry.
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(pollInterval):
					continue
				}
			}
			return "", fmt.Errorf("invoicexpress: invoices.generate-pdf: %w", err)
		}
		if statusCode == http.StatusAccepted {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(pollInterval):
				continue
			}
		}
		return resp.Output.PDFURL, nil
	}
}

// CreatePartialPayment creates a partial payment/receipt for a document.
func (s *InvoicesService) CreatePartialPayment(ctx context.Context, id int64, req *PartialPaymentRequest) (*PartialPayment, error) {
	path := fmt.Sprintf("/documents/%d/partial_payments.json", id)
	body := struct {
		PartialPayment *PartialPaymentRequest `json:"partial_payment"`
	}{PartialPayment: req}
	var resp struct {
		PartialPayment PartialPayment `json:"partial_payment"`
	}
	if err := s.client.do(ctx, http.MethodPost, path, nil, body, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.create-partial-payment: %w", err)
	}
	return &resp.PartialPayment, nil
}

// CancelPartialPayment cancels a partial payment.
func (s *InvoicesService) CancelPartialPayment(ctx context.Context, documentID, paymentID int64) error {
	path := fmt.Sprintf("/documents/%d/partial_payments/%d.json", documentID, paymentID)
	if err := s.client.do(ctx, http.MethodPut, path, nil, nil, nil); err != nil {
		return fmt.Errorf("invoicexpress: invoices.cancel-partial-payment: %w", err)
	}
	return nil
}

// GetQRCode returns the QR code for a document.
func (s *InvoicesService) GetQRCode(ctx context.Context, id int64) (*QRCode, error) {
	path := fmt.Sprintf("/api/qr_codes/%d.json", id)
	var resp struct {
		QRCode QRCode `json:"qr_code"`
	}
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: invoices.get-qr-code: %w", err)
	}
	return &resp.QRCode, nil
}

// ListAll returns all invoice documents across all pages for the given type.
func (s *InvoicesService) ListAll(ctx context.Context, docType DocumentType) ([]Invoice, error) {
	var all []Invoice
	page := 1
	for {
		invoices, pageInfo, err := s.List(ctx, docType, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, invoices...)
		if page >= pageInfo.TotalPages || len(invoices) == 0 {
			break
		}
		page++
	}
	return all, nil
}
