package invoicexpress

// DocumentType represents the type of invoice/estimate/guide document.
type DocumentType string

const (
	// Invoice document types.
	DocumentTypeInvoice        DocumentType = "invoices"
	DocumentTypeSimplified     DocumentType = "simplified_invoices"
	DocumentTypeInvoiceReceipt DocumentType = "invoice_receipts"
	DocumentTypeCreditNote     DocumentType = "credit_notes"
	DocumentTypeDebitNote      DocumentType = "debit_notes"

	// Estimate document types.
	DocumentTypeQuote    DocumentType = "quotes"
	DocumentTypeProforma DocumentType = "proformas"
	DocumentTypeFeesNote DocumentType = "fees_notes"

	// Guide document types.
	DocumentTypeShipping   DocumentType = "shippings"
	DocumentTypeTransport  DocumentType = "transports"
	DocumentTypeDevolution DocumentType = "devolutions"
)

// DocumentState represents the state of a document.
type DocumentState string

const (
	StateFinalized  DocumentState = "finalized"
	StateDeleted    DocumentState = "deleted"
	StateCanceled   DocumentState = "canceled"
	StateSettled    DocumentState = "settled"
	StateUnsettled  DocumentState = "unsettled"
	StateSecondCopy DocumentState = "second_copy"
)

// PaymentMechanism represents the payment method.
type PaymentMechanism string

const (
	PaymentMechanismTransfer     PaymentMechanism = "TB"
	PaymentMechanismMultiBanco   PaymentMechanism = "MB"
	PaymentMechanismCash         PaymentMechanism = "DIN"
	PaymentMechanismDebitCard    PaymentMechanism = "CD"
	PaymentMechanismCreditCard   PaymentMechanism = "CC"
	PaymentMechanismCheck        PaymentMechanism = "CH"
	PaymentMechanismMBWay        PaymentMechanism = "MW"
	PaymentMechanismCompensation PaymentMechanism = "CO"
	PaymentMechanismOther        PaymentMechanism = "OU"
)

// ListOptions holds pagination parameters for list endpoints.
type ListOptions struct {
	Page    int
	PerPage int
}

// PageInfo holds pagination metadata returned in list responses.
type PageInfo struct {
	CurrentPage  int `json:"current_page"`
	TotalPages   int `json:"total_pages"`
	TotalEntries int `json:"total_entries"`
	PerPage      int `json:"per_page"`
}

// TaxRef is a reference to a tax by name.
type TaxRef struct {
	ID    int64   `json:"id,omitempty"`
	Name  string  `json:"name,omitempty"`
	Value float64 `json:"value,omitempty"`
}

// GlobalDiscount represents a discount applied to the whole document.
type GlobalDiscount struct {
	ValueType string  `json:"value_type"` // "percentage" or "amount"
	Value     Decimal `json:"value"`
}

// MBReference represents a Multibanco payment reference.
type MBReference struct {
	Entity    string  `json:"entity"`
	Value     Decimal `json:"value"`
	Reference string  `json:"reference"`
}

// ClientRef is used when creating/updating documents to reference a client.
type ClientRef struct {
	Name         string             `json:"name"`
	Code         string             `json:"code,omitempty"`
	Email        string             `json:"email,omitempty"`
	Address      string             `json:"address,omitempty"`
	City         string             `json:"city,omitempty"`
	PostalCode   string             `json:"postal_code,omitempty"`
	Country      string             `json:"country,omitempty"`
	FiscalID     string             `json:"fiscal_id,omitempty"`
	Website      string             `json:"website,omitempty"`
	Phone        string             `json:"phone,omitempty"`
	Fax          string             `json:"fax,omitempty"`
	Observations string             `json:"observations,omitempty"`
	SendOptions  *ClientSendOptions `json:"send_options,omitempty"`
}

// ClientSendOptions configures how documents are sent to a client.
type ClientSendOptions struct {
	SendBy       []string `json:"send_by,omitempty"`
	SendRevision bool     `json:"send_revision,omitempty"`
}

// ItemRef is used when creating/updating documents to reference an item.
// Monetary fields are Decimal so amounts are sent exactly (e.g. mirroring a
// Stripe charge) without float rounding.
type ItemRef struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	UnitPrice   Decimal  `json:"unit_price"`
	Quantity    Decimal  `json:"quantity"`
	Unit        string   `json:"unit,omitempty"`
	Discount    *Decimal `json:"discount,omitempty"`
	Tax         *TaxRef  `json:"tax,omitempty"`
}

