package matcher

import (
	"sync"
)

// UploadState represents the .upload-state.json structure
type UploadState struct {
	LastUpload string                     `json:"last_upload"`
	Products   map[string]UploadedProduct `json:"uploaded_products"` // SKU → Product
}

// UploadedProduct represents a product with uploaded images
type UploadedProduct struct {
	PartNumber     string   `json:"part_number"`
	ProductName    string   `json:"product_name"`
	Images         []string `json:"images"`          // Public IDs
	CloudinaryURLs []string `json:"cloudinary_urls"` // Cloudinary URLs
	UploadedAt     string   `json:"uploaded_at"`
}

// CloudinaryImage represents image URLs from Cloudinary
type CloudinaryImage struct {
	Path         string `json:"path"`
	CloudinaryURL string `json:"cloudinary_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	MainURL      string `json:"main_url"`
	ZoomURL      string `json:"zoom_url"`
}

// MatchedProduct represents a product matched with its images
type MatchedProduct struct {
	// Excel data
	SKU      string
	Name     string
	Price    float64
	Stock    int
	Category string

	// Cloudinary images
	Images     []CloudinaryImage
	HasImages  bool
	ImageCount int

	// Metadata
	SourceFile  string
	SourceSheet string
	RowNumber   int
}

// MatchResult contains the results of matching products to images
type MatchResult struct {
	Matched       []MatchedProduct
	Unmatched     []MatchedProduct
	TotalProducts int
	MatchRate     float64
	mu            sync.RWMutex
}

// AddMatched safely adds a matched product (thread-safe)
func (mr *MatchResult) AddMatched(product MatchedProduct) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.Matched = append(mr.Matched, product)
	mr.TotalProducts++
}

// AddUnmatched safely adds an unmatched product (thread-safe)
func (mr *MatchResult) AddUnmatched(product MatchedProduct) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.Unmatched = append(mr.Unmatched, product)
	mr.TotalProducts++
}

// Finalize calculates the final match rate
func (mr *MatchResult) Finalize() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if mr.TotalProducts > 0 {
		mr.MatchRate = float64(len(mr.Matched)) / float64(mr.TotalProducts)
	}
}

// GetMatched safely retrieves matched products
func (mr *MatchResult) GetMatched() []MatchedProduct {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return mr.Matched
}

// GetUnmatched safely retrieves unmatched products
func (mr *MatchResult) GetUnmatched() []MatchedProduct {
	mr.mu.RUnlock()
	defer mr.mu.RUnlock()
	return mr.Unmatched
}
