.PHONY: all build test clean lint fmt install

# Variables
BINARY_NAME=aws-sso-util
BINARY_PATH=./cmd/aws-sso-util
BUILD_DIR=./dist
GO=go
GOFLAGS=-v

# Get the current version from git tags or default
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build flags
LDFLAGS=-ldflags "-X main.version=${VERSION}"

all: clean lint test build

build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${BINARY_PATH}

install:
	@echo "Installing ${BINARY_NAME}..."
	${GO} install ${GOFLAGS} ${LDFLAGS} ${BINARY_PATH}

test:
	@echo "Running tests..."
	${GO} test ${GOFLAGS} -race -coverprofile=coverage.out ./...

coverage: test
	@echo "Generating coverage report..."
	${GO} tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt:
	@echo "Formatting code..."
	${GO} fmt ./...

clean:
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -f coverage.out coverage.html

# Cross compilation targets
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 ${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${BINARY_PATH}
	GOOS=linux GOARCH=arm64 ${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${BINARY_PATH}

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p ${BUILD_DIR}
	GOOS=darwin GOARCH=amd64 ${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${BINARY_PATH}
	GOOS=darwin GOARCH=arm64 ${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${BINARY_PATH}

build-windows:
	@echo "Building for Windows..."
	@mkdir -p ${BUILD_DIR}
	GOOS=windows GOARCH=amd64 ${GO} build ${GOFLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BINARY_PATH}

# Development helpers
run: build
	@echo "Running ${BINARY_NAME}..."
	${BUILD_DIR}/${BINARY_NAME}

deps:
	@echo "Downloading dependencies..."
	${GO} mod download
	${GO} mod tidy

update-deps:
	@echo "Updating dependencies..."
	${GO} get -u ./...
	${GO} mod tidy

help:
	@echo "Available targets:"
	@echo "  make build       - Build the binary"
	@echo "  make install     - Install the binary to GOPATH/bin"
	@echo "  make test        - Run tests"
	@echo "  make coverage    - Generate test coverage report"
	@echo "  make lint        - Run linter"
	@echo "  make fmt         - Format code"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make build-all   - Build for all platforms"
	@echo "  make deps        - Download dependencies"
	@echo "  make update-deps - Update dependencies"