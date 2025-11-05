package convex

import (
	"fmt"
	"strings"
	"time"

	"github.com/geraldbahati/image-sourcer/internal/generator"
)

// MapperConfig holds mapping configuration
type MapperConfig struct {
	DefaultCurrency       string
	DefaultCurrencySymbol string
	MarkNewArrivals       bool
}

// DefaultMapperConfig returns default mapper configuration
func DefaultMapperConfig() MapperConfig {
	return MapperConfig{
		DefaultCurrency:       "KES",
		DefaultCurrencySymbol: "KSh",
		MarkNewArrivals:       true,
	}
}

// MapToConvexProduct converts EnrichedProduct to ConvexProduct matching your mutation schema
func MapToConvexProduct(enriched generator.EnrichedProduct, categoryID *string, config MapperConfig) ConvexProduct {
	// Ensure we have a valid base price (Convex requires non-zero)
	basePrice := enriched.BasePriceKes
	if basePrice == 0 {
		basePrice = 10.0 // Default minimum price
	}

	// Calculate pricing fields with defaults
	comparePrice := enriched.ComparePriceKes
	costPrice := float64(0) // Default cost price to 0

	// Calculate discounts if compare price exists
	var discountAmount, discountPercentage *float64
	if comparePrice > 0 && comparePrice > basePrice {
		discount := comparePrice - basePrice
		discountAmount = &discount
		percentage := (discount / comparePrice) * 100
		discountPercentage = &percentage
	}

	// Calculate profit if cost price exists and is greater than 0
	var profitAmount, profitMargin, markupPercentage *float64
	if costPrice > 0 && basePrice > costPrice {
		profit := basePrice - costPrice
		profitAmount = &profit
		margin := (profit / basePrice) * 100
		profitMargin = &margin
		markup := (profit / costPrice) * 100
		markupPercentage = &markup
	}

	// Generate availableSizes (empty array if no sizes)
	availableSizes := mapSizeOptions(enriched.AvailableSizes)
	if availableSizes == nil {
		availableSizes = []string{} // Empty array instead of nil
	}

	// Create base product with ALL required fields
	product := ConvexProduct{
		// Basic Information (REQUIRED)
		Name:        enriched.Name,
		Slug:        enriched.Slug,
		Description: enriched.Description,
		SKU:         enriched.SKU,
		Barcode:     stringPtr(enriched.Barcode),

		// Pricing Structure - KES Input (REQUIRED)
		BasePrice:    basePrice,
		ComparePrice: float64PtrOrZero(comparePrice), // Send 0 if not available
		CostPrice:    float64PtrOrZero(costPrice),    // Send 0 if not available

		// USD Pricing (optional - Convex will calculate)
		BasePriceUsd:    float64PtrOrZero(enriched.BasePriceUsd),
		ComparePriceUsd: float64PtrOrZero(enriched.ComparePriceUsd),
		CostPriceUsd:    float64PtrOrZero(0), // Send 0 for Convex to calculate

		// Tax Configuration (optional - use defaults)
		TaxIncluded: boolPtr(false),
		TaxClass:    stringPtr("standard"),
		TaxRate:     float64Ptr(16.0), // Default VAT 16%

		// Profit & Markup Calculations
		ProfitMargin:     profitMargin,
		ProfitAmount:     profitAmount,
		MarkupPercentage: markupPercentage,

		// Discount Configuration
		DiscountAmount:     discountAmount,
		DiscountPercentage: discountPercentage,

		// Currency Settings
		Currency:       stringPtr("KES"),
		CurrencySymbol: stringPtr("KSh"),

		// Sale & Deal Configuration
		IsOnSale: boolPtr(comparePrice > 0 && comparePrice > basePrice),

		// Legacy pricing fields
		Price:         float64Ptr(basePrice),
		OriginalPrice: float64Ptr(comparePrice),
		Discount:      discountPercentage,

		// Images (REQUIRED: image)
		Image:     getImageOrPlaceholder(enriched.PrimaryImage),
		Images:    enriched.Images,
		Thumbnail: stringPtr(enriched.ThumbnailURL),

		// Category
		CategoryID: categoryID,
		Category:   stringPtr(enriched.Category),

		// Product Attributes
		Tags:         enriched.Tags,
		Badges:       enriched.Badges,
		IsNew:        boolPtr(enriched.IsNew),
		Featured:     boolPtr(false), // Default
		IsBestSeller: boolPtr(false), // Default
		IsDailyDeal:  boolPtr(false), // Default
		IsNewArrival: boolPtr(config.MarkNewArrivals && enriched.IsNew),

		// Deal expiration (optional)
		DealExpiresAt: nil,

		// Inventory & Stock (REQUIRED: isActive)
		Stock:    intPtr(enriched.Stock),
		IsActive: enriched.IsActive,

		// Physical Properties
		Weight:     nil, // Not available in enriched data
		Dimensions: nil, // Not available in enriched data

		// Variants
		AvailableSizes:  availableSizes, // Use pre-computed empty array
		AvailableColors: mapColorOptions(enriched.AvailableColors),
		Variants:        mapVariants(enriched.Variants),

		// Features
		Features: mapFeatures(enriched.Features),

		// Additional metadata
		Metadata: nil,

		// SEO Fields - ALL fields with defaults
		SEOTitle:             stringPtr(enriched.SEO.Title),
		SEODescription:       stringPtr(enriched.SEO.Description),
		SEOKeywords:          enriched.SEO.Keywords,
		MetaRobots:           stringPtr("index, follow"),
		CanonicalUrl:         stringPtrEmpty(""), // Empty string instead of nil
		OpenGraphTitle:       stringPtr(enriched.SEO.OpenGraphTitle),
		OpenGraphDescription: stringPtr(enriched.SEO.OpenGraphDesc),
		OpenGraphImage:       stringPtr(enriched.SEO.OpenGraphImage),
		OpenGraphType:        stringPtr("product"),
		TwitterCard:          stringPtr(enriched.SEO.TwitterCard),
		TwitterTitle:         stringPtr(enriched.SEO.TwitterTitle),
		TwitterDescription:   stringPtr(enriched.SEO.TwitterDescription),
		TwitterImage:         stringPtr(enriched.SEO.TwitterImage),
		StructuredData:       make(map[string]interface{}), // Empty object
		SEOScore:             float64PtrOrZero(0),          // Default score 0
		FocusKeyword:         stringPtrEmpty(""),           // Empty string
		ReadabilityScore:     float64PtrOrZero(0),          // Default score 0

		// Specifications - included in same mutation
		Specifications: mapSpecificationsToInput(enriched.Specifications),
	}

	return product
}