// InvoiceCreateRequest holds data for creating an invoice document.
type InvoiceCreateRequest struct {
	Date           Date            `json:"date"`
	DueDate        Date            `json:"due_date,omitempty"`
	Reference      string          `json:"reference,omitempty"`
	Observations   string          `json:"observations,omitempty"`
	Retention      string          `json:"retention,omitempty"`
	TaxExemption   string          `json:"tax_exemption,omitempty"`
	SequenceID     string          `json:"sequence_id,omitempty"`
	CurrencyCode   string          `json:"currency_code,omitempty"`
	Rate           string          `json:"rate,omitempty"`
	Client         ClientRef       `json:"client"`
	Items          []ItemRef       `json:"items"`
	MBReference    string          `json:"mb_reference,omitempty"`
	OwnerInvoiceID int64           `json:"owner_invoice_id,omitempty"`
	GlobalDiscount *GlobalDiscount `json:"global_discount,omitempty"`
	ProprietaryUID string          `json:"proprietary_uid,omitempty"`
}

// InvoiceUpdateRequest holds data for updating an invoice document.
type InvoiceUpdateRequest = InvoiceCreateRequest

// Invoice is the full invoice document as returned by the API.
type Invoice struct {
	ID                     int64         `json:"id"`
	Status                 string        `json:"status"`
	Archived               bool          `json:"archived"`
	Type                   string        `json:"type"`
	SequenceNumber         string        `json:"sequence_number"`
	InvertedSequenceNumber string        `json:"inverted_sequence_number"`
	ATCUD                  string        `json:"atcud"`
	SequenceID             string        `json:"sequence_id"`
	Date                   Date          `json:"date"`
	DueDate                Date          `json:"due_date"`
	Permalink              string        `json:"permalink"`
	SAFTHash               string        `json:"saft_hash"`
	Sum                    Decimal       `json:"sum"`
	Discount               Decimal       `json:"discount"`
	BeforeTaxes            Decimal       `json:"before_taxes"`
	Taxes                  Decimal       `json:"taxes"`
	Total                  Decimal       `json:"total"`
	Currency               string        `json:"currency"`
	Client                 ClientSummary `json:"client"`
	Items                  []InvoiceItem `json:"items"`
	MBReference            *MBReference  `json:"mb_reference,omitempty"`
	Reference              string        `json:"reference"`
	Observations           string        `json:"observations"`
	TaxExemption           string        `json:"tax_exemption"`
	ProprietaryUID         string        `json:"proprietary_uid"`
}

// Estimate is an estimate document (quote, proforma, fees note). Estimates
// share the document shape with Invoice — estimate-specific fields are a
// subset — so this is an alias for ergonomic, self-documenting return types.
type Estimate = Invoice

// Guide is a transport/shipping/devolution guide document. Guides share the
// core document shape with Invoice (alias). Guide-specific transport fields
// beyond the common set are not yet modeled — see the README limitations.
type Guide = Invoice

// ClientSummary is the client info embedded in invoice responses.
type ClientSummary struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Code    string `json:"code"`
	Email   string `json:"email"`
}

// InvoiceItem is the item info embedded in invoice responses.
type InvoiceItem struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UnitPrice   Decimal `json:"unit_price"`
	Unit        string  `json:"unit"`
	Quantity    Decimal `json:"quantity"`
	Tax         TaxRef  `json:"tax"`
	Discount    Decimal `json:"discount"`
	Subtotal    Decimal `json:"subtotal"`
	TaxAmount   Decimal `json:"tax_amount"`
	Total       Decimal `json:"total"`
}

// ChangeStateRequest holds the data for a state transition.
type ChangeStateRequest struct {
	State   DocumentState `json:"state"`
	Message string        `json:"message,omitempty"`
}

// EmailClientRef is the client portion of an email request.
type EmailClientRef struct {
	Email string `json:"email"`
	Save  string `json:"save,omitempty"`
}

// EmailRequest holds the data for sending a document by email.
type EmailRequest struct {
	Client  EmailClientRef `json:"client"`
	Subject string         `json:"subject"`
	Body    string         `json:"body"`
	CC      string         `json:"cc,omitempty"`
	BCC     string         `json:"bcc,omitempty"`
	Logo    string         `json:"logo,omitempty"`
}

// PartialPaymentRequest holds data for creating a partial payment.
type PartialPaymentRequest struct {
	PaymentMechanism PaymentMechanism `json:"payment_mechanism"`
	Note             string           `json:"note,omitempty"`
	Serie            string           `json:"serie,omitempty"`
	Amount           Decimal          `json:"amount"`
	PaymentDate      Date             `json:"payment_date"`
}

