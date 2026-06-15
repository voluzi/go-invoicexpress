# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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

## [0.0.0] - initial
- Initial implementation: invoices, estimates, guides, clients, items,
  sequences, taxes, SAF-T, accounts; `Date` type; async PDF/SAF-T polling.
