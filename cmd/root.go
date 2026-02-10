// Package cmd implements the CLI commands for PagePipe using Cobra.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pagepipe",
	Short: "PagePipe — convert website URLs into structured outputs",
	Long: `PagePipe is a deterministic ingestion pipeline that converts website URLs
into Markdown, PDF, JSON, or Embeddings.

Usage:
  pagepipe convert <url> [flags]`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
