.PHONY: build install clean test

# Build the protohost binary
build:
	@echo "Building protohost..."
	go build -o protohost cmd/protohost/main.go
	@echo "Build complete: ./protohost"

# Install protohost to ~/go/bin (or $GOPATH/bin)
install:
	@echo "Installing protohost..."
	go install ./cmd/protohost
	@echo "Installed to $(shell go env GOPATH)/bin/protohost"
	@echo ""
	@echo "Make sure $(shell go env GOPATH)/bin is in your PATH"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f protohost
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run go mod tidy
tidy:
	@echo "Tidying go modules..."
	go mod tidy

# Show help
help:
	@echo "Protohost Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build the protohost binary"
	@echo "  install  - Install protohost to your GOPATH/bin"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  tidy     - Run go mod tidy"
	@echo "  help     - Show this help message"
