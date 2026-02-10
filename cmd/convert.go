// Package cmd — convert command.
// This is the main command that orchestrates the pipeline:
// fetch → extract → normalize → render → write.
//
// It handles flag validation, renderer selection, and the --only / --all modes.
package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gaurav-prasanna/pagepipe/core"
	"github.com/gaurav-prasanna/pagepipe/core/extract"
	"github.com/gaurav-prasanna/pagepipe/core/fetch"
	"github.com/gaurav-prasanna/pagepipe/core/normalize"
	"github.com/gaurav-prasanna/pagepipe/core/output"
	"github.com/gaurav-prasanna/pagepipe/core/render"
	"github.com/gaurav-prasanna/pagepipe/crawl"
	"github.com/spf13/cobra"
)

// Flag variables.
var (
	flagOnly       bool
	flagAll        bool
	flagPDF        bool
	flagMarkdown   bool
	flagJSON       bool
	flagEmbeddings bool
	flagModel      string
	flagChunkSize  int
	flagOutputDir  string
)

var convertCmd = &cobra.Command{
	Use:   "convert <url>",
	Short: "Convert a URL to the specified output format",
	Long: `Convert fetches a webpage, extracts main content, normalizes it to Markdown,
and converts it to the specified output format (PDF, Markdown, JSON, or Embeddings).

Examples:
  pagepipe convert https://example.com --markdown
  pagepipe convert https://example.com --json --output_dir ./out
  pagepipe convert https://example.com --all --pdf
  pagepipe convert https://example.com --embeddings --model nomic-embed-text`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	// Mode flags.
	convertCmd.Flags().BoolVar(&flagOnly, "only", false, "Convert only the given URL (default)")
	convertCmd.Flags().BoolVar(&flagAll, "all", false, "Convert all discovered sub-pages")

	// Output format flags (mutually exclusive).
	convertCmd.Flags().BoolVar(&flagPDF, "pdf", false, "Output PDF")
	convertCmd.Flags().BoolVar(&flagMarkdown, "markdown", false, "Output Markdown")
	convertCmd.Flags().BoolVar(&flagJSON, "json", false, "Output structured JSON")
	convertCmd.Flags().BoolVar(&flagEmbeddings, "embeddings", false, "Output embeddings")

	// Embedding-specific flags.
	convertCmd.Flags().StringVar(&flagModel, "model", "", "Embedding model (required with --embeddings)")
	convertCmd.Flags().IntVar(&flagChunkSize, "chunk_size", 512, "Token chunk size for embeddings")

	// Output directory.
	convertCmd.Flags().StringVar(&flagOutputDir, "output_dir", "", "Output directory (default: current directory)")
}

func runConvert(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	// --- Validate flags ---
	if err := validateFlags(); err != nil {
		return err
	}

	// Validate URL.
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid URL: %s (must include scheme, e.g. https://example.com)", rawURL)
	}

	// Select renderer.
	renderer, err := selectRenderer()
	if err != nil {
		return err
	}

	// Initialize pipeline components.
	fetcher := fetch.New()
	extractor := extract.New()
	normalizer := normalize.New()

	writer, err := output.New(flagOutputDir)
	if err != nil {
		return fmt.Errorf("initializing output writer: %w", err)
	}

	ctx := context.Background()

	if flagAll {
		return runAll(ctx, rawURL, fetcher, extractor, normalizer, renderer, writer)
	}
	return runOnly(ctx, rawURL, fetcher, extractor, normalizer, renderer, writer)
}

// runOnly processes a single URL through the pipeline.
func runOnly(
	ctx context.Context,
	rawURL string,
	fetcher core.Fetcher,
	extractor core.Extractor,
	normalizer core.Normalizer,
	renderer core.Renderer,
	writer *output.Writer,
) error {
	data, meta, err := processURL(ctx, rawURL, fetcher, extractor, normalizer, renderer)
	if err != nil {
		return err
	}
	_ = meta

	path, err := writer.WriteOnly(rawURL, data, renderer.Extension())
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "✓ Written: %s\n", path)
	return nil
}

// runAll discovers all internal pages and processes each through the pipeline.
func runAll(
	ctx context.Context,
	rawURL string,
	fetcher core.Fetcher,
	extractor core.Extractor,
	normalizer core.Normalizer,
	renderer core.Renderer,
	writer *output.Writer,
) error {
	fmt.Fprintf(os.Stdout, "Discovering pages from %s...\n", rawURL)

	// Discover all internal URLs.
	urls, err := crawl.DiscoverAll(ctx, rawURL, fetcher)
	if err != nil {
		return fmt.Errorf("discovering pages: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Found %d pages to process\n", len(urls))

	var errCount int
	for i, pageURL := range urls {
		fmt.Fprintf(os.Stdout, "[%d/%d] Processing %s\n", i+1, len(urls), pageURL)

		data, _, err := processURL(ctx, pageURL, fetcher, extractor, normalizer, renderer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Error: %v\n", err)
			errCount++
			continue
		}

		path, err := writer.WriteAll(pageURL, data, renderer.Extension())
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Write error: %v\n", err)
			errCount++
			continue
		}
		fmt.Fprintf(os.Stdout, "  ✓ Written: %s\n", path)
	}

	if errCount > 0 {
		fmt.Fprintf(os.Stderr, "\n%d/%d pages failed\n", errCount, len(urls))
	}
	return nil
}

