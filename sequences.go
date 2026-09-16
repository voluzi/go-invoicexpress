package invoicexpress

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	Sequence Sequence
}

// sequenceListResponse is the JSON response for a list of sequences.
type sequenceListResponse struct {
	Sequences  []Sequence `json:"sequences"`
	Pagination PageInfo   `json:"pagination"`
}

func (r *sequenceListResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	payload, ok := raw["sequences"]
	if !ok {
		return errors.New("invoicexpress: response carries no sequences")
	}
	if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return errors.New("invoicexpress: response carries null sequences")
	}
	if err := json.Unmarshal(payload, &r.Sequences); err != nil {
		return fmt.Errorf("invoicexpress: decode sequences: %w", err)
	}
	for i := range r.Sequences {
		if r.Sequences[i].ID == 0 {
			return fmt.Errorf("invoicexpress: sequence at index %d has no id", i)
		}
	}
	if pagination, ok := raw["pagination"]; ok {
		if err := json.Unmarshal(pagination, &r.Pagination); err != nil {
			return fmt.Errorf("invoicexpress: decode sequence pagination: %w", err)
		}
	}
	return nil
}

func (r SequenceCreateRequest) MarshalJSON() ([]byte, error) {
	defaultSequence := ""
	if r.DefaultSequence {
		defaultSequence = "1"
	}
	return json.Marshal(struct {
		Serie           string `json:"serie"`
		DefaultSequence string `json:"default_sequence,omitempty"`
	}{Serie: r.SerieNumber, DefaultSequence: defaultSequence})
}

func (r *sequenceResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	payload, ok := raw["sequences"]
	if !ok {
		payload, ok = raw["sequence"]
	}
	if !ok {
		return errors.New("invoicexpress: response carries no sequence")
	}
	if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return errors.New("invoicexpress: response carries a null sequence")
	}
	if err := json.Unmarshal(payload, &r.Sequence); err != nil {
		return fmt.Errorf("invoicexpress: decode sequence: %w", err)
	}
	if r.Sequence.ID == 0 {
		return errors.New("invoicexpress: sequence in response has no id")
	}
	return nil
}

// UnmarshalJSON accepts both response names used for the sequence series.
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

// Register registers an existing sequence with the Portuguese Tax Authority.
// It does not retry automatically because registration is a one-shot state
// transition.
func (s *SequencesService) Register(ctx context.Context, id int64) ([]Sequence, error) {
	path := fmt.Sprintf("/sequences/%d/register.json", id)
	var resp sequenceListResponse
	if err := s.client.doWithoutRetry(ctx, http.MethodPut, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("invoicexpress: sequences.register: %w", err)
	}
	return resp.Sequences, nil
}

// SetCurrent sets a sequence as the default.
func (s *SequencesService) SetCurrent(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/sequences/%d/set_current.json", id)
	if err := s.client.do(ctx, http.MethodPut, path, nil, nil, nil); err != nil {
		return fmt.Errorf("invoicexpress: sequences.set-current: %w", err)
	}
	return nil
}
