GO      ?= go
BINARY  := bin/nx
PKGS    := ./...
CORE    := ./internal/...
E2E_CONTAINER ?= nx-e2e
E2E_PORT ?= 8081
E2E_PASS ?= nx-e2e-Pass1!

.PHONY: help build test test-unit test-integration e2e-up e2e-down e2e-test lint fmt tidy clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-18s %s\n", $$1, $$2}'

build: ## Build ./bin/nx
	$(GO) build -o $(BINARY) ./cmd/nx

test: test-unit test-integration ## Unit + integration tests with -race (no e2e)
	$(GO) test $(PKGS) -race -cover

test-unit: ## Unit tests for internal packages with -race and coverage
	$(GO) test $(CORE) -race -cover

test-integration: ## Integration tests under test/integration with -race
	$(GO) test ./test/integration/... -race

e2e-up: ## Start and provision a local Nexus container for e2e tests
	bash test/e2e/up.sh

e2e-down: ## Stop and remove the e2e Nexus container
	docker rm -f $(E2E_CONTAINER)

e2e-test: ## Run e2e tests against the container from e2e-up
	NX_E2E_URL=http://localhost:$(E2E_PORT) NX_E2E_PASS=$(E2E_PASS) \
		$(GO) test ./test/e2e/... -tags=e2e -race -v

lint: ## golangci-lint run (zero warnings required)
	golangci-lint run ./...

fmt: ## gofmt -s and goimports
	gofmt -s -w .
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .

tidy: ## go mod tidy
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf bin
