package invoicexpress

import (
	"context"
	"time"
)

// This file defines an interface per service so consumers can mock the client
// in their own tests (e.g. a Stripe webhook that issues invoices) without
// hitting the network. The concrete *Service types satisfy them — the
// compile-time assertions at the bottom keep the interfaces in sync.

// InvoicesAPI is the behaviour of InvoicesService.
type InvoicesAPI interface {
	Create(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Invoice, error)
	CreateAndFinalize(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Invoice, error)
	Get(ctx context.Context, docType DocumentType, id int64) (*Invoice, error)
	List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Invoice, *PageInfo, error)
	ListAll(ctx context.Context, docType DocumentType) ([]Invoice, error)
	Update(ctx context.Context, docType DocumentType, id int64, req *InvoiceUpdateRequest) error
	ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Invoice, error)
	RelatedDocuments(ctx context.Context, docType DocumentType, id int64) ([]Invoice, error)
	SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error
	GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error)
	CreatePartialPayment(ctx context.Context, id int64, req *PartialPaymentRequest) (*PartialPayment, error)
	CancelPartialPayment(ctx context.Context, receiptID int64, message string) error
	GetQRCode(ctx context.Context, id int64) (*QRCode, error)
}

// EstimatesAPI is the behaviour of EstimatesService.
type EstimatesAPI interface {
	Create(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Estimate, error)
	CreateAndFinalize(ctx context.Context, docType DocumentType, req *InvoiceCreateRequest) (*Estimate, error)
	Get(ctx context.Context, docType DocumentType, id int64) (*Estimate, error)
	List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Estimate, *PageInfo, error)
	ListAll(ctx context.Context, docType DocumentType) ([]Estimate, error)
	Update(ctx context.Context, docType DocumentType, id int64, req *InvoiceUpdateRequest) error
	ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Estimate, error)
	SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error
	GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error)
}

// GuidesAPI is the behaviour of GuidesService.
type GuidesAPI interface {
	Create(ctx context.Context, docType DocumentType, req *GuideCreateRequest) (*Guide, error)
	CreateAndFinalize(ctx context.Context, docType DocumentType, req *GuideCreateRequest) (*Guide, error)
	Get(ctx context.Context, docType DocumentType, id int64) (*Guide, error)
	List(ctx context.Context, docType DocumentType, opts *ListOptions) ([]Guide, *PageInfo, error)
	ListAll(ctx context.Context, docType DocumentType) ([]Guide, error)
	Update(ctx context.Context, docType DocumentType, id int64, req *GuideUpdateRequest) error
	ChangeState(ctx context.Context, docType DocumentType, id int64, state DocumentState, message string) (*Guide, error)
	SendByEmail(ctx context.Context, docType DocumentType, id int64, req *EmailRequest) error
	GeneratePDF(ctx context.Context, id int64, pollInterval time.Duration) (string, error)
}

// ClientsAPI is the behaviour of ClientsService.
type ClientsAPI interface {
	List(ctx context.Context, opts *ListOptions) ([]Customer, *PageInfo, error)
	ListAll(ctx context.Context) ([]Customer, error)
	Get(ctx context.Context, id int64) (*Customer, error)
	Create(ctx context.Context, req *ClientCreateRequest) (*Customer, error)
	Update(ctx context.Context, id int64, req *ClientUpdateRequest) error
	FindByName(ctx context.Context, name string) ([]Customer, error)
	FindByCode(ctx context.Context, code string) (*Customer, error)
	ListInvoices(ctx context.Context, clientID int64, opts *ListOptions) ([]Invoice, *PageInfo, error)
}

// ItemsAPI is the behaviour of ItemsService.
type ItemsAPI interface {
	List(ctx context.Context, opts *ListOptions) ([]Item, *PageInfo, error)
	ListAll(ctx context.Context) ([]Item, error)
	Get(ctx context.Context, id int64) (*Item, error)
	Create(ctx context.Context, req *ItemCreateRequest) (*Item, error)
	Update(ctx context.Context, id int64, req *ItemUpdateRequest) error
	Delete(ctx context.Context, id int64) error
}

// TaxesAPI is the behaviour of TaxesService.
type TaxesAPI interface {
	List(ctx context.Context) ([]Tax, error)
	FindByName(ctx context.Context, name string) (*Tax, error)
	Get(ctx context.Context, id int64) (*Tax, error)
	Create(ctx context.Context, req *TaxCreateRequest) (*Tax, error)
	Update(ctx context.Context, id int64, req *TaxUpdateRequest) error
	Delete(ctx context.Context, id int64) error
}

// SequencesAPI is the behaviour of SequencesService.
type SequencesAPI interface {
	List(ctx context.Context) ([]Sequence, error)
	Get(ctx context.Context, id int64) (*Sequence, error)
	Create(ctx context.Context, req *SequenceCreateRequest) (*Sequence, error)
	SetCurrent(ctx context.Context, id int64) error
}

// SAFTAPI is the behaviour of SAFTService.
type SAFTAPI interface {
	Export(ctx context.Context, month, year int, pollInterval time.Duration) (*SAFTExportResult, error)
}

// AccountsAPI is the behaviour of AccountsService.
type AccountsAPI interface {
	List(ctx context.Context) ([]Account, error)
	Get(ctx context.Context, id int64) (*Account, error)
}

// Compile-time guarantees that the concrete services satisfy their interfaces.
var (
	_ InvoicesAPI  = (*InvoicesService)(nil)
	_ EstimatesAPI = (*EstimatesService)(nil)
	_ GuidesAPI    = (*GuidesService)(nil)
	_ ClientsAPI   = (*ClientsService)(nil)
	_ ItemsAPI     = (*ItemsService)(nil)
	_ TaxesAPI     = (*TaxesService)(nil)
	_ SequencesAPI = (*SequencesService)(nil)
	_ SAFTAPI      = (*SAFTService)(nil)
	_ AccountsAPI  = (*AccountsService)(nil)
)
