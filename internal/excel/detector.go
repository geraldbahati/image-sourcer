package excel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// DetectedColumn represents a column that has been detected
type DetectedColumn struct {
	OriginalName string
	ColumnType   string  // "sku", "name", "price", "stock", "image"
	Confidence   float64 // 0.0 - 1.0
	Index        int     // Column index in sheet
}

// DetectionResult contains the results of column detection for a sheet
type DetectionResult struct {
	SheetName  string
	Columns    map[string]DetectedColumn // columnType → DetectedColumn
	Confidence float64                   // Overall confidence
	Issues     []string                  // Missing columns, low confidence warnings
}

// ColumnPattern defines patterns for detecting column types
type ColumnPattern struct {
	ExactMatches  []string
	FuzzyPatterns []string
	Validator     func([]string) float64
}

// Pre-compiled regex for performance
var (
	alphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	numericRegex      = regexp.MustCompile(`^[0-9.,]+$`)
)

// Column patterns for detection (initialized once for performance)
var columnPatterns = map[string]ColumnPattern{
	"sku": {
		ExactMatches: []string{
			"sku", "partnumber", "partnum", "partno", "itemcode", "productcode",
			"code", "id", "itemid", "productid", "part",
		},
		FuzzyPatterns: []string{"sku", "part", "item", "code", "number"},
		Validator:     validateSKU,
	},
	"name": {
		ExactMatches: []string{
			"name", "productname", "itemname", "description", "title",
			"product", "item", "productdescription", "hplaptop", "laptop",
			"monitor", "desktop",
		},
		FuzzyPatterns: []string{"name", "description", "title", "product"},
		Validator:     validateName,
	},
	"price": {
		ExactMatches: []string{
			"price", "unitprice", "cost", "unitcost", "baseprice",
			"retailprice", "sellingprice", "amount", "value",
		},
		FuzzyPatterns: []string{"price", "cost", "amount", "value"},
		Validator:     validatePrice,
	},
	"stock": {
		ExactMatches: []string{
			"stock", "qty", "quantity", "available", "availability", "inventory",
			"instock", "onhand", "units", "availableqty", "stockqty",
		},
		FuzzyPatterns: []string{"stock", "qty", "quantity", "available", "availability"},
		Validator:     validateStock,
	},
	"image": {
		ExactMatches: []string{
			"image", "imageurl", "photo", "picture", "img",
			"imagepath", "thumbnail", "productimage",
		},
		FuzzyPatterns: []string{"image", "photo", "picture", "url"},
		Validator:     validateImage,
	},
}

// String builder pool for performance (reuse builders)
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// getStringBuilder gets a builder from the pool
func getStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// putStringBuilder returns a builder to the pool
func putStringBuilder(sb *strings.Builder) {
	stringBuilderPool.Put(sb)
}

// DetectColumns detects column types in a sheet with high performance
func DetectColumns(sheet ExcelSheet, confidenceThreshold float64) DetectionResult {
	result := DetectionResult{
		SheetName:  sheet.Name,
		Columns:    make(map[string]DetectedColumn),
		Issues:     make([]string, 0),
		Confidence: 0.0,
	}

	// Pre-extract all column values once (performance optimization)
	columnValues := make(map[string][]string, len(sheet.Headers))
	for _, header := range sheet.Headers {
		values := make([]string, 0, len(sheet.Rows))
		for _, row := range sheet.Rows {
			if val, ok := row.Cells[header]; ok {
				values = append(values, val)
			} else {
				values = append(values, "")
			}
		}
		columnValues[header] = values
	}

	// Detect each column type
	for i, header := range sheet.Headers {
		detected := detectColumnType(header, columnValues[header], i, confidenceThreshold)
		if detected != nil {
			// Check if we already have this column type
			if existing, found := result.Columns[detected.ColumnType]; found {
				// Keep the one with higher confidence
				if detected.Confidence > existing.Confidence {
					result.Columns[detected.ColumnType] = *detected
				}
			} else {
				result.Columns[detected.ColumnType] = *detected
			}
		}
	}

	// Check for missing required columns
	requiredColumns := []string{"sku", "name"}
	for _, req := range requiredColumns {
		if _, found := result.Columns[req]; !found {
			result.Issues = append(result.Issues, fmt.Sprintf("Missing required column: %s", req))
		}
	}

	// Check for low confidence warnings
	for colType, col := range result.Columns {
		if col.Confidence < 0.85 {
			result.Issues = append(result.Issues,
				fmt.Sprintf("Low confidence for %s: %.0f%% (column: %s)",
					colType, col.Confidence*100, col.OriginalName))
		}
	}

	// Calculate overall confidence
	result.Confidence = calculateOverallConfidence(result.Columns)

	return result
}

