# Strata Makefile

.PHONY: build run test clean dev seed-admin tidy

# Build the application
build:
	go build -o bin/strata ./cmd/strata

# Run the application
run: build
	./bin/strata

# Run in development mode (with live reload if air is installed)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed, using go run"; \
		go run ./cmd/strata; \
	fi

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Seed admin user (requires EMAIL env var)
seed-admin:
	@if [ -z "$(EMAIL)" ]; then \
		echo "Usage: make seed-admin EMAIL=admin@example.com"; \
		exit 1; \
	fi
	./bin/strata seed-admin --email=$(EMAIL)

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
		go vet ./...; \
	fi

# Generate (if needed)
generate:
	go generate ./...

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/strata ./cmd/strata

# Docker build (if Dockerfile exists)
docker-build:
	docker build -t strata:latest .

# Show help
help:
	@echo "Strata Makefile targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run the application"
	@echo "  dev         - Run in development mode"
	@echo "  test        - Run tests"
	@echo "  test-cover  - Run tests with coverage"
	@echo "  clean       - Clean build artifacts"
	@echo "  tidy        - Tidy dependencies"
	@echo "  seed-admin  - Seed admin user (EMAIL=... required)"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  build-prod  - Build for production"
	@echo "  help        - Show this help"
