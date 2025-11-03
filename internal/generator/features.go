package generator

import (
	"strings"
)

// GenerateFeatures creates product features based on category and name
func GenerateFeatures(name, category string, defaultFeatures []ProductFeature) []ProductFeature {
	features := make([]ProductFeature, 0)

	// Always add default features first
	features = append(features, defaultFeatures...)

	// Add category-specific features
	categoryFeatures := generateCategoryFeatures(category, name)
	features = append(features, categoryFeatures...)

	// Limit to 6 features maximum for UI
	if len(features) > 6 {
		features = features[:6]
	}

	return features
}

// generateCategoryFeatures generates features specific to product category
func generateCategoryFeatures(category, name string) []ProductFeature {
	features := make([]ProductFeature, 0, 3)
	categoryLower := strings.ToLower(category)
	nameLower := strings.ToLower(name)

	// Printers & Toners
	if strings.Contains(categoryLower, "printer") || strings.Contains(categoryLower, "toner") || strings.Contains(categoryLower, "cartridge") {
		if strings.Contains(nameLower, "wireless") || strings.Contains(nameLower, "wifi") {
			features = append(features, ProductFeature{
				Icon:        "Wifi",
				Title:       "Wireless Printing",
				Description: "Print from anywhere with Wi-Fi connectivity",
			})
		}

		features = append(features, ProductFeature{
			Icon:        "Printer",
			Title:       "High Yield",
			Description: "More pages per cartridge for cost savings",
		})
	}

	// Laptops & Desktops
	if strings.Contains(categoryLower, "laptop") || strings.Contains(categoryLower, "desktop") || strings.Contains(categoryLower, "psg") {
		features = append(features, ProductFeature{
			Icon:        "Cpu",
			Title:       "Powerful Performance",
			Description: "Handle demanding tasks with ease",
		})

		if strings.Contains(nameLower, "gaming") {
			features = append(features, ProductFeature{
				Icon:        "Gamepad2",
				Title:       "Gaming Ready",
				Description: "Optimized for immersive gaming experience",
			})
		} else if strings.Contains(nameLower, "pro") || strings.Contains(nameLower, "business") {
			features = append(features, ProductFeature{
				Icon:        "Briefcase",
				Title:       "Business Class",
				Description: "Professional-grade reliability",
			})
		}
	}

	// Monitors
	if strings.Contains(categoryLower, "monitor") {
		features = append(features, ProductFeature{
			Icon:        "Monitor",
			Title:       "Crystal Clear Display",
			Description: "Vibrant colors and sharp details",
		})

		if strings.Contains(nameLower, "4k") {
			features = append(features, ProductFeature{
				Icon:        "Sparkles",
				Title:       "4K Resolution",
				Description: "Ultra HD clarity for stunning visuals",
			})
		}
	}

	// UPS & Power
	if strings.Contains(categoryLower, "ups") || strings.Contains(categoryLower, "power") {
		features = append(features, ProductFeature{
			Icon:        "Zap",
			Title:       "Backup Power",
			Description: "Protect devices during outages",
		})

		features = append(features, ProductFeature{
			Icon:        "Battery",
			Title:       "Extended Runtime",
			Description: "Keep working during power interruptions",
		})
	}

	// Storage (SSD/HDD/RAM)
	if strings.Contains(categoryLower, "ssd") || strings.Contains(categoryLower, "storage") || strings.Contains(categoryLower, "ram") {
		features = append(features, ProductFeature{
			Icon:        "HardDrive",
			Title:       "Fast Performance",
			Description: "Lightning-fast data access speeds",
		})

		if strings.Contains(nameLower, "nvme") {
			features = append(features, ProductFeature{
				Icon:        "Zap",
				Title:       "NVMe Speed",
				Description: "Up to 7x faster than SATA SSDs",
			})
		}
	}

	// Peripherals (Mouse, Keyboard)
	if strings.Contains(categoryLower, "mouse") || strings.Contains(categoryLower, "keyboard") || strings.Contains(categoryLower, "logitech") {
		if strings.Contains(nameLower, "wireless") {
			features = append(features, ProductFeature{
				Icon:        "Wifi",
				Title:       "Wireless Freedom",
				Description: "No cables, more flexibility",
			})
		}

		features = append(features, ProductFeature{
			Icon:        "Hand",
			Title:       "Ergonomic Design",
			Description: "Comfortable for extended use",
		})
	}

	// Security & Cameras
	if strings.Contains(categoryLower, "camera") || strings.Contains(categoryLower, "security") || strings.Contains(categoryLower, "cp plus") {
		features = append(features, ProductFeature{
			Icon:        "Eye",
			Title:       "24/7 Monitoring",
			Description: "Round-the-clock security surveillance",
		})

		if strings.Contains(nameLower, "4k") || strings.Contains(nameLower, "5mp") {
			features = append(features, ProductFeature{
				Icon:        "Focus",
				Title:       "High Resolution",
				Description: "Crystal clear video capture",
			})
		}
	}

	// Epson/Consumables
	if strings.Contains(categoryLower, "epson") || strings.Contains(categoryLower, "consumable") {
		features = append(features, ProductFeature{
			Icon:        "DropletIcon",
			Title:       "Authentic Quality",
			Description: "Genuine products for best results",
		})
	}

	return features
}
