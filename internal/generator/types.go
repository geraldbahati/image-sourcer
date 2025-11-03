package generator

import (
	"time"
)

// EnrichedProduct represents a fully enriched product ready for Convex
type EnrichedProduct struct {
	// Basic Info
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"` // URL-friendly
	Category string  `json:"category"`

	// Description
	Description string `json:"description"`

	// Pricing (USD - Source of Truth)
	BasePriceUsd    float64 `json:"basePriceUsd"`
	ComparePriceUsd float64 `json:"comparePriceUsd,omitempty"`

	// Pricing (KES - Calculated)
	BasePriceKes    float64 `json:"basePriceKes"`
	ComparePriceKes float64 `json:"comparePriceKes,omitempty"`

	// Stock
	Stock    int    `json:"stock"`
	IsActive bool   `json:"isActive"`
	Barcode  string `json:"barcode,omitempty"`

	// Images
	PrimaryImage  string   `json:"image"`
	Images        []string `json:"images,omitempty"`
	ThumbnailURL  string   `json:"thumbnail,omitempty"`

	// Product Attributes
	Tags     []string `json:"tags,omitempty"`
	Badges   []string `json:"badges,omitempty"`
	IsNew    bool     `json:"isNew"`
	Featured bool     `json:"featured"`

	// Variants (for products with color/size variations)
	IsVariantProduct bool              `json:"isVariantProduct"`
	Variants         []ProductVariant  `json:"variants,omitempty"`
	AvailableColors  []ColorOption     `json:"availableColors,omitempty"`
	AvailableSizes   []SizeOption      `json:"availableSizes,omitempty"`

	// Features (for product detail page)
	Features []ProductFeature `json:"features,omitempty"`

	// SEO
	SEO SEOMetadata `json:"seo"`

	// Specifications
	Specifications map[string][]Specification `json:"specifications,omitempty"`

	// Metadata
	Source SourceMetadata `json:"source"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProductVariant represents a color/size variant of a product
type ProductVariant struct {
	ID          string      `json:"id"`           // Variant SKU
	Name        string      `json:"name"`         // e.g., "HP 305A Black"
	Type        string      `json:"type"`         // "color", "size", or "color-size"
	Color       *ColorOption `json:"color,omitempty"`
	Size        *SizeOption  `json:"size,omitempty"`
	SKU         string      `json:"sku"`
	PriceUsd    float64     `json:"priceUsd"`
	PriceKes    float64     `json:"priceKes"`
	Stock       int         `json:"stock"`
	Images      []string    `json:"images,omitempty"`
	IsAvailable bool        `json:"isAvailable"`
}

// ColorOption represents an available color
type ColorOption struct {
	Name  string `json:"name"`  // e.g., "Black", "Cyan"
	Value string `json:"value"` // Hex color code, e.g., "#000000"
}

// SizeOption represents an available size
type SizeOption struct {
	Name  string `json:"name"`  // e.g., "128GB", "M"
	Value string `json:"value"` // Normalized value
}

// ProductFeature represents a feature with icon and description
type ProductFeature struct {
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SEOMetadata contains all SEO-related fields
type SEOMetadata struct {
	Title              string   `json:"seoTitle"`
	Description        string   `json:"seoDescription"`
	Keywords           []string `json:"seoKeywords,omitempty"`
	OpenGraphTitle     string   `json:"openGraphTitle,omitempty"`
	OpenGraphDesc      string   `json:"openGraphDescription,omitempty"`
	OpenGraphImage     string   `json:"openGraphImage,omitempty"`
	TwitterCard        string   `json:"twitterCard,omitempty"`
	TwitterTitle       string   `json:"twitterTitle,omitempty"`
	TwitterDescription string   `json:"twitterDescription,omitempty"`
	TwitterImage       string   `json:"twitterImage,omitempty"`
}

// Specification represents a single spec item
type Specification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Order int    `json:"order"`
}

// SourceMetadata tracks where the product came from
type SourceMetadata struct {
	File      string `json:"file"`
	Sheet     string `json:"sheet"`
	RowNumber int    `json:"rowNumber"`
}

// EnrichmentConfig holds configuration for the enrichment process
type EnrichmentConfig struct {
	// Pricing
	ExchangeRate   float64 // USD to KES
	SmartRounding  bool    // Round KES prices to nearest 10/50/100

	// Description
	DescriptionStyle  string // "professional", "casual", "technical"
	MaxDescLength     int    // Maximum description length
	MinDescLength     int    // Minimum description length

	// SEO
	GenerateOGTags      bool // Generate Open Graph tags
	GenerateTwitterTags bool // Generate Twitter Card tags

	// Features
	DefaultFeatures []ProductFeature // Default features for all products

	// Variants
	DetectVariants bool // Auto-detect and group product variants

	// Performance
	MaxWorkers int // Concurrent enrichment workers
	BufferSize int // Channel buffer size
}

// DefaultEnrichmentConfig returns default configuration
func DefaultEnrichmentConfig() EnrichmentConfig {
	return EnrichmentConfig{
		ExchangeRate:        130.0, // USD to KES
		SmartRounding:       true,
		DescriptionStyle:    "professional",
		MaxDescLength:       500,
		MinDescLength:       100,
		GenerateOGTags:      true,
		GenerateTwitterTags: true,
		DetectVariants:      true, // Enable automatic variant detection
		DefaultFeatures: []ProductFeature{
			{
				Icon:        "Truck",
				Title:       "Fast Shipping",
				Description: "Ships within 24-48 hours",
			},
			{
				Icon:        "Shield",
				Title:       "Authentic Products",
				Description: "100% genuine products",
			},
			{
				Icon:        "RefreshCw",
				Title:       "Easy Returns",
				Description: "14-day return policy",
			},
		},
		MaxWorkers: 8,
		BufferSize: 100,
	}
}

// EnrichmentResult contains the results of enrichment
type EnrichmentResult struct {
	Products       []EnrichedProduct
	TotalProcessed int
	TotalEnriched  int
	Errors         []EnrichmentError
	Duration       time.Duration
}

// EnrichmentError represents an error during enrichment
type EnrichmentError struct {
	SKU     string
	Message string
}
