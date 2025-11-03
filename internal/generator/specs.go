package generator

import (
	"regexp"
	"strings"
)

// ExtractSpecifications extracts product specifications from name and category
func ExtractSpecifications(name, category, sku string) map[string][]Specification {
	specs := make(map[string][]Specification)

	// General specifications (always present)
	general := []Specification{
		{Name: "SKU", Value: sku, Order: 1},
		{Name: "Category", Value: category, Order: 2},
		{Name: "Brand", Value: extractBrand(name), Order: 3},
	}

	// Remove empty brand
	if general[2].Value == "" {
		general[2].Value = "Generic"
	}

	specs["General"] = general

	// Extract technical specs based on product type
	technical := extractTechnicalSpecs(name, category)
	if len(technical) > 0 {
		specs["Technical"] = technical
	}

	// Extract dimensions/capacity if present
	dimensions := extractDimensionsOrCapacity(name)
	if len(dimensions) > 0 {
		specs["Dimensions & Capacity"] = dimensions
	}

	return specs
}

// extractTechnicalSpecs extracts technical specifications from product name
func extractTechnicalSpecs(name, category string) []Specification {
	specs := make([]Specification, 0)
	nameLower := strings.ToLower(name)
	order := 1

	// Memory/RAM
	if ramMatch := regexp.MustCompile(`(\d+)\s*(gb|tb)\s*(ddr[345]|ram|memory)`).FindStringSubmatch(nameLower); ramMatch != nil {
		specs = append(specs, Specification{
			Name:  "Memory",
			Value: strings.ToUpper(ramMatch[0]),
			Order: order,
		})
		order++
	}

	// Storage (SSD/HDD)
	if storageMatch := regexp.MustCompile(`(\d+)\s*(gb|tb)\s*(ssd|nvme|sata|m\.2)`).FindStringSubmatch(nameLower); storageMatch != nil {
		specs = append(specs, Specification{
			Name:  "Storage",
			Value: strings.ToUpper(storageMatch[0]),
			Order: order,
		})
		order++
	}

	// Processor
	if procMatch := regexp.MustCompile(`(intel\s+core\s+i[3579]|amd\s+ryzen\s+[3579]|core\s+i[3579])`).FindStringSubmatch(nameLower); procMatch != nil {
		specs = append(specs, Specification{
			Name:  "Processor",
			Value: strings.Title(procMatch[0]),
			Order: order,
		})
		order++
	}

	// Display size
	if sizeMatch := regexp.MustCompile(`(\d+\.?\d*)\s*(inch|")`).FindStringSubmatch(nameLower); sizeMatch != nil {
		specs = append(specs, Specification{
			Name:  "Display Size",
			Value: sizeMatch[1] + `"`,
			Order: order,
		})
		order++
	}

	// Resolution
	if strings.Contains(nameLower, "4k") {
		specs = append(specs, Specification{
			Name:  "Resolution",
			Value: "4K Ultra HD",
			Order: order,
		})
		order++
	} else if strings.Contains(nameLower, "full hd") || strings.Contains(nameLower, "1080p") {
		specs = append(specs, Specification{
			Name:  "Resolution",
			Value: "Full HD 1080p",
			Order: order,
		})
		order++
	}

	// Connectivity
	connectivity := make([]string, 0)
	if strings.Contains(nameLower, "wifi") || strings.Contains(nameLower, "wi-fi") {
		connectivity = append(connectivity, "Wi-Fi")
	}
	if strings.Contains(nameLower, "bluetooth") {
		connectivity = append(connectivity, "Bluetooth")
	}
	if strings.Contains(nameLower, "usb") {
		connectivity = append(connectivity, "USB")
	}
	if strings.Contains(nameLower, "ethernet") {
		connectivity = append(connectivity, "Ethernet")
	}

	if len(connectivity) > 0 {
		specs = append(specs, Specification{
			Name:  "Connectivity",
			Value: strings.Join(connectivity, ", "),
			Order: order,
		})
		order++
	}

	// Print technology (for printers/toners)
	if strings.Contains(nameLower, "laser") {
		specs = append(specs, Specification{
			Name:  "Technology",
			Value: "Laser",
			Order: order,
		})
		order++
	} else if strings.Contains(nameLower, "inkjet") || strings.Contains(nameLower, "ink") {
		specs = append(specs, Specification{
			Name:  "Technology",
			Value: "Inkjet",
			Order: order,
		})
		order++
	}

	// Color/Mono (for printers/toners)
	if strings.Contains(nameLower, "color") || strings.Contains(nameLower, "colour") {
		specs = append(specs, Specification{
			Name:  "Print Type",
			Value: "Color",
			Order: order,
		})
		order++
	} else if strings.Contains(nameLower, "mono") || strings.Contains(nameLower, "black") {
		specs = append(specs, Specification{
			Name:  "Print Type",
			Value: "Monochrome",
			Order: order,
		})
		order++
	}

	return specs
}

// extractDimensionsOrCapacity extracts size/capacity information
func extractDimensionsOrCapacity(name string) []Specification {
	specs := make([]Specification, 0)
	nameLower := strings.ToLower(name)
	order := 1

	// Capacity (batteries, UPS, etc.)
	if capacityMatch := regexp.MustCompile(`(\d+)\s*(va|w|wh|mah)`).FindStringSubmatch(nameLower); capacityMatch != nil {
		specs = append(specs, Specification{
			Name:  "Capacity",
			Value: strings.ToUpper(capacityMatch[0]),
			Order: order,
		})
		order++
	}

	// Weight
	if weightMatch := regexp.MustCompile(`(\d+\.?\d*)\s*(kg|g|lbs)`).FindStringSubmatch(nameLower); weightMatch != nil {
		specs = append(specs, Specification{
			Name:  "Weight",
			Value: weightMatch[0],
			Order: order,
		})
		order++
	}

	// Form factor
	if strings.Contains(nameLower, "2.5\"") || strings.Contains(nameLower, "2.5 inch") {
		specs = append(specs, Specification{
			Name:  "Form Factor",
			Value: `2.5"`,
			Order: order,
		})
		order++
	} else if strings.Contains(nameLower, "m.2") {
		specs = append(specs, Specification{
			Name:  "Form Factor",
			Value: "M.2",
			Order: order,
		})
		order++
	}

	return specs
}
