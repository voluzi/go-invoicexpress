package invoicexpress

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GuidesService handles guide document operations (shippings, transports, devolutions).
type GuidesService struct {
	client *Client
}

// guideWrapper is used for JSON serialization of guide requests.
type guideWrapper struct {
	Guide interface{} `json:"guide"`
}

// guideResponse is the JSON response for a single guide.
type guideResponse struct {
	Guide Guide `json:"guide"`
}

// guideListResponse is the JSON response for a list of guides.
type guideListResponse struct {
	Guides     []Guide  `json:"guides"`
	Pagination PageInfo `json:"pagination"`
}

// Create creates a new guide document. The request is validated client-side
// before any network call.
func (s *GuidesService) Create(ctx context.Context, docType DocumentType, req *GuideCreateRequest) (*Guide, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/%s.json", docType)
	var resp guideResponse
	if err := s.client.do(ctx, http.MethodPost, path, nil, guideWrapper{Guide: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.create: %w", err)
	}
	return &resp.Guide, nil
}

// Get retrieves a guide document by ID.
func (s *GuidesService) Get(ctx context.Context, docType DocumentType, id int64) (*Guide, error) {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	var resp guideResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.get: %w", err)
	}
	return &resp.Guide, nil
}

// List returns a paginated list of guide documents.
func (s *GuidesService) List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Guide, *PageInfo, error) {
	path := fmt.Sprintf("/%s.json", docType)
	var resp guideListResponse
	if err := s.client.do(ctx, http.MethodGet, path, paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: guides.list: %w", err)
	}
	return resp.Guides, &resp.Pagination, nil
}

// Update updates an existing guide document.
func (s *GuidesService) Update(ctx context.Context, docType DocumentType, id int64, req *GuideUpdateRequest) error {
	path := fmt.Sprintf("/%s/%d.json", docType, id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, guideWrapper{Guide: req}, nil); err != nil {
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
		Guide ChangeStateRequest `json:"guide"`
	}{Guide: ChangeStateRequest{State: state, Message: message}}
	var resp guideResponse
	if err := s.client.do(ctx, http.MethodPut, path, nil, body, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: guides.change-state: %w", err)
	}
	return &resp.Guide, nil
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

// GeneratePDF starts PDF generation and polls until the PDF is ready.
func (s *GuidesService) GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	path := fmt.Sprintf("/api/pdf/%d.json", id)
	for {
		var resp pdfResponse
		statusCode, err := s.client.doWithStatus(ctx, http.MethodGet, path, nil, nil, &resp)
		if err != nil {
			if statusCode == http.StatusAccepted {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(pollInterval):
					continue
				}
			}
			return "", fmt.Errorf("invoicexpress: guides.generate-pdf: %w", err)
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
