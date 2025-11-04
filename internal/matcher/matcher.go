package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/geraldbahati/image-sourcer/internal/excel"
)

// MatcherConfig holds configuration for the matcher
type MatcherConfig struct {
	MaxWorkers int  // Maximum concurrent matchers (default: 8)
	BufferSize int  // Channel buffer size (default: 100)
}

// DefaultMatcherConfig returns default configuration
func DefaultMatcherConfig() MatcherConfig {
	return MatcherConfig{
		MaxWorkers: 8,
		BufferSize: 100,
	}
}

// Matcher handles efficient matching of Excel products to uploaded images
type Matcher struct {
	config MatcherConfig
	state  *UploadState
}

// NewMatcher creates a new matcher with the given configuration
func NewMatcher(config MatcherConfig) *Matcher {
	return &Matcher{
		config: config,
		state:  &UploadState{Products: make(map[string]UploadedProduct)},
	}
}

// LoadUploadState loads the upload state from file
func (m *Matcher) LoadUploadState(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read upload state: %w", err)
	}

	if err := json.Unmarshal(data, &m.state); err != nil {
		return fmt.Errorf("failed to parse upload state: %w", err)
	}

	return nil
}

// MatchProducts matches Excel products to uploaded images with high performance
func (m *Matcher) MatchProducts(
	ctx context.Context,
	parseResult *excel.ParseResult,
	detectionResults map[string]excel.DetectionResult, // sheetName → detection
) (*MatchResult, error) {
	result := &MatchResult{
		Matched:   make([]MatchedProduct, 0),
		Unmatched: make([]MatchedProduct, 0),
	}

	// Create channel for products to match
	productChan := make(chan matchJob, m.config.BufferSize)
	var wg sync.WaitGroup

	// Start matcher workers
	for i := 0; i < m.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.matchWorker(ctx, productChan, result)
		}()
	}

	// Send products to workers
	go func() {
		defer close(productChan)

		for _, file := range parseResult.GetFiles() {
			for _, sheet := range file.Sheets {
				// Get detection result for this sheet
				sheetKey := fmt.Sprintf("%s:%s", file.FileName, sheet.Name)
				detection, ok := detectionResults[sheetKey]
				if !ok {
					continue
				}

				// Get column mappings
				skuCol, hasSKU := detection.Columns["sku"]
				nameCol, hasName := detection.Columns["name"]
				if !hasSKU || !hasName {
					continue // Skip sheets without required columns
				}

				priceCol, hasPrice := detection.Columns["price"]
				stockCol, hasStock := detection.Columns["stock"]

				// Process each row
				for _, row := range sheet.Rows {
					select {
					case <-ctx.Done():
						return
					default:
						job := matchJob{
							file:     file,
							sheet:    sheet,
							row:      row,
							skuCol:   skuCol.OriginalName,
							nameCol:  nameCol.OriginalName,
							priceCol: "",
							stockCol: "",
							hasPrice: hasPrice,
							hasStock: hasStock,
						}

						if hasPrice {
							job.priceCol = priceCol.OriginalName
						}
						if hasStock {
							job.stockCol = stockCol.OriginalName
						}

						productChan <- job
					}
				}
			}
		}
	}()

	// Wait for all workers to finish
	wg.Wait()

	// Calculate final match rate
	result.Finalize()

	return result, nil
}

// matchJob represents a product matching job
type matchJob struct {
	file     *excel.ExcelFile
	sheet    excel.ExcelSheet
	row      excel.ExcelRow
	skuCol   string
	nameCol  string
	priceCol string
	stockCol string
	hasPrice bool
	hasStock bool
}

// matchWorker processes matching jobs
func (m *Matcher) matchWorker(ctx context.Context, jobs <-chan matchJob, result *MatchResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return // Channel closed
			}

			// Extract product data
			sku := job.row.Cells[job.skuCol]
			name := job.row.Cells[job.nameCol]

			// Skip if missing required fields
			if sku == "" || name == "" {
				continue
			}

			// Create base product
			product := MatchedProduct{
				SKU:         strings.ToUpper(strings.TrimSpace(sku)),
				Name:        strings.TrimSpace(name),
				Category:    job.sheet.Name, // Use sheet name as category
				SourceFile:  job.file.FileName,
				SourceSheet: job.sheet.Name,
				RowNumber:   job.row.RowNumber,
			}

			// Extract optional fields
			if job.hasPrice {
				if priceStr, ok := job.row.Cells[job.priceCol]; ok {
					product.Price = parsePrice(priceStr)
				}
			}

			if job.hasStock {
				if stockStr, ok := job.row.Cells[job.stockCol]; ok {
					product.Stock = parseStock(stockStr)
				}
			}

			// Try to match to upload state (O(1) hash map lookup)
			if uploaded, found := m.state.Products[product.SKU]; found {
				product.Images = convertToCloudinaryImages(uploaded.CloudinaryURLs)
				product.HasImages = true
				product.ImageCount = len(uploaded.CloudinaryURLs)
				result.AddMatched(product)
			} else {
				// Try alternate SKU formats (common variations)
				alternates := generateSKUAlternates(product.SKU)
				matched := false

				for _, altSKU := range alternates {
					if uploaded, found := m.state.Products[altSKU]; found {
						product.Images = convertToCloudinaryImages(uploaded.CloudinaryURLs)
						product.HasImages = true
						product.ImageCount = len(uploaded.CloudinaryURLs)
						result.AddMatched(product)
						matched = true
						break
					}
				}

				if !matched {
					product.HasImages = false
					product.ImageCount = 0
					result.AddUnmatched(product)
				}
			}
		}
	}
}

