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

// MapToConvexProduct converts EnrichedProduct to ConvexProduct
func MapToConvexProduct(enriched generator.EnrichedProduct, categoryID *string, config MapperConfig) ConvexProduct {
	now := float64(time.Now().UnixMilli())

	// Create base product
	product := ConvexProduct{
		// Basic Info
		Name:        enriched.Name,
		Slug:        enriched.Slug,
		Description: enriched.Description,
		SKU:         enriched.SKU,
		Barcode:     stringPtr(enriched.Barcode),

		// USD Pricing (Source of Truth)
		BasePriceUsd:    enriched.BasePriceUsd,
		ComparePriceUsd: float64Ptr(enriched.ComparePriceUsd),

		// Calculated KES Pricing
		BasePriceKes:       float64Ptr(enriched.BasePriceKes),
		ComparePriceKes:    float64Ptr(enriched.ComparePriceKes),
		LastCurrencyUpdate: float64Ptr(now),

		// Legacy price fields (map to KES)
		Price:         enriched.BasePriceKes, // Legacy field
		OriginalPrice: float64Ptr(enriched.ComparePriceKes),
		Discount:      calculateDiscountPercentage(enriched.BasePriceKes, enriched.ComparePriceKes),

		// Images - use placeholder if no primary image
		Image:     getImageOrPlaceholder(enriched.PrimaryImage),
		Images:    enriched.Images,
		Thumbnail: stringPtr(enriched.ThumbnailURL),

		// Category
		CategoryID: categoryID,
		Category:   stringPtr(enriched.Category),

		// Product Attributes
		Tags:   enriched.Tags,
		Badges: enriched.Badges,
		IsNew:  boolPtr(enriched.IsNew),

		// Mark products without images as new arrivals
		IsNewArrival: boolPtr(config.MarkNewArrivals && enriched.IsNew),

		// Inventory
		Stock:    intPtr(enriched.Stock),
		IsActive: enriched.IsActive,

		// Features
		Features: mapFeatures(enriched.Features),

		// SEO
		SEOTitle:              stringPtr(enriched.SEO.Title),
		SEODescription:        stringPtr(enriched.SEO.Description),
		SEOKeywords:           enriched.SEO.Keywords,
		OpenGraphTitle:        stringPtr(enriched.SEO.OpenGraphTitle),
		OpenGraphDescription:  stringPtr(enriched.SEO.OpenGraphDesc),
		OpenGraphImage:        stringPtr(enriched.SEO.OpenGraphImage),
		TwitterCard:           stringPtr(enriched.SEO.TwitterCard),
		TwitterTitle:          stringPtr(enriched.SEO.TwitterTitle),
		TwitterDescription:    stringPtr(enriched.SEO.TwitterDescription),
		TwitterImage:          stringPtr(enriched.SEO.TwitterImage),

		// Search field
		SearchField: generateSearchField(enriched),

		// Timestamps
		CreatedAt: float64(enriched.CreatedAt.UnixMilli()),
		UpdatedAt: float64(enriched.UpdatedAt.UnixMilli()),
	}

	// Map variants if product has variants
	if enriched.IsVariantProduct && len(enriched.Variants) > 0 {
		product.Variants = mapVariants(enriched.Variants)
		product.AvailableColors = mapColorOptions(enriched.AvailableColors)
		product.AvailableSizes = mapSizeOptions(enriched.AvailableSizes)
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

// mapVariants converts generator.ProductVariant to ConvexVariant
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

		// Map pricing
		if v.PriceUsd > 0 {
			variant.PriceUsd = float64Ptr(v.PriceUsd)
			variant.PriceKes = float64Ptr(v.PriceKes)
			variant.Price = float64Ptr(v.PriceKes) // Legacy field
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

// MapToProductSpecifications converts enriched specifications to Convex format
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

	// Image is not required - products without images will use placeholder
	// if product.Image == "" {
	// 	return fmt.Errorf("product image is required")
	// }

	if product.BasePriceUsd < 0 {
		return fmt.Errorf("product price cannot be negative")
	}

	if product.SearchField == "" {
		return fmt.Errorf("search field is required")
	}

	return nil
}
