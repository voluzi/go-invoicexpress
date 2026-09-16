.PHONY: help test test-race cover lint fmt vet tidy check release-check

GO ?= go

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

test: ## Run tests
	$(GO) test ./...

test-race: ## Run tests with the race detector
	$(GO) test -race ./...

cover: ## Run tests and print coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

fmt: ## Format the code
	gofmt -w .

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint (must be installed)
	golangci-lint run

tidy: ## Tidy go.mod
	$(GO) mod tidy

check: fmt vet test-race ## Format, vet, and test with race

release-check: ## Validate release metadata (TAG=vX.Y.Z)
	@test -n "$(TAG)" || (echo "TAG is required (for example: TAG=v0.3.0)" >&2; exit 2)
	@$(GO) run ./cmd/release-metadata -tag "$(TAG)"