// convertToCloudinaryImages converts CloudinaryURL strings to CloudinaryImage objects with transformations
func convertToCloudinaryImages(cloudinaryURLs []string) []CloudinaryImage {
	images := make([]CloudinaryImage, len(cloudinaryURLs))

	for i, url := range cloudinaryURLs {
		// Extract the base URL and public ID from the Cloudinary URL
		// URL format: https://res.cloudinary.com/{cloud_name}/image/upload/v{version}/{public_id}.{ext}

		images[i] = CloudinaryImage{
			Path:          extractPublicID(url),
			CloudinaryURL: url,
			ThumbnailURL:  generateThumbnailURL(url),
			MainURL:       generateMainURL(url),
			ZoomURL:       generateZoomURL(url),
		}
	}

	return images
}

// extractPublicID extracts the public ID from a Cloudinary URL
func extractPublicID(url string) string {
	// Extract everything after /upload/
	parts := strings.Split(url, "/upload/")
	if len(parts) < 2 {
		return ""
	}

	// Remove version if present (v1234567890)
	path := parts[1]
	if strings.HasPrefix(path, "v") {
		// Remove version number
		slashIdx := strings.Index(path, "/")
		if slashIdx > 0 {
			path = path[slashIdx+1:]
		}
	}

	// Remove file extension
	dotIdx := strings.LastIndex(path, ".")
	if dotIdx > 0 {
		path = path[:dotIdx]
	}

	return path
}

// generateThumbnailURL generates thumbnail transformation URL (300x300)
func generateThumbnailURL(baseURL string) string {
	return insertTransformation(baseURL, "w_300,h_300,c_fill,f_auto,q_auto")
}

// generateMainURL generates main image transformation URL (1000x1000)
func generateMainURL(baseURL string) string {
	return insertTransformation(baseURL, "w_1000,h_1000,c_pad,b_white,f_auto,q_auto")
}

// generateZoomURL generates zoom transformation URL (2000x2000)
func generateZoomURL(baseURL string) string {
	return insertTransformation(baseURL, "w_2000,h_2000,c_fit,f_auto,q_auto:good")
}

// insertTransformation inserts transformation parameters into Cloudinary URL
func insertTransformation(url, transformation string) string {
	// URL format: https://res.cloudinary.com/{cloud_name}/image/upload/v{version}/{public_id}.{ext}
	// Insert transformation after /upload/: https://res.cloudinary.com/{cloud_name}/image/upload/{transformation}/v{version}/{public_id}.{ext}

	uploadIdx := strings.Index(url, "/upload/")
	if uploadIdx < 0 {
		return url
	}

	// Insert transformation after /upload/
	return url[:uploadIdx+8] + transformation + "/" + url[uploadIdx+8:]
}

// generateSKUAlternates generates common SKU variations for better matching
func generateSKUAlternates(sku string) []string {
	alternates := make([]string, 0, 4)

	// Original with different cases
	alternates = append(alternates, strings.ToUpper(sku))
	alternates = append(alternates, strings.ToLower(sku))

	// Without dashes/underscores
	noDash := strings.ReplaceAll(sku, "-", "")
	noDash = strings.ReplaceAll(noDash, "_", "")
	if noDash != sku {
		alternates = append(alternates, strings.ToUpper(noDash))
	}

	// With dashes instead of underscores
	withDash := strings.ReplaceAll(sku, "_", "-")
	if withDash != sku {
		alternates = append(alternates, strings.ToUpper(withDash))
	}

	return alternates
}

// parsePrice extracts price from string (optimized)
func parsePrice(s string) float64 {
	if s == "" {
		return 0.0
	}

	// Fast path: clean numeric string
	cleaned := cleanNumericString(s)
	price, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0.0
	}

	return price
}

// parseStock extracts stock from string (optimized)
func parseStock(s string) int {
	if s == "" {
		return 0
	}

	stock, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}

	return stock
}

// cleanNumericString removes non-numeric characters (optimized with strings.Builder)
func cleanNumericString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s)) // Pre-allocate

	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// MatchProductsWithDefaults is a convenience function using default config
func MatchProductsWithDefaults(
	ctx context.Context,
	statePath string,
	parseResult *excel.ParseResult,
	detectionResults map[string]excel.DetectionResult,
) (*MatchResult, error) {
	matcher := NewMatcher(DefaultMatcherConfig())

	if err := matcher.LoadUploadState(statePath); err != nil {
		return nil, err
	}

	return matcher.MatchProducts(ctx, parseResult, detectionResults)
}
