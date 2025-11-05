package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geraldbahati/image-sourcer/internal/convex"
	"github.com/geraldbahati/image-sourcer/internal/generator"
)

func main() {
	// Create a sample enriched product
	enriched := generator.EnrichedProduct{
		Name:        "HP 305A Black LaserJet Toner",
		Slug:        "hp-305a-black-laserjet-toner",
		Description: "Professional toner cartridge for HP LaserJet printers",
		SKU:         "CE410A",
		Barcode:     "123456789",

		// Pricing
		BasePriceKes:    12500,
		ComparePriceKes: 15000,
		BasePriceUsd:    96.15,
		ComparePriceUsd: 115.38,

		// Images
		PrimaryImage:  "https://via.placeholder.com/800x800.png",
		Images:        []string{"https://via.placeholder.com/800x800.png"},
		ThumbnailURL:  "https://via.placeholder.com/300x300.png",

		// Category
		Category: "Toner Cartridges",

		// Attributes
		Tags:     []string{"toner", "hp", "black"},
		Badges:   []string{"bestseller"},
		IsNew:    true,
		IsActive: true,
		Stock:    50,

		// Features
		Features: []generator.ProductFeature{
			{
				Icon:        "printer",
				Title:       "High Quality",
				Description: "Professional grade printing",
			},
		},

		// SEO
		SEO: generator.SEOMetadata{
			Title:            "HP 305A Black Toner - Professional Quality",
			Description:      "Get professional printing quality with HP 305A Black LaserJet Toner",
			Keywords:         []string{"hp", "toner", "305a", "black"},
			OpenGraphTitle:   "HP 305A Black Toner",
			OpenGraphDesc:    "Professional quality toner",
			OpenGraphImage:   "https://via.placeholder.com/1200x630.png",
			TwitterCard:      "summary_large_image",
			TwitterTitle:     "HP 305A Black Toner",
			TwitterDescription: "Professional quality toner",
			TwitterImage:     "https://via.placeholder.com/1200x630.png",
		},

		// Specifications
		Specifications: map[string][]generator.Specification{
			"Technical": {
				{Name: "Color", Value: "Black", Order: 0},
				{Name: "Page Yield", Value: "2,400 pages", Order: 1},
			},
			"Compatibility": {
				{Name: "Compatible Models", Value: "HP LaserJet Pro 300/400", Order: 0},
			},
		},

		// Variants (example with color variants)
		IsVariantProduct: true,
		AvailableColors: []generator.ColorOption{
			{Name: "Black", Value: "#000000"},
			{Name: "Cyan", Value: "#00FFFF"},
		},
		AvailableSizes: []generator.SizeOption{},
		Variants: []generator.ProductVariant{
			{
				ID:       "CE410A",
				Name:     "HP 305A Black",
				Type:     "color",
				Color:    &generator.ColorOption{Name: "Black", Value: "#000000"},
				SKU:      "CE410A",
				PriceKes: 12500,
				PriceUsd: 96.15,
				Stock:    50,
				Images:   []string{"https://via.placeholder.com/800x800.png"},
			},
			{
				ID:       "CE411A",
				Name:     "HP 305A Cyan",
				Type:     "color",
				Color:    &generator.ColorOption{Name: "Cyan", Value: "#00FFFF"},
				SKU:      "CE411A",
				PriceKes: 12500,
				PriceUsd: 96.15,
				Stock:    30,
				Images:   []string{"https://via.placeholder.com/800x800-cyan.png"},
			},
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Map to Convex format
	config := convex.DefaultMapperConfig()
	convexProduct := convex.MapToConvexProduct(enriched, nil, config)

	// Convert to JSON to see the payload
	jsonBytes, err := json.MarshalIndent(convexProduct, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println("=== CONVEX PRODUCT PAYLOAD ===")
	fmt.Println(string(jsonBytes))
	fmt.Println("\n=== FIELD COUNT ===")

	// Count fields
	var data map[string]interface{}
	json.Unmarshal(jsonBytes, &data)

	fieldCount := 0
	for key, value := range data {
		if value != nil {
			fieldCount++
			fmt.Printf("✓ %s\n", key)
		}
	}

	fmt.Printf("\nTotal fields: %d\n", fieldCount)
}
