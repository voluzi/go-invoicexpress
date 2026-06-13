# go-invoicexpress

A Go client library for the [InvoiceXpress API v2](https://invoicexpress.com/api/documentation).

## Installation

```bash
go get github.com/voluzi/go-invoicexpress
```

## Quick Start

```go
import "github.com/voluzi/go-invoicexpress"

client := invoicexpress.NewClient("your-account-name", "your-api-key")
```

## Usage

### Invoices

```go
ctx := context.Background()

// Create an invoice
inv, err := client.Invoices.Create(ctx, invoicexpress.DocumentTypeInvoice, &invoicexpress.InvoiceCreateRequest{
    Date:    invoicexpress.NewDate(time.Now()),
    DueDate: invoicexpress.NewDate(time.Now().AddDate(0, 0, 30)),
    Client: invoicexpress.ClientRef{
        Name: "Acme Corp",
        Code: "ACME",
    },
    Items: []invoicexpress.ItemRef{
        {Name: "Consulting", UnitPrice: 100, Quantity: 8, Tax: &invoicexpress.TaxRef{Name: "IVA23"}},
    },
})

// Finalize
inv, err = client.Invoices.ChangeState(ctx, invoicexpress.DocumentTypeInvoice, inv.ID, invoicexpress.StateFinalized, "")

// Send by email
err = client.Invoices.SendByEmail(ctx, invoicexpress.DocumentTypeInvoice, inv.ID, &invoicexpress.EmailRequest{
    Client:  invoicexpress.EmailClientRef{Email: "billing@acme.com"},
    Subject: "Your invoice",
    Body:    "Please find your invoice attached.",
})

// Get PDF URL (polls until ready)
pdfURL, err := client.Invoices.GeneratePDF(ctx, inv.ID, 2*time.Second)

// List with pagination
invoices, pageInfo, err := client.Invoices.List(ctx, invoicexpress.DocumentTypeInvoice, &invoicexpress.ListOptions{
    Page:    1,
    PerPage: 25,
})

// Get QR code
qr, err := client.Invoices.GetQRCode(ctx, inv.ID)

// Partial payment
payment, err := client.Invoices.CreatePartialPayment(ctx, inv.ID, &invoicexpress.PartialPaymentRequest{
    PaymentMechanism: invoicexpress.PaymentMechanismTransfer,
    Amount:           500.00,
    PaymentDate:      invoicexpress.NewDate(time.Now()),
})
```

### Document Types

**Invoices:**
- `DocumentTypeInvoice` — `invoices`
- `DocumentTypeSimplified` — `simplified_invoices`
- `DocumentTypeInvoiceReceipt` — `invoice_receipts`
- `DocumentTypeCreditNote` — `credit_notes`
- `DocumentTypeDebitNote` — `debit_notes`

**Estimates:**
- `DocumentTypeQuote` — `quotes`
- `DocumentTypeProforma` — `proformas`
- `DocumentTypeFeesNote` — `fees_notes`

**Guides:**
- `DocumentTypeShipping` — `shippings`
- `DocumentTypeTransport` — `transports`
- `DocumentTypeDevolution` — `devolutions`

### Clients (Customers)

```go
// Create
customer, err := client.Clients.Create(ctx, &invoicexpress.ClientCreateRequest{
    Name:    "Acme Corp",
    Code:    "ACME",
    Email:   "billing@acme.com",
    Country: "Portugal",
})

// Find by name or code
customers, err := client.Clients.FindByName(ctx, "Acme")
customer, err := client.Clients.FindByCode(ctx, "ACME")

// List all (auto-paginated)
all, err := client.Clients.ListAll(ctx)
```

### Items

```go
item, err := client.Items.Create(ctx, &invoicexpress.ItemCreateRequest{
    Name:      "Consulting",
    UnitPrice: 100.00,
    Unit:      "hour",
    Tax:       &invoicexpress.TaxRef{Name: "IVA23"},
})
```

### Taxes & Sequences

```go
taxes, err := client.Taxes.List(ctx)
sequences, err := client.Sequences.List(ctx)
err = client.Sequences.SetCurrent(ctx, sequenceID)
```

### SAF-T Export

```go
// Polls until the export is ready
result, err := client.SAFT.Export(ctx, 3, 2024, 3*time.Second) // March 2024
fmt.Println(result.XMLURL)
```

### Estimates

```go
quote, err := client.Estimates.Create(ctx, invoicexpress.DocumentTypeQuote, &invoicexpress.InvoiceCreateRequest{...})
```

### Guides

```go
shipping, err := client.Guides.Create(ctx, invoicexpress.DocumentTypeShipping, &invoicexpress.GuideCreateRequest{
    Date:        invoicexpress.NewDate(time.Now()),
    Client:      invoicexpress.ClientRef{Name: "Acme Corp", Code: "ACME"},
    Items:       []invoicexpress.ItemRef{{Name: "Goods", UnitPrice: 100, Quantity: 1}},
    AddressFrom: &invoicexpress.AddressInfo{City: "Lisbon", Country: "Portugal"},
    AddressTo:   &invoicexpress.AddressInfo{City: "Porto", Country: "Portugal"},
})
```

## Error Handling

```go
inv, err := client.Invoices.Get(ctx, invoicexpress.DocumentTypeInvoice, 12345)
if err != nil {
    if invoicexpress.IsNotFound(err) {
        // 404 — document doesn't exist
    }
    if invoicexpress.IsUnprocessable(err) {
        // 422 — validation error
    }
    // Other errors
    var apiErr *invoicexpress.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Body)
    }
}
```

## Dates

Dates in the InvoiceXpress API are in `dd/mm/yyyy` format. The library handles this transparently via the `Date` type:

```go
// Create a Date from time.Time
d := invoicexpress.NewDate(time.Now())

// Dates unmarshal automatically from API responses
fmt.Println(inv.Date.Time) // time.Time
```

## License

MIT