// processURL runs a single URL through the full pipeline.
func processURL(
	ctx context.Context,
	rawURL string,
	fetcher core.Fetcher,
	extractor core.Extractor,
	normalizer core.Normalizer,
	renderer core.Renderer,
) ([]byte, core.PageMetadata, error) {
	// 1. Fetch
	result, err := fetcher.Fetch(ctx, rawURL)
	if err != nil {
		return nil, core.PageMetadata{}, fmt.Errorf("fetch: %w", err)
	}

	// 2. Extract main content
	content, err := extractor.Extract(result.HTML)
	if err != nil {
		return nil, core.PageMetadata{}, fmt.Errorf("extract: %w", err)
	}

	// 3. Normalize to Markdown
	markdown, err := normalizer.Normalize(content)
	if err != nil {
		return nil, core.PageMetadata{}, fmt.Errorf("normalize: %w", err)
	}

	// Build metadata from URL and fetched HTML.
	meta := buildMetadata(rawURL, result.HTML)

	// 4. Render to output format
	data, err := renderer.Render(markdown, meta)
	if err != nil {
		return nil, core.PageMetadata{}, fmt.Errorf("render: %w", err)
	}

	return data, meta, nil
}

// buildMetadata constructs PageMetadata from the URL and raw HTML.
func buildMetadata(rawURL string, html string) core.PageMetadata {
	parsed, _ := url.Parse(rawURL)

	title := extractTitle(html)
	lang := extractLang(html)

	return core.PageMetadata{
		URL:       rawURL,
		Domain:    parsed.Host,
		Path:      parsed.Path,
		Title:     title,
		Language:  lang,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// extractTitle pulls the <title> content from raw HTML.
func extractTitle(html string) string {
	// Simple regex-free extraction for performance.
	start := findTag(html, "<title>")
	if start == -1 {
		return ""
	}
	// findTag returns index AFTER the tag, so for </title> we need
	// the index BEFORE it (i.e., subtract the tag length).
	endTag := findTag(html, "</title>")
	if endTag == -1 || endTag <= start {
		return ""
	}
	// endTag points after "</title>", subtract len("</title>") to get before it.
	end := endTag - len("</title>")
	if end <= start {
		return ""
	}
	return html[start:end]
}

// extractLang pulls the lang attribute from the <html> tag.
func extractLang(html string) string {
	idx := findTag(html, "lang=\"")
	if idx == -1 {
		return "en" // sensible default
	}
	end := idx
	for end < len(html) && html[end] != '"' {
		end++
	}
	return html[idx:end]
}

// findTag returns the index immediately after the given tag string.
func findTag(html, tag string) int {
	for i := 0; i <= len(html)-len(tag); i++ {
		if html[i:i+len(tag)] == tag {
			return i + len(tag)
		}
	}
	return -1
}

// validateFlags checks that exactly one output format is chosen and
// that --only and --all are not both specified.
func validateFlags() error {
	// Check mutually exclusive mode flags.
	if flagOnly && flagAll {
		return fmt.Errorf("--only and --all are mutually exclusive")
	}

	// Count output formats.
	formatCount := 0
	if flagPDF {
		formatCount++
	}
	if flagMarkdown {
		formatCount++
	}
	if flagJSON {
		formatCount++
	}
	if flagEmbeddings {
		formatCount++
	}

	if formatCount == 0 {
		return fmt.Errorf("exactly one output format is required: --pdf, --markdown, --json, or --embeddings")
	}
	if formatCount > 1 {
		return fmt.Errorf("only one output format allowed per run (got %d)", formatCount)
	}

	// --model is required with --embeddings.
	if flagEmbeddings && flagModel == "" {
		return fmt.Errorf("--model is required when using --embeddings")
	}

	return nil
}

// selectRenderer creates the appropriate Renderer based on flags.
func selectRenderer() (core.Renderer, error) {
	switch {
	case flagMarkdown:
		return render.NewMarkdownRenderer(), nil
	case flagJSON:
		return render.NewJSONRenderer(), nil
	case flagPDF:
		return render.NewPDFRenderer(), nil
	case flagEmbeddings:
		return render.NewEmbeddingsRenderer(flagModel, flagChunkSize), nil
	default:
		return nil, fmt.Errorf("no output format selected")
	}
}
