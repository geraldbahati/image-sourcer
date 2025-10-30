# Image Uploader for E-commerce

A high-performance Go CLI tool for batch uploading product images to Cloudinary with intelligent optimization and state management.

## Features

- **Concurrent uploads** - 20 workers by default for blazing fast processing
- **CSV-based manifest** - Easy product management with spreadsheets
- **Automatic optimization** - Smart image transformations and format conversion
- **State tracking** - Incremental uploads, resume capability, skip duplicates
- **Error handling** - Comprehensive retry logic with exponential backoff
- **Detailed reports** - Upload summaries with Cloudinary URLs for database import
- **Progress tracking** - Real-time progress bars and statistics

## Performance

- **Speed**: ~200 images/minute with concurrent uploads
- **Efficiency**: One upload, multiple transformations on-demand
- **Scalability**: Handles 10 to 10,000+ products seamlessly

## Quick Start

### 1. Setup Configuration

```bash
# Copy example config
cp config.example.yaml config.yaml

# Edit config.yaml and add your Cloudinary credentials
# Get credentials from: https://cloudinary.com/console
```

### 2. Prepare Your CSV

Create a CSV file with your product images:

```csv
part_number,product_name,image_path,is_primary
ABC123,Blue Widget Pro,/path/to/images/widget1.jpg,true
ABC123,Blue Widget Pro,/path/to/images/widget2.jpg,false
ABC123,Blue Widget Pro,/path/to/images/widget3.jpg,false
XYZ456,Red Gadget,/path/to/images/gadget1.jpg,true
XYZ456,Red Gadget,/path/to/images/gadget2.jpg,false
```

### 3. Run the Upload

```bash
# Build the application
go build -o uploader ./cmd/uploader

# Upload your images
./uploader upload --csv products.csv
```

## Usage

### Basic Commands

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

## Image Transformations

The tool automatically creates optimized URLs for different use cases:

### Original (Optimized)
```
https://res.cloudinary.com/yourcloud/image/upload/f_auto,q_auto/products/ABC123/ABC123_001.jpg
```
- Auto-format (WebP, AVIF, JPG based on browser)
- Auto-quality (intelligent compression)

### Thumbnail (300x300)
```
https://res.cloudinary.com/yourcloud/image/upload/w_300,h_300,c_fill,f_auto,q_auto/products/ABC123/ABC123_001.jpg
```
- Perfect for product listings and grids

### Product Main (1000x1000)
```
https://res.cloudinary.com/yourcloud/image/upload/w_1000,h_1000,c_pad,b_white,f_auto,q_auto/products/ABC123/ABC123_001.jpg
```
- Primary product page image with white background

### Zoom View (2000x2000)
```
https://res.cloudinary.com/yourcloud/image/upload/w_2000,h_2000,c_fit,f_auto,q_auto:good/products/ABC123/ABC123_001.jpg
```
- High-quality detail view

## Configuration

Edit `config.yaml` to customize behavior:

```yaml
cloudinary:
  cloud_name: "your-cloud-name"
  api_key: "your-api-key"
  api_secret: "your-api-secret"
  folder: "products"

upload:
  concurrent_workers: 20      # Adjust based on your network
  retry_attempts: 3
  retry_delay: 2s
  timeout: 30s

transformations:
  format: "auto"              # Auto-detect best format
  quality: "auto:good"        # High quality for products
  max_width: 2000
  max_height: 2000
  strip_metadata: true        # Remove EXIF data
```

## State Management

The tool tracks uploaded products in `.upload-state.json`:

- **Incremental uploads** - Only new products are uploaded
- **Resume capability** - Interrupted uploads can be resumed
- **Idempotent** - Safe to re-run without duplicates

### Workflow Example

**Week 1: Upload 200 products**
```bash
./uploader upload --csv products.csv
# Uploads all 200 products (~10 minutes)
```

**Week 2: Add 25 new products**
```bash
# Add 25 new rows to products.csv
./uploader upload --csv products.csv
# Only uploads 25 new products (~1 minute)
# Automatically skips existing 200
```

## Output Reports

After each upload, find detailed reports in the `reports/` directory:

```csv
part_number,product_name,original_path,cloudinary_url,thumbnail_url,main_url,zoom_url,status,error_message
ABC123,Blue Widget Pro,/path/img1.jpg,https://...,https://...,https://...,https://...,success,
ABC123,Blue Widget Pro,/path/img2.jpg,https://...,https://...,https://...,https://...,success,
XYZ456,Red Gadget,/path/img3.jpg,,,,,failed,File not found
```

Import this CSV into your e-commerce database!

## Error Handling

The tool handles common issues gracefully:

- **Network interruption** - Auto-retry with exponential backoff
- **Cloudinary rate limit** - Pause and resume
- **Invalid images** - Skip and log, continue processing
- **Missing files** - Log error, continue with others
- **API errors** - Detailed logging with context

## Project Structure

```
image-uploader/
├── cmd/uploader/          # CLI entry point
├── internal/
│   ├── config/           # Configuration loading
│   ├── csv/              # CSV parsing
│   ├── cloudinary/       # Cloudinary client
│   ├── image/            # Image validation
│   ├── state/            # State tracking
│   ├── worker/           # Worker pool
│   └── report/           # Report generation
├── reports/              # Upload reports
├── config.yaml           # Your configuration
└── .upload-state.json    # State tracking
```

## Requirements

- Go 1.24.2 or later
- Cloudinary account ([Sign up free](https://cloudinary.com/users/register/free))

## Installation

```bash
# Clone the repository
git clone https://github.com/geraldbahati/image-sourcer.git
cd image-sourcer

# Install dependencies
go mod download

# Build
go build -o uploader ./cmd/uploader

# Run
./uploader upload --csv products.csv
```

## Development

```bash
# Run tests
go test ./...

# Run with race detection
go run -race ./cmd/uploader upload --csv test.csv

# Build for production
go build -ldflags="-s -w" -o uploader ./cmd/uploader
```

## Documentation

- [APPROACH.md](APPROACH.md) - Detailed technical architecture and design decisions
- [Cloudinary Documentation](https://cloudinary.com/documentation)

## Cost Estimation

### Cloudinary Free Tier
- 25 GB storage
- 25 GB bandwidth/month
- 25,000 transformations/month

### Your Usage (200 products × 4 images)
- ~800 images × 2MB = ~1.6 GB storage
- Well within free tier!

## Troubleshooting

### Images not uploading
- Check Cloudinary credentials in `config.yaml`
- Verify image paths in CSV are absolute paths
- Ensure images are valid formats (JPG, PNG, WebP)

### Slow uploads
- Increase `concurrent_workers` in config
- Check network bandwidth
- Verify Cloudinary plan limits

### State file issues
- Delete `.upload-state.json` to reset
- Use `--force` flag to ignore state

## Contributing

Contributions welcome! Please open an issue or submit a pull request.

## License

MIT License - feel free to use for your projects.

## Support

For issues or questions:
- Open an issue on GitHub
- Check [APPROACH.md](APPROACH.md) for technical details

---

**Built with Go for speed, reliability, and performance.**
