package excel

import (
	"sync"
)

// ExcelFile represents a parsed Excel file with all its sheets
type ExcelFile struct {
	FilePath string
	FileName string
	Sheets   []ExcelSheet
}

// ExcelSheet represents a single sheet in an Excel file
type ExcelSheet struct {
	Name     string
	Headers  []string   // Row 1: column names
	Rows     []ExcelRow // Data rows (pre-allocated for performance)
	RowCount int
}

// ExcelRow represents a single row of data
type ExcelRow struct {
	RowNumber int                // Excel row number (1-indexed)
	Cells     map[string]string  // Header → Value mapping
}

// ParseResult contains the result of parsing multiple Excel files
type ParseResult struct {
	Files         []*ExcelFile
	TotalFiles    int
	TotalSheets   int
	TotalRows     int
	Errors        []ParseError
	mu            sync.RWMutex // Protects concurrent writes
}

// ParseError represents an error that occurred during parsing
type ParseError struct {
	FilePath string
	Sheet    string
	RowNum   int
	Message  string
}

// AddFile safely adds a file to the result (thread-safe)
func (pr *ParseResult) AddFile(file *ExcelFile) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.Files = append(pr.Files, file)
	pr.TotalFiles++
	pr.TotalSheets += len(file.Sheets)
	for _, sheet := range file.Sheets {
		pr.TotalRows += sheet.RowCount
	}
}

// AddError safely adds an error (thread-safe)
func (pr *ParseResult) AddError(err ParseError) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.Errors = append(pr.Errors, err)
}

// GetFiles safely retrieves all files
func (pr *ParseResult) GetFiles() []*ExcelFile {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.Files
}

// GetErrors safely retrieves all errors
func (pr *ParseResult) GetErrors() []ParseError {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.Errors
}

// rowPool is a sync.Pool for reusing row maps (memory optimization)
var rowPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]string, 20) // Pre-allocate for ~20 columns
	},
}

// getRowMap gets a map from the pool
func getRowMap() map[string]string {
	return rowPool.Get().(map[string]string)
}

// putRowMap returns a map to the pool after clearing it
func putRowMap(m map[string]string) {
	// Clear the map before returning to pool
	for k := range m {
		delete(m, k)
	}
	rowPool.Put(m)
}
