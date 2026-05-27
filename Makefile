# nbcli — modern Netbox CLI + TUI
#
# Common targets:
#   make build    - compile bin/nbcli with version metadata
#   make run      - go run ./cmd/nbcli (passes ARGS)
#   make test     - go test ./...
#   make lint     - golangci-lint run
#   make vuln     - govulncheck ./...
#   make tidy     - go mod tidy
#   make ci       - lint + test + vuln, what CI runs
#   make release  - goreleaser snapshot (no publish)

BINARY        := nbcli
PKG           := github.com/ravinald/nbcli
VERSION_PKG   := $(PKG)/internal/version
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE          ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X $(VERSION_PKG).Version=$(VERSION) \
  -X $(VERSION_PKG).Commit=$(COMMIT) \
  -X $(VERSION_PKG).Date=$(DATE)

GO            ?= go
GOFLAGS       ?=
GOBIN         := $(shell $(GO) env GOPATH)/bin

.PHONY: all build run install test cover lint vuln tidy fmt clean ci release help

all: build

build: ## Build the binary into bin/$(BINARY) with embedded version metadata.
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)

run: ## go run ./cmd/$(BINARY) ARGS=...
	$(GO) run ./cmd/$(BINARY) $(ARGS)

install: ## go install with ldflags into $GOPATH/bin
	$(GO) install $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' ./cmd/$(BINARY)

test: ## Run unit tests with race detector.
	$(GO) test -race -count=1 ./...

cover: ## Run tests with coverage; opens coverage.html.
	$(GO) test -race -count=1 -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint.
	golangci-lint run ./...

vuln: ## Run govulncheck against the call graph.
	govulncheck ./...

tidy: ## go mod tidy.
	$(GO) mod tidy

fmt: ## gofmt + goimports.
	$(GO) fmt ./...

clean: ## Remove build artifacts.
	rm -rf bin dist coverage.out coverage.html

ci: lint test vuln ## What CI runs. Fails on any sub-target failure.

release: ## Build a local goreleaser snapshot (no publish).
	goreleaser release --snapshot --clean

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