// mapFeatures converts generator.ProductFeature to ConvexFeature
func mapFeatures(features []generator.ProductFeature) []ConvexFeature {
	if len(features) == 0 {
		return nil
	}

	convexFeatures := make([]ConvexFeature, len(features))
	for i, f := range features {
		convexFeatures[i] = ConvexFeature{
			Icon:        f.Icon,
			Title:       f.Title,
			Description: f.Description,
		}
	}

	return convexFeatures
}

// mapVariants converts generator.ProductVariant to ConvexVariant matching your schema
func mapVariants(variants []generator.ProductVariant) []ConvexVariant {
	if len(variants) == 0 {
		return nil
	}

	convexVariants := make([]ConvexVariant, len(variants))
	for i, v := range variants {
		variant := ConvexVariant{
			ID:     v.ID,
			Name:   v.Name,
			Type:   v.Type,
			SKU:    stringPtr(v.SKU),
			Stock:  intPtr(v.Stock),
			Images: v.Images,
		}

		// Map color
		if v.Color != nil {
			variant.Color = &ConvexColorOption{
				Name:  v.Color.Name,
				Value: v.Color.Value,
			}
		}

		// Map size
		if v.Size != nil {
			variant.Size = stringPtr(v.Size.Name)
		}

		// Map pricing - KES prices (required), USD prices (optional)
		if v.PriceKes > 0 {
			variant.Price = float64Ptr(v.PriceKes)     // KES price input
			variant.PriceUsd = float64Ptr(v.PriceUsd) // USD price (calculated)

			// For variants, use original price from enrichment if available
			// (variants don't have separate OriginalPriceKes field)
			variant.OriginalPrice = nil           // No original price for variants by default
			variant.OriginalPriceUsd = nil
		}

		convexVariants[i] = variant
	}

	return convexVariants
}

// mapColorOptions converts generator.ColorOption to ConvexColorOption
func mapColorOptions(colors []generator.ColorOption) []ConvexColorOption {
	if len(colors) == 0 {
		return nil
	}

	convexColors := make([]ConvexColorOption, len(colors))
	for i, c := range colors {
		convexColors[i] = ConvexColorOption{
			Name:  c.Name,
			Value: c.Value,
		}
	}

	return convexColors
}

// mapSizeOptions converts generator.SizeOption to string array
func mapSizeOptions(sizes []generator.SizeOption) []string {
	if len(sizes) == 0 {
		return nil
	}

	sizeNames := make([]string, len(sizes))
	for i, s := range sizes {
		sizeNames[i] = s.Name
	}

	return sizeNames
}

