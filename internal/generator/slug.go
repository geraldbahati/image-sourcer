package generator

import (
	"fmt"
	"strings"

	"github.com/gosimple/slug"
)

// GenerateSlug creates a URL-friendly slug from product name and SKU
func GenerateSlug(name string, sku string) string {
	// Clean the name
	cleaned := strings.TrimSpace(name)

	// Remove URLs if present (some products have URLs as names)
	if strings.HasPrefix(cleaned, "http") {
		// Use SKU as base for slug if name is a URL
		cleaned = sku
	}

	// Generate slug
	baseSlug := slug.Make(cleaned)

	// If slug is empty or too short, use SKU
	if baseSlug == "" || len(baseSlug) < 3 {
		baseSlug = slug.Make(sku)
	}

	// Ensure uniqueness by appending SKU if needed
	// This prevents collisions for products with similar names
	if !strings.Contains(baseSlug, strings.ToLower(sku)) {
		baseSlug = fmt.Sprintf("%s-%s", baseSlug, slug.Make(sku))
	}

	return baseSlug
}

// GenerateSlugWithCounter creates a slug with a counter for duplicates
func GenerateSlugWithCounter(name string, sku string, counter int) string {
	baseSlug := GenerateSlug(name, sku)
	if counter > 0 {
		return fmt.Sprintf("%s-%d", baseSlug, counter)
	}
	return baseSlug
}
