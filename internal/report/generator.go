package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/geraldbahati/image-sourcer/internal/cloudinary"
)

// Generator handles report generation
type Generator struct {
	reportDir string
}

// NewGenerator creates a new report generator
func NewGenerator(reportDir string) *Generator {
	return &Generator{
		reportDir: reportDir,
	}
}

// Generate generates a CSV report from upload results
func (g *Generator) Generate(results []cloudinary.UploadResult) (string, error) {
	// Ensure report directory exists
	if err := os.MkdirAll(g.reportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate report filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("upload-report_%s.csv", timestamp)
	reportPath := filepath.Join(g.reportDir, filename)

	// Create CSV file
	file, err := os.Create(reportPath)
	if err != nil {
		return "", fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"part_number",
		"product_name",
		"original_path",
		"cloudinary_url",
		"thumbnail_url",
		"main_url",
		"zoom_url",
		"public_id",
		"format",
		"width",
		"height",
		"size_bytes",
		"status",
		"error_message",
		"uploaded_at",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write results
	for _, result := range results {
		status := "success"
		if !result.Success {
			status = "failed"
		}

		row := []string{
			result.ProductImage.PartNumber,
			result.ProductImage.ProductName,
			result.ProductImage.ImagePath,
			result.CloudinaryURL,
			result.ThumbnailURL,
			result.MainURL,
			result.ZoomURL,
			result.PublicID,
			result.Format,
			fmt.Sprintf("%d", result.Width),
			fmt.Sprintf("%d", result.Height),
			fmt.Sprintf("%d", result.Bytes),
			status,
			result.Error,
			result.UploadedAt.Format(time.RFC3339),
		}

		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return reportPath, nil
}

// GenerateSummary generates a summary of upload results
func (g *Generator) GenerateSummary(results []cloudinary.UploadResult) Summary {
	summary := Summary{
		TotalImages:    len(results),
		SuccessCount:   0,
		FailedCount:    0,
		TotalBytes:     0,
		Products:       make(map[string]int),
		Errors:         make(map[string]int),
	}

	for _, result := range results {
		if result.Success {
			summary.SuccessCount++
			summary.TotalBytes += int64(result.Bytes)
		} else {
			summary.FailedCount++
			if result.Error != "" {
				summary.Errors[result.Error]++
			}
		}

		// Count images per product
		summary.Products[result.ProductImage.PartNumber]++
	}

	summary.ProductCount = len(summary.Products)
	return summary
}

// Summary represents upload summary statistics
type Summary struct {
	TotalImages  int
	SuccessCount int
	FailedCount  int
	ProductCount int
	TotalBytes   int64
	Products     map[string]int // part_number -> image count
	Errors       map[string]int // error message -> count
}

// SuccessRate returns the success rate as a percentage
func (s *Summary) SuccessRate() float64 {
	if s.TotalImages == 0 {
		return 0
	}
	return float64(s.SuccessCount) / float64(s.TotalImages) * 100
}

// FormatSize returns the total size in human-readable format
func (s *Summary) FormatSize() string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	bytes := s.TotalBytes
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
