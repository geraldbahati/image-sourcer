package generator

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// VariantDetector handles variant detection and grouping
type VariantDetector struct {
	minGroupSize int // Minimum products to consider as variants
}

// NewVariantDetector creates a new variant detector
func NewVariantDetector() *VariantDetector {
	return &VariantDetector{
		minGroupSize: 2, // At least 2 products to be variants
	}
}

// DetectVariants analyzes products and groups them into parent + variants
func (vd *VariantDetector) DetectVariants(ctx context.Context, products []EnrichedProduct) ([]EnrichedProduct, error) {
	// Step 1: Group products by base name and category
	groups := vd.groupByBaseName(products)

	// Step 2: Detect which groups are actual variants
	variantGroups := vd.filterVariantGroups(groups)

	// Step 3: Create parent products with variants
	parentProducts := vd.createParentProducts(variantGroups)

	// Step 4: Merge with non-variant products
	result := vd.mergeProducts(products, variantGroups, parentProducts)

	return result, nil
}

// groupByBaseName groups products by their base name (without color/size)
func (vd *VariantDetector) groupByBaseName(products []EnrichedProduct) map[string][]EnrichedProduct {
	groups := make(map[string][]EnrichedProduct)

	for _, product := range products {
		// Extract base name
		baseName := GetBaseProductName(product.Name)

		// Create group key: baseName + category (products in different categories aren't variants)
		groupKey := fmt.Sprintf("%s|%s", strings.ToLower(baseName), strings.ToLower(product.Category))

		groups[groupKey] = append(groups[groupKey], product)
	}

	return groups
}

// filterVariantGroups filters groups to find actual variant groups
func (vd *VariantDetector) filterVariantGroups(groups map[string][]EnrichedProduct) map[string][]EnrichedProduct {
	variantGroups := make(map[string][]EnrichedProduct)

	for key, group := range groups {
		// Skip if group is too small
		if len(group) < vd.minGroupSize {
			continue
		}

		// Check if this group is actually variants
		if vd.isVariantGroup(group) {
			variantGroups[key] = group
		}
	}

	return variantGroups
}

// isVariantGroup determines if a group of products are variants of each other
func (vd *VariantDetector) isVariantGroup(products []EnrichedProduct) bool {
	if len(products) < 2 {
		return false
	}

	// Check 1: At least one product must have color or size
	hasColorOrSize := false
	for _, p := range products {
		if HasColor(p.Name) || HasSize(p.Name) {
			hasColorOrSize = true
			break
		}
	}

	if !hasColorOrSize {
		return false
	}

	// Check 2: Products should have similar SKU patterns (optional but strong indicator)
	// Example: CE410A, CE411A, CE412A (sequential)
	if vd.hasSimilarSKUs(products) {
		return true
	}

	// Check 3: Products have different colors/sizes
	uniqueColors := make(map[string]bool)
	uniqueSizes := make(map[string]bool)

	for _, p := range products {
		if color := ExtractColor(p.Name); color != nil {
			uniqueColors[color.Name] = true
		}
		if size := ExtractSize(p.Name); size != nil {
			uniqueSizes[size.Name] = true
		}
	}

	// If we have multiple different colors or sizes, it's likely variants
	return len(uniqueColors) >= 2 || len(uniqueSizes) >= 2
}

// hasSimilarSKUs checks if SKUs follow a sequential pattern
func (vd *VariantDetector) hasSimilarSKUs(products []EnrichedProduct) bool {
	if len(products) < 2 {
		return false
	}

	// Extract SKU prefixes and suffixes
	// Example: CE410A -> prefix: CE, middle: 410, suffix: A
	type skuParts struct {
		prefix string
		number int
		suffix string
	}

	// Regex to extract SKU pattern: (prefix)(number)(suffix)
	skuPattern := regexp.MustCompile(`^([A-Z]+)(\d+)([A-Z]*)$`)

	parts := make([]skuParts, 0, len(products))
	for _, p := range products {
		matches := skuPattern.FindStringSubmatch(p.SKU)
		if len(matches) >= 3 {
			var num int
			fmt.Sscanf(matches[2], "%d", &num)
			parts = append(parts, skuParts{
				prefix: matches[1],
				number: num,
				suffix: matches[3],
			})
		}
	}

	// Check if all parts have same prefix and suffix
	if len(parts) < 2 {
		return false
	}

	firstPrefix := parts[0].prefix
	firstSuffix := parts[0].suffix

	for _, p := range parts[1:] {
		if p.prefix != firstPrefix || p.suffix != firstSuffix {
			return false
		}
	}

	// Check if numbers are sequential or close
	numbers := make([]int, len(parts))
	for i, p := range parts {
		numbers[i] = p.number
	}

	sort.Ints(numbers)

	// Check if numbers are within a reasonable range (max 20 apart)
	maxDiff := numbers[len(numbers)-1] - numbers[0]
	return maxDiff <= 20
}

// createParentProducts creates parent products with variants
func (vd *VariantDetector) createParentProducts(variantGroups map[string][]EnrichedProduct) map[string]EnrichedProduct {
	parentProducts := make(map[string]EnrichedProduct)

	// Use a worker pool for concurrent processing
	var wg sync.WaitGroup
	resultChan := make(chan struct {
		key    string
		parent EnrichedProduct
	}, len(variantGroups))

	// Process groups concurrently
	semaphore := make(chan struct{}, 8) // Max 8 concurrent workers

	for key, group := range variantGroups {
		wg.Add(1)
		go func(k string, g []EnrichedProduct) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			parent := vd.createParentFromGroup(g)
			resultChan <- struct {
				key    string
				parent EnrichedProduct
			}{k, parent}
		}(key, group)
	}

	// Close channel when all workers done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		parentProducts[result.key] = result.parent
	}

	return parentProducts
}

