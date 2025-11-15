.PHONY: build install clean test run help

# Build variables
BINARY_NAME=linear
BUILD_DIR=build
VERSION?=0.1.0
LDFLAGS=-ldflags "-X github.com/linear-cli/linear/cmd.Version=$(VERSION)"

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)
	@echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)"

# Install to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "✓ Cleaned"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run the application
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Formatted"

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "Install golangci-lint for linting"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "✓ Built for multiple platforms"

# Help
help:
	@echo "Linear CLI - Makefile targets:"
	@echo "  make build      - Build the application"
	@echo "  make install    - Install to /usr/local/bin"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run tests"
	@echo "  make run        - Build and run"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Lint code"
	@echo "  make deps       - Download dependencies"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make help       - Show this help"
