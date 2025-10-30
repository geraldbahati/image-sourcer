package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/geraldbahati/image-sourcer/internal/config"
	"github.com/geraldbahati/image-sourcer/internal/state"
)

func newResetCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset upload state",
		Long:  "Clear the upload state file (allows re-uploading all products)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReset(configFile)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "config.yaml", "Path to config file")

	return cmd
}

func runReset(configFile string) error {
	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.State.Enabled {
		return fmt.Errorf("state tracking is disabled in config")
	}

	// Load and clear state
	tracker := state.NewTracker(cfg.State.File)
	if err := tracker.Load(); err != nil {
		fmt.Printf("Warning: failed to load state: %v\n", err)
	}

	uploadedCount := tracker.GetUploadedCount()

	// Delete state file
	if err := tracker.Delete(); err != nil {
		return fmt.Errorf("failed to reset state: %w", err)
	}

	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Println()
	fmt.Printf("%s Upload state reset successfully!\n", green("✅"))
	fmt.Printf("  • Cleared %d products from state\n", uploadedCount)
	fmt.Printf("  • All products can now be re-uploaded\n")
	fmt.Println()

	return nil
}
