package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "uploader",
		Short: "Image Uploader for E-commerce",
		Long: `A high-performance Go CLI tool for batch uploading product images 
to Cloudinary with intelligent optimization and state management.`,
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newUploadCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newStatsCmd())
	rootCmd.AddCommand(newResetCmd())
	rootCmd.AddCommand(newAnalyzeCmd())
	rootCmd.AddCommand(newEnrichCmd())
	rootCmd.AddCommand(newImportCmd())

	return rootCmd
}
