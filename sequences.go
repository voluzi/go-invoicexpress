package invoicexpress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SequencesService handles sequence operations.
type SequencesService struct {
	client *Client
}

// sequenceWrapper is used for JSON serialization of sequence requests.
type sequenceWrapper struct {
	Sequence interface{} `json:"sequence"`
}

// sequenceResponse is the JSON response for a single sequence.
type sequenceResponse struct {
	Sequence Sequence `json:"sequence"`
}

// sequenceListResponse is the JSON response for a list of sequences.
type sequenceListResponse struct {
	Sequences  []Sequence `json:"sequences"`
	Pagination PageInfo   `json:"pagination"`
}

// UnmarshalJSON fills SerieNumber from either key the API uses for it. A
// returned sequence names the series "serie"; the create request names the
// same thing "serie_number". Decoding only the documented key left the series
// silently empty on every sequence the API returned.
func (s *Sequence) UnmarshalJSON(data []byte) error {
	type sequenceFields Sequence
	var v struct {
		sequenceFields
		Serie string `json:"serie"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = Sequence(v.sequenceFields)
	if v.Serie != "" {
		s.SerieNumber = v.Serie
	}
	return nil
}

// List returns the first page of sequences. Use ListPage to control pagination
// or ListAll to fetch every page.
func (s *SequencesService) List(ctx context.Context) ([]Sequence, error) {
	sequences, _, err := s.ListPage(ctx, nil)
	return sequences, err
}

// ListPage returns a single page of sequences along with the pagination
// metadata.
func (s *SequencesService) ListPage(ctx context.Context, opts *ListOptions) ([]Sequence, *PageInfo, error) {
	var resp sequenceListResponse
	if err := s.client.do(ctx, http.MethodGet, "/sequences.json", paginationParams(opts), nil, &resp); err != nil {
		return nil, nil, fmt.Errorf("invoicexpress: sequences.list: %w", err)
	}
	return resp.Sequences, &resp.Pagination, nil
}

// ListAll returns all sequences across all pages.
func (s *SequencesService) ListAll(ctx context.Context) ([]Sequence, error) {
	var all []Sequence
	page := 1
	for {
		sequences, pageInfo, err := s.ListPage(ctx, &ListOptions{Page: page, PerPage: 25})
		if err != nil {
			return nil, err
		}
		all = append(all, sequences...)
		if page >= pageInfo.TotalPages || len(sequences) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// Get retrieves a sequence by ID.
func (s *SequencesService) Get(ctx context.Context, id int64) (*Sequence, error) {
	path := fmt.Sprintf("/sequences/%d.json", id)
	var resp sequenceResponse
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: sequences.get: %w", err)
	}
	return &resp.Sequence, nil
}

// Create creates a new sequence. The request is validated client-side before
// any network call, consistent with the other Create methods.
func (s *SequencesService) Create(ctx context.Context, req *SequenceCreateRequest) (*Sequence, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var resp sequenceResponse
	if err := s.client.do(ctx, http.MethodPost, "/sequences.json", nil, sequenceWrapper{Sequence: req}, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: sequences.create: %w", err)
	}
	return &resp.Sequence, nil
}

// SetCurrent sets a sequence as the default.
func (s *SequencesService) SetCurrent(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/sequences/%d/set_current.json", id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, nil, nil); err != nil {
		return fmt.Errorf("invoicexpress: sequences.set-current: %w", err)
	}
	return nil
}
