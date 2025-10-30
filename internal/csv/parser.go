package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProductImage represents a single product image entry from the CSV
type ProductImage struct {
	PartNumber  string
	ProductName string
	ImagePath   string
	IsPrimary   bool
	RowNumber   int // For error report
}

// Product represents a product with all its images
type Product struct {
	PartNumber  string
	ProductName string
	Images      []ProductImage
}

// Manifest represents the parsed CSV data
type Manifest struct {
	Products map[string]*Product //keyed by part_number
	Images   []ProductImage      // all images in order
}

// Parser handles CSV parsing
type Parser struct {
	filePath string
}

// NewParser creates a new CSV parser
func NewParser(filePath string) *Parser {
	return &Parser{filePath: filePath}
}

// Parse reads and parses the CSV file
func (p *Parser) Parse() (*Manifest, error) {
	// Open CSV file
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Parser header
	header := records[0]
	columnIndexes, err := parseHeader(header)
	if err != nil {
		return nil, err
	}

	// Parse records
	manifest := &Manifest{
		Products: make(map[string]*Product),
		Images:   make([]ProductImage, 0),
	}

	for i, record := range records[1:] {
		rowNumber := i + 2 // +2 because we skip header (1) and arrays are 0-indexed

		if len(record) == 0 || (len(record) == 1 && record[0] == "") {
			// Skip empty rows
			continue
		}

		image, err := parseRecord(record, columnIndexes, rowNumber)
		if err != nil {
			return nil, fmt.Errorf("error on row %d: %w", rowNumber, err)
		}

		// Add to manifest
		manifest.Images = append(manifest.Images, image)

		// Group by product
		if product, exists := manifest.Products[image.PartNumber]; exists {
			product.Images = append(product.Images, image)
		} else {
			manifest.Products[image.PartNumber] = &Product{
				PartNumber:  image.PartNumber,
				ProductName: image.ProductName,
				Images:      []ProductImage{image},
			}
		}
	}

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return manifest, nil
}

// columnIndexes holds the index of each column in the CSV
type columnIndexes struct {
	partNumber  int
	productName int
	imagePath   int
	isPrimary   int
}

// parseHeader parses the CSV header and returns column indexes
func parseHeader(header []string) (*columnIndexes, error) {
	indexes := &columnIndexes{
		partNumber:  -1,
		productName: -1,
		imagePath:   -1,
		isPrimary:   -1,
	}

	for i, col := range header {
		col = strings.TrimSpace(strings.ToLower(col))
		switch col {
		case "part_number", "partnumber", "part number":
			indexes.partNumber = i
		case "product_name", "productname", "product name", "name":
			indexes.productName = i
		case "image_path", "imagepath", "image path", "path":
			indexes.imagePath = i
		case "is_primary", "isprimary", "primary":
			indexes.isPrimary = i
		}
	}

	// Validate required columns
	if indexes.partNumber == -1 {
		return nil, fmt.Errorf("required column 'part_number' not found in CSV header")
	}
	if indexes.productName == -1 {
		return nil, fmt.Errorf("required column 'product_name' not found in CSV header")
	}
	if indexes.imagePath == -1 {
		return nil, fmt.Errorf("required column 'image_path' not found in CSV header")
	}

	// is_primary is optional, defaults to false

	return indexes, nil
}

// parseRecord parses a single CSV record
func parseRecord(record []string, indexes *columnIndexes, rowNumber int) (ProductImage, error) {
	image := ProductImage{
		RowNumber: rowNumber,
		IsPrimary: false, // default
	}

	// Validate record length
	maxIndex := max(indexes.partNumber, indexes.productName, indexes.imagePath)
	if indexes.isPrimary > maxIndex {
		maxIndex = indexes.isPrimary
	}
	if len(record) <= maxIndex {
		return image, fmt.Errorf("insufficient columns in record (expected at least %d, got %d)", maxIndex+1, len(record))
	}

	// Parse part_number
	image.PartNumber = strings.TrimSpace(record[indexes.partNumber])
	if image.PartNumber == "" {
		return image, fmt.Errorf("part_number is empty")
	}

	// Parse product_name
	image.ProductName = strings.TrimSpace(record[indexes.productName])
	if image.ProductName == "" {
		return image, fmt.Errorf("product_name is empty")
	}

	// Parse image_path
	image.ImagePath = strings.TrimSpace(record[indexes.imagePath])
	if image.ImagePath == "" {
		return image, fmt.Errorf("image_path is empty")
	}

	// Make path absolute if it's not already
	if !filepath.IsAbs(image.ImagePath) {
		absPath, err := filepath.Abs(image.ImagePath)
		if err == nil {
			image.ImagePath = absPath
		}
	}

	// Parse is_primary (optional)
	if indexes.isPrimary != -1 && indexes.isPrimary < len(record) {
		primaryStr := strings.ToLower(strings.TrimSpace(record[indexes.isPrimary]))
		if primaryStr == "true" || primaryStr == "1" || primaryStr == "yes" {
			image.IsPrimary = true
		}
	}

	return image, nil
}

// Validate checks if the manifest is valid
func (m *Manifest) Validate() error {
	if len(m.Products) == 0 {
		return fmt.Errorf("no products found in CSV")
	}

	if len(m.Images) == 0 {
		return fmt.Errorf("no images found in CSV")
	}

	// Validate each product has at least one image
	for partNumber, product := range m.Products {
		if len(product.Images) == 0 {
			return fmt.Errorf("product %s has no images", partNumber)
		}
	}

	return nil
}

// GetProductCount returns the number of unique products
func (m *Manifest) GetProductCount() int {
	return len(m.Products)
}

// GetImageCount returns the total number of images
func (m *Manifest) GetImageCount() int {
	return len(m.Images)
}

// GetProductList returns a list of all products
func (m *Manifest) GetProductList() []*Product {
	products := make([]*Product, 0, len(m.Products))
	for _, product := range m.Products {
		products = append(products, product)
	}
	return products
}

