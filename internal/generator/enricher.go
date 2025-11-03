package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// NormalizedProduct represents the input from the analyze phase
type NormalizedProduct struct {
	SKU           string   `json:"sku"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Price         float64  `json:"price"`
	Stock         int      `json:"stock"`
	PrimaryImage  string   `json:"primary_image"`
	GalleryImages []string `json:"gallery_images"`
	ThumbnailURL  string   `json:"thumbnail_url"`
	MainURL       string   `json:"main_url"`
	ZoomURL       string   `json:"zoom_url"`
	HasImages     bool     `json:"has_images"`
	Source        struct {
		File      string `json:"file"`
		Sheet     string `json:"sheet"`
		RowNumber int    `json:"row_number"`
	} `json:"source"`
}

// NormalizedInput represents the input JSON structure
type NormalizedInput struct {
	GeneratedAt   string              `json:"generated_at"`
	TotalProducts int                 `json:"total_products"`
	WithImages    int                 `json:"with_images"`
	WithoutImages int                 `json:"without_images"`
	Products      []NormalizedProduct `json:"products"`
}

// Enricher handles concurrent product enrichment
type Enricher struct {
	config EnrichmentConfig
}

// NewEnricher creates a new enricher with the given configuration
func NewEnricher(config EnrichmentConfig) *Enricher {
	return &Enricher{config: config}
}

// EnrichProducts enriches all products concurrently
func (e *Enricher) EnrichProducts(ctx context.Context, products []NormalizedProduct) (*EnrichmentResult, error) {
	startTime := time.Now()

	result := &EnrichmentResult{
		Products: make([]EnrichedProduct, 0, len(products)),
		Errors:   make([]EnrichmentError, 0),
	}

	// Create channels
	jobChan := make(chan NormalizedProduct, e.config.BufferSize)
	resultChan := make(chan EnrichedProduct, e.config.BufferSize)
	errorChan := make(chan EnrichmentError, e.config.BufferSize)

	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < e.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.enrichWorker(ctx, jobChan, resultChan, errorChan)
		}()
	}

	// Start result collector
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for enriched := range resultChan {
			result.Products = append(result.Products, enriched)
			result.TotalEnriched++
		}
	}()

	// Start error collector
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for err := range errorChan {
			result.Errors = append(result.Errors, err)
		}
	}()

	// Send jobs to workers
	go func() {
		for _, product := range products {
			select {
			case <-ctx.Done():
				return
			case jobChan <- product:
				result.TotalProcessed++
			}
		}
		close(jobChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)
	close(errorChan)

	// Wait for collectors
	collectorWg.Wait()

	// Step 4: Detect and group variants (if enabled)
	if e.config.DetectVariants {
		detector := NewVariantDetector()
		variantProducts, err := detector.DetectVariants(ctx, result.Products)
		if err != nil {
			return nil, fmt.Errorf("variant detection failed: %w", err)
		}
		result.Products = variantProducts
	}

	result.Duration = time.Since(startTime)

	return result, nil
}

// enrichWorker processes enrichment jobs
func (e *Enricher) enrichWorker(ctx context.Context, jobs <-chan NormalizedProduct, results chan<- EnrichedProduct, errors chan<- EnrichmentError) {
	for {
		select {
		case <-ctx.Done():
			return
		case product, ok := <-jobs:
			if !ok {
				return
			}

			enriched, err := e.enrichSingleProduct(product)
			if err != nil {
				errors <- EnrichmentError{
					SKU:     product.SKU,
					Message: err.Error(),
				}
				continue
			}

			results <- enriched
		}
	}
}

// enrichSingleProduct enriches a single product
func (e *Enricher) enrichSingleProduct(product NormalizedProduct) (EnrichedProduct, error) {
	now := time.Now()

	enriched := EnrichedProduct{
		// Basic info
		SKU:      product.SKU,
		Name:     product.Name,
		Category: product.Category,
		Stock:    product.Stock,
		IsActive: true, // Default to active

		// Generate slug
		Slug: GenerateSlug(product.Name, product.SKU),

		// Pricing (USD - source of truth)
		BasePriceUsd: product.Price,

		// Images
		PrimaryImage: product.PrimaryImage,
		ThumbnailURL: product.ThumbnailURL,

		// Source
		Source: SourceMetadata{
			File:      product.Source.File,
			Sheet:     product.Source.Sheet,
			RowNumber: product.Source.RowNumber,
		},

		// Timestamps
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add gallery images if present
	if len(product.GalleryImages) > 0 {
		enriched.Images = product.GalleryImages
	}

	// Calculate KES pricing
	basePriceKes, comparePriceKes := CalculatePricing(product.Price, 0, e.config)
	enriched.BasePriceKes = basePriceKes
	enriched.ComparePriceKes = comparePriceKes

	// Generate description
	enriched.Description = GenerateDescription(
		product.Name,
		product.Category,
		e.config.DescriptionStyle,
		e.config.MaxDescLength,
	)

	// Generate SEO metadata
	enriched.SEO = GenerateSEO(
		product.Name,
		enriched.Description,
		product.Category,
		enriched.Slug,
		product.Price,
		product.PrimaryImage,
		e.config,
	)

	// Extract specifications
	enriched.Specifications = ExtractSpecifications(
		product.Name,
		product.Category,
		product.SKU,
	)

	// Generate features
	enriched.Features = GenerateFeatures(
		product.Name,
		product.Category,
		e.config.DefaultFeatures,
	)

	// Generate tags (from category and name)
	enriched.Tags = generateTags(product.Name, product.Category)

	// Generate badges
	enriched.Badges = generateBadges(product.Name, product.Category, product.HasImages)

	// Mark as new if no images yet
	enriched.IsNew = !product.HasImages

	return enriched, nil
}

// generateTags creates product tags
func generateTags(name, category string) []string {
	tags := make([]string, 0, 5)

	// Add category as tag
	tags = append(tags, category)

	// Add brand as tag
	brand := extractBrand(name)
	if brand != "" {
		tags = append(tags, brand)
	}

	// Add product type
	productType := extractProductType(name, category)
	if productType != "" && productType != category {
		tags = append(tags, productType)
	}

	return tags
}

// generateBadges creates product badges
func generateBadges(name, category string, hasImages bool) []string {
	badges := make([]string, 0, 3)

	nameLower := strings.ToLower(name)

	// New Arrival badge for products without images
	if !hasImages {
		badges = append(badges, "New Arrival")
	}

	// Special badges based on keywords
	if strings.Contains(nameLower, "pro") || strings.Contains(nameLower, "professional") {
		badges = append(badges, "Professional")
	}

	if strings.Contains(nameLower, "gaming") {
		badges = append(badges, "Gaming")
	}

	if strings.Contains(nameLower, "eco") {
		badges = append(badges, "Eco-Friendly")
	}

	if strings.Contains(nameLower, "premium") {
		badges = append(badges, "Premium")
	}

	return badges
}

// LoadNormalizedInput loads normalized products from JSON file
func LoadNormalizedInput(path string) (*NormalizedInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var input NormalizedInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &input, nil
}

// SaveEnrichedOutput saves enriched products to JSON file
func SaveEnrichedOutput(path string, result *EnrichmentResult) error {
	output := map[string]interface{}{
		"generated_at":    time.Now().Format(time.RFC3339),
		"total_products":  len(result.Products),
		"total_enriched":  result.TotalEnriched,
		"total_errors":    len(result.Errors),
		"processing_time": result.Duration.String(),
		"products":        result.Products,
		"errors":          result.Errors,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
