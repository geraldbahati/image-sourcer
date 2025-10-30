package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/geraldbahati/image-sourcer/internal/csv"
	"github.com/geraldbahati/image-sourcer/internal/image"
)

func newValidateCmd() *cobra.Command {
	var csvFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate CSV file and images",
		Long:  "Validate the CSV manifest and check if all image files exist and are valid",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(csvFile)
		},
	}

	cmd.Flags().StringVarP(&csvFile, "csv", "c", "", "Path to CSV file (required)")
	cmd.MarkFlagRequired("csv")

	return cmd
}

func runValidate(csvFile string) error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println("🔍 Validating CSV and images...")
	fmt.Println()

	// Parse CSV
	parser := csv.NewParser(csvFile)
	manifest, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("CSV validation failed: %w", err)
	}

	fmt.Printf("✓ CSV parsed successfully\n")
	fmt.Printf("  • Products: %d\n", manifest.GetProductCount())
	fmt.Printf("  • Images: %d\n\n", manifest.GetImageCount())

	// Validate images
	fmt.Println("Checking image files...")
	validator := image.DefaultValidator()
	
	validCount := 0
	invalidCount := 0
	warningCount := 0

	for _, img := range manifest.Images {
		result := validator.Validate(img.ImagePath)
		
		if result.Valid {
			validCount++
			if result.Width < 500 || result.Height < 500 {
				warningCount++
				fmt.Printf("  %s %s: %dx%d (low resolution)\n", 
					yellow("⚠"), 
					img.PartNumber,
					result.Width,
					result.Height)
			}
		} else {
			invalidCount++
			fmt.Printf("  %s %s: %s\n", 
				red("✗"), 
				img.PartNumber,
				result.Error)
		}
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	if invalidCount == 0 {
		fmt.Printf("%s All images validated successfully!\n", green("✅"))
		fmt.Printf("  • Valid: %d\n", validCount)
		if warningCount > 0 {
			fmt.Printf("  • Warnings: %d (low resolution)\n", warningCount)
		}
	} else {
		fmt.Printf("%s Validation failed\n", red("✗"))
		fmt.Printf("  • Valid: %d\n", validCount)
		fmt.Printf("  • Invalid: %d\n", invalidCount)
		if warningCount > 0 {
			fmt.Printf("  • Warnings: %d\n", warningCount)
		}
		return fmt.Errorf("found %d invalid images", invalidCount)
	}

	fmt.Println()
	return nil
}
