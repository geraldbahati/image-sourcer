package generator

import (
	"regexp"
	"strings"
	"sync"
)

// Pre-compiled regex patterns for performance
var (
	// Color patterns
	colorPattern = regexp.MustCompile(`(?i)\b(black|white|cyan|magenta|yellow|red|blue|green|gray|grey|silver|gold|rose|pink|purple|orange|brown|beige|navy|teal|lime|violet|indigo|maroon|olive|aqua|coral|lavender|mint|peach|tan|turquoise|cream|ivory)\b`)

	// Size patterns - storage
	storagePattern = regexp.MustCompile(`(?i)\b(\d+)\s*(gb|tb|mb)\b`)

	// Size patterns - RAM
	ramPattern = regexp.MustCompile(`(?i)\b(\d+)\s*(gb|mb)\s*(ddr[345]|ram|memory)\b`)

	// Size patterns - display
	displayPattern = regexp.MustCompile(`(?i)\b(\d+\.?\d*)\s*(inch|"|in)\b`)

	// Size patterns - clothing
	clothingSizePattern = regexp.MustCompile(`(?i)\b(xxs|xs|s|m|l|xl|xxl|xxxl|2xl|3xl|4xl)\b`)

	// Size patterns - numeric sizes (shoe sizes, etc.)
	numericSizePattern = regexp.MustCompile(`(?i)\bsize\s*(\d+)\b`)

	// String builder pool for performance
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// colorHexMap maps color names to hex codes
var colorHexMap = map[string]string{
	// Basic colors
	"black":   "#000000",
	"white":   "#FFFFFF",
	"gray":    "#808080",
	"grey":    "#808080",
	"silver":  "#C0C0C0",

	// Primary colors
	"red":     "#FF0000",
	"blue":    "#0000FF",
	"yellow":  "#FFFF00",
	"green":   "#00FF00",

	// Secondary colors
	"cyan":    "#00FFFF",
	"magenta": "#FF00FF",
	"orange":  "#FFA500",
	"purple":  "#800080",
	"pink":    "#FFC0CB",
	"brown":   "#A52A2A",

	// Tertiary colors
	"gold":      "#FFD700",
	"rose":      "#FF007F",
	"beige":     "#F5F5DC",
	"navy":      "#000080",
	"teal":      "#008080",
	"lime":      "#00FF00",
	"violet":    "#EE82EE",
	"indigo":    "#4B0082",
	"maroon":    "#800000",
	"olive":     "#808000",
	"aqua":      "#00FFFF",
	"coral":     "#FF7F50",
	"lavender":  "#E6E6FA",
	"mint":      "#98FF98",
	"peach":     "#FFDAB9",
	"tan":       "#D2B48C",
	"turquoise": "#40E0D0",
	"cream":     "#FFFDD0",
	"ivory":     "#FFFFF0",
}

// ExtractColor extracts color from product name
func ExtractColor(name string) *ColorOption {
	nameLower := strings.ToLower(name)

	// Find color match
	matches := colorPattern.FindStringSubmatch(nameLower)
	if len(matches) < 2 {
		return nil
	}

	colorName := matches[1]

	// Capitalize first letter for display
	colorNameDisplay := strings.Title(colorName)

	// Get hex code
	hexCode, exists := colorHexMap[colorName]
	if !exists {
		hexCode = "#CCCCCC" // Default gray for unknown colors
	}

	return &ColorOption{
		Name:  colorNameDisplay,
		Value: hexCode,
	}
}

// ExtractAllColors extracts all colors from product name (for multi-color products)
func ExtractAllColors(name string) []ColorOption {
	nameLower := strings.ToLower(name)

	// Find all color matches
	matches := colorPattern.FindAllStringSubmatch(nameLower, -1)
	if len(matches) == 0 {
		return nil
	}

	colors := make([]ColorOption, 0, len(matches))
	seen := make(map[string]bool, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		colorName := match[1]

		// Skip duplicates
		if seen[colorName] {
			continue
		}
		seen[colorName] = true

		// Capitalize first letter for display
		colorNameDisplay := strings.Title(colorName)

		// Get hex code
		hexCode, exists := colorHexMap[colorName]
		if !exists {
			hexCode = "#CCCCCC"
		}

		colors = append(colors, ColorOption{
			Name:  colorNameDisplay,
			Value: hexCode,
		})
	}

	return colors
}

// ExtractSize extracts size information from product name
func ExtractSize(name string) *SizeOption {
	nameLower := strings.ToLower(name)

	// Try storage size first (most common in product names)
	if matches := storagePattern.FindStringSubmatch(nameLower); len(matches) >= 3 {
		size := strings.ToUpper(matches[1] + matches[2])
		return &SizeOption{
			Name:  size,
			Value: normalizeStorageSize(matches[1], matches[2]),
		}
	}

	// Try RAM size
	if matches := ramPattern.FindStringSubmatch(nameLower); len(matches) >= 3 {
		size := strings.ToUpper(matches[1] + matches[2])
		return &SizeOption{
			Name:  size + " RAM",
			Value: normalizeStorageSize(matches[1], matches[2]),
		}
	}

	// Try display size
	if matches := displayPattern.FindStringSubmatch(nameLower); len(matches) >= 2 {
		size := matches[1] + "\""
		return &SizeOption{
			Name:  size,
			Value: matches[1],
		}
	}

	// Try clothing size
	if matches := clothingSizePattern.FindStringSubmatch(nameLower); len(matches) >= 2 {
		size := strings.ToUpper(matches[1])
		return &SizeOption{
			Name:  size,
			Value: size,
		}
	}

	// Try numeric size
	if matches := numericSizePattern.FindStringSubmatch(nameLower); len(matches) >= 2 {
		size := "Size " + matches[1]
		return &SizeOption{
			Name:  size,
			Value: matches[1],
		}
	}

	return nil
}

// normalizeStorageSize normalizes storage sizes to GB for comparison
func normalizeStorageSize(value, unit string) string {
	unit = strings.ToLower(unit)

	builder := builderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		builderPool.Put(builder)
	}()

	builder.WriteString(value)

	switch unit {
	case "tb":
		builder.WriteString("000") // Convert TB to GB
		builder.WriteString("GB")
	case "mb":
		builder.WriteString("MB")
	default: // gb
		builder.WriteString("GB")
	}

	return builder.String()
}

