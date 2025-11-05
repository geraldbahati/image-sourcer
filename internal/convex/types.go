package convex

import "time"

// ConvexProduct represents a product in the Convex schema - matches your mutation exactly
type ConvexProduct struct {
	// Basic Information (REQUIRED)
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SKU         string `json:"sku"`
	Barcode     *string `json:"barcode,omitempty"`

	// Pricing Structure - KES Input (REQUIRED: basePrice)
	BasePrice    float64  `json:"basePrice"` // Input price in KES
	ComparePrice *float64 `json:"comparePrice,omitempty"` // Compare price in KES
	CostPrice    *float64 `json:"costPrice,omitempty"` // Cost price in KES

	// USD Pricing (calculated internally by Convex)
	BasePriceUsd    *float64 `json:"basePriceUsd,omitempty"`
	ComparePriceUsd *float64 `json:"comparePriceUsd,omitempty"`
	CostPriceUsd    *float64 `json:"costPriceUsd,omitempty"`

	// Tax Configuration
	TaxIncluded *bool    `json:"taxIncluded,omitempty"`
	TaxClass    *string  `json:"taxClass,omitempty"`
	TaxRate     *float64 `json:"taxRate,omitempty"`

	// Profit & Markup Calculations
	ProfitMargin     *float64 `json:"profitMargin,omitempty"`
	ProfitAmount     *float64 `json:"profitAmount,omitempty"`
	MarkupPercentage *float64 `json:"markupPercentage,omitempty"`

	// Discount Configuration
	DiscountAmount     *float64 `json:"discountAmount,omitempty"`
	DiscountPercentage *float64 `json:"discountPercentage,omitempty"`

	// Currency Settings
	Currency       *string `json:"currency,omitempty"`
	CurrencySymbol *string `json:"currencySymbol,omitempty"`

	// Sale & Deal Configuration
	IsOnSale *bool `json:"isOnSale,omitempty"`

	// Legacy pricing fields for backward compatibility
	Price         *float64 `json:"price,omitempty"`
	OriginalPrice *float64 `json:"originalPrice,omitempty"`
	Discount      *float64 `json:"discount,omitempty"`

	// Images (REQUIRED: image)
	Image     string   `json:"image"`
	Images    []string `json:"images,omitempty"`
	Thumbnail *string  `json:"thumbnail,omitempty"`

	// Category
	CategoryID *string `json:"categoryId,omitempty"`
	Category   *string `json:"category,omitempty"`

	// Product Attributes
	Tags         []string `json:"tags,omitempty"`
	Badges       []string `json:"badges,omitempty"`
	IsNew        *bool    `json:"isNew,omitempty"`
	Featured     *bool    `json:"featured,omitempty"`
	IsBestSeller *bool    `json:"isBestSeller,omitempty"`
	IsDailyDeal  *bool    `json:"isDailyDeal,omitempty"`
	IsNewArrival *bool    `json:"isNewArrival,omitempty"`

	// Deal expiration
	DealExpiresAt *float64 `json:"dealExpiresAt,omitempty"`

	// Inventory & Stock (REQUIRED: isActive)
	Stock    *int `json:"stock,omitempty"`
	IsActive bool `json:"isActive"`

	// Physical Properties
	Weight     *float64           `json:"weight,omitempty"`
	Dimensions *ConvexDimensions `json:"dimensions,omitempty"`

	// Variants
	AvailableSizes  []string            `json:"availableSizes"` // Always include, even if empty
	AvailableColors []ConvexColorOption `json:"availableColors,omitempty"`
	Variants        []ConvexVariant     `json:"variants,omitempty"`

	// Features
	Features []ConvexFeature `json:"features,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// SEO Fields
	SEOTitle             *string  `json:"seoTitle,omitempty"`
	SEODescription       *string  `json:"seoDescription,omitempty"`
	SEOKeywords          []string `json:"seoKeywords,omitempty"`
	MetaRobots           *string  `json:"metaRobots,omitempty"`
	CanonicalUrl         *string  `json:"canonicalUrl,omitempty"`
	OpenGraphTitle       *string  `json:"openGraphTitle,omitempty"`
	OpenGraphDescription *string  `json:"openGraphDescription,omitempty"`
	OpenGraphImage       *string  `json:"openGraphImage,omitempty"`
	OpenGraphType        *string  `json:"openGraphType,omitempty"`
	TwitterCard          *string  `json:"twitterCard,omitempty"`
	TwitterTitle         *string  `json:"twitterTitle,omitempty"`
	TwitterDescription   *string  `json:"twitterDescription,omitempty"`
	TwitterImage         *string  `json:"twitterImage,omitempty"`
	StructuredData       map[string]interface{} `json:"structuredData"` // Always include, even if empty
	SEOScore             *float64 `json:"seoScore,omitempty"`
	FocusKeyword         *string  `json:"focusKeyword,omitempty"`
	ReadabilityScore     *float64 `json:"readabilityScore,omitempty"`

	// Product specifications - included in same mutation
	Specifications []ConvexProductSpecificationInput `json:"specifications,omitempty"`
}

// ConvexDimensions represents product dimensions
type ConvexDimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

// ConvexVariant represents a product variant in Convex schema
type ConvexVariant struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Type            string             `json:"type"` // "color", "size", "color-size"
	Color           *ConvexColorOption `json:"color,omitempty"`
	Size            *string            `json:"size,omitempty"`
	SKU             *string            `json:"sku,omitempty"`
	Price           *float64           `json:"price,omitempty"`           // KES price input
	OriginalPrice   *float64           `json:"originalPrice,omitempty"`   // KES original price input
	PriceUsd        *float64           `json:"priceUsd,omitempty"`        // USD price (calculated)
	OriginalPriceUsd *float64          `json:"originalPriceUsd,omitempty"` // USD original price (calculated)
	Stock           *int               `json:"stock,omitempty"`
	Images          []string           `json:"images,omitempty"`
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

// ConvexProductSpecificationInput represents specifications for input (matches your mutation schema)
type ConvexProductSpecificationInput struct {
	ID       string                    `json:"id"`
	Category string                    `json:"category"`
	Specs    []ConvexSpecificationInput `json:"specs"`
	Order    *int                      `json:"order,omitempty"`
	IsActive bool                      `json:"isActive"`
}

// ConvexSpecificationInput represents a single spec for input
type ConvexSpecificationInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Order *int   `json:"order,omitempty"`
}

// ConvexProductSpecification represents specifications in Convex (legacy - for separate API calls)
type ConvexProductSpecification struct {
	ProductID string                `json:"productId"`
	Category  string                `json:"category"`
	Specs     []ConvexSpecification `json:"specs"`
	Order     *int                  `json:"order,omitempty"`
	IsActive  bool                  `json:"isActive"`
	CreatedAt float64               `json:"createdAt"`
	UpdatedAt float64               `json:"updatedAt"`
}

// ConvexSpecification represents a single spec (legacy - for separate API calls)
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
	Value        interface{} `json:"value,omitempty"`
	Error        *ConvexError `json:"error,omitempty"`
	Status       string      `json:"status,omitempty"`        // Alternative response format
	ErrorMessage string      `json:"errorMessage,omitempty"`  // Alternative error format
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
