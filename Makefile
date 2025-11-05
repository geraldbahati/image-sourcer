.PHONY: help build test clean dev install run-analyze run-enrich run-import pipeline \
        validate-csv generate-csv upload-images stats reset test-payload check-deps \
        build-prod build-dev lint format

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=uploader
BUILD_DIR=./bin
CMD_DIR=./cmd/uploader
INTERNAL_DIR=./internal
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

# File paths (customize these)
EXCEL_FILE?=products.xlsx
ANALYZED_FILE?=analyzed_products.json
ENRICHED_FILE?=enriched_products.json
CSV_FILE?=products.csv
IMAGE_DIR?=./images
CONFIG_FILE?=config.yaml

# Build configuration
WORKERS?=20
DRY_RUN?=false

##@ Help

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Build the application (default)
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-dev: ## Build with race detection (development)
	@echo "🔨 Building $(BINARY_NAME) with race detection..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -race $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Development build complete"

build-prod: ## Build optimized production binary
	@echo "🔨 Building optimized production binary..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Production build complete"

install: build ## Build and install to $GOPATH/bin
	@echo "📦 Installing $(BINARY_NAME)..."
	@$(GO) install $(CMD_DIR)
	@echo "✅ Installed to $GOPATH/bin/$(BINARY_NAME)"

##@ Testing

test: ## Run all tests
	@echo "🧪 Running tests..."
	@$(GO) test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	@$(GO) test -race -v ./...

test-payload: build ## Test Convex payload generation
	@echo "🧪 Testing Convex payload generation..."
	@$(GO) run test_payload.go

##@ Code Quality

lint: ## Run linter (requires golangci-lint)
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run ./...

format: ## Format code with gofmt
	@echo "📝 Formatting code..."
	@gofmt -s -w .
	@echo "✅ Code formatted"

check-deps: ## Check for missing dependencies
	@echo "🔍 Checking dependencies..."
	@$(GO) mod verify
	@$(GO) mod tidy
	@echo "✅ Dependencies OK"

##@ CSV Generation

generate-csv: build ## Generate CSV from images directory
	@echo "📝 Generating CSV from $(IMAGE_DIR)..."
	@$(BUILD_DIR)/$(BINARY_NAME) generate --dir $(IMAGE_DIR) --output $(CSV_FILE)
	@echo "✅ CSV generated: $(CSV_FILE)"

generate-csv-interactive: build ## Generate CSV with interactive mode
	@echo "📝 Generating CSV with interactive mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) generate --dir $(IMAGE_DIR) --output $(CSV_FILE) --interactive
	@echo "✅ CSV generated: $(CSV_FILE)"

validate-csv: build ## Validate CSV file
	@echo "🔍 Validating CSV: $(CSV_FILE)..."
	@$(BUILD_DIR)/$(BINARY_NAME) validate --csv $(CSV_FILE)
	@echo "✅ CSV validation complete"

##@ Image Upload (Cloudinary)

upload-images: build validate-csv ## Upload images to Cloudinary
	@echo "☁️  Uploading images from $(CSV_FILE)..."
	@$(BUILD_DIR)/$(BINARY_NAME) upload --csv $(CSV_FILE) --workers $(WORKERS)
	@echo "✅ Upload complete"

upload-images-dry-run: build ## Dry run upload (validate only)
	@echo "🔍 Dry run upload validation..."
	@$(BUILD_DIR)/$(BINARY_NAME) upload --csv $(CSV_FILE) --dry-run
	@echo "✅ Validation complete"

upload-images-resume: build ## Resume interrupted upload
	@echo "♻️  Resuming upload..."
	@$(BUILD_DIR)/$(BINARY_NAME) upload --csv $(CSV_FILE) --resume
	@echo "✅ Upload resumed"

##@ Product Pipeline (Excel → Convex)

run-analyze: build ## Step 1: Analyze Excel file
	@echo "📊 Analyzing Excel file: $(EXCEL_FILE)..."
	@$(BUILD_DIR)/$(BINARY_NAME) analyze --excel $(EXCEL_FILE) --output $(ANALYZED_FILE)
	@echo "✅ Analysis complete: $(ANALYZED_FILE)"

run-enrich: build ## Step 2: Enrich products with AI
	@echo "🤖 Enriching products from $(ANALYZED_FILE)..."
	@$(BUILD_DIR)/$(BINARY_NAME) enrich --input $(ANALYZED_FILE) --output $(ENRICHED_FILE)
	@echo "✅ Enrichment complete: $(ENRICHED_FILE)"

