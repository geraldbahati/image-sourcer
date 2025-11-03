package generator

import (
	"fmt"
	"strings"
)

// GenerateSEO creates comprehensive SEO metadata for a product
func GenerateSEO(name, description, category, slug string, price float64, primaryImage string, config EnrichmentConfig) SEOMetadata {
	seo := SEOMetadata{}

	// Generate SEO title (max 60 chars for Google)
	seo.Title = generateSEOTitle(name, category)

	// Generate SEO description (max 160 chars for Google)
	seo.Description = generateSEODescription(name, description, price)

	// Generate keywords
	seo.Keywords = generateKeywords(name, category)

	// Generate Open Graph tags if enabled
	if config.GenerateOGTags {
		seo.OpenGraphTitle = seo.Title
		seo.OpenGraphDesc = seo.Description
		seo.OpenGraphImage = primaryImage
	}

	// Generate Twitter Card tags if enabled
	if config.GenerateTwitterTags {
		seo.TwitterCard = "summary_large_image"
		seo.TwitterTitle = seo.Title
		seo.TwitterDescription = seo.Description
		seo.TwitterImage = primaryImage
	}

	return seo
}

// generateSEOTitle creates an optimized title (max 60 chars)
func generateSEOTitle(name, category string) string {
	// Format: "Product Name - Category | Store Name"
	title := fmt.Sprintf("%s - %s | OM IT Distribution", name, category)

	// Truncate if too long
	if len(title) > 60 {
		maxNameLength := 60 - len(category) - len(" - ... | OM IT")
		if maxNameLength > 20 {
			truncatedName := name
			if len(name) > maxNameLength {
				truncatedName = name[:maxNameLength] + "..."
			}
			title = fmt.Sprintf("%s - %s | OM IT", truncatedName, category)
		} else {
			// Very long category, just use product name
			if len(name) > 45 {
				title = name[:42] + "... | OM IT"
			} else {
				title = name + " | OM IT Distribution"
			}
		}
	}

	return title
}

// generateSEODescription creates an optimized description (max 160 chars)
func generateSEODescription(name, description string, price float64) string {
	// Start with product name and price
	var desc string

	if price > 0 {
		desc = fmt.Sprintf("Buy %s for $%.2f. ", name, price)
	} else {
		desc = fmt.Sprintf("Shop %s. ", name)
	}

	// Add description snippet
	remaining := 160 - len(desc) - len(" Order now!")
	if remaining > 50 && description != "" {
		snippet := description
		if len(snippet) > remaining {
			snippet = snippet[:remaining-3] + "..."
		}
		desc += snippet + " "
	} else {
		desc += "Premium quality products with fast shipping. "
	}

	desc += "Order now!"

	// Final safety truncation
	if len(desc) > 160 {
		desc = desc[:157] + "..."
	}

	return desc
}

// generateKeywords extracts relevant keywords from name and category
func generateKeywords(name, category string) []string {
	keywords := make([]string, 0, 10)

	// Add category as primary keyword
	keywords = append(keywords, strings.ToLower(category))

	// Extract brand if present
	brand := extractBrand(name)
	if brand != "" {
		keywords = append(keywords, strings.ToLower(brand))
	}

	// Extract product type
	productType := extractProductType(name, category)
	if productType != "" && productType != strings.ToLower(category) {
		keywords = append(keywords, productType)
	}

	// Add common tech keywords
	nameLower := strings.ToLower(name)
	techKeywords := []string{
		"wireless", "bluetooth", "wifi", "4k", "hd", "laser", "inkjet",
		"gaming", "portable", "compact", "professional", "business",
		"enterprise", "eco-friendly", "smart", "digital", "usb", "ssd",
		"nvme", "sata", "ddr4", "ddr5", "rgb", "mechanical",
	}

	for _, keyword := range techKeywords {
		if strings.Contains(nameLower, keyword) {
			keywords = append(keywords, keyword)
		}
	}

	// Limit to 10 keywords
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

// extractBrand extracts brand name from product name
func extractBrand(name string) string {
	// Common brands
	brands := []string{
		"HP", "Dell", "Lenovo", "Asus", "Acer", "Apple", "Microsoft",
		"Logitech", "Canon", "Epson", "Brother", "Samsung", "LG",
		"Lexar", "SanDisk", "WD", "Western Digital", "Seagate",
		"Toshiba", "KIOXIA", "Intel", "AMD", "NVIDIA",
		"Promate", "Mercury", "D-Link", "TP-Link", "Ubiquiti",
		"Kaspersky", "Norton", "McAfee",
	}

	nameUpper := strings.ToUpper(name)
	for _, brand := range brands {
		if strings.Contains(nameUpper, strings.ToUpper(brand)) {
			return brand
		}
	}

	return ""
}

// extractProductType extracts product type from name and category
func extractProductType(name, category string) string {
	nameLower := strings.ToLower(name)

	productTypes := map[string][]string{
		"toner": {"toner", "cartridge"},
		"printer": {"printer", "mfp", "multifunction"},
		"laptop": {"laptop", "notebook"},
		"desktop": {"desktop", "pc", "workstation"},
		"monitor": {"monitor", "display", "screen"},
		"keyboard": {"keyboard"},
		"mouse": {"mouse", "mice"},
		"ssd": {"ssd", "solid state"},
		"hdd": {"hdd", "hard drive"},
		"ram": {"ram", "memory"},
		"ups": {"ups", "power supply"},
		"router": {"router", "access point"},
		"switch": {"switch", "hub"},
		"webcam": {"webcam", "camera"},
		"headset": {"headset", "headphone"},
	}

	for productType, keywords := range productTypes {
		for _, keyword := range keywords {
			if strings.Contains(nameLower, keyword) {
				return productType
			}
		}
	}

	// Fallback to category
	return strings.ToLower(category)
}
