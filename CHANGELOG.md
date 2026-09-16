# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.1] - 2026-09-16

### Added
- Clients carry the `language` that decides what language InvoiceXpress
  produces their documents in. Absent from the published API reference, but
  returned on every client read and accepted on write.
- `NullableString`, with `String` and `Null` constructors, for a write field
  whose "account default" is JSON null rather than an empty string.

### Fixed
- A `NullableString` whose token is rejected is left exactly as it was found,
  so a refused value cannot be marked as set and written back on the next
  request.

## [0.3.0] - 2026-09-16

### Added
- Existing clients can be referenced by ID, code, or name when creating any
  document family.
- Guide requests support typed loading timestamps, addresses, vehicle and
  loading/delivery-site fields, with complete create validation.
- Sequence creation can opt into the default sequence, existing sequences can
  be registered with the Tax Authority, and registration metadata is decoded.
- `SAFTExportResult.URL`, `ErrNoSAFTDocuments`, full partial-payment receipts,
  account state fields, and structured `APIError.Code` values.
- A tag-triggered GitHub Release workflow and dependency-free release metadata
  checker that fail closed when the tag, version, or changelog drift.

### Changed
- Invoice create sends `proprietary_uid` beside `invoice`, as required by the
  provider contract. Invoice updates omit it.
- Estimate writes use the fixed `quote` envelope and guide writes use the fixed
  `shipping` envelope for every subtype.
- Payment mechanisms now emit the documented legal codes for cash,
  compensation, bank checks, and checks or vouchers.
- Sequence requests serialize `serie` and responses accept both documented
  plural and legacy singular wrappers.

### Fixed
- Partial payments decode the returned `receipt` and reject missing or id-less
  legal documents.
- PDF polling decodes `output.pdfUrl`; SAF-T export decodes top-level `url` or
  `Url`. Completed responses without a usable URL are errors.
- Related documents, client-name lookup, client invoice listing, and account
  lookup use their documented method, route, envelope, and response fields.
- Document `sequence_id` accepts string, integer, or null without losing the
  source-compatible string field.

## [0.2.2] - 2026-09-16

### Fixed
- Reject decimal rates that underflow to zero instead of silently treating an
  unrepresentable rate as a real 0% tax.

## [0.2.1] - 2026-09-16

### Fixed
- Reject null tax rates, id-less documents and document-list entries, and empty
  successful response bodies outside the asynchronous pollers.

## [0.2.0] - 2026-09-16

### Added
- Core InvoiceXpress services for invoices, estimates, guides, clients, items,
  sequences, taxes, SAF-T, and accounts.
- Exact decimal money, tolerant tax-rate and flag types, typed errors,
  client-side validation, bounded retries, rate limiting, service interfaces,
  examples, CI, and contributor documentation.

### Changed
- Tax and document response types reflect observed InvoiceXpress wire shapes,
  including numeric strings, integer flags, nullable fields, and concrete
  document-family envelopes.

### Fixed
- Reject missing, null, ambiguous, family-mismatched, or otherwise unusable
  document responses instead of returning zero-valued legal documents.
- Redact API keys from transport and API errors, enforce cancellation messages,
  and validate required item amounts without rejecting legitimate zero prices.

## [0.0.0] - initial

- Initial implementation.
