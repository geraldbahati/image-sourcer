package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type generateFlags struct {
	imageDir    string
	outputCSV   string
	pattern     string
	interactive bool
}

func newGenerateCmd() *cobra.Command {
	flags := &generateFlags{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate CSV from images folder",
		Long:  "Automatically scan an images folder and generate a CSV manifest file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(flags)
		},
	}

	cmd.Flags().StringVarP(&flags.imageDir, "dir", "d", "", "Directory containing product images (required)")
	cmd.Flags().StringVarP(&flags.outputCSV, "output", "o", "products.csv", "Output CSV file path")
	cmd.Flags().StringVarP(&flags.pattern, "pattern", "p", "", "Part number pattern (regex, e.g. '^[A-Z0-9]+_')")
	cmd.Flags().BoolVarP(&flags.interactive, "interactive", "i", false, "Interactive mode (prompt for product names)")

	cmd.MarkFlagRequired("dir")

	return cmd
}

type productGroup struct {
	PartNumber  string
	ProductName string
	Images      []string
}

func runGenerate(flags *generateFlags) error {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("\n%s\n", cyan("📁 CSV Generator"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Scan directory for images
	fmt.Printf("📂 Scanning directory: %s\n\n", flags.imageDir)

	imageFiles, err := findImageFiles(flags.imageDir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(imageFiles) == 0 {
		return fmt.Errorf("no image files found in directory")
	}

	fmt.Printf("✓ Found %s image files\n\n", green(fmt.Sprintf("%d", len(imageFiles))))

	// Group images by part number
	fmt.Println("🔍 Grouping images by part number...")
	groups := groupImagesByPartNumber(imageFiles, flags.pattern)

	fmt.Printf("✓ Detected %s products\n\n", green(fmt.Sprintf("%d", len(groups))))

	// Show preview
	fmt.Println("Preview of detected products:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, group := range groups {
		if i >= 5 {
			fmt.Printf("\n... and %d more products\n", len(groups)-5)
			break
		}
		fmt.Printf("  • %s - %s (%d images)\n",
			yellow(group.PartNumber),
			group.ProductName,
			len(group.Images))
	}
	fmt.Println()

	// Interactive mode - ask for product names
	if flags.interactive {
		fmt.Println("📝 Enter product names (press Enter to keep auto-detected name):")
		fmt.Println()
		for i := range groups {
			fmt.Printf("[%d/%d] %s\n", i+1, len(groups), groups[i].PartNumber)
			fmt.Printf("  Current: %s\n", groups[i].ProductName)
			fmt.Print("  New name (or Enter to keep): ")

			var input string
			fmt.Scanln(&input)
			if input != "" {
				groups[i].ProductName = input
			}
			fmt.Println()
		}
	}

	// Generate CSV
	fmt.Printf("📝 Generating CSV: %s\n", flags.outputCSV)
	if err := generateCSV(groups, flags.outputCSV); err != nil {
		return fmt.Errorf("failed to generate CSV: %w", err)
	}

	fmt.Printf("\n%s CSV generated successfully!\n", green("✅"))
	fmt.Printf("  • Products: %d\n", len(groups))
	fmt.Printf("  • Total images: %d\n", countTotalImages(groups))
	fmt.Printf("  • File: %s\n", flags.outputCSV)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review: %s\n", yellow(fmt.Sprintf("open %s", flags.outputCSV)))
	fmt.Printf("  2. Validate: %s\n", yellow(fmt.Sprintf("./uploader validate --csv %s", flags.outputCSV)))
	fmt.Printf("  3. Upload: %s\n", yellow(fmt.Sprintf("./uploader upload --csv %s", flags.outputCSV)))
	fmt.Println()

	return nil
}

func findImageFiles(dir string) ([]string, error) {
	var imageFiles []string

	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if validExtensions[ext] {
			absPath, _ := filepath.Abs(path)
			imageFiles = append(imageFiles, absPath)
		}

		return nil
	})

	return imageFiles, err
}

func groupImagesByPartNumber(imageFiles []string, pattern string) []productGroup {
	groups := make(map[string]*productGroup)

	for _, imagePath := range imageFiles {
		filename := filepath.Base(imagePath)

		// Extract part number
		partNumber := extractPartNumber(filename, pattern)
		if partNumber == "" {
			partNumber = "UNKNOWN"
		}

		// Create or update group
		if _, exists := groups[partNumber]; !exists {
			groups[partNumber] = &productGroup{
				PartNumber:  partNumber,
				ProductName: extractProductName(filename),
				Images:      []string{},
			}
		}

		groups[partNumber].Images = append(groups[partNumber].Images, imagePath)
	}

	// Convert to slice and sort
	result := make([]productGroup, 0, len(groups))
	for _, group := range groups {
		// Sort images within each group
		sort.Strings(group.Images)
		result = append(result, *group)
	}

	// Sort groups by part number
	sort.Slice(result, func(i, j int) bool {
		return result[i].PartNumber < result[j].PartNumber
	})

	return result
}

func extractPartNumber(filename, pattern string) string {
	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// If custom pattern provided, use it
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil {
			matches := re.FindStringSubmatch(nameWithoutExt)
			if len(matches) > 0 {
				return matches[0]
			}
		}
	}

	// Default: Extract part before first underscore or dash with number
	// Pattern: ABC123_something or ABC123-1 or ABC123 something
	re := regexp.MustCompile(`^([A-Z0-9]+)(?:[-_\s]|$)`)
	matches := re.FindStringSubmatch(nameWithoutExt)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback: Use first word/token
	parts := strings.FieldsFunc(nameWithoutExt, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) > 0 {
		return parts[0]
	}

	return nameWithoutExt
}

func extractProductName(filename string) string {
	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove part number and numbers at the end
	// Pattern: Remove leading part number
	re := regexp.MustCompile(`^[A-Z0-9]+[-_\s]*`)
	productName := re.ReplaceAllString(nameWithoutExt, "")

	// Remove trailing numbers (like -1, -2, _1, _2)
	re = regexp.MustCompile(`[-_\s]*\d+$`)
	productName = re.ReplaceAllString(productName, "")

	// Replace underscores and multiple spaces with single space
	productName = strings.ReplaceAll(productName, "_", " ")
	productName = regexp.MustCompile(`\s+`).ReplaceAllString(productName, " ")

	// Clean up
	productName = strings.TrimSpace(productName)

	if productName == "" {
		productName = "Product Name (Please update)"
	}

	return productName
}

func generateCSV(groups []productGroup, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"part_number", "product_name", "image_path", "is_primary"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, group := range groups {
		for i, imagePath := range group.Images {
			isPrimary := "FALSE"
			if i == 0 {
				isPrimary = "TRUE"
			}

			row := []string{
				group.PartNumber,
				group.ProductName,
				imagePath,
				isPrimary,
			}

			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

func countTotalImages(groups []productGroup) int {
	count := 0
	for _, group := range groups {
		count += len(group.Images)
	}
	return count
}