// PartialPayment is the payment receipt returned by the API.
type PartialPayment struct {
	ID               int64            `json:"id"`
	Amount           Decimal          `json:"amount"`
	PaymentDate      Date             `json:"payment_date"`
	PaymentMechanism PaymentMechanism `json:"payment_mechanism"`
	Note             string           `json:"note"`
	Serie            string           `json:"serie"`
}

// QRCode holds the QR code data for a document.
type QRCode struct {
	URL  string `json:"url"`
	Data string `json:"data"`
}

// Customer represents a customer/client in InvoiceXpress.
type Customer struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	City         string `json:"city"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
	FiscalID     string `json:"fiscal_id"`
	Website      string `json:"website"`
	Phone        string `json:"phone"`
	Fax          string `json:"fax"`
	Observations string `json:"observations"`
}

// ClientCreateRequest holds data for creating a client.
type ClientCreateRequest struct {
	Name         string             `json:"name"`
	Code         string             `json:"code,omitempty"`
	Email        string             `json:"email,omitempty"`
	Address      string             `json:"address,omitempty"`
	City         string             `json:"city,omitempty"`
	PostalCode   string             `json:"postal_code,omitempty"`
	Country      string             `json:"country,omitempty"`
	FiscalID     string             `json:"fiscal_id,omitempty"`
	Website      string             `json:"website,omitempty"`
	Phone        string             `json:"phone,omitempty"`
	Fax          string             `json:"fax,omitempty"`
	Observations string             `json:"observations,omitempty"`
	SendOptions  *ClientSendOptions `json:"send_options,omitempty"`
}

// ClientUpdateRequest holds data for updating a client.
type ClientUpdateRequest = ClientCreateRequest

// Item represents a product/service item in InvoiceXpress.
type Item struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UnitPrice   Decimal `json:"unit_price"`
	Unit        string  `json:"unit"`
	Discount    Decimal `json:"discount"`
	Tax         TaxRef  `json:"tax"`
}

// ItemCreateRequest holds data for creating an item.
type ItemCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	UnitPrice   Decimal  `json:"unit_price"`
	Unit        string   `json:"unit,omitempty"`
	Discount    *Decimal `json:"discount,omitempty"`
	Tax         *TaxRef  `json:"tax,omitempty"`
}

// ItemUpdateRequest holds data for updating an item.
type ItemUpdateRequest = ItemCreateRequest

// Sequence represents a document numbering sequence.
type Sequence struct {
	ID              int64  `json:"id"`
	SerieNumber     string `json:"serie_number"`
	DefaultSequence bool   `json:"default_sequence"`
}

// SequenceCreateRequest holds data for creating a sequence.
type SequenceCreateRequest struct {
	SerieNumber string `json:"serie_number"`
}

// Tax represents a tax rate in InvoiceXpress.
type Tax struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Region    string  `json:"region"`
	IsDefault bool    `json:"is_default"`
}

// TaxCreateRequest holds data for creating a tax.
type TaxCreateRequest struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Region    string  `json:"region,omitempty"`
	IsDefault bool    `json:"is_default,omitempty"`
}

// TaxUpdateRequest holds data for updating a tax.
type TaxUpdateRequest = TaxCreateRequest

// SAFTExportResult holds the result of a SAF-T export.
type SAFTExportResult struct {
	PDFURL string `json:"pdf_url"`
	XMLURL string `json:"xml_url"`
}

// Account represents an InvoiceXpress account.
type Account struct {
	ID           int64  `json:"id"`
	Organization string `json:"organization"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Country      string `json:"country"`
	FiscalID     string `json:"fiscal_id"`
	Subdomain    string `json:"subdomain"`
}

// AddressInfo holds address details used in guides.
type AddressInfo struct {
	Detail     string `json:"detail,omitempty"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// GuideCreateRequest holds data for creating a guide document.
type GuideCreateRequest struct {
	Date           Date         `json:"date"`
	DueDate        Date         `json:"due_date,omitempty"`
	Reference      string       `json:"reference,omitempty"`
	Observations   string       `json:"observations,omitempty"`
	Retention      string       `json:"retention,omitempty"`
	TaxExemption   string       `json:"tax_exemption,omitempty"`
	SequenceID     string       `json:"sequence_id,omitempty"`
	Client         ClientRef    `json:"client"`
	Items          []ItemRef    `json:"items"`
	AddressFrom    *AddressInfo `json:"address_from,omitempty"`
	AddressTo      *AddressInfo `json:"address_to,omitempty"`
	ProprietaryUID string       `json:"proprietary_uid,omitempty"`
}

// GuideUpdateRequest holds data for updating a guide document.
type GuideUpdateRequest = GuideCreateRequest
