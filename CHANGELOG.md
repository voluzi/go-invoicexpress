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

### Documented
- A zero `Date` on an `omitempty` field marshals to JSON `null` (omitempty does
  not apply to structs); InvoiceXpress treats it as an absent optional date.

## [0.0.0] - initial
- Initial implementation: invoices, estimates, guides, clients, items,
  sequences, taxes, SAF-T, accounts; `Date` type; async PDF/SAF-T polling.
