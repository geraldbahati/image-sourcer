package convex

import "time"

// ConvexProduct represents a product in the Convex schema
type ConvexProduct struct {
	// Basic Info
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	SKU         string  `json:"sku"`
	Barcode     *string `json:"barcode,omitempty"`

	// USD Pricing (Source of Truth)
	BasePriceUsd    float64  `json:"basePriceUsd"`
	ComparePriceUsd *float64 `json:"comparePriceUsd,omitempty"`
	CostPriceUsd    *float64 `json:"costPriceUsd,omitempty"`

	// Calculated KES Pricing
	BasePriceKes       *float64 `json:"basePriceKes,omitempty"`
	ComparePriceKes    *float64 `json:"comparePriceKes,omitempty"`
	CostPriceKes       *float64 `json:"costPriceKes,omitempty"`
	LastCurrencyUpdate *float64 `json:"lastCurrencyUpdate,omitempty"`

	// Legacy price fields (for compatibility)
	Price         float64  `json:"price"`         // Maps to basePriceKes
	OriginalPrice *float64 `json:"originalPrice,omitempty"` // Maps to comparePriceKes
	Discount      *float64 `json:"discount,omitempty"`

	// Images
	Image     string   `json:"image"`
	Images    []string `json:"images,omitempty"`
	Thumbnail *string  `json:"thumbnail,omitempty"`

	// Category
	CategoryID *string `json:"categoryId,omitempty"`
	Category   *string `json:"category,omitempty"`

	// Product Attributes
	Tags            []string `json:"tags,omitempty"`
	Badges          []string `json:"badges,omitempty"`
	IsNew           *bool    `json:"isNew,omitempty"`
	Featured        *bool    `json:"featured,omitempty"`
	IsBestSeller    *bool    `json:"isBestSeller,omitempty"`
	IsDailyDeal     *bool    `json:"isDailyDeal,omitempty"`
	IsNewArrival    *bool    `json:"isNewArrival,omitempty"`
	DealExpiresAt   *float64 `json:"dealExpiresAt,omitempty"`

	// Inventory
	Stock    *int  `json:"stock,omitempty"`
	IsActive bool  `json:"isActive"`

	// Variants
	Variants         []ConvexVariant     `json:"variants,omitempty"`
	AvailableColors  []ConvexColorOption `json:"availableColors,omitempty"`
	AvailableSizes   []string            `json:"availableSizes,omitempty"`

	// Features
	Features []ConvexFeature `json:"features,omitempty"`

	// SEO
	SEOTitle              *string  `json:"seoTitle,omitempty"`
	SEODescription        *string  `json:"seoDescription,omitempty"`
	SEOKeywords           []string `json:"seoKeywords,omitempty"`
	OpenGraphTitle        *string  `json:"openGraphTitle,omitempty"`
	OpenGraphDescription  *string  `json:"openGraphDescription,omitempty"`
	OpenGraphImage        *string  `json:"openGraphImage,omitempty"`
	TwitterCard           *string  `json:"twitterCard,omitempty"`
	TwitterTitle          *string  `json:"twitterTitle,omitempty"`
	TwitterDescription    *string  `json:"twitterDescription,omitempty"`
	TwitterImage          *string  `json:"twitterImage,omitempty"`

	// Search field
	SearchField string `json:"searchField"`

	// Timestamps
	CreatedAt float64 `json:"createdAt"`
	UpdatedAt float64 `json:"updatedAt"`
}

// ConvexVariant represents a product variant in Convex schema
type ConvexVariant struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Type            string              `json:"type"` // "color", "size", "color-size"
	Color           *ConvexColorOption  `json:"color,omitempty"`
	Size            *string             `json:"size,omitempty"`
	SKU             *string             `json:"sku,omitempty"`
	PriceUsd        *float64            `json:"priceUsd,omitempty"`
	OriginalPriceUsd *float64           `json:"originalPriceUsd,omitempty"`
	PriceKes        *float64            `json:"priceKes,omitempty"`
	OriginalPriceKes *float64           `json:"originalPriceKes,omitempty"`
	Price           *float64            `json:"price,omitempty"` // Legacy - maps to priceKes
	OriginalPrice   *float64            `json:"originalPrice,omitempty"` // Legacy - maps to originalPriceKes
	Stock           *int                `json:"stock,omitempty"`
	Images          []string            `json:"images,omitempty"`
}

// ConvexColorOption represents a color option
type ConvexColorOption struct {
	Name   string   `json:"name"`
	Value  string   `json:"value"` // Hex color
	Images []string `json:"images,omitempty"`
}

// ConvexFeature represents a product feature
type ConvexFeature struct {
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ConvexProductSpecification represents specifications in Convex
type ConvexProductSpecification struct {
	ProductID string                `json:"productId"`
	Category  string                `json:"category"`
	Specs     []ConvexSpecification `json:"specs"`
	Order     *int                  `json:"order,omitempty"`
	IsActive  bool                  `json:"isActive"`
	CreatedAt float64               `json:"createdAt"`
	UpdatedAt float64               `json:"updatedAt"`
}

// ConvexSpecification represents a single spec
type ConvexSpecification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Order *int   `json:"order,omitempty"`
}

// ConvexCategory represents a category
type ConvexCategory struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	Image       *string `json:"image,omitempty"`
	ParentID    *string `json:"parentId,omitempty"`
	IsActive    bool    `json:"isActive"`
	Order       *int    `json:"order,omitempty"`
	CreatedAt   float64 `json:"createdAt"`
	UpdatedAt   float64 `json:"updatedAt"`
}

// ConvexMutationRequest represents a Convex mutation request
type ConvexMutationRequest struct {
	Path string                 `json:"path"`
	Args map[string]interface{} `json:"args"`
}

// ConvexMutationResponse represents a Convex mutation response
type ConvexMutationResponse struct {
	Value interface{}            `json:"value,omitempty"`
	Error *ConvexError           `json:"error,omitempty"`
}

// ConvexError represents a Convex error
type ConvexError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ImportConfig holds configuration for import
type ImportConfig struct {
	ConvexURL      string
	ConvexAPIKey   string
	BatchSize      int           // Products per batch
	MaxWorkers     int           // Concurrent workers
	RetryAttempts  int           // Retry attempts per request
	RetryDelay     time.Duration // Initial retry delay
	RequestTimeout time.Duration // HTTP request timeout
}

// DefaultImportConfig returns default configuration
func DefaultImportConfig() ImportConfig {
	return ImportConfig{
		BatchSize:      10,  // 10 products per batch
		MaxWorkers:     5,   // 5 concurrent workers
		RetryAttempts:  3,   // 3 retry attempts
		RetryDelay:     time.Second * 2,
		RequestTimeout: time.Second * 30,
	}
}

// ImportResult contains import results
type ImportResult struct {
	TotalProducts       int
	ImportedProducts    int
	ImportedCategories  int
	ImportedSpecs       int
	Errors              []ImportError
	Duration            time.Duration
	CategoryIDMap       map[string]string // Maps category name to Convex ID
}

// ImportError represents an import error
type ImportError struct {
	ProductSKU string
	Category   string
	Message    string
	Retry      int
}

// ImportProgress tracks import progress
type ImportProgress struct {
	TotalProducts      int
	ProcessedProducts  int
	ImportedProducts   int
	FailedProducts     int
	ProcessedCategories int
}