// createParentFromGroup creates a parent product from a group of variants
func (vd *VariantDetector) createParentFromGroup(products []EnrichedProduct) EnrichedProduct {
	if len(products) == 0 {
		return EnrichedProduct{}
	}

	// Use the first product as base
	parent := products[0]

	// Set base name (without color/size)
	baseName := GetBaseProductName(parent.Name)
	parent.Name = baseName

	// Generate parent SKU from base name
	parent.SKU = generateParentSKU(baseName, parent.Category)
	parent.Slug = GenerateSlug(baseName, parent.SKU)

	// Mark as variant product
	parent.IsVariantProduct = true

	// Initialize variants array
	parent.Variants = make([]ProductVariant, 0, len(products))

	// Track available colors and sizes
	colorMap := make(map[string]ColorOption)
	sizeMap := make(map[string]SizeOption)

	// Find the lowest price for parent
	lowestPriceUsd := parent.BasePriceUsd
	lowestPriceKes := parent.BasePriceKes

	// Create variants from all products in group
	for _, product := range products {
		variant := vd.createVariant(product)
		parent.Variants = append(parent.Variants, variant)

		// Track colors
		if variant.Color != nil {
			colorMap[variant.Color.Name] = *variant.Color
		}

		// Track sizes
		if variant.Size != nil {
			sizeMap[variant.Size.Name] = *variant.Size
		}

		// Track lowest price
		if product.BasePriceUsd > 0 && (lowestPriceUsd == 0 || product.BasePriceUsd < lowestPriceUsd) {
			lowestPriceUsd = product.BasePriceUsd
			lowestPriceKes = product.BasePriceKes
		}
	}

	// Set parent price to lowest variant price
	parent.BasePriceUsd = lowestPriceUsd
	parent.BasePriceKes = lowestPriceKes

	// Convert maps to slices
	parent.AvailableColors = make([]ColorOption, 0, len(colorMap))
	for _, color := range colorMap {
		parent.AvailableColors = append(parent.AvailableColors, color)
	}

	parent.AvailableSizes = make([]SizeOption, 0, len(sizeMap))
	for _, size := range sizeMap {
		parent.AvailableSizes = append(parent.AvailableSizes, size)
	}

	// Sort for consistent output
	sort.Slice(parent.AvailableColors, func(i, j int) bool {
		return parent.AvailableColors[i].Name < parent.AvailableColors[j].Name
	})

	sort.Slice(parent.AvailableSizes, func(i, j int) bool {
		return parent.AvailableSizes[i].Name < parent.AvailableSizes[j].Name
	})

	return parent
}

// createVariant creates a ProductVariant from an EnrichedProduct
func (vd *VariantDetector) createVariant(product EnrichedProduct) ProductVariant {
	variant := ProductVariant{
		ID:          product.SKU,
		Name:        product.Name,
		Type:        GetVariantType(product.Name),
		Color:       ExtractColor(product.Name),
		Size:        ExtractSize(product.Name),
		SKU:         product.SKU,
		PriceUsd:    product.BasePriceUsd,
		PriceKes:    product.BasePriceKes,
		Stock:       product.Stock,
		Images:      product.Images,
		IsAvailable: product.Stock > 0 || product.IsActive,
	}

	return variant
}

// mergeProducts merges parent products with non-variant products
func (vd *VariantDetector) mergeProducts(originalProducts []EnrichedProduct, variantGroups map[string][]EnrichedProduct, parentProducts map[string]EnrichedProduct) []EnrichedProduct {
	// Create a set of SKUs that are part of variant groups
	variantSKUs := make(map[string]bool)
	for _, group := range variantGroups {
		for _, product := range group {
			variantSKUs[product.SKU] = true
		}
	}

	// Add parent products
	result := make([]EnrichedProduct, 0, len(originalProducts))
	for _, parent := range parentProducts {
		result = append(result, parent)
	}

	// Add non-variant products
	for _, product := range originalProducts {
		if !variantSKUs[product.SKU] {
			result = append(result, product)
		}
	}

	return result
}

// generateParentSKU generates a SKU for the parent product
func generateParentSKU(baseName, category string) string {
	// Create a clean SKU from base name and category
	// Example: "HP 305A LaserJet Toner" + "Toners & Cartridges" -> "HP-305A-TONER"

	// Extract key words
	words := strings.Fields(baseName)
	if len(words) == 0 {
		return "PRODUCT"
	}

	// Take first 3 significant words
	var skuParts []string
	for _, word := range words {
		cleaned := strings.ToUpper(regexp.MustCompile(`[^A-Z0-9]+`).ReplaceAllString(strings.ToUpper(word), ""))
		if cleaned != "" && len(cleaned) > 1 {
			skuParts = append(skuParts, cleaned)
			if len(skuParts) >= 3 {
				break
			}
		}
	}

	if len(skuParts) == 0 {
		return "PRODUCT"
	}

	return strings.Join(skuParts, "-")
}
