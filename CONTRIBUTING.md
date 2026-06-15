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

This project follows [SemVer](https://semver.org). Tag releases as `vX.Y.Z`
and move the `[Unreleased]` section of the changelog under the new version.
