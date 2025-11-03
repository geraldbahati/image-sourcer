package excel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

// ParserConfig holds configuration for the Excel parser
type ParserConfig struct {
	MaxWorkers      int  // Maximum concurrent file parsers (default: 4)
	MaxSheetWorkers int  // Maximum concurrent sheet parsers per file (default: 2)
	BufferSize      int  // Channel buffer size (default: 10)
	SkipEmptyRows   bool // Skip rows with all empty cells (default: true)
}

// DefaultParserConfig returns default configuration optimized for performance
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		MaxWorkers:      4,  // Process 4 files concurrently
		MaxSheetWorkers: 2,  // Process 2 sheets per file concurrently
		BufferSize:      10,
		SkipEmptyRows:   true,
	}
}

// Parser handles concurrent Excel file parsing
type Parser struct {
	config ParserConfig
}

// NewParser creates a new parser with the given configuration
func NewParser(config ParserConfig) *Parser {
	return &Parser{config: config}
}

// ParseFiles parses multiple Excel files concurrently
func (p *Parser) ParseFiles(ctx context.Context, filePaths []string) (*ParseResult, error) {
	result := &ParseResult{
		Files:  make([]*ExcelFile, 0, len(filePaths)),
		Errors: make([]ParseError, 0),
	}

	// Create worker pool for file parsing
	fileChan := make(chan string, p.config.BufferSize)
	var wg sync.WaitGroup

	// Start file parser workers
	for i := 0; i < p.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.fileWorker(ctx, fileChan, result)
		}()
	}

	// Send files to workers
	go func() {
		for _, filePath := range filePaths {
			select {
			case <-ctx.Done():
				return
			case fileChan <- filePath:
			}
		}
		close(fileChan)
	}()

	// Wait for all workers to finish
	wg.Wait()

	return result, nil
}

// fileWorker processes files from the channel
func (p *Parser) fileWorker(ctx context.Context, fileChan <-chan string, result *ParseResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case filePath, ok := <-fileChan:
			if !ok {
				return // Channel closed
			}

			// Parse the file
			excelFile, err := p.parseFile(ctx, filePath)
			if err != nil {
				result.AddError(ParseError{
					FilePath: filePath,
					Message:  err.Error(),
				})
				continue
			}

			result.AddFile(excelFile)
		}
	}
}

// parseFile parses a single Excel file
func (p *Parser) parseFile(ctx context.Context, filePath string) (*ExcelFile, error) {
	// Open Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail
		}
	}()

	// Get all sheet names
	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, fmt.Errorf("no sheets found in file")
	}

	excelFile := &ExcelFile{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		Sheets:   make([]ExcelSheet, 0, len(sheetNames)),
	}

	// Parse sheets concurrently
	sheetChan := make(chan string, len(sheetNames))
	resultChan := make(chan ExcelSheet, len(sheetNames))
	errorChan := make(chan error, len(sheetNames))

	var wg sync.WaitGroup

	// Start sheet parser workers
	for i := 0; i < p.config.MaxSheetWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sheetName := range sheetChan {
				select {
				case <-ctx.Done():
					return
				default:
					sheet, err := p.parseSheet(f, sheetName)
					if err != nil {
						errorChan <- fmt.Errorf("sheet %s: %w", sheetName, err)
						continue
					}
					resultChan <- sheet
				}
			}
		}()
	}

	// Send sheet names to workers
	go func() {
		for _, sheetName := range sheetNames {
			sheetChan <- sheetName
		}
		close(sheetChan)
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	for sheet := range resultChan {
		excelFile.Sheets = append(excelFile.Sheets, sheet)
	}

	// Check for errors (non-blocking)
	select {
	case err := <-errorChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	return excelFile, nil
}

// parseSheet parses a single sheet (optimized for performance)
func (p *Parser) parseSheet(f *excelize.File, sheetName string) (ExcelSheet, error) {
	// Get all rows at once (more efficient than row-by-row)
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		return ExcelSheet{}, fmt.Errorf("invalid sheet: %w", err)
	}

	// Find header row (may not be row 1 due to title rows)
	headerRowIndex := -1
	var headers []string

	maxRows := 10
	if len(rows) < maxRows {
		maxRows = len(rows)
	}

	for i := 0; i < maxRows; i++ { // Check first 10 rows
		if isValidHeaderRow(rows[i]) {
			headerRowIndex = i
			headers = normalizeHeaders(rows[i])
			break
		}
	}

	if headerRowIndex == -1 || len(headers) == 0 {
		return ExcelSheet{}, fmt.Errorf("no headers found")
	}

	// Pre-allocate slice for data rows (performance optimization)
	estimatedRows := len(rows) - headerRowIndex - 1
	dataRows := make([]ExcelRow, 0, estimatedRows)

	// Process data rows (skip rows before and including header)
	for i, row := range rows[headerRowIndex+1:] {
		// Skip empty rows if configured
		if p.config.SkipEmptyRows && isEmptyRow(row) {
			continue
		}

		// Get a map from the pool for reuse
		cells := make(map[string]string, len(headers))

		// Populate cells
		for j, cellValue := range row {
			if j < len(headers) {
				trimmed := strings.TrimSpace(cellValue)
				if trimmed != "" {
					cells[headers[j]] = trimmed
				}
			}
		}

		// Only add row if it has data
		if len(cells) > 0 {
			dataRows = append(dataRows, ExcelRow{
				RowNumber: headerRowIndex + i + 2, // Excel row number (1-indexed)
				Cells:     cells,
			})
		}
	}

	return ExcelSheet{
		Name:     sheetName,
		Headers:  headers,
		Rows:     dataRows,
		RowCount: len(dataRows),
	}, nil
}

// isEmptyRow checks if all cells in a row are empty
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// isValidHeaderRow checks if a row looks like a header row
func isValidHeaderRow(row []string) bool {
	if len(row) == 0 {
		return false
	}

	// Count non-empty cells
	nonEmptyCount := 0
	for _, cell := range row {
		trimmed := strings.TrimSpace(cell)
		if trimmed != "" {
			nonEmptyCount++

			// Headers typically have certain keywords
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "part") ||
				strings.Contains(lower, "sku") ||
				strings.Contains(lower, "description") ||
				strings.Contains(lower, "price") ||
				strings.Contains(lower, "availability") ||
				strings.Contains(lower, "name") ||
				strings.Contains(lower, "product") {
				return true // Likely a header row
			}
		}
	}

	// If at least 2 non-empty cells, might be a header
	return nonEmptyCount >= 2
}

// normalizeHeaders cleans up header names (trim spaces, etc.)
func normalizeHeaders(row []string) []string {
	normalized := make([]string, 0, len(row))
	for _, cell := range row {
		trimmed := strings.TrimSpace(cell)
		// Keep even empty headers to maintain column positions
		normalized = append(normalized, trimmed)
	}
	return normalized
}

// FindExcelFiles finds all .xlsx files in a directory (recursive)
func FindExcelFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Single file
	if !info.IsDir() {
		if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("not an Excel file: %s", path)
	}

	// Directory - find all .xlsx files
	var files []string
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".xlsx") {
			// Skip temporary Excel files (start with ~$)
			if !strings.HasPrefix(filepath.Base(p), "~$") {
				files = append(files, p)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Excel files found in: %s", path)
	}

	return files, nil
}

// ParseExcelFiles is a convenience function to parse Excel files with default config
func ParseExcelFiles(ctx context.Context, paths []string) (*ParseResult, error) {
	parser := NewParser(DefaultParserConfig())
	return parser.ParseFiles(ctx, paths)
}
