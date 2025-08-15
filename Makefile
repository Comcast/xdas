# Makefile for building, testing, and managing the xdas project
# Usage: Run `make <target>` to execute a specific target.
# Default target is `all`, which builds the project.
#
# Security Notes:
# - Ensure you trust the source of this Makefile before executing it.
# - Avoid running `make` commands with elevated privileges (e.g., as root).
# - Verify all dependencies and tools (e.g., Go) are installed from trusted sources.
# - Review custom flags or variables passed via the command line to ensure they do not introduce vulnerabilities.

# Default architecture
GOARCH ?= amd64
# Default operating system
GOOS ?= linux
AppName := xdas
GOHOSTARCH = $(shell go env GOHOSTARCH)  # Host architecture
GOHOSTOS = $(shell go env GOHOSTOS)      # Host operating system

# Default target
all: build  ## Build all components of the project

.PHONY: local
local: GOARCH = $(GOHOSTARCH)
local: GOOS = $(GOHOSTOS)
local: Version = local
local: build  ## Build for the local environment (host OS and architecture)

.PHONY: build
build:  ## Build a version of the xdas application
	@echo "Building the application for GOOS=$(GOOS) GOARCH=$(GOARCH)..."
	GOOS=${GOOS} GOARCH=${GOARCH} go build -v -o bin/xdasgo-${GOOS}-${GOARCH} \
	  -ldflags="-X main.AppName=${AppName} -X main.AppVersion=${Version} -X main.BuildTime=`date -u +%F_%T_%Z`" \
	  ./cmd/xdas

.PHONY: keygen
keygen: ## Generate an encryption key (for secure data handling)
	@echo "Generating encryption key..."
	# Note: Key generation may involve sensitive data. Ensure the output is stored securely.
	go build -v -o bin/keygen ./cmd/keygen

.PHONY: test
test: ## Run all tests (ensure the code passes tests before deployment)
	@echo "Running tests on the host system..."
	GOOS=${GOHOSTOS} GOARCH=${GOHOSTARCH} go test ./...

.PHONY: clean
clean: ## Remove temporary files and binaries
	@echo "Cleaning up temporary files and binaries..."
	# Note: This will remove all build artifacts. Ensure you do not delete important files accidentally.
	go clean ./...

# Additional Notes:
# - The `GOARCH` and `GOOS` variables can be overridden to build for other platforms (e.g., `make build GOOS=windows GOARCH=amd64`).
# - Before running `make`, ensure all dependencies (e.g., Go compiler) are installed and properly configured in your environment.
# - Review the `-ldflags` used during build to ensure no sensitive data (e.g., hardcoded secrets) is embedded in the binary.
# - Use `make clean` to clean up build artifacts before sharing the codebase to avoid exposing sensitive information.
