package cloudinary

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/geraldbahati/image-sourcer/internal/csv"
)

// UploadResult represents the result of an image upload
type UploadResult struct {
	Success        bool
	ProductImage   csv.ProductImage
	CloudinaryURL  string
	PublicID       string
	ThumbnailURL   string
	MainURL        string
	ZoomURL        string
	Format         string
	Width          int
	Height         int
	Bytes          int
	Error          string
	UploadedAt     time.Time
}

// Uploader handles image uploads to Cloudinary
type Uploader struct {
	client *Client
}

// NewUploader creates a new uploader
func NewUploader(client *Client) *Uploader {
	return &Uploader{
		client: client,
	}
}

// Upload uploads a single image to Cloudinary
func (u *Uploader) Upload(ctx context.Context, productImage csv.ProductImage) UploadResult {
	result := UploadResult{
		Success:      false,
		ProductImage: productImage,
		UploadedAt:   time.Now(),
	}

	// Generate public ID based on part number and image index
	publicID := u.generatePublicID(productImage)
	
	// Build upload parameters
	params := u.buildUploadParams(publicID)

	// Perform upload
	uploadResult, err := u.client.GetUploadAPI().Upload(ctx, productImage.ImagePath, params)
	if err != nil {
		result.Error = fmt.Sprintf("upload failed: %v", err)
		return result
	}

	// Populate result
	result.Success = true
	result.CloudinaryURL = uploadResult.SecureURL
	result.PublicID = uploadResult.PublicID
	result.Format = uploadResult.Format
	result.Width = uploadResult.Width
	result.Height = uploadResult.Height
	result.Bytes = uploadResult.Bytes

	// Generate transformation URLs
	result.ThumbnailURL = u.generateTransformationURL(uploadResult.PublicID, "thumbnail")
	result.MainURL = u.generateTransformationURL(uploadResult.PublicID, "product_main")
	result.ZoomURL = u.generateTransformationURL(uploadResult.PublicID, "product_zoom")

	return result
}

// generatePublicID creates a public ID for the image
func (u *Uploader) generatePublicID(productImage csv.ProductImage) string {
	// Get base filename without extension
	baseFilename := filepath.Base(productImage.ImagePath)
	ext := filepath.Ext(baseFilename)
	nameWithoutExt := baseFilename[:len(baseFilename)-len(ext)]

	// Format: products/{part_number}/{part_number}_{filename}
	folder := u.client.config.Cloudinary.Folder
	publicID := fmt.Sprintf("%s/%s/%s_%s", 
		folder, 
		productImage.PartNumber, 
		productImage.PartNumber,
		nameWithoutExt,
	)

	return publicID
}

// buildUploadParams builds the upload parameters
func (u *Uploader) buildUploadParams(publicID string) uploader.UploadParams {
	cfg := u.client.config

	// Helper function to create bool pointers
	boolPtr := func(b bool) *bool { return &b }

	params := uploader.UploadParams{
		PublicID:       publicID,
		Overwrite:      boolPtr(true),
		ResourceType:   "image",
		UniqueFilename: boolPtr(false),
		UseFilename:    boolPtr(false),
	}

	// Add transformations if configured
	if cfg.Transformations.MaxWidth > 0 || cfg.Transformations.MaxHeight > 0 {
		transformation := fmt.Sprintf("c_limit,w_%d,h_%d,f_%s,q_%s",
			cfg.Transformations.MaxWidth,
			cfg.Transformations.MaxHeight,
			cfg.Transformations.Format,
			cfg.Transformations.Quality,
		)
		params.Transformation = transformation
	}

	// Add eager transformations as string
	if len(cfg.Eager) > 0 {
		eagerTransformations := ""
		for i, et := range cfg.Eager {
			if i > 0 {
				eagerTransformations += "|"
			}
			eagerTransformations += et.Transformation
		}
		params.Eager = eagerTransformations
	}

	return params
}

// generateTransformationURL generates a URL with specific transformation
func (u *Uploader) generateTransformationURL(publicID, transformationName string) string {
	cfg := u.client.config
	
	// Find the transformation by name
	var transformation string
	for _, et := range cfg.Eager {
		if et.Name == transformationName {
			transformation = et.Transformation
			break
		}
	}

	if transformation == "" {
		// Fallback to default transformations
		switch transformationName {
		case "thumbnail":
			transformation = "w_300,h_300,c_fill,f_auto,q_auto"
		case "product_main":
			transformation = "w_1000,h_1000,c_pad,b_white,f_auto,q_auto"
		case "product_zoom":
			transformation = "w_2000,h_2000,c_fit,f_auto,q_auto:good"
		}
	}

	// Build URL
	baseURL := fmt.Sprintf("https://res.cloudinary.com/%s/image/upload", cfg.Cloudinary.CloudName)
	if transformation != "" {
		return fmt.Sprintf("%s/%s/%s", baseURL, transformation, publicID)
	}
	return fmt.Sprintf("%s/%s", baseURL, publicID)
}

// UploadWithRetry uploads an image with retry logic
func (u *Uploader) UploadWithRetry(ctx context.Context, productImage csv.ProductImage) UploadResult {
	cfg := u.client.config
	maxAttempts := cfg.Upload.RetryAttempts + 1 // +1 for initial attempt
	retryDelay := cfg.Upload.RetryDelay

	var result UploadResult
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-ctx.Done():
				result.Error = "upload cancelled"
				return result
			case <-time.After(retryDelay * time.Duration(attempt)): // Exponential backoff
			}
		}

		// Create context with timeout
		uploadCtx, cancel := context.WithTimeout(ctx, cfg.Upload.Timeout)
		result = u.Upload(uploadCtx, productImage)
		cancel()

		if result.Success {
			return result
		}

		// Check if error is retryable
		if !isRetryableError(result.Error) {
			break
		}
	}

	return result
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(errorMsg string) bool {
	// Network errors, timeouts, and rate limits are retryable
	// File not found, invalid image, etc. are not retryable
	retryablePatterns := []string{
		"timeout",
		"connection",
		"network",
		"rate limit",
		"temporary",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		indexOf(s, substr) >= 0))
}

// indexOf finds the index of substring in string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