// detectColumnType detects the type of a single column
func detectColumnType(header string, values []string, colIndex int, threshold float64) *DetectedColumn {
	normalized := normalizeHeader(header)

	bestMatch := &DetectedColumn{
		OriginalName: header,
		Index:        colIndex,
		Confidence:   0.0,
	}

	// Try each column type
	for colType, pattern := range columnPatterns {
		confidence := 0.0

		// 1. Check exact matches (fastest path)
		for _, exact := range pattern.ExactMatches {
			if normalized == exact {
				confidence = 1.0
				break
			}
		}

		// 2. If no exact match, try fuzzy matching
		if confidence < 1.0 {
			for _, fuzzy := range pattern.FuzzyPatterns {
				score := fuzzyMatch(normalized, fuzzy)
				if score > confidence {
					confidence = score
				}
			}
		}

		// 3. Validate with actual data if reasonable confidence
		if confidence > 0.5 {
			// Sample validation (use subset for performance on large datasets)
			sampleSize := min(len(values), 100) // Only validate first 100 rows
			sample := values[:sampleSize]

			dataConfidence := pattern.Validator(sample)

			// Weighted average: 40% header match + 60% data validation
			confidence = (confidence * 0.4) + (dataConfidence * 0.6)
		}

		// Keep best match
		if confidence > bestMatch.Confidence {
			bestMatch.ColumnType = colType
			bestMatch.Confidence = confidence
		}
	}

	// Only return if confidence above threshold
	if bestMatch.Confidence >= threshold {
		return bestMatch
	}

	return nil
}

// normalizeHeader normalizes a header for comparison (cached for performance)
func normalizeHeader(s string) string {
	sb := getStringBuilder()
	defer putStringBuilder(sb)

	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// fuzzyMatch computes similarity using optimized Levenshtein distance
func fuzzyMatch(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Quick length check
	lenDiff := len(s1) - len(s2)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 1.0
	}

	// If length difference is too large, skip expensive calculation
	if float64(lenDiff)/float64(maxLen) > 0.5 {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	return 1.0 - (float64(distance) / float64(maxLen))
}

// levenshteinDistance computes edit distance (optimized with single array)
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Use single array optimization (space: O(n) instead of O(m*n))
	prev := make([]int, len(s2)+1)
	for i := range prev {
		prev[i] = i
	}

	for i := 1; i <= len(s1); i++ {
		curr := i
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			temp := min(
				prev[j]+1,      // deletion
				curr+1,         // insertion
				prev[j-1]+cost, // substitution
			)

			prev[j-1] = curr
			curr = temp
		}
		prev[len(s2)] = curr
	}

	return prev[len(s2)]
}

// Validators - optimized for performance

func validateSKU(values []string) float64 {
	valid := 0
	total := 0

	for _, v := range values {
		if v == "" {
			continue
		}
		total++

		// SKUs are typically alphanumeric with dashes/underscores, reasonably short
		if len(v) > 0 && len(v) < 100 && alphanumericRegex.MatchString(v) {
			valid++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(valid) / float64(total)
}

func validateName(values []string) float64 {
	valid := 0
	total := 0

	for _, v := range values {
		if v == "" {
			continue
		}
		total++

		// Names should be non-empty, reasonable length
		if len(v) >= 3 && len(v) <= 500 {
			valid++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(valid) / float64(total)
}

func validatePrice(values []string) float64 {
	valid := 0
	total := 0

	for _, v := range values {
		if v == "" {
			total++ // Count empty as valid (price might be missing)
			valid++
			continue
		}
		total++

		// Clean and parse price
		cleaned := cleanPriceString(v)
		if _, err := strconv.ParseFloat(cleaned, 64); err == nil {
			valid++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(valid) / float64(total)
}

func validateStock(values []string) float64 {
	valid := 0
	total := 0

	for _, v := range values {
		if v == "" {
			total++
			valid++
			continue
		}
		total++

		trimmed := strings.TrimSpace(v)

		// Stock can be integer or text like "Ex-Stock", "Available", "Sold out"
		if _, err := strconv.Atoi(trimmed); err == nil {
			valid++
		} else {
			// Check for common stock status text
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "stock") ||
				strings.Contains(lower, "available") ||
				strings.Contains(lower, "arrival") ||
				strings.Contains(lower, "sold") ||
				strings.Contains(lower, "offer") ||
				strings.Contains(lower, "expected") ||
				strings.Contains(lower, "limited") {
				valid++
			}
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(valid) / float64(total)
}

func validateImage(values []string) float64 {
	valid := 0
	total := 0

	for _, v := range values {
		if v == "" {
			continue
		}
		total++

		// Images are typically URLs or file paths
		lower := strings.ToLower(v)
		if strings.HasPrefix(lower, "http") ||
			strings.HasPrefix(lower, "/") ||
			strings.Contains(lower, ".jpg") ||
			strings.Contains(lower, ".png") ||
			strings.Contains(lower, ".jpeg") {
			valid++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(valid) / float64(total)
}

// cleanPriceString removes currency symbols and formatting
func cleanPriceString(s string) string {
	sb := getStringBuilder()
	defer putStringBuilder(sb)

	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// calculateOverallConfidence computes overall confidence from detected columns
func calculateOverallConfidence(columns map[string]DetectedColumn) float64 {
	if len(columns) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, col := range columns {
		totalConfidence += col.Confidence
	}

	return totalConfidence / float64(len(columns))
}

// Helper functions

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
