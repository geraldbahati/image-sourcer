package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"github.com/geraldbahati/image-sourcer/internal/cloudinary"
	"github.com/geraldbahati/image-sourcer/internal/config"
	"github.com/geraldbahati/image-sourcer/internal/csv"
	"github.com/geraldbahati/image-sourcer/internal/image"
	"github.com/geraldbahati/image-sourcer/internal/report"
	"github.com/geraldbahati/image-sourcer/internal/state"
	"github.com/geraldbahati/image-sourcer/internal/worker"
)

type uploadFlags struct {
	csvFile    string
	configFile string
	dryRun     bool
	force      bool
	workers    int
}

func newUploadCmd() *cobra.Command {
	flags := &uploadFlags{}

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload images to Cloudinary",
		Long:  "Upload product images to Cloudinary from a CSV manifest file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpload(flags)
		},
	}

	cmd.Flags().StringVarP(&flags.csvFile, "csv", "c", "", "Path to CSV file (required)")
	cmd.Flags().StringVar(&flags.configFile, "config", "config.yaml", "Path to config file")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Validate only, don't upload")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Force re-upload (ignore state)")
	cmd.Flags().IntVarP(&flags.workers, "workers", "w", 0, "Number of concurrent workers (0 = use config)")

	cmd.MarkFlagRequired("csv")

	return cmd
}

func runUpload(flags *uploadFlags) error {
	startTime := time.Now()

	// Print header
	printHeader()

	// Load configuration
	cfg, err := config.Load(flags.configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override worker count if specified
	if flags.workers > 0 {
		cfg.Upload.ConcurrentWorkers = flags.workers
	}

	// Parse CSV
	fmt.Printf("📄 Reading CSV: %s\n", flags.csvFile)
	parser := csv.NewParser(flags.csvFile)
	manifest, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("✓ Found %s for %s\n\n", 
		green(fmt.Sprintf("%d images", manifest.GetImageCount())),
		green(fmt.Sprintf("%d products", manifest.GetProductCount())))

	// Validate images
	fmt.Println("🔍 Validating images...")
	validator := image.DefaultValidator()
	
	imagePaths := make([]string, len(manifest.Images))
	for i, img := range manifest.Images {
		imagePaths[i] = img.ImagePath
	}
	
	validationResults := validator.ValidateBatch(imagePaths)
	
	invalidCount := 0
	for i, result := range validationResults {
		if !result.Valid {
			invalidCount++
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("  %s %s: %s\n", red("✗"), manifest.Images[i].ImagePath, result.Error)
		}
	}

	if invalidCount > 0 {
		return fmt.Errorf("%d images failed validation", invalidCount)
	}

	fmt.Printf("✓ All images validated successfully\n\n")

	// If dry run, stop here
	if flags.dryRun {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\n%s Dry run mode - no images uploaded\n", yellow("ℹ"))
		return nil
	}

	// Initialize state tracker
	tracker := state.NewTracker(cfg.State.File)
	if cfg.State.Enabled {
		if err := tracker.Load(); err != nil {
			fmt.Printf("⚠ Warning: failed to load state: %v\n", err)
		}
	}

	// Filter out already uploaded products if not forcing
	imagesToUpload := manifest.Images
	skippedCount := 0
	
	if !flags.force && cfg.State.Enabled {
		filtered := make([]csv.ProductImage, 0)
		for _, img := range manifest.Images {
			if !tracker.IsProductUploaded(img.PartNumber) {
				filtered = append(filtered, img)
			} else {
				skippedCount++
			}
		}
		imagesToUpload = filtered
	}

	if skippedCount > 0 {
		fmt.Printf("✓ Skipping %d already uploaded images\n", skippedCount)
	}

	if len(imagesToUpload) == 0 {
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("\n%s All products already uploaded!\n", cyan("✓"))
		return nil
	}

	fmt.Printf("📤 Uploading %d images...\n\n", len(imagesToUpload))

	// Create Cloudinary client
	client, err := cloudinary.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Cloudinary client: %w", err)
	}

	// Create uploader
	uploader := cloudinary.NewUploader(client)

	// Create worker pool
	pool := worker.NewPool(cfg.Upload.ConcurrentWorkers, uploader)
	pool.Start()

	// Create progress bar
	bar := progressbar.NewOptions(len(imagesToUpload),
		progressbar.OptionSetDescription("Uploading"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Submit jobs
	go func() {
		for i, img := range imagesToUpload {
			pool.Submit(worker.Job{
				ProductImage: img,
				Index:        i,
			})
		}
		pool.Close()
	}()

	// Collect results
	results := make([]cloudinary.UploadResult, 0, len(imagesToUpload))
	successCount := 0
	failedCount := 0

	for result := range pool.Results() {
		results = append(results, result.UploadResult)
		
		if result.UploadResult.Success {
			successCount++
		} else {
			failedCount++
			// Print error for failed uploads
			if cfg.Output.Verbose {
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("\n%s %s: %s\n", 
					red("✗"), 
					result.UploadResult.ProductImage.PartNumber,
					result.UploadResult.Error)
			}
		}
		
		bar.Add(1)
	}

	fmt.Println() // New line after progress bar

	// Update state
	if cfg.State.Enabled && !flags.force {
		// Group results by product
		productResults := make(map[string][]cloudinary.UploadResult)
		for _, r := range results {
			if r.Success {
				partNum := r.ProductImage.PartNumber
				productResults[partNum] = append(productResults[partNum], r)
			}
		}

		// Mark products as uploaded
		for partNum, prodResults := range productResults {
			publicIDs := make([]string, len(prodResults))
			urls := make([]string, len(prodResults))
			productName := ""
			
			for i, r := range prodResults {
				publicIDs[i] = r.PublicID
				urls[i] = r.CloudinaryURL
				if productName == "" {
					productName = r.ProductImage.ProductName
				}
			}
			
			tracker.MarkProductUploaded(partNum, productName, publicIDs, urls)
		}

		if err := tracker.Save(); err != nil {
			fmt.Printf("⚠ Warning: failed to save state: %v\n", err)
		}
	}

	// Generate report
	generator := report.NewGenerator(cfg.Output.ReportDir)
	reportPath, err := generator.Generate(results)
	if err != nil {
		fmt.Printf("⚠ Warning: failed to generate report: %v\n", err)
	}

	summary := generator.GenerateSummary(results)

	// Print summary
	printSummary(summary, startTime, reportPath)

	if failedCount > 0 {
		return fmt.Errorf("%d images failed to upload", failedCount)
	}

	return nil
}

func printHeader() {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Printf("\n%s\n", cyan("🚀 Image Uploader v1.0"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func printSummary(summary report.Summary, startTime time.Time, reportPath string) {
	duration := time.Since(startTime)
	
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Printf("%s\n", green("✅ Upload Complete!"))
	
	fmt.Printf("  • Processed: %d products\n", summary.ProductCount)
	fmt.Printf("  • Uploaded: %d/%d images (%.1f%%)\n", 
		summary.SuccessCount, 
		summary.TotalImages, 
		summary.SuccessRate())
	
	if summary.FailedCount > 0 {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Printf("  • Failed: %s\n", red(fmt.Sprintf("%d", summary.FailedCount)))
	}
	
	fmt.Printf("  • Total size: %s\n", summary.FormatSize())
	fmt.Printf("  • Duration: %v\n", duration.Round(time.Second))

	if reportPath != "" {
		fmt.Printf("\n📊 Report: %s\n", reportPath)
	}
	
	fmt.Println()
}
