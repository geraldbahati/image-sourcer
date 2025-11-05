# Quick Start Guide - Image Sourcer with Convex Integration

This guide shows you how to use the Makefile to streamline your entire workflow from building to importing products to Convex.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Common Workflows](#common-workflows)
- [Command Reference](#command-reference)
- [Customizing Paths](#customizing-paths)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Go 1.21 or higher
- Cloudinary account (for image uploads)
- Convex account (for product import)
- Excel file with product data

## Initial Setup

### 1. Clone and Setup

```bash
# Setup development environment (downloads dependencies, creates config.yaml)
make setup

# Edit config.yaml with your credentials
nano config.yaml
```

### 2. Build the Application

```bash
# Standard build
make build

# Or development build with race detection
make build-dev

# Or optimized production build
make build-prod
```

### 3. Verify Setup

```bash
# Show configuration
make info

# Test payload generation
make test-payload
```

## Common Workflows

### Workflow 1: Full Product Import Pipeline (Excel → Convex)

This is the most common workflow for importing products to Convex:

```bash
# Run the complete pipeline in one command
make pipeline EXCEL_FILE=products.xlsx

# Or run steps individually:
make run-analyze EXCEL_FILE=products.xlsx           # Step 1: Analyze Excel
make run-enrich ANALYZED_FILE=analyzed_products.json # Step 2: Enrich with AI
make run-import ENRICHED_FILE=enriched_products.json # Step 3: Import to Convex
```

**What happens:**
1. **Analyze**: Parses Excel file, extracts product data
2. **Enrich**: Adds AI-generated SEO, descriptions, specifications
3. **Import**: Uploads to Convex with all 57 fields

### Workflow 2: Image Upload to Cloudinary

Generate CSV from images and upload to Cloudinary:

```bash
# Step 1: Generate CSV from images folder
make generate-csv IMAGE_DIR=./my-images CSV_FILE=products.csv

# Step 2: Validate the CSV
make validate-csv CSV_FILE=products.csv

# Step 3: Upload images (dry-run first)
make upload-images-dry-run CSV_FILE=products.csv

# Step 4: Upload for real
make upload-images CSV_FILE=products.csv WORKERS=20
```

### Workflow 3: Development Mode

For active development with automatic rebuilds:

```bash
# Development build and test
make dev

# Or watch for changes and auto-rebuild (requires entr)
make dev-watch
```

### Workflow 4: Testing Before Production

Always test before running on production data:

```bash
# Test the full pipeline without actual import
make pipeline-dry-run EXCEL_FILE=test-products.xlsx

# Test just the import step
make run-import-dry-run ENRICHED_FILE=enriched_products.json

# Test payload generation
make test-payload
```

## Command Reference

### 🔨 Build Commands

```bash
make build              # Standard build → ./bin/uploader
make build-dev          # Development build with race detection
make build-prod         # Optimized production binary
make install            # Install to $GOPATH/bin
```

### 🧪 Testing Commands

```bash
make test               # Run all tests
make test-coverage      # Generate coverage report
make test-race          # Run tests with race detection
make test-payload       # Test Convex payload generation (57 fields)
```

### 📊 Product Pipeline Commands

```bash
# Individual steps
make run-analyze        # Step 1: Analyze Excel → JSON
make run-enrich         # Step 2: Enrich with AI
make run-import         # Step 3: Import to Convex

# Combined
make pipeline           # Run all steps: analyze → enrich → import
make pipeline-dry-run   # Test pipeline without actual import

# With custom files
make run-analyze EXCEL_FILE=my-products.xlsx ANALYZED_FILE=output.json
make run-enrich ANALYZED_FILE=input.json ENRICHED_FILE=enriched.json
make run-import ENRICHED_FILE=enriched.json CONFIG_FILE=custom-config.yaml
```

### 📝 CSV & Image Commands

```bash
make generate-csv              # Generate CSV from images
make generate-csv-interactive  # Generate CSV with prompts
make validate-csv              # Validate CSV structure

make upload-images             # Upload to Cloudinary
make upload-images-dry-run     # Validate without uploading
make upload-images-resume      # Resume interrupted upload
```

### 🛠️ Utility Commands

```bash
make stats              # Show upload statistics
make reset              # Reset upload state
make clean              # Remove build artifacts
make clean-all          # Remove all generated files
make format             # Format code with gofmt
make lint               # Run linter (requires golangci-lint)
make check-deps         # Verify dependencies
```

### 📋 Info Commands

```bash
make help               # Show all available commands
make info               # Show configuration and paths
make version            # Show application version
```

## Customizing Paths

You can override default file paths using environment variables:

```bash
# Custom Excel file
make pipeline EXCEL_FILE=data/products-2024.xlsx

# Custom output paths
make run-analyze \
  EXCEL_FILE=input/products.xlsx \
  ANALYZED_FILE=output/analyzed.json

# Custom image directory and CSV
make generate-csv \
  IMAGE_DIR=/path/to/images \
  CSV_FILE=custom-products.csv

# Custom number of workers
make upload-images WORKERS=30

# Custom config file
make run-import CONFIG_FILE=production.yaml
```

### Setting Defaults

Create a `.env` file or export variables:

```bash
# In your shell or .bashrc/.zshrc
export EXCEL_FILE=data/products.xlsx
export WORKERS=25
export CONFIG_FILE=production.yaml

# Then just run:
make pipeline
```

## Advanced Usage

### Chain Multiple Commands

```bash
# Build, test, and run pipeline
make build && make test && make pipeline

# Clean, rebuild, and test
make clean && make build-prod && make test-coverage
```

### Parallel Testing

```bash
# Run tests while building
make build & make test & wait
```

### Production Deployment

```bash
# Build optimized binary
make build-prod

# Run with production config
./bin/uploader import \
  --input enriched_products.json \
  --config production.yaml \
  --workers 30
```

## Examples

### Example 1: Complete New Product Import

```bash
# 1. Setup (first time only)
make setup
# Edit config.yaml with your Cloudinary and Convex credentials

# 2. Generate CSV from images
make generate-csv IMAGE_DIR=./product-photos

# 3. Upload images to Cloudinary
make upload-images CSV_FILE=products.csv

# 4. Prepare Excel file with product data
# (manually create products.xlsx or use existing one)

# 5. Run full pipeline to Convex
make pipeline EXCEL_FILE=products.xlsx

# 6. Check results
make stats
```

### Example 2: Update Existing Products

```bash
# 1. Create new enriched data
make run-analyze EXCEL_FILE=updated-products.xlsx
make run-enrich

# 2. Dry-run to verify
make run-import-dry-run

# 3. Import to Convex
make run-import
```

### Example 3: Development Workflow

```bash
# 1. Make code changes
# 2. Format and lint
make format && make lint

# 3. Run tests
make test-race

# 4. Test payload generation
make test-payload

# 5. Build and test full pipeline
make dev
make pipeline-dry-run
```

## Troubleshooting

### Build Fails

```bash
# Check Go version
go version  # Should be 1.21+

# Check dependencies
make check-deps

# Clean and rebuild
make clean && make build
```

### Upload Fails

```bash
# Verify config
make info

# Check Cloudinary credentials in config.yaml
cat config.yaml

# Try dry-run first
make upload-images-dry-run

# Check state file
cat .upload-state.json
```

### Import Fails

```bash
# Verify payload structure
make test-payload

# Check enriched data format
cat enriched_products.json | jq '.' | head -50

# Try dry-run first
make run-import-dry-run

# Check Convex deployment URL in config.yaml
grep convex config.yaml
```

### Pipeline Fails Mid-Way

```bash
# Check which step completed
ls -la *.json

# Resume from failed step
make run-enrich  # If analyze completed
make run-import  # If enrich completed
```

### Reset Everything

```bash
# Nuclear option - start fresh
make clean-all
make reset
make setup
```

## Performance Tips

### Optimize Upload Speed

```bash
# Increase workers (default: 20)
make upload-images WORKERS=40

# But watch for rate limits
# Cloudinary free tier: ~200 requests/hour
```

### Optimize Import Speed

```bash
# Increase batch size in config.yaml
convex:
  batch_size: 100      # Default: 50
  max_workers: 30      # Default: 20
```

### Development Speed

```bash
# Use dev build for faster compilation
make build-dev

# Skip tests during rapid iteration
make build  # Instead of make dev

# Use dry-run to skip actual API calls
make pipeline-dry-run
```

## File Structure

After running the pipeline, you'll have:

```
image-sourcer/
├── bin/
│   └── uploader                    # Built binary
├── config.yaml                     # Your configuration
├── products.csv                    # Generated CSV (if using images)
├── analyzed_products.json          # After analyze step
├── enriched_products.json          # After enrich step
├── .upload-state.json             # Upload state tracking
├── reports/
│   └── upload_report_*.csv        # Upload reports
└── coverage.html                   # Test coverage (if generated)
```

## Next Steps

1. **Read the main README**: `cat README.md`
2. **Check the approach doc**: `cat APPROACH.md`
3. **Review Convex integration**: `cat /tmp/convex_integration_complete.md`
4. **Customize config**: `nano config.yaml`
5. **Run your first pipeline**: `make pipeline EXCEL_FILE=your-products.xlsx`

## Getting Help

```bash
# Show all available commands
make help

# Show current configuration
make info

# Test setup
make test-payload
make test
```

For more details, see:
- `README.md` - Complete documentation
- `APPROACH.md` - Technical architecture
- `CLAUDE.md` - Project context for Claude Code

---

**Pro Tip**: Add commonly used commands as shell aliases:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias up-build='make build'
alias up-test='make test-payload'
alias up-pipeline='make pipeline'
alias up-clean='make clean-all'
```

Then just run: `up-pipeline EXCEL_FILE=products.xlsx`
