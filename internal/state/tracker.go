package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// UploadedProduct represents a product that has been uploaded
type UploadedProduct struct {
	PartNumber     string    `json:"part_number"`
	ProductName    string    `json:"product_name"`
	Images         []string  `json:"images"`          // Public IDs
	CloudinaryURLs []string  `json:"cloudinary_urls"`
	UploadedAt     time.Time `json:"uploaded_at"`
}

// State represents the upload state
type State struct {
	LastUpload       time.Time                  `json:"last_upload"`
	UploadedProducts map[string]*UploadedProduct `json:"uploaded_products"` // keyed by part_number
	mu               sync.RWMutex                `json:"-"`
}

// Tracker manages upload state
type Tracker struct {
	filePath string
	state    *State
	mu       sync.Mutex
}

// NewTracker creates a new state tracker
func NewTracker(filePath string) *Tracker {
	return &Tracker{
		filePath: filePath,
		state: &State{
			UploadedProducts: make(map[string]*UploadedProduct),
		},
	}
}

// Load loads the state from disk
func (t *Tracker) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty state
		return nil
	}

	// Read file
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	// Initialize map if nil
	if state.UploadedProducts == nil {
		state.UploadedProducts = make(map[string]*UploadedProduct)
	}

	t.state = &state
	return nil
}

// Save saves the state to disk
func (t *Tracker) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state.mu.Lock()
	t.state.LastUpload = time.Now()
	t.state.mu.Unlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	if err := os.WriteFile(t.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// IsProductUploaded checks if a product has been uploaded
func (t *Tracker) IsProductUploaded(partNumber string) bool {
	t.state.mu.RLock()
	defer t.state.mu.RUnlock()

	_, exists := t.state.UploadedProducts[partNumber]
	return exists
}

// MarkProductUploaded marks a product as uploaded
func (t *Tracker) MarkProductUploaded(partNumber, productName string, publicIDs, cloudinaryURLs []string) {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()

	t.state.UploadedProducts[partNumber] = &UploadedProduct{
		PartNumber:     partNumber,
		ProductName:    productName,
		Images:         publicIDs,
		CloudinaryURLs: cloudinaryURLs,
		UploadedAt:     time.Now(),
	}
}

// GetUploadedProduct gets information about an uploaded product
func (t *Tracker) GetUploadedProduct(partNumber string) (*UploadedProduct, bool) {
	t.state.mu.RLock()
	defer t.state.mu.RUnlock()

	product, exists := t.state.UploadedProducts[partNumber]
	return product, exists
}

// GetUploadedCount returns the number of uploaded products
func (t *Tracker) GetUploadedCount() int {
	t.state.mu.RLock()
	defer t.state.mu.RUnlock()

	return len(t.state.UploadedProducts)
}

// GetLastUploadTime returns the last upload time
func (t *Tracker) GetLastUploadTime() time.Time {
	t.state.mu.RLock()
	defer t.state.mu.RUnlock()

	return t.state.LastUpload
}

// Clear clears all state
func (t *Tracker) Clear() {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()

	t.state.UploadedProducts = make(map[string]*UploadedProduct)
	t.state.LastUpload = time.Time{}
}

// Delete deletes the state file
func (t *Tracker) Delete() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := os.Remove(t.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	t.state = &State{
		UploadedProducts: make(map[string]*UploadedProduct),
	}

	return nil
}

// GetAllUploadedProducts returns all uploaded products
func (t *Tracker) GetAllUploadedProducts() []*UploadedProduct {
	t.state.mu.RLock()
	defer t.state.mu.RUnlock()

	products := make([]*UploadedProduct, 0, len(t.state.UploadedProducts))
	for _, product := range t.state.UploadedProducts {
		products = append(products, product)
	}
	return products
}
