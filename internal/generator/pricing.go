package generator

import (
	"math"
)

// CalculatePricing calculates KES prices from USD with smart rounding
func CalculatePricing(basePriceUsd, comparePriceUsd float64, config EnrichmentConfig) (basePriceKes, comparePriceKes float64) {
	// Convert USD to KES
	baseKes := basePriceUsd * config.ExchangeRate
	compareKes := comparePriceUsd * config.ExchangeRate

	// Apply smart rounding if enabled
	if config.SmartRounding {
		basePriceKes = smartRound(baseKes)
		if comparePriceUsd > 0 {
			comparePriceKes = smartRound(compareKes)
		}
	} else {
		basePriceKes = math.Round(baseKes)
		comparePriceKes = math.Round(compareKes)
	}

	return basePriceKes, comparePriceKes
}

// smartRound rounds prices to psychologically appealing values
func smartRound(price float64) float64 {
	if price == 0 {
		return 0
	}

	// For very small prices (< 100), round to nearest 10
	if price < 100 {
		return math.Round(price/10) * 10
	}

	// For prices 100-500, round to nearest 50
	if price < 500 {
		return math.Round(price/50) * 50
	}

	// For prices 500-1000, round to nearest 100
	if price < 1000 {
		return math.Round(price/100) * 100
	}

	// For prices 1000-5000, round to nearest 500
	if price < 5000 {
		return math.Round(price/500) * 500
	}

	// For prices 5000-10000, round to nearest 1000
	if price < 10000 {
		return math.Round(price/1000) * 1000
	}

	// For very high prices, round to nearest 5000
	return math.Round(price/5000) * 5000
}

// GeneratePricingTiers creates pricing tiers for display
func GeneratePricingTiers(basePriceKes float64) (tier string) {
	if basePriceKes == 0 {
		return "Price on Request"
	}

	if basePriceKes < 1000 {
		return "Budget"
	}

	if basePriceKes < 10000 {
		return "Mid-Range"
	}

	if basePriceKes < 50000 {
		return "Premium"
	}

	return "Enterprise"
}
