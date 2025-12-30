.PHONY: build build-cover test test-unit test-integration test-e2e e2e e2e-cli e2e-tui e2e-tmux e2e-tmux-cover e2e-update coverage coverage-all clean

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)
COVERAGE_THRESHOLD := 80
COVERDIR := $(shell pwd)/.coverdata

# Build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/currier ./cmd/currier

# Build with coverage instrumentation
build-cover:
	@mkdir -p bin
	go build -cover -ldflags "$(LDFLAGS)" -o bin/currier-cover ./cmd/currier
	@echo "Built coverage-instrumented binary: bin/currier-cover"

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	go test -v -race ./internal/...

# Run integration tests only
test-integration:
	go test -v -race ./tests/integration/...

# Run E2E tests
test-e2e: e2e

# Run all E2E tests
e2e:
	go test -v -timeout 5m ./e2e/...

# Run CLI E2E tests only
e2e-cli:
	go test -v -timeout 5m ./e2e/cli/...

# Run TUI E2E tests only
e2e-tui:
	go test -v -timeout 5m ./e2e/tui/...

# Run tmux integration tests (real binary testing)
e2e-tmux:
	go test -v -timeout 10m ./e2e/tmux/...

# Run tmux tests with coverage collection
e2e-tmux-cover: build-cover
	@mkdir -p $(COVERDIR)
	@rm -rf $(COVERDIR)/*
	GOCOVERDIR=$(COVERDIR) CURRIER_BINARY=$(shell pwd)/bin/currier-cover go test -v -timeout 10m ./e2e/tmux/...
	@echo "E2E coverage data written to $(COVERDIR)"

# Update golden files
e2e-update:
	UPDATE_GOLDEN=1 go test -v -timeout 5m ./e2e/...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Check coverage meets threshold
coverage-check:
	@go test -coverprofile=coverage.out ./... > /dev/null
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc) -eq 1 ]; then \
		echo "FAIL: Coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "PASS: Coverage meets threshold"

# Run all tests with combined coverage (unit + e2e)
coverage-all: build-cover
	@echo "=== Running unit tests with coverage ==="
	@mkdir -p $(COVERDIR)/unit
	go test -coverprofile=$(COVERDIR)/unit.out -covermode=atomic ./internal/...
	@echo ""
	@echo "=== Running e2e tmux tests with coverage ==="
	@mkdir -p $(COVERDIR)/e2e
	@rm -rf $(COVERDIR)/e2e/*
	GOCOVERDIR=$(COVERDIR)/e2e CURRIER_BINARY=$(shell pwd)/bin/currier-cover go test -v -timeout 10m ./e2e/tmux/... || true
	@echo ""
	@echo "=== Merging coverage data ==="
	@if [ -d "$(COVERDIR)/e2e" ] && [ "$$(ls -A $(COVERDIR)/e2e 2>/dev/null)" ]; then \
		go tool covdata textfmt -i=$(COVERDIR)/e2e -o=$(COVERDIR)/e2e.out; \
		echo "E2E coverage converted to $(COVERDIR)/e2e.out"; \
	else \
		echo "No e2e coverage data found"; \
		touch $(COVERDIR)/e2e.out; \
	fi
	@echo ""
	@echo "=== Coverage Summary ==="
	@echo "Unit test coverage:"
	@go tool cover -func=$(COVERDIR)/unit.out | grep total || true
	@if [ -s "$(COVERDIR)/e2e.out" ]; then \
		echo "E2E test coverage:"; \
		go tool cover -func=$(COVERDIR)/e2e.out | grep total || true; \
	fi
	@echo ""
	@echo "Coverage files:"
	@echo "  Unit: $(COVERDIR)/unit.out"
	@echo "  E2E:  $(COVERDIR)/e2e.out"
	@echo ""
	@echo "To view coverage: go tool cover -html=$(COVERDIR)/unit.out"

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html .coverdata/

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/currier-darwin-amd64 ./cmd/currier
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/currier-darwin-arm64 ./cmd/currier
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/currier-linux-amd64 ./cmd/currier
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/currier-linux-arm64 ./cmd/currier
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/currier-windows-amd64.exe ./cmd/currier

# Run the application
run: build
	./bin/currier

# Full quality check
check: fmt vet test coverage-check
