package generator

import (
	"fmt"
	"strings"
)

// GenerateDescription creates a product description from name and category
func GenerateDescription(name, category string, style string, maxLength int) string {
	// Extract key features from name
	features := extractFeatures(name)

	// Choose template based on category
	template := selectTemplate(category, style)

	// Generate description
	desc := fmt.Sprintf(template, name, category, formatFeatures(features))

	// Ensure length constraints
	if len(desc) > maxLength {
		desc = desc[:maxLength-3] + "..."
	}

	// Ensure minimum quality
	if len(desc) < 50 {
		desc = fmt.Sprintf("The %s is a high-quality product perfect for your needs. "+
			"This %s offers excellent performance and reliability.", name, category)
	}

	return strings.TrimSpace(desc)
}

// extractFeatures extracts key features from product name
func extractFeatures(name string) []string {
	features := make([]string, 0)
	nameLower := strings.ToLower(name)

	// Common features to look for
	featureKeywords := map[string]string{
		"wireless":    "wireless connectivity",
		"bluetooth":   "Bluetooth support",
		"wifi":        "Wi-Fi enabled",
		"4k":          "4K resolution",
		"hd":          "HD quality",
		"laser":       "laser technology",
		"inkjet":      "inkjet printing",
		"gaming":      "gaming performance",
		"pro":         "professional grade",
		"portable":    "portable design",
		"compact":     "compact form factor",
		"enterprise":  "enterprise-ready",
		"business":    "business-class",
		"eco":         "eco-friendly",
		"smart":       "smart features",
	}

	for keyword, feature := range featureKeywords {
		if strings.Contains(nameLower, keyword) {
			features = append(features, feature)
		}
	}

	return features
}

// selectTemplate chooses appropriate template based on category and style
func selectTemplate(category, style string) string {
	categoryLower := strings.ToLower(category)

	// Category-specific templates
	if strings.Contains(categoryLower, "toner") || strings.Contains(categoryLower, "cartridge") {
		return "The %s is a premium %s designed for optimal print quality and page yield. %s"
	}

	if strings.Contains(categoryLower, "printer") {
		return "The %s is a reliable %s offering exceptional print quality and efficiency. %s Perfect for home and office use."
	}

	if strings.Contains(categoryLower, "laptop") || strings.Contains(categoryLower, "desktop") {
		return "The %s is a powerful %s built for productivity and performance. %s Ideal for professional and personal computing needs."
	}

	if strings.Contains(categoryLower, "monitor") || strings.Contains(categoryLower, "display") {
		return "The %s is a high-quality %s delivering crystal-clear visuals. %s Enhance your viewing experience with vibrant colors and sharp details."
	}

	if strings.Contains(categoryLower, "ups") || strings.Contains(categoryLower, "power") {
		return "The %s is a reliable %s ensuring uninterrupted power supply. %s Protect your devices with dependable backup power."
	}

	if strings.Contains(categoryLower, "ssd") || strings.Contains(categoryLower, "storage") {
		return "The %s is a fast and reliable %s for all your data storage needs. %s Experience lightning-fast read/write speeds."
	}

	if strings.Contains(categoryLower, "mouse") || strings.Contains(categoryLower, "keyboard") {
		return "The %s is an ergonomic %s designed for comfort and precision. %s Enhance your productivity with superior control."
	}

	if strings.Contains(categoryLower, "camera") || strings.Contains(categoryLower, "security") {
		return "The %s is a professional-grade %s for comprehensive surveillance. %s Ensure peace of mind with reliable security monitoring."
	}

	// Default template
	switch style {
	case "technical":
		return "The %s is a high-performance %s engineered for demanding applications. %s Built with cutting-edge technology."
	case "casual":
		return "Check out the %s - a great %s that gets the job done! %s You'll love what it can do."
	default: // professional
		return "The %s is a premium %s delivering exceptional quality and performance. %s Designed to meet your professional needs."
	}
}

// formatFeatures converts feature list to descriptive text
func formatFeatures(features []string) string {
	if len(features) == 0 {
		return ""
	}

	if len(features) == 1 {
		return fmt.Sprintf("Features %s.", features[0])
	}

	if len(features) == 2 {
		return fmt.Sprintf("Features %s and %s.", features[0], features[1])
	}

	// 3 or more features
	featureText := "Features " + strings.Join(features[:len(features)-1], ", ") +
		", and " + features[len(features)-1] + "."

	return featureText
}