run-import: build ## Step 3: Import to Convex
	@echo "📤 Importing products to Convex from $(ENRICHED_FILE)..."
	@$(BUILD_DIR)/$(BINARY_NAME) import --input $(ENRICHED_FILE) --config $(CONFIG_FILE)
	@echo "✅ Import complete"

run-import-dry-run: build ## Dry run import (validate only)
	@echo "🔍 Dry run import validation..."
	@$(BUILD_DIR)/$(BINARY_NAME) import --input $(ENRICHED_FILE) --dry-run
	@echo "✅ Validation complete"

pipeline: build run-analyze run-enrich run-import ## Run full pipeline: Excel → Analyze → Enrich → Import
	@echo "🎉 Full pipeline complete!"

pipeline-dry-run: build ## Test full pipeline without actual import
	@echo "🔍 Running full pipeline in dry-run mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) analyze --excel $(EXCEL_FILE) --output $(ANALYZED_FILE)
	@$(BUILD_DIR)/$(BINARY_NAME) enrich --input $(ANALYZED_FILE) --output $(ENRICHED_FILE)
	@$(BUILD_DIR)/$(BINARY_NAME) import --input $(ENRICHED_FILE) --dry-run
	@echo "✅ Dry-run pipeline complete"

##@ Utilities

stats: build ## Show upload statistics
	@echo "📊 Upload Statistics:"
	@$(BUILD_DIR)/$(BINARY_NAME) stats

reset: ## Reset upload state
	@echo "🗑️  Resetting upload state..."
	@$(BUILD_DIR)/$(BINARY_NAME) reset
	@echo "✅ State reset complete"

clean: ## Clean build artifacts and generated files
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f .upload-state.json
	@rm -rf reports/
	@echo "✅ Clean complete"

clean-all: clean ## Clean everything including generated data files
	@echo "🧹 Cleaning all generated files..."
	@rm -f $(ANALYZED_FILE) $(ENRICHED_FILE) $(CSV_FILE)
	@echo "✅ Deep clean complete"

##@ Development

dev: build-dev test ## Development build and test
	@echo "✅ Development environment ready"

dev-watch: ## Watch for changes and rebuild (requires entr)
	@which entr > /dev/null || (echo "❌ entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)" && exit 1)
	@echo "👀 Watching for changes..."
	@find . -name '*.go' | entr -r make build-dev

setup: ## Setup development environment
	@echo "⚙️  Setting up development environment..."
	@$(GO) mod download
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@cp -n config.example.yaml config.yaml || echo "config.yaml already exists"
	@echo "✅ Setup complete. Edit config.yaml with your credentials."

##@ Docker (Optional)

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .
	@echo "✅ Docker image built"

docker-run: ## Run in Docker container
	@echo "🐳 Running in Docker..."
	@docker run --rm -v $(PWD)/config.yaml:/app/config.yaml $(BINARY_NAME):latest

##@ Examples

example-full: ## Example: Full workflow with sample data
	@echo "📚 Running example full workflow..."
	@make build
	@echo "\n1️⃣  Generating CSV from images..."
	@make generate-csv IMAGE_DIR=./test-images CSV_FILE=test.csv
	@echo "\n2️⃣  Validating CSV..."
	@make validate-csv CSV_FILE=test.csv
	@echo "\n3️⃣  Uploading images (dry-run)..."
	@make upload-images-dry-run CSV_FILE=test.csv
	@echo "\n4️⃣  Analyzing products..."
	@make run-analyze EXCEL_FILE=test-products.xlsx
	@echo "\n5️⃣  Enriching products..."
	@make run-enrich
	@echo "\n6️⃣  Importing to Convex (dry-run)..."
	@make run-import-dry-run
	@echo "\n✅ Example workflow complete!"

example-quick: build ## Example: Quick test with payload generation
	@echo "📚 Running quick test..."
	@make test-payload
	@echo "✅ Quick test complete!"

##@ Info

info: ## Show configuration and environment info
	@echo "📋 Configuration Info:"
	@echo "  Go Version:    $$($(GO) version)"
	@echo "  Binary Name:   $(BINARY_NAME)"
	@echo "  Build Dir:     $(BUILD_DIR)"
	@echo "  Config File:   $(CONFIG_FILE)"
	@echo "  Workers:       $(WORKERS)"
	@echo ""
	@echo "📁 File Paths:"
	@echo "  Excel File:    $(EXCEL_FILE)"
	@echo "  Analyzed File: $(ANALYZED_FILE)"
	@echo "  Enriched File: $(ENRICHED_FILE)"
	@echo "  CSV File:      $(CSV_FILE)"
	@echo "  Image Dir:     $(IMAGE_DIR)"

version: build ## Show application version
	@$(BUILD_DIR)/$(BINARY_NAME) version || echo "Version command not implemented"
