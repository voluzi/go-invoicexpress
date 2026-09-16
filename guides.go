package invoicexpress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GuidesService handles guide document operations (shippings, transports, devolutions).
type GuidesService struct {
	client *Client
}

// shippingWrapper is the fixed request envelope for every guide subtype.
// Responses remain keyed by their concrete document type.
type shippingWrapper struct {
	Shipping interface{} `json:"shipping"`
}

type guideUpdatePayload struct {
	request *GuideUpdateRequest
}

func (p guideUpdatePayload) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(p.request)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for name, value := range fields {
		if string(value) == "null" {
			delete(fields, name)
		}
	}
	if client, ok := fields["client"]; ok && string(client) == "{}" {
		delete(fields, "client")
	}
	return json.Marshal(fields)
}

// Create creates a new guide document. The request is validated client-side
// before any network call.
func (s *GuidesService) Create(ctx context.Context, docType DocumentType, req *GuideCreateRequest) (*Guide, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/%s.json", docType)
	resp := documentEnvelope{docType: docType}
	if err := s.client.do(ctx, http.MethodPost, path, nil, shippingWrapper{Shipping: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.create: %w", err)
	}
	return &resp.doc, nil
}

// CreateAndFinalize creates a guide document and immediately transitions it to
// the finalized state, mirroring InvoicesService.CreateAndFinalize.
func (s *GuidesService) CreateAndFinalize(ctx context.Context, docType DocumentType, req *GuideCreateRequest) (*Guide, error) {
	guide, err := s.Create(ctx, docType, req)
	if err != nil {
		return nil, err
	}
	finalized, err := s.ChangeState(ctx, docType, guide.ID, StateFinalized, "")
	if err != nil {
		return guide, fmt.Errorf("invoicexpress: guides.create-and-finalize: created id=%d but finalize failed: %w", guide.ID, err)
	}
	if finalized == nil || finalized.ID == 0 {
		return guide, nil
	}
	return finalized, nil
}

// Get retrieves a guide document by ID.
func (s *GuidesService) Get(ctx context.Context, docType DocumentType, id int64) (*Guide, error) {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	resp := documentEnvelope{docType: docType}
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.get: %w", err)
	}
	return &resp.doc, nil
}

// List returns a paginated list of guide documents.
func (s *GuidesService) List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Guide, *PageInfo, error) {
	path := fmt.Sprintf("/%s.json", docType)
	resp := documentListEnvelope{docType: docType}
	if err := s.client.do(ctx, http.MethodGet, path, paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: guides.list: %w", err)
	}
	return resp.docs, &resp.page, nil
}

// Update updates an existing guide document.
func (s *GuidesService) Update(ctx context.Context, docType DocumentType, id int64, req *GuideUpdateRequest) error {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	body := guideUpdatePayload{request: req}
	if err := s.client.do(ctx, http.MethodPut, path, nil, shippingWrapper{Shipping: body}, nil); err != nil {
		return fmt.Errorf("invoicexpress: guides.update: %w", err)
	}
	return nil
}

// ChangeState transitions a guide document to a new state. Message is required
// for the canceled state (enforced client-side).
func (s *GuidesService) ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Guide, error) {
	if err := requireCancelMessage(state, message); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/%s/%d/change-state.json", docType, id)
	body := struct {
		Shipping ChangeStateRequest `json:"shipping"`
	}{Shipping: ChangeStateRequest{State: state, Message: message}}
	resp := documentEnvelope{docType: docType}
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.change-state: %w", err)
	}
	return &resp.doc, nil
}

// SendByEmail sends a guide document by email.
func (s *GuidesService) SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error {
	path := fmt.Sprintf("/%s/%d/email-document.json", docType, id)
	body := struct {
		Message *EmailRequest `json:"message"`
	}{Message: req}
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, nil); err != nil {
		return fmt.Errorf("invoicexpress: guides.send-by-email: %w", err)
	}
	return nil
}

// ListAll returns all guide documents across all pages for the given type.
func (s *GuidesService) ListAll(ctx context.Context, docType DocumentType) ([]Guide, error) {
	var all []Guide
	page := 1
	for {
		guides, pageInfo, err := s.List(ctx, docType, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, guides...)
		if page >= pageInfo.TotalPages || len(guides) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// GeneratePDF starts PDF generation and polls until the PDF is ready.
func (s *GuidesService) GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error) {
	url, err := s.client.pollPDF(ctx, id, pollInterval)
	if err != nil {
		return "", fmt.Errorf("invoicexpress: guides.generate-pdf: %w", err)
	}
	return url, nil
}
