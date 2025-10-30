# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a high-performance Go CLI tool for batch uploading product images to Cloudinary with intelligent optimization and state management. The tool is designed for e-commerce applications, targeting ~200 products with multiple images per product (~600-800 total images).

## Development Commands

### Build and Run
```bash
# Build the application
go build -o uploader ./cmd/uploader

# Run tests
go test ./...

# Run with race detection (recommended during development)
go run -race ./cmd/uploader upload --csv test.csv

# Build for production (optimized)
go build -ldflags="-s -w" -o uploader ./cmd/uploader
```

### Application Usage
```bash
# Upload images from CSV
./uploader upload --csv products.csv

# Dry run (validate only, no upload)
./uploader upload --csv products.csv --dry-run

# Resume previous upload
./uploader upload --csv products.csv --resume

# Force re-upload (ignore state)
./uploader upload --csv products.csv --force

# Custom worker count
./uploader upload --csv products.csv --workers 30

# Validate CSV only
./uploader validate --csv products.csv

# Show upload statistics
./uploader stats

# Clean up state
./uploader reset
```

### Setup
```bash
# Install dependencies
go mod download

# Copy example config and edit with Cloudinary credentials
cp config.example.yaml config.yaml
```

## Architecture

### High-Level Design

The application uses a **worker pool pattern** for concurrent image uploads:

```
CSV Input → Parser → Job Queue → Worker Pool (20 workers) → Cloudinary API
                                       ↓
                              State Tracker + Report Generator
```

**Key architectural decisions:**
- **Concurrent Processing**: 20 workers by default for ~200 images/minute throughput
- **State Management**: Tracks uploaded products in `.upload-state.json` for incremental uploads and resume capability
- **Idempotent Operations**: Safe to re-run; automatically skips already-uploaded products
- **Worker Pool Pattern**: Controlled concurrency prevents API rate limiting while maximizing throughput

### Project Structure

```
image-sourcer/
├── cmd/uploader/          # CLI entry point (currently empty, needs implementation)
├── internal/
│   ├── config/           # Configuration loading (Viper-based, validates Cloudinary creds)
│   ├── csv/              # CSV parsing (part_number, product_name, image_path, is_primary)
│   ├── cloudinary/       # Cloudinary SDK wrapper and upload logic
│   ├── image/            # Image validation
│   ├── state/            # State tracking (.upload-state.json persistence)
│   ├── worker/           # Worker pool implementation
│   └── report/           # Report generation (CSV with Cloudinary URLs)
├── reports/              # Generated upload reports
├── config.yaml           # User configuration (gitignored, use config.example.yaml)
└── .upload-state.json    # State tracking file (auto-generated)
```

### Core Packages

**internal/config**
- Uses Viper for YAML configuration
- Validates all required Cloudinary credentials
- Configures: workers, retry logic, transformations, state tracking, output settings
- Key struct: `Config` with nested configs for Cloudinary, Upload, Transformations, State, Output

**internal/csv**
- Parses CSV manifest with columns: `part_number, product_name, image_path, is_primary`
- Groups images by product (part_number)
- Returns `Manifest` struct with both product-grouped and flat image lists
- Validates CSV structure and required columns

**internal/cloudinary**
- Wraps official Cloudinary Go SDK (`github.com/cloudinary/cloudinary-go/v2`)
- Uploads to folder structure: `/products/{part_number}/{part_number}_001.jpg`
- Applies automatic transformations: format=auto, quality=auto
- Generates multiple URL variants (original, thumbnail, main, zoom)

**internal/state**
- Persists upload history to `.upload-state.json`
- Tracks which products have been uploaded to avoid duplicates
- Enables resume functionality for interrupted uploads
- Stores Cloudinary URLs for each uploaded image

**internal/worker**
- Implements concurrent worker pool pattern
- Configurable worker count (default: 20)
- Channels-based job distribution
- Collects results from all workers

**internal/report**
- Generates CSV reports in `reports/` directory
- Includes: part_number, product_name, original_path, cloudinary_url, thumbnail_url, main_url, zoom_url, status, error_message
- Reports can be imported directly into e-commerce databases

### Cloudinary Integration

**URL Transformation Strategy:**
Single upload generates multiple on-demand transformations:

