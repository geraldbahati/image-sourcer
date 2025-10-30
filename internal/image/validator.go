package image

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult holds the result of image validation
type ValidationResult struct {
	Valid    bool
	FilePath string
	Error    string
	FileSize int64
	Width    int
	Height   int
	Format   string
}

// Validator handles image validation
type Validator struct {
	maxFileSize int64 // in bytes
	minWidth    int
	minHeight   int
}

// NewValidator creates a new image validator
func NewValidator(maxFileSizeMB int, minWidth, minHeight int) *Validator {
	return &Validator{
		maxFileSize: int64(maxFileSizeMB) * 1024 * 1024, // Convert MB to bytes
		minWidth:    minWidth,
		minHeight:   minHeight,
	}
}

// DefaultValidator creates a validator with default settings
func DefaultValidator() *Validator {
	return NewValidator(
		50,   // 50MB max file size
		100,  // 100px minimum width
		100,  // 100px minimum height
	)
}

// Validate validates a single image file
func (v *Validator) Validate(filePath string) ValidationResult {
	result := ValidationResult{
		FilePath: filePath,
		Valid:    false,
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Error = "file does not exist"
			return result
		}
		result.Error = fmt.Sprintf("error accessing file: %v", err)
		return result
	}

	// Check if it's a regular file (not a directory)
	if fileInfo.IsDir() {
		result.Error = "path is a directory, not a file"
		return result
	}

	// Get file size
	result.FileSize = fileInfo.Size()

	// Check file size
	if result.FileSize == 0 {
		result.Error = "file is empty"
		return result
	}

	if v.maxFileSize > 0 && result.FileSize > v.maxFileSize {
		result.Error = fmt.Sprintf("file size (%d bytes) exceeds maximum (%d bytes)", result.FileSize, v.maxFileSize)
		return result
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !isValidImageExtension(ext) {
		result.Error = fmt.Sprintf("unsupported file extension: %s (supported: .jpg, .jpeg, .png, .gif, .webp)", ext)
		return result
	}

	// Open and decode image to verify it's valid and get dimensions
	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("error opening file: %v", err)
		return result
	}
	defer file.Close()

	// Decode image
	img, format, err := image.DecodeConfig(file)
	if err != nil {
		result.Error = fmt.Sprintf("invalid image file or unsupported format: %v", err)
		return result
	}

	result.Width = img.Width
	result.Height = img.Height
	result.Format = format

	// Check dimensions
	if v.minWidth > 0 && img.Width < v.minWidth {
		result.Error = fmt.Sprintf("image width (%dpx) is less than minimum (%dpx)", img.Width, v.minWidth)
		return result
	}

	if v.minHeight > 0 && img.Height < v.minHeight {
		result.Error = fmt.Sprintf("image height (%dpx) is less than minimum (%dpx)", img.Height, v.minHeight)
		return result
	}

	// All checks passed
	result.Valid = true
	return result
}

// ValidateBatch validates multiple image files
func (v *Validator) ValidateBatch(filePaths []string) []ValidationResult {
	results := make([]ValidationResult, len(filePaths))
	for i, path := range filePaths {
		results[i] = v.Validate(path)
	}
	return results
}

// isValidImageExtension checks if the file extension is a supported image format
func isValidImageExtension(ext string) bool {
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
	}
	return validExtensions[ext]
}

// FormatFileSize formats file size in human-readable format
func FormatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