// mapSpecificationsToInput converts enriched specifications to input format for mutation
func mapSpecificationsToInput(specifications map[string][]generator.Specification) []ConvexProductSpecificationInput {
	if len(specifications) == 0 {
		return nil
	}

	specs := make([]ConvexProductSpecificationInput, 0, len(specifications))
	order := 0

	for category, specList := range specifications {
		if len(specList) == 0 {
			continue
		}

		// Generate category ID
		categoryID := fmt.Sprintf("spec-cat-%s-%d", generateSlugSimple(category), order)

		// Convert spec list with IDs
		convexSpecs := make([]ConvexSpecificationInput, len(specList))
		for i, spec := range specList {
			// Generate spec ID
			specID := fmt.Sprintf("spec-%s-%d", generateSlugSimple(spec.Name), i)

			convexSpecs[i] = ConvexSpecificationInput{
				ID:    specID,
				Name:  spec.Name,
				Value: spec.Value,
				Order: intPtr(spec.Order),
			}
		}

		specs = append(specs, ConvexProductSpecificationInput{
			ID:       categoryID,
			Category: category,
			Specs:    convexSpecs,
			Order:    intPtr(order),
			IsActive: true,
		})

		order++
	}

	return specs
}

// MapToProductSpecifications converts enriched specifications to Convex format (legacy - for separate API calls)
func MapToProductSpecifications(enriched generator.EnrichedProduct, productID string) []ConvexProductSpecification {
	if len(enriched.Specifications) == 0 {
		return nil
	}

	now := float64(time.Now().UnixMilli())
	specs := make([]ConvexProductSpecification, 0, len(enriched.Specifications))

	order := 0
	for category, specList := range enriched.Specifications {
		if len(specList) == 0 {
			continue
		}

		// Convert spec list
		convexSpecs := make([]ConvexSpecification, len(specList))
		for i, spec := range specList {
			convexSpecs[i] = ConvexSpecification{
				Name:  spec.Name,
				Value: spec.Value,
				Order: intPtr(spec.Order),
			}
		}

		specs = append(specs, ConvexProductSpecification{
			ProductID: productID,
			Category:  category,
			Specs:     convexSpecs,
			Order:     intPtr(order),
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		})

		order++
	}

	return specs
}

// generateSearchField creates a concatenated search field
func generateSearchField(enriched generator.EnrichedProduct) string {
	parts := []string{
		enriched.Name,
		enriched.Description,
		enriched.Category,
		enriched.SKU,
	}

	// Add tags
	if len(enriched.Tags) > 0 {
		parts = append(parts, strings.Join(enriched.Tags, " "))
	}

	// Add variant names if applicable
	if len(enriched.Variants) > 0 {
		for _, v := range enriched.Variants {
			parts = append(parts, v.Name)
		}
	}

	return strings.Join(parts, " ")
}

// calculateDiscountPercentage calculates discount percentage
func calculateDiscountPercentage(basePrice, comparePrice float64) *float64 {
	if comparePrice == 0 || basePrice == 0 || comparePrice <= basePrice {
		return nil
	}

	discount := ((comparePrice - basePrice) / comparePrice) * 100
	return &discount
}

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func float64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// float64PtrOrZero always returns a pointer, even for zero values
func float64PtrOrZero(f float64) *float64 {
	return &f
}

// stringPtrEmpty returns empty string pointer instead of nil for empty strings
func stringPtrEmpty(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// getImageOrPlaceholder returns the image URL or a placeholder if empty
func getImageOrPlaceholder(image string) string {
	if image == "" {
		// Use a transparent placeholder image URL
		return "https://via.placeholder.com/800x800.png?text=No+Image"
	}
	return image
}

// generateSlugSimple creates a simple slug from a string
func generateSlugSimple(s string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove any character that's not alphanumeric or hyphen
	result := ""
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result += string(ch)
		}
	}
	// Remove consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	// Trim hyphens from ends
	result = strings.Trim(result, "-")
	return result
}

// ValidateConvexProduct validates a Convex product before import
func ValidateConvexProduct(product ConvexProduct) error {
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	if product.Slug == "" {
		return fmt.Errorf("product slug is required")
	}

	if product.SKU == "" {
		return fmt.Errorf("product SKU is required")
	}

	if product.Image == "" {
		return fmt.Errorf("product image is required")
	}

	if product.BasePrice < 0 {
		return fmt.Errorf("product basePrice cannot be negative")
	}

	// SearchField is not in the mutation schema, removed

	return nil
}
