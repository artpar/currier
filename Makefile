.PHONY: build test test-unit test-integration coverage clean

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)
COVERAGE_THRESHOLD := 80

# Build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/currier ./cmd/currier

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	go test -v -race ./internal/...

# Run integration tests only
test-integration:
	go test -v -race ./tests/integration/...

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

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

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