1. **Original (Optimized)**: `f_auto,q_auto` - Auto-format (WebP/AVIF) and quality
2. **Thumbnail**: `w_300,h_300,c_fill,f_auto,q_auto` - Product grids
3. **Main**: `w_1000,h_1000,c_pad,b_white,f_auto,q_auto` - Product page
4. **Zoom**: `w_2000,h_2000,c_fit,f_auto,q_auto:good` - Detail view

**Benefits:**
- One upload → infinite transformations
- 30-50% smaller files with auto-format
- CDN delivery globally
- No storage bloat from multiple versions

### CSV Format

Expected CSV structure:
```csv
part_number,product_name,image_path,is_primary
ABC123,Blue Widget Pro,/path/to/images/widget1.jpg,true
ABC123,Blue Widget Pro,/path/to/images/widget2.jpg,false
XYZ456,Red Gadget,/path/to/images/gadget1.jpg,true
```

**Key Points:**
- Multiple images per product grouped by `part_number`
- One primary image per product (`is_primary=true`)
- Image paths should be absolute paths
- CSV can be generated from existing databases

### State Management

The `.upload-state.json` file tracks:
- Last upload timestamp
- Uploaded products by part_number
- Image count and Cloudinary URLs per product
- Upload timestamps per product

**Workflow Example:**
1. Week 1: Upload 200 products → ~10 minutes
2. Week 2: Add 25 new products to CSV → Only uploads 25 new → ~1 minute
3. Automatic skip of existing 200 products

### Error Handling

- **Network interruption**: Auto-retry with exponential backoff
- **Cloudinary rate limit**: Pause and resume
- **Invalid images**: Skip, log, continue processing
- **Missing files**: Log error, continue with others
- **Timeout**: Retry up to 3 times (configurable)

### Performance Targets

- **Sequential**: ~40 images/minute (single worker)
- **Concurrent**: ~200 images/minute (20 workers)
- **800 images**: 5-10 minutes total
- **Memory**: ~50-100 MB
- **CPU**: Low (I/O bound)

## Development Notes

### Implementation Status
- ✅ Configuration loading (`internal/config/config.go`)
- ✅ CSV parsing structure (`internal/csv/parser.go` - partial)
- ❌ CLI entry point (`cmd/uploader/main.go` - needs implementation)
- ❌ Cloudinary client wrapper
- ❌ Worker pool implementation
- ❌ State tracking
- ❌ Image validation
- ❌ Report generation

### Key Dependencies
- `github.com/cloudinary/cloudinary-go/v2` - Official Cloudinary SDK
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/schollz/progressbar/v3` - Progress bars
- `github.com/fatih/color` - Colored terminal output

### Configuration

Edit `config.yaml` (copy from `config.example.yaml`):
- **Cloudinary credentials**: Required (cloud_name, api_key, api_secret)
- **Worker count**: Default 20, adjust based on network (1-100)
- **Retry logic**: 3 attempts with 2s delay
- **Transformations**: Eager transformations for thumbnail/main/zoom
- **State tracking**: Enabled by default with `.upload-state.json`

**Important**: `config.yaml` is gitignored. Never commit API credentials.

### Testing Considerations
- Test with small CSV first (5-10 products)
- Use `--dry-run` flag to validate without uploading
- Test resume functionality by interrupting upload
- Verify state file accuracy after uploads
- Test with missing image files to verify error handling

### Security
- API credentials stored in `config.yaml` (gitignored)
- Use environment variables in production
- Validate file types before upload
- Check file sizes to prevent huge uploads
- Strip EXIF metadata (configured in transformations)

## Additional Resources

- **APPROACH.md**: Comprehensive technical architecture and design decisions
- **README.md**: User-facing documentation with setup and usage
- **Cloudinary Docs**: https://cloudinary.com/documentation
- **Cloudinary Go SDK**: https://github.com/cloudinary/cloudinary-go

## Cost Considerations

**Cloudinary Free Tier:**
- 25 GB storage
- 25 GB bandwidth/month
- 25,000 transformations/month

**Expected Usage (200 products × 4 images):**
- ~800 images × 2MB = ~1.6 GB storage
- Well within free tier
