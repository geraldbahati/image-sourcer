package convex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geraldbahati/image-sourcer/internal/generator"
)

// Importer handles concurrent product imports to Convex
type Importer struct {
	client *Client
	config ImportConfig
	mapper MapperConfig
}

// NewImporter creates a new Convex importer
func NewImporter(config ImportConfig) *Importer {
	return &Importer{
		client: NewClient(config),
		config: config,
		mapper: DefaultMapperConfig(),
	}
}

// Import imports enriched products to Convex with concurrent processing
func (im *Importer) Import(ctx context.Context, enrichedData *generator.EnrichmentResult) (*ImportResult, error) {
	startTime := time.Now()

	result := &ImportResult{
		TotalProducts: len(enrichedData.Products),
		CategoryIDMap: make(map[string]string),
		Errors:        make([]ImportError, 0),
	}

	// Step 1: Create categories
	categoryMap, err := im.createCategories(ctx, enrichedData.Products)
	if err != nil {
		return nil, fmt.Errorf("failed to create categories: %w", err)
	}
	result.CategoryIDMap = categoryMap
	result.ImportedCategories = len(categoryMap)

	// Step 2: Import products with concurrent workers
	importedCount, productIDMap, importErrors := im.importProducts(ctx, enrichedData.Products, categoryMap)
	result.ImportedProducts = importedCount
	result.Errors = append(result.Errors, importErrors...)

	// Step 3: Import product specifications
	specsCount, specsErrors := im.importSpecifications(ctx, enrichedData.Products, productIDMap)
	result.ImportedSpecs = specsCount
	result.Errors = append(result.Errors, specsErrors...)

	result.Duration = time.Since(startTime)

	return result, nil
}

// createCategories creates all unique categories
func (im *Importer) createCategories(ctx context.Context, products []generator.EnrichedProduct) (map[string]string, error) {
	// Extract unique categories
	categorySet := make(map[string]bool)
	for _, product := range products {
		if product.Category != "" {
			categorySet[product.Category] = true
		}
	}

	// Create categories concurrently
	categoryMap := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent category creations
	semaphore := make(chan struct{}, 3) // Max 3 concurrent category creations

	for categoryName := range categorySet {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Get or create category
			categoryID, err := im.client.GetOrCreateCategory(ctx, name)
			if err != nil {
				// Log error but continue - we'll handle missing categories later
				return
			}

			mu.Lock()
			categoryMap[name] = categoryID
			mu.Unlock()
		}(categoryName)
	}

	wg.Wait()

	return categoryMap, nil
}

// importProducts imports products with concurrent worker pool
func (im *Importer) importProducts(ctx context.Context, products []generator.EnrichedProduct, categoryMap map[string]string) (int, map[string]string, []ImportError) {
	// Create channels
	jobChan := make(chan importJob, im.config.BatchSize)
	resultChan := make(chan importJobResult, im.config.BatchSize)

	// Track product IDs for specifications
	productIDMap := make(map[string]string)
	var productIDMu sync.Mutex

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < im.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			im.importWorker(ctx, jobChan, resultChan, categoryMap)
		}()
	}

	// Start result collector
	var collectorWg sync.WaitGroup
	errors := make([]ImportError, 0)
	importedCount := 0

	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for result := range resultChan {
			if result.Error != nil {
				errors = append(errors, ImportError{
					ProductSKU: result.SKU,
					Category:   result.Category,
					Message:    result.Error.Error(),
					Retry:      result.Attempt,
				})
			} else {
				importedCount++
				// Track product ID for specifications
				productIDMu.Lock()
				productIDMap[result.SKU] = result.ProductID
				productIDMu.Unlock()
			}
		}
	}()

	// Send jobs to workers
	go func() {
		for i, product := range products {
			select {
			case <-ctx.Done():
				close(jobChan)
				return
			case jobChan <- importJob{
				Product: product,
				Index:   i,
			}:
			}
		}
		close(jobChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Wait for result collector
	collectorWg.Wait()

	return importedCount, productIDMap, errors
}

// importWorker processes import jobs
func (im *Importer) importWorker(ctx context.Context, jobs <-chan importJob, results chan<- importJobResult, categoryMap map[string]string) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get category ID
		var categoryID *string
		if job.Product.Category != "" {
			if id, exists := categoryMap[job.Product.Category]; exists {
				categoryID = &id
			}
		}

		// Map to Convex format
		convexProduct := MapToConvexProduct(job.Product, categoryID, im.mapper)

		// Validate product
		if err := ValidateConvexProduct(convexProduct); err != nil {
			results <- importJobResult{
				SKU:      job.Product.SKU,
				Category: job.Product.Category,
				Error:    fmt.Errorf("validation failed: %w", err),
			}
			continue
		}

		// Import product with retry
		productID, err := im.client.CreateProduct(ctx, convexProduct)
		if err != nil {
			results <- importJobResult{
				SKU:      job.Product.SKU,
				Category: job.Product.Category,
				Error:    err,
				Attempt:  im.config.RetryAttempts,
			}
			continue
		}

		results <- importJobResult{
			SKU:       job.Product.SKU,
			Category:  job.Product.Category,
			ProductID: productID,
		}
	}
}

// importSpecifications imports product specifications
func (im *Importer) importSpecifications(ctx context.Context, products []generator.EnrichedProduct, productIDMap map[string]string) (int, []ImportError) {
	errors := make([]ImportError, 0)
	importedCount := 0

	// Use semaphore to limit concurrent spec imports
	semaphore := make(chan struct{}, im.config.MaxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, product := range products {
		// Skip products without specifications or products that weren't imported
		productID, exists := productIDMap[product.SKU]
		if !exists || len(product.Specifications) == 0 {
			continue
		}

		// Map specifications
		specs := MapToProductSpecifications(product, productID)
		if len(specs) == 0 {
			continue
		}

		// Import each specification category
		for _, spec := range specs {
			wg.Add(1)
			go func(s ConvexProductSpecification, sku string) {
				defer wg.Done()

				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				_, err := im.client.CreateProductSpecification(ctx, s)
				mu.Lock()
				if err != nil {
					errors = append(errors, ImportError{
						ProductSKU: sku,
						Category:   s.Category,
						Message:    fmt.Sprintf("spec import failed: %v", err),
					})
				} else {
					importedCount++
				}
				mu.Unlock()
			}(spec, product.SKU)
		}
	}

	wg.Wait()

	return importedCount, errors
}

// Ping tests the Convex connection
func (im *Importer) Ping(ctx context.Context) error {
	return im.client.Ping(ctx)
}

// Close closes the importer and releases resources
func (im *Importer) Close() error {
	return im.client.Close()
}

// importJob represents a product import job
type importJob struct {
	Product generator.EnrichedProduct
	Index   int
}

// importJobResult represents the result of an import job
type importJobResult struct {
	SKU       string
	Category  string
	ProductID string
	Error     error
	Attempt   int
}

// LoadEnrichedData loads enriched products from JSON file
func LoadEnrichedData(path string) (*generator.EnrichmentResult, error) {
	// This is a placeholder - we'll use the existing loader
	// The actual implementation will reuse generator.LoadEnrichedOutput
	return nil, fmt.Errorf("not implemented - use generator.LoadEnrichedOutput")
}
