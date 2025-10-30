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
# Generate CSV from images folder (scans directory and creates manifest)
./uploader generate --dir /path/to/images --output products.csv

# Generate CSV with custom part number pattern
./uploader generate --dir /path/to/images --pattern "^[A-Z0-9]+" --interactive

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
├── cmd/uploader/          # CLI entry point
│   ├── main.go           # Cobra CLI setup with all commands
│   ├── generate.go       # Generate CSV from images folder
│   ├── upload.go         # Upload command with progress tracking
│   ├── validate.go       # CSV validation command
│   ├── stats.go          # Statistics display command
│   └── reset.go          # State reset command
├── internal/
│   ├── config/           # Configuration loading (Viper-based, validates Cloudinary creds)
│   ├── csv/              # CSV parsing (part_number, product_name, image_path, is_primary)
│   ├── cloudinary/       # Cloudinary SDK wrapper and upload logic
│   │   ├── client.go     # Cloudinary client wrapper with Ping()
│   │   └── uploader.go   # Upload logic with retry and transformations
│   ├── image/            # Image validation (file type, size, dimensions)
│   ├── state/            # State tracking (.upload-state.json persistence)
│   ├── worker/           # Worker pool implementation with goroutines
│   └── report/           # Report generation (CSV with Cloudinary URLs)
├── reports/              # Generated upload reports
├── config.yaml           # User configuration (gitignored, use config.example.yaml)
└── .upload-state.json    # State tracking file (auto-generated)
```

### Core Packages

**cmd/uploader**
- Full Cobra-based CLI with 5 commands: generate, upload, validate, stats, reset
- `generate.go`: Scans image directories, extracts part numbers, auto-generates CSV manifests
- `upload.go`: Main upload command with progress bars, dry-run mode, resume capability
- `validate.go`: Pre-flight CSV validation without uploading
- `stats.go`: Display upload statistics from state file
- `reset.go`: Clear state and start fresh

**internal/config** (144 lines)
- Uses Viper for YAML configuration
- Validates all required Cloudinary credentials
- Configures: workers, retry logic, transformations, state tracking, output settings
- Key struct: `Config` with nested configs for Cloudinary, Upload, Transformations, State, Output

**internal/csv** (253 lines)
- Parses CSV manifest with columns: `part_number, product_name, image_path, is_primary`
- Groups images by product (part_number)
- Returns `Manifest` struct with both product-grouped and flat image lists
- Validates CSV structure and required columns
- Full validation with row-level error reporting

**internal/cloudinary** (306 lines)
- Wraps official Cloudinary Go SDK (`github.com/cloudinary/cloudinary-go/v2`)
- `client.go`: Client wrapper with Ping() for connection testing
- `uploader.go`: Upload logic with retry, transformations, URL generation
- Uploads to folder structure: `/products/{part_number}/{part_number}_001.jpg`
- Applies automatic transformations: format=auto, quality=auto
- Generates multiple URL variants (original, thumbnail, main, zoom)
- Retry with exponential backoff

**internal/state** (182 lines)
- Thread-safe state management with RWMutex
- Persists upload history to `.upload-state.json`
- Tracks which products have been uploaded to avoid duplicates
- Enables resume functionality for interrupted uploads
- Stores Cloudinary URLs for each uploaded image
- Methods: Load(), Save(), IsProductUploaded(), MarkProductUploaded(), Clear()

**internal/worker** (120 lines)
- Implements concurrent worker pool pattern with goroutines
- Configurable worker count (default: 20)
- Buffered channels for job distribution and result collection
- Context-based cancellation support
- Graceful shutdown with WaitGroup
- Methods: Start(), Submit(), Results(), Close(), Cancel()

**internal/image** (172 lines)
- Validates image files before upload
- Checks file types (JPG, PNG, GIF, WebP, BMP)
- Validates file existence and accessibility
- Checks file sizes and dimensions
- Supports WebP format

**internal/report** (171 lines)
- Generates CSV reports in `reports/` directory
- Includes: part_number, product_name, original_path, cloudinary_url, thumbnail_url, main_url, zoom_url, status, error_message
- Reports can be imported directly into e-commerce databases
- Timestamped report files

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

## Special Features

### CSV Generation from Folders

The `generate` command is a powerful feature that scans a directory of images and automatically creates a CSV manifest:

```bash
# Basic usage - scans folder and creates products.csv
./uploader generate --dir /path/to/images

# Custom output file
./uploader generate --dir /path/to/images --output manifest.csv

# Custom regex pattern for part number extraction
./uploader generate --dir /path/to/images --pattern "^[A-Z]{3}\d{4}"

# Interactive mode - prompts for product names
./uploader generate --dir /path/to/images --interactive
```

**How it works:**
1. Recursively scans the specified directory for image files (.jpg, .jpeg, .png, .gif, .webp, .bmp)
2. Extracts part numbers from filenames using regex (default: `^[A-Z0-9]+`)
3. Groups images by part number
4. Auto-generates product names from filenames
5. Sets the first image in each group as primary
6. Outputs a properly formatted CSV ready for upload

**Supported filename patterns:**
- `ABC123_001.jpg` → Part: ABC123
- `XYZ-456-blue-widget.png` → Part: XYZ, Name: blue widget
- `PROD789 image 1.jpg` → Part: PROD789, Name: image

## Development Notes

### Implementation Status

**Fully Implemented** (1,348+ lines of Go code)
- ✅ Configuration loading (`internal/config/config.go` - 144 lines)
- ✅ CSV parsing (`internal/csv/parser.go` - 253 lines)
- ✅ CLI entry point (`cmd/uploader/` - main.go + 5 command files)
- ✅ Cloudinary client wrapper (`internal/cloudinary/` - 306 lines)
- ✅ Worker pool implementation (`internal/worker/pool.go` - 120 lines)
- ✅ State tracking (`internal/state/tracker.go` - 182 lines)
- ✅ Image validation (`internal/image/validator.go` - 172 lines)
- ✅ Report generation (`internal/report/generator.go` - 171 lines)

**Commands Available:**
- ✅ `generate` - Auto-generate CSV from image folders
- ✅ `upload` - Upload with progress, retry, resume
- ✅ `validate` - Pre-flight CSV validation
- ✅ `stats` - Display upload statistics
- ✅ `reset` - Clear state file

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
