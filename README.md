# go-invoicexpress

[![CI](https://github.com/voluzi/go-invoicexpress/actions/workflows/ci.yml/badge.svg)](https://github.com/voluzi/go-invoicexpress/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/voluzi/go-invoicexpress.svg)](https://pkg.go.dev/github.com/voluzi/go-invoicexpress)

A Go client for the [InvoiceXpress API v2](https://invoicexpress.com/api/documentation) — the Portuguese certified invoicing platform.

- Invoices, invoice-receipts, credit/debit notes, estimates, guides
- Clients, items, sequences, taxes, SAF-T export, accounts
- Exact decimal money (no float rounding), typed errors, client-side validation
- Automatic retries with backoff + `Retry-After`, optional client-side rate limiting
- Zero third-party dependencies

## Install

```bash
go get github.com/voluzi/go-invoicexpress
```

## Quick start

```go
import ix "github.com/voluzi/go-invoicexpress"

client := ix.NewClient("my-account", os.Getenv("INVOICEXPRESS_API_KEY"))
ctx := context.Background()

// Issue a finalized invoice-receipt (fatura-recibo) in one call.
inv, err := client.Invoices.CreateAndFinalize(ctx, ix.DocumentTypeInvoiceReceipt, &ix.InvoiceCreateRequest{
    Date:   ix.NewDate(time.Now()),
    Client: ix.ClientRef{Name: "ACME, Lda", FiscalID: "500000000", Email: "geral@acme.pt"},
    Items: []ix.ItemRef{{
        Name:      "Plano Pro (mensal)",
        UnitPrice: ix.NewDecimal("29.00"),
        Quantity:  ix.NewDecimal("1"),
        Tax:       &ix.TaxRef{Name: "IVA23"},
    }},
})
if err != nil {
    log.Fatal(err)
}

pdfURL, _ := client.Invoices.GeneratePDF(ctx, inv.ID, 2*time.Second)
_ = client.Invoices.SendByEmail(ctx, ix.DocumentTypeInvoiceReceipt, inv.ID, &ix.EmailRequest{
    Client:  ix.EmailClientRef{Email: "geral@acme.pt"},
    Subject: "A sua fatura",
    Body:    "Obrigado!",
})
```

## Configuration

```go
client := ix.NewClient("my-account", "api-key",
    ix.WithTimeout(20*time.Second),
    ix.WithRetry(ix.RetryConfig{MaxAttempts: 5, BaseDelay: 500*time.Millisecond, MaxDelay: 10*time.Second}),
    ix.WithRateLimit(10, 5),      // ≤10 req/s, burst 5 — stays under server limits
    ix.WithUserAgent("acme/1.0"),
    ix.WithHTTPClient(myHTTPClient),
)
```

Retries apply automatically to HTTP 429 (always) and 5xx (idempotent methods),
with exponential backoff + jitter, honoring the `Retry-After` header. Pass
`WithRetry(ix.RetryConfig{MaxAttempts: 1})` to disable.

## Money

All amounts use the `Decimal` type so values are sent and stored exactly — never
as a `float64`. This matters when mirroring an upstream charge (e.g. a Stripe
payment) onto a legal invoice.

```go
ix.NewDecimal("29.99")          // exact, recommended
ix.DecimalFromFloat(29.99, 2)   // when you only have a float
amount.String()                 // "29.99"
amount.Float64()                // for display/aggregation only
```

`Decimal` decodes from either a JSON string (`"29.99"`) or number (`29.99`).
Tax *rates* (percentages) remain `float64`.

## Errors

```go
inv, err := client.Invoices.Get(ctx, ix.DocumentTypeInvoice, id)
switch {
case ix.IsNotFound(err):
    // 404
case ix.IsUnprocessable(err):
    if e, ok := ix.AsAPIError(err); ok {
        log.Printf("validation: %v", e.Errors) // parsed field messages
    }
case ix.IsRateLimited(err):
    // 429 after retries exhausted
}
```

Helpers: `IsNotFound`, `IsUnprocessable`, `IsRateLimited`, `IsUnauthorized`,
`IsConflict`, `AsAPIError`, and `IsValidation` for client-side validation errors.

## Validation

`Invoices.Create` / `CreateAndFinalize` validate required fields before any
network call. You can also call `req.Validate()` yourself.

## Mocking

Each service has an interface (`ix.InvoicesAPI`, `ix.ClientsAPI`, …) so you can
fake the client in your own tests without the network.

## Documents

| Group | Types |
|-------|-------|
| Invoices | `invoices`, `simplified_invoices`, `invoice_receipts`, `credit_notes`, `debit_notes` |
| Estimates | `quotes`, `proformas`, `fees_notes` |
| Guides | `shippings`, `transports`, `devolutions` |

Use the `DocumentType*` constants. A draft document has no fiscal value until
finalized (`ChangeState(..., StateFinalized, "")`, or `CreateAndFinalize`).

## Coverage & limitations

Covered: documents (create/get/list/update/change-state/related/email/PDF),
partial payments, QR codes, clients (incl. find-by-name/code), items, taxes,
sequences, SAF-T export, accounts.

Known limitations (PRs welcome):

- **Estimates and guides decode into the shared document shape** (`Estimate` and
  `Guide` are aliases of the document type). Transport-specific guide fields
  beyond the common set are not yet modeled.
- **Invoices are never deleted** — Portuguese law forbids deleting a finalized
  document. Cancel instead via `ChangeState(..., StateCanceled, reason)`.
- **Monetary amounts use `Decimal`; tax rates/percentages use `float64`.**
- Validation-error parsing from 422 bodies is best-effort across the shapes the
  API has used; the raw body is always available in `APIError.Body`.
- This client does not compute VAT. For cross-border EU VAT (per-country rates,
  reverse charge, OSS) determine the tax upstream (e.g. Stripe Tax) and pass the
  resulting amounts and `TaxExemption` code through.

## License

[MIT](LICENSE)
