package cloudinary

import (
	"context"
	"fmt"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/geraldbahati/image-sourcer/internal/config"
)

// Client wraps the Cloudinary client
type Client struct {
	cld    *cloudinary.Cloudinary
	config *config.Config
}

// NewClient creates a new Cloudinary client
func NewClient(cfg *config.Config) (*Client, error) {
	// Create Cloudinary instance
	cld, err := cloudinary.NewFromParams(
		cfg.Cloudinary.CloudName,
		cfg.Cloudinary.APIKey,
		cfg.Cloudinary.APISecret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudinary client: %w", err)
	}

	return &Client{
		cld:    cld,
		config: cfg,
	}, nil
}

// GetUploadAPI returns the Cloudinary upload API
func (c *Client) GetUploadAPI() *uploader.API {
	return &c.cld.Upload
}

// GetCloudinary returns the underlying Cloudinary instance
func (c *Client) GetCloudinary() *cloudinary.Cloudinary {
	return c.cld
}

// Ping tests the connection to Cloudinary
func (c *Client) Ping(ctx context.Context) error {
	// Try a simple API call to verify credentials
	_, err := c.cld.Admin.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Cloudinary: %w", err)
	}
	return nil
}

// GetConfig returns the configuration
func (c *Client) GetConfig() *config.Config {
	return c.config
}
