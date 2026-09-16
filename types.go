package invoicexpress

import (
	"bytes"
	"encoding/json"
	"fmt"
)

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
	PaymentMechanismTransfer       PaymentMechanism = "TB"
	PaymentMechanismMultiBanco     PaymentMechanism = "MB"
	PaymentMechanismCash           PaymentMechanism = "NU"
	PaymentMechanismDebitCard      PaymentMechanism = "CD"
	PaymentMechanismCreditCard     PaymentMechanism = "CC"
	PaymentMechanismCheck          PaymentMechanism = "CH"
	PaymentMechanismCheckOrVoucher PaymentMechanism = "CO"
	// PaymentMechanismMBWay is retained for compatibility. MW is not listed in
	// the provider's documented payment-mechanism table.
	PaymentMechanismMBWay        PaymentMechanism = "MW"
	PaymentMechanismCompensation PaymentMechanism = "CS"
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
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// Value is a Rate, not a float64: the API writes this field as a JSON
	// string on some endpoints and a number on others.
	Value Rate `json:"value,omitempty"`
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

// ClientRef identifies an existing client by positive ID, then code, then
// name. InvoiceXpress applies that precedence when more than one is present.
type ClientRef struct {
	ID           int64  `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Code         string `json:"code,omitempty"`
	Email        string `json:"email,omitempty"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	Country      string `json:"country,omitempty"`
	FiscalID     string `json:"fiscal_id,omitempty"`
	Website      string `json:"website,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Fax          string `json:"fax,omitempty"`
	Observations string `json:"observations,omitempty"`
	// Language the client's documents are produced in, as a two-letter code:
	// "en" for English, empty to leave the account's own default in place.
	//
	// Absent from the published API reference, but the field is returned on
	// every client read and accepted on write — an account can be seen holding
	// a mix of "en" and null. Only values observed in use should be sent: a
	// guess here changes the language of a legal document.
	Language    string             `json:"language,omitempty"`
	SendOptions *ClientSendOptions `json:"send_options,omitempty"`
}

// ClientSendOptions configures how documents are sent to a client.
//
// Known omitempty limitations: encoding/json's omitempty does NOT omit an empty
// (non-nil) slice, so an empty SendBy still marshals as "send_by":[]. Likewise
// omitempty on a bool omits only false, so an explicit SendRevision:false is
// indistinguishable from an unset value and will not be sent.
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
	ProprietaryUID string          `json:"-"`
}

// InvoiceUpdateRequest holds data for updating an invoice document.
type InvoiceUpdateRequest = InvoiceCreateRequest

// Invoice is the full invoice document as returned by the API.
type Invoice struct {
	ID                     int64         `json:"id"`
	Status                 string        `json:"status"`
	Archived               Flag          `json:"archived"`
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

// UnmarshalJSON accepts both documented numeric sequence identifiers and the
// string form returned by older endpoints while keeping SequenceID source
// compatible as a string.
func (i *Invoice) UnmarshalJSON(data []byte) error {
	type invoiceFields Invoice
	decoded := invoiceFields{}
	aux := struct {
		SequenceID json.RawMessage `json:"sequence_id"`
		*invoiceFields
	}{invoiceFields: &decoded}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.SequenceID) > 0 {
		sequenceID, err := decodeSequenceID(aux.SequenceID)
		if err != nil {
			return err
		}
		decoded.SequenceID = sequenceID
	}
	*i = Invoice(decoded)
	return nil
}

func decodeSequenceID(data []byte) (string, error) {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return "", nil
	}
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return "", fmt.Errorf("decode sequence_id: %w", err)
		}
		return value, nil
	}
	start := 0
	if len(data) > 0 && data[0] == '-' {
		start = 1
	}
	if start == len(data) {
		return "", fmt.Errorf("decode sequence_id: expected string, integer, or null")
	}
	for _, digit := range data[start:] {
		if digit < '0' || digit > '9' {
			return "", fmt.Errorf("decode sequence_id: expected string, integer, or null")
		}
	}
	return string(data), nil
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
	Receipt          Invoice          `json:"-"`
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
	// Language the client's documents are produced in ("en"), or empty for the
	// account's own default. Returned on every client read.
	Language string `json:"language"`
}

// ClientCreateRequest holds data for creating a client.
type ClientCreateRequest struct {
	Name         string `json:"name"`
	Code         string `json:"code,omitempty"`
	Email        string `json:"email,omitempty"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	Country      string `json:"country,omitempty"`
	FiscalID     string `json:"fiscal_id,omitempty"`
	Website      string `json:"website,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Fax          string `json:"fax,omitempty"`
	Observations string `json:"observations,omitempty"`
	// Language the client's documents are produced in: "en", or empty to leave
	// the account's default. See ClientRef.Language.
	Language    string             `json:"language,omitempty"`
	SendOptions *ClientSendOptions `json:"send_options,omitempty"`
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
//
// SerieNumber accepts both response keys observed for the sequence series.
// DefaultSequence is a Flag because the API writes it as 1/0, not true/false.
type Sequence struct {
	ID                                     int64  `json:"id"`
	SerieNumber                            string `json:"serie_number"`
	DefaultSequence                        Flag   `json:"default_sequence"`
	CurrentInvoiceNumber                   int64  `json:"current_invoice_number"`
	CurrentInvoiceSequenceID               int64  `json:"current_invoice_sequence_id"`
	CurrentInvoiceValidationCode           string `json:"current_invoice_validation_code"`
	CurrentInvoiceReceiptNumber            int64  `json:"current_invoice_receipt_number"`
	CurrentInvoiceReceiptSequenceID        int64  `json:"current_invoice_receipt_sequence_id"`
	CurrentInvoiceReceiptValidationCode    string `json:"current_invoice_receipt_validation_code"`
	CurrentSimplifiedInvoiceNumber         int64  `json:"current_simplified_invoice_number"`
	CurrentSimplifiedInvoiceSequenceID     int64  `json:"current_simplified_invoice_sequence_id"`
	CurrentSimplifiedInvoiceValidationCode string `json:"current_simplified_invoice_validation_code"`
	CurrentCreditNoteNumber                int64  `json:"current_credit_note_number"`
	CurrentCreditNoteSequenceID            int64  `json:"current_credit_note_sequence_id"`
	CurrentCreditNoteValidationCode        string `json:"current_credit_note_validation_code"`
	CurrentDebitNoteNumber                 int64  `json:"current_debit_note_number"`
	CurrentDebitNoteSequenceID             int64  `json:"current_debit_note_sequence_id"`
	CurrentDebitNoteValidationCode         string `json:"current_debit_note_validation_code"`
	CurrentReceiptNumber                   int64  `json:"current_receipt_number"`
	CurrentReceiptSequenceID               int64  `json:"current_receipt_sequence_id"`
	CurrentReceiptValidationCode           string `json:"current_receipt_validation_code"`
	CurrentShippingNumber                  int64  `json:"current_shipping_number"`
	CurrentShippingSequenceID              int64  `json:"current_shipping_sequence_id"`
	CurrentShippingValidationCode          string `json:"current_shipping_validation_code"`
	CurrentTransportNumber                 int64  `json:"current_transport_number"`
	CurrentTransportSequenceID             int64  `json:"current_transport_sequence_id"`
	CurrentTransportValidationCode         string `json:"current_transport_validation_code"`
	CurrentDevolutionNumber                int64  `json:"current_devolution_number"`
	CurrentDevolutionSequenceID            int64  `json:"current_devolution_sequence_id"`
	CurrentDevolutionValidationCode        string `json:"current_devolution_validation_code"`
	CurrentProformaNumber                  int64  `json:"current_proforma_number"`
	CurrentProformaSequenceID              int64  `json:"current_proforma_sequence_id"`
	CurrentProformaValidationCode          string `json:"current_proforma_validation_code"`
	CurrentQuoteNumber                     int64  `json:"current_quote_number"`
	CurrentQuoteSequenceID                 int64  `json:"current_quote_sequence_id"`
	CurrentQuoteValidationCode             string `json:"current_quote_validation_code"`
	CurrentFeesNoteNumber                  int64  `json:"current_fees_note_number"`
	CurrentFeesNoteSequenceID              int64  `json:"current_fees_note_sequence_id"`
	CurrentFeesNoteValidationCode          string `json:"current_fees_note_validation_code"`
	CurrentVATMOSSInvoiceNumber            int64  `json:"current_vat_moss_invoice_number"`
	CurrentVATMOSSInvoiceSequenceID        int64  `json:"current_vat_moss_invoice_sequence_id"`
	CurrentVATMOSSInvoiceValidationCode    string `json:"current_vat_moss_invoice_validation_code"`
	CurrentVATMOSSCreditNoteNumber         int64  `json:"current_vat_moss_credit_note_number"`
	CurrentVATMOSSCreditNoteSequenceID     int64  `json:"current_vat_moss_credit_note_sequence_id"`
	CurrentVATMOSSCreditNoteValidationCode string `json:"current_vat_moss_credit_note_validation_code"`
	CurrentVATMOSSReceiptNumber            int64  `json:"current_vat_moss_receipt_number"`
	CurrentVATMOSSReceiptSequenceID        int64  `json:"current_vat_moss_receipt_sequence_id"`
	CurrentVATMOSSReceiptValidationCode    string `json:"current_vat_moss_receipt_validation_code"`
}

// SequenceCreateRequest holds data for creating a sequence.
type SequenceCreateRequest struct {
	SerieNumber     string `json:"-"`
	DefaultSequence bool   `json:"-"`
}

// Tax represents a tax rate in InvoiceXpress.
//
// The wire shapes here are the API's, not the documentation's: /taxes.json
// returns the rate as a string ("23.0"), marks the account default with
// "default_tax": 1 rather than "is_default": true, and sends null for an
// unset region or code.
type Tax struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Value     Rate   `json:"value"`
	Region    string `json:"region"`
	Code      string `json:"code"`
	IsDefault Flag   `json:"default_tax"`
}

// TaxCreateRequest holds data for creating a tax. Rate marshals as a JSON
// number, the shape this endpoint is documented to take.
//
// IsDefault keeps the "is_default" key: tax creation is not documented, and
// the response-side name ("default_tax") is not evidence for the request. It
// is unverified against the live API — this library never creates taxes.
type TaxCreateRequest struct {
	Name      string `json:"name"`
	Value     Rate   `json:"value"`
	Region    string `json:"region,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

// TaxUpdateRequest holds data for updating a tax.
type TaxUpdateRequest = TaxCreateRequest

// SAFTExportResult holds the result of a SAF-T export.
type SAFTExportResult struct {
	URL string `json:"url"`
	// Deprecated: InvoiceXpress returns one archive URL, not a PDF URL.
	PDFURL string `json:"pdf_url"`
	// Deprecated: InvoiceXpress returns one archive URL, not an XML URL.
	XMLURL string `json:"xml_url"`
}

// Account represents an InvoiceXpress account.
type Account struct {
	ID           int64  `json:"id"`
	Organization string `json:"organization_name"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Country      string `json:"country"`
	FiscalID     string `json:"fiscal_id"`
	Subdomain    string `json:"subdomain"`
	State        string `json:"state"`
	ATConfigured bool   `json:"at_configured"`
	Trial        bool   `json:"trial"`
}

func (a *Account) UnmarshalJSON(data []byte) error {
	type accountFields Account
	var v struct {
		accountFields
		OrganizationName   string `json:"organization_name"`
		LegacyOrganization string `json:"organization"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*a = Account(v.accountFields)
	if v.OrganizationName != "" {
		a.Organization = v.OrganizationName
	} else {
		a.Organization = v.LegacyOrganization
	}
	return nil
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
	Date                 Date         `json:"date"`
	DueDate              Date         `json:"due_date"`
	LoadedAt             DateTime     `json:"loaded_at"`
	LicensePlate         string       `json:"license_plate,omitempty"`
	Reference            string       `json:"reference,omitempty"`
	Observations         string       `json:"observations,omitempty"`
	Retention            string       `json:"retention,omitempty"`
	TaxExemption         string       `json:"tax_exemption,omitempty"`
	SequenceID           string       `json:"sequence_id,omitempty"`
	ManualSequenceNumber string       `json:"manual_sequence_number,omitempty"`
	Client               ClientRef    `json:"client"`
	Items                []ItemRef    `json:"items"`
	AddressFrom          *AddressInfo `json:"address_from"`
	AddressTo            *AddressInfo `json:"address_to"`
	TaxExemptionReason   string       `json:"tax_exemption_reason,omitempty"`
	LoadSite             string       `json:"load_site,omitempty"`
	DeliverySite         string       `json:"delivery_site,omitempty"`
	ProprietaryUID       string       `json:"proprietary_uid,omitempty"`
}

// GuideUpdateRequest holds data for a partial guide update. Guides.Update
// omits unset create-only fields from the request body.
type GuideUpdateRequest = GuideCreateRequest
