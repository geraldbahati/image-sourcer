package convex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// Client is a Convex HTTP client with connection pooling and retry logic
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     ImportConfig
}

// NewClient creates a new Convex client with optimized settings
func NewClient(config ImportConfig) *Client {
	// Create HTTP client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	return &Client{
		baseURL: config.ConvexURL,
		apiKey:  config.ConvexAPIKey,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.RequestTimeout,
		},
		config: config,
	}
}

// Mutation executes a Convex mutation with retry logic
func (c *Client) Mutation(ctx context.Context, path string, args map[string]interface{}) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.config.RetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := c.executeMutation(ctx, path, args)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on context cancellation or certain errors
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Don't retry on validation errors (4xx)
		if convexErr, ok := err.(*ConvexError); ok {
			if convexErr.Code == "INVALID_ARGUMENT" || convexErr.Code == "ALREADY_EXISTS" {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.config.RetryAttempts+1, lastErr)
}

// executeMutation performs a single mutation request
func (c *Client) executeMutation(ctx context.Context, path string, args map[string]interface{}) (interface{}, error) {
	// Prepare request payload
	payload := ConvexMutationRequest{
		Path: path,
		Args: args,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/mutation", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Convex %s", c.apiKey))
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var mutationResp ConvexMutationResponse
	if err := json.Unmarshal(body, &mutationResp); err != nil {
		// DEBUG: Print raw body if parse fails
		fmt.Printf("DEBUG: Failed to parse response for %s. Raw body: %s\n", path, string(body))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// DEBUG: Print first response to understand structure
	if path == "products/createProduct:create" {
		fmt.Printf("DEBUG: Sample response for %s:\n  Body: %s\n  Value: %+v\n  Error: %+v\n  Status: %s\n  ErrorMessage: %s\n",
			path, string(body), mutationResp.Value, mutationResp.Error, mutationResp.Status, mutationResp.ErrorMessage)
	}

	// Check for Convex errors (handles both response formats)
	if mutationResp.Error != nil {
		return nil, mutationResp.Error
	}

	// Alternative error format: {status: "error", errorMessage: "..."}
	if mutationResp.Status == "error" && mutationResp.ErrorMessage != "" {
		return nil, &ConvexError{
			Code:    "VALIDATION_ERROR",
			Message: mutationResp.ErrorMessage,
		}
	}

	return mutationResp.Value, nil
}

// CreateProduct creates a product in Convex
func (c *Client) CreateProduct(ctx context.Context, product ConvexProduct) (string, error) {
	// Convert product struct to map and pass fields directly (not nested)
	// The Convex mutation expects args matching the productSchema, not nested under "product"
	productBytes, err := json.Marshal(product)
	if err != nil {
		return "", fmt.Errorf("failed to marshal product: %w", err)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(productBytes, &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal product to map: %w", err)
	}

	// Remove fields that Convex adds automatically (not in validator)
	delete(args, "createdAt")
	delete(args, "updatedAt")
	delete(args, "lastCurrencyUpdate") // Added by handler, not in input schema
	delete(args, "searchField")        // Computed by handler from name/desc/tags

	// Clean up variant fields that aren't in the Convex schema
	if variants, ok := args["variants"].([]interface{}); ok {
		for _, variant := range variants {
			if v, ok := variant.(map[string]interface{}); ok {
				delete(v, "priceKes")
				delete(v, "originalPriceKes")
			}
		}
	}

	result, err := c.Mutation(ctx, "products/createProduct:create", args)
	if err != nil {
		return "", err
	}

	// Handle object response: {productId, slug, message}
	if resultMap, ok := result.(map[string]interface{}); ok {
		if productID, exists := resultMap["productId"]; exists {
			if id, ok := productID.(string); ok {
				return id, nil
			}
		}
		return "", fmt.Errorf("productId field missing or invalid in response: %+v", resultMap)
	}

	// Fallback: Handle direct string response (for backward compatibility)
	if id, ok := result.(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("unexpected result type: %T, value: %+v", result, result)
}

// CreateCategory creates a category in Convex
func (c *Client) CreateCategory(ctx context.Context, category ConvexCategory) (string, error) {
	// Convert category struct to map and pass fields directly
	categoryBytes, err := json.Marshal(category)
	if err != nil {
		return "", fmt.Errorf("failed to marshal category: %w", err)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(categoryBytes, &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal category to map: %w", err)
	}

	// Remove fields that Convex adds automatically (not in validator)
	delete(args, "createdAt")
	delete(args, "updatedAt")

	result, err := c.Mutation(ctx, "categories:create", args)
	if err != nil {
		return "", err
	}

	// Extract category ID from result
	if id, ok := result.(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("unexpected result type: %T", result)
}

// GetOrCreateCategory gets or creates a category by name
func (c *Client) GetOrCreateCategory(ctx context.Context, name string) (string, error) {
	// Note: categories:getByName doesn't exist in Convex backend
	// Just try to create - if it exists, it will fail and we can handle that

	now := float64(time.Now().UnixMilli())
	category := ConvexCategory{
		Name:      name,
		Slug:      generateSlug(name),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	id, err := c.CreateCategory(ctx, category)
	if err != nil {
		// If category already exists, this will fail
		// For now, we'll return the error - in production you'd query existing categories
		return "", fmt.Errorf("failed to create category %s: %w", name, err)
	}

	return id, nil
}

// CreateProductSpecification creates product specifications
func (c *Client) CreateProductSpecification(ctx context.Context, spec ConvexProductSpecification) (string, error) {
	// Convert spec struct to map and pass fields directly
	specBytes, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal specification: %w", err)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(specBytes, &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal specification to map: %w", err)
	}

	// Remove fields that Convex adds automatically (not in validator)
	delete(args, "createdAt")
	delete(args, "updatedAt")

	result, err := c.Mutation(ctx, "productSpecifications:create", args)
	if err != nil {
		return "", err
	}

	// Extract ID from result
	if id, ok := result.(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("unexpected result type: %T", result)
}

// BatchCreateProducts creates multiple products in a batch (sequential for data integrity)
func (c *Client) BatchCreateProducts(ctx context.Context, products []ConvexProduct) ([]string, []error) {
	ids := make([]string, len(products))
	errors := make([]error, len(products))

	for i, product := range products {
		select {
		case <-ctx.Done():
			errors[i] = ctx.Err()
			continue
		default:
		}

		id, err := c.CreateProduct(ctx, product)
		if err != nil {
			errors[i] = err
		} else {
			ids[i] = id
		}
	}

	return ids, errors
}

// Ping tests the connection to Convex
func (c *Client) Ping(ctx context.Context) error {
	// Use a simple query to test connectivity
	args := map[string]interface{}{}

	_, err := c.Mutation(ctx, "system:ping", args)
	if err != nil {
		// If ping mutation doesn't exist, try a basic request
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create ping request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: HTTP %d", resp.StatusCode)
		}
	}

	return nil
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// generateSlug creates a URL-friendly slug from a string
func generateSlug(s string) string {
	// Simple slug generation - replace spaces with hyphens and lowercase
	// For production, use a proper slug library
	s = fmt.Sprintf("%s", s)
	// This is a placeholder - in production, use gosimple/slug
	return s
}

// Error implements error interface for ConvexError
func (e *ConvexError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
