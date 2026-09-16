# Contributing

Thanks for your interest in improving go-invoicexpress!

## Development

Requirements: Go 1.24+ (see `go.mod`).

```bash
make check      # gofmt + go vet + go test -race
make cover      # coverage summary
make lint       # golangci-lint (install separately)
```

All code must be `gofmt`-clean, pass `go vet`, and keep tests green with the
race detector. New behaviour needs tests — the suite is httptest-based and
requires no network or credentials.

## Pull requests

- Keep changes focused; one logical change per PR.
- Update `CHANGELOG.md` under `[Unreleased]`.
- Follow the existing error-wrapping convention:
  `fmt.Errorf("invoicexpress: <service>.<method>: %w", err)`.
- Money goes through the `Decimal` type, never `float64`.

## Reporting issues

Please include the InvoiceXpress endpoint involved, the request you made
(redacting your API key), and the response or error you observed.

## Releasing

This project follows [SemVer](https://semver.org). Before creating a tag:

1. Set the exported `Version` constant to a stable `X.Y.Z` value.
2. Add exactly one nonempty `## [X.Y.Z] - YYYY-MM-DD` changelog section.
3. Run the full local checks and metadata validator:

   ```bash
   make check
   make lint
   go build ./...
   make release-check TAG=vX.Y.Z > /tmp/release-notes.md
   ```

4. Review `/tmp/release-notes.md`, then create and push the matching `vX.Y.Z`
   tag only with maintainer approval.

The tag workflow repeats build, vet, and race tests, validates the metadata,
and creates a GitHub Release from that changelog section. It does not create or
move tags and does not publish binaries, containers, or package-manager assets.
