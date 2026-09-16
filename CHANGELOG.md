# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Repository hygiene for open-source: `SECURITY.md` (private vulnerability
  reporting), Dependabot for GitHub Actions + Go modules, and issue/PR templates.
- `ValidPortugueseNIF` and `Taxes.FindByName` — guards against InvoiceXpress's
  silent fallbacks (a bad NIF degrades to "Consumidor Final"; an unknown tax
  name applies the default rate, neither erroring).
- `examples/draft-invoice` smoke-test command (with an opt-in `-finalize` flag).
- Functional options for `NewClient`: `WithBaseURL`, `WithHTTPClient`,
  `WithUserAgent`, `WithTimeout`, `WithRetry`, `WithRateLimit`.
- Automatic retries with exponential backoff + full jitter on HTTP 429 and 5xx
  (idempotent methods), honoring the `Retry-After` header.
- Client-side token-bucket rate limiter (`WithRateLimit`).
- Typed error helpers using `errors.As`: `AsAPIError`, `IsNotFound`,
  `IsUnprocessable`, `IsRateLimited`, `IsUnauthorized`, `IsConflict`.
- Parsing of structured validation messages from 422 bodies into `APIError.Errors`.
- `Decimal` type for exact monetary values (no float rounding), applied to all
  amount fields. Tolerant of JSON string or number on decode.
- Client-side request validation (`Validate()` + `IsValidation`); `Invoices.Create`
  validates before any network call.
- `Invoices.CreateAndFinalize` convenience helper.
- Per-service interfaces (`InvoicesAPI`, `ClientsAPI`, …) for mocking in consumers.
- Default `User-Agent` carrying the library `Version`; bounded response reads.
- Test suite (httptest), CI (build/vet/test/lint matrix), `golangci-lint` config,
  and `Makefile`.

### Fixed
- `IsNotFound` / `IsUnprocessable` now match wrapped errors (previously a type
  assertion that always failed because services wrap with `fmt.Errorf`).
- `Clients.ListInvoices` used `POST`; corrected to `GET`.
- `buildURL` no longer mutates the caller's `url.Values`.
- Removed dead code.

### Fixed (review round)
- **Security:** the `api_key` (a query parameter) could leak into a caller's logs
  via `*url.Error` on transport failures — it's now redacted in returned errors.
- `Decimal.MarshalJSON` now escapes via `json.Marshal` (a raw value with quotes
  or backslashes previously produced invalid JSON).
- `Decimal.UnmarshalJSON` no longer panics on malformed input (e.g. a lone `"`);
  the string branch decodes via `json.Unmarshal`.
- `Validate()` is now called by `Estimates`/`Guides`/`Clients`/`Items` `Create`
  (previously only `Invoices.Create`), matching the documented contract.
- Item validation now rejects a line item missing **either** unit price or
  quantity (was `&&`, only caught both-missing).
- **`CancelPartialPayment` was using the wrong endpoint.** Corrected to
  `PUT /receipts/{id}/change-state.json` with the required
  `{"receipt":{"state":"canceled","message":...}}` body. Signature changed to
  `CancelPartialPayment(ctx, receiptID int64, message string)`.

### Fixed (review round 2)
- `Decimal.UnmarshalJSON` now rejects non-numeric input — booleans, objects,
  arrays, and non-numeric strings (e.g. `"not-a-number"`) error instead of being
  silently stored. Added `ParseDecimal(string) (Decimal, error)` and `Valid()`.
- `CancelPartialPayment` and `ChangeState` now enforce a non-empty message
  client-side when canceling (the API requires a reason).
- `ItemCreateRequest.Validate` now requires `unit_price` (per the API), not just
  name.
- API error bodies are scrubbed of the `api_key` before being placed in
  `APIError.Body`, in case a proxy echoes the request URL.

### Documented
- A zero `Date` on an `omitempty` field marshals to JSON `null` (omitempty does
  not apply to structs); InvoiceXpress treats it as an absent optional date.
- `Update` methods do not validate (pass-through for partial updates); only
  `Create` validates client-side. Corrected the misleading doc comment.

## [0.2.0] - 2026-09-16

Decoding fixes for wire shapes the API actually uses, which the published
examples contradict. A consumer could not read a tax table at all, and document
reads failed silently.

### Changed (breaking)
- `Tax.Value` and `TaxRef.Value` are now `Rate`; `Tax.IsDefault`,
  `Invoice.Archived` and `Sequence.DefaultSequence` are now `Flag`. Both have
  the expected underlying types (`float64`, `bool`), so a boolean field still
  reads as one, but arithmetic and returns need an explicit conversion.
- `Tax.IsDefault` is tagged `default_tax`, the key the API actually sends. The
  previous `is_default` matched nothing, so the account-default flag had never
  once been true.
- `Sequence.SerieNumber` now fills from `serie` (what the API returns) as well
  as `serie_number` (what it accepts), instead of coming back empty.

### Fixed
- **Tax rates arrive as JSON strings** (`"23.0"`) from `/taxes.json` and
  `/items.json`, and as numbers elsewhere. `Rate` accepts both and rejects
  anything that is not a plain decimal, so a nonsense value can never be read
  as 0% and stamped on a document.
- **Document responses are keyed by document type** — `{"invoice_receipt": …}`
  and `{"invoice_receipts": […]}`, not the documented fixed `{"invoice": …}`.
  Decoding the documented key did not error: it left the value zero, so `Get`
  returned an empty document and `ListAll` an empty slice, both with a nil
  error. An idempotency check that scans that list concluded "nothing exists"
  every time and would issue a duplicate legal document.
- A response carrying no document, a null document, an unrelated document type
  or an error body is now an error rather than a zero-valued document.
- `CreateAndFinalize` no longer discards the created document when the
  state change answers without one.
- Booleans written as `1`/`0` (`default_tax`, `default_sequence`) decode.

### Note
Request bodies deliberately keep the documented fixed key (`{"invoice": …}`).
The docs are the only evidence about what the endpoints accept, so they were
not changed on the strength of the response shapes.

## [0.0.0] - initial
- Initial implementation: invoices, estimates, guides, clients, items,
  sequences, taxes, SAF-T, accounts; `Date` type; async PDF/SAF-T polling.