// RemoveColorFromName removes color from product name (for base name extraction)
func RemoveColorFromName(name string) string {
	// Use regex to remove color words
	cleaned := colorPattern.ReplaceAllString(name, "")

	// Clean up extra spaces
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return strings.TrimSpace(cleaned)
}

// RemoveSizeFromName removes size from product name (for base name extraction)
func RemoveSizeFromName(name string) string {
	// Remove storage sizes
	cleaned := storagePattern.ReplaceAllString(name, "")

	// Remove RAM sizes
	cleaned = ramPattern.ReplaceAllString(cleaned, "")

	// Remove display sizes
	cleaned = displayPattern.ReplaceAllString(cleaned, "")

	// Remove clothing sizes
	cleaned = clothingSizePattern.ReplaceAllString(cleaned, "")

	// Remove numeric sizes
	cleaned = numericSizePattern.ReplaceAllString(cleaned, "")

	// Clean up extra spaces
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return strings.TrimSpace(cleaned)
}

// GetBaseProductName removes both color and size to get base product name
func GetBaseProductName(name string) string {
	// Remove color first
	cleaned := RemoveColorFromName(name)

	// Then remove size
	cleaned = RemoveSizeFromName(cleaned)

	// Clean up common suffixes/prefixes
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimSuffix(cleaned, "-")
	cleaned = strings.TrimPrefix(cleaned, "-")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// HasColor checks if a product name contains a color
func HasColor(name string) bool {
	return colorPattern.MatchString(strings.ToLower(name))
}

// HasSize checks if a product name contains a size
func HasSize(name string) bool {
	nameLower := strings.ToLower(name)
	return storagePattern.MatchString(nameLower) ||
		ramPattern.MatchString(nameLower) ||
		displayPattern.MatchString(nameLower) ||
		clothingSizePattern.MatchString(nameLower) ||
		numericSizePattern.MatchString(nameLower)
}

// GetVariantType determines the type of variant (color, size, or both)
func GetVariantType(name string) string {
	hasColor := HasColor(name)
	hasSize := HasSize(name)

	if hasColor && hasSize {
		return "color-size"
	} else if hasColor {
		return "color"
	} else if hasSize {
		return "size"
	}

	return ""
}
