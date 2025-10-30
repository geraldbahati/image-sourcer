package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/geraldbahati/image-sourcer/internal/config"
	"github.com/geraldbahati/image-sourcer/internal/state"
)

func newStatsCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show upload statistics",
		Long:  "Display statistics about previously uploaded products",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats(configFile)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "config.yaml", "Path to config file")

	return cmd
}

func runStats(configFile string) error {
	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.State.Enabled {
		return fmt.Errorf("state tracking is disabled in config")
	}

	// Load state
	tracker := state.NewTracker(cfg.State.File)
	if err := tracker.Load(); err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Get statistics
	uploadedCount := tracker.GetUploadedCount()
	lastUpload := tracker.GetLastUploadTime()
	products := tracker.GetAllUploadedProducts()

	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	
	fmt.Println()
	fmt.Printf("%s\n", cyan("📊 Upload Statistics"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if uploadedCount == 0 {
		fmt.Println("No products uploaded yet.")
		fmt.Println()
		return nil
	}

	fmt.Printf("Total products uploaded: %s\n", green(fmt.Sprintf("%d", uploadedCount)))
	
	if !lastUpload.IsZero() {
		fmt.Printf("Last upload: %s\n", lastUpload.Format("2006-01-02 15:04:05"))
	}

	// Count total images
	totalImages := 0
	for _, product := range products {
		totalImages += len(product.Images)
	}
	fmt.Printf("Total images: %s\n", green(fmt.Sprintf("%d", totalImages)))

	fmt.Println("\nRecent products:")
	// Show last 10 products
	count := min(10, len(products))
	for i := 0; i < count; i++ {
		p := products[i]
		fmt.Printf("  • %s - %s (%d images)\n", 
			p.PartNumber, 
			p.ProductName,
			len(p.Images))
	}

	if len(products) > 10 {
		fmt.Printf("\n  ... and %d more\n", len(products)-10)
	}

	fmt.Println()
	return nil
}
