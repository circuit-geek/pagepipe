// Package output handles file naming and writing for PagePipe outputs.
// In --only mode, filenames are derived from the domain (e.g., example_com.md).
// In --all mode, filenames mirror the URL path structure.
package output

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Writer writes rendered output to disk.
type Writer struct {
	OutputDir string
}

// New creates a Writer targeting the given output directory.
// If outputDir is empty, it defaults to the current working directory.
func New(outputDir string) (*Writer, error) {
	if outputDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		outputDir = wd
	}

	// Ensure the output directory exists.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	return &Writer{OutputDir: outputDir}, nil
}

// WriteOnly writes output for --only mode.
// Filename: domain_path.ext (e.g., example_com.md).
func (w *Writer) WriteOnly(rawURL string, data []byte, ext string) (string, error) {
	name := filenameFromURL(rawURL)
	path := filepath.Join(w.OutputDir, name+ext)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", path, err)
	}
	return path, nil
}

// WriteAll writes output for --all mode, mirroring the URL path structure.
// Example: https://site.com/docs/intro → ./docs/intro.md
func (w *Writer) WriteAll(rawURL string, data []byte, ext string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	// Build the path from the URL.
	urlPath := strings.TrimSuffix(parsed.Path, "/")
	if urlPath == "" || urlPath == "/" {
		urlPath = "/index"
	}
	// Remove leading slash for filepath.Join.
	urlPath = strings.TrimPrefix(urlPath, "/")

	fullPath := filepath.Join(w.OutputDir, urlPath+ext)

	// Ensure parent directories exist.
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", fullPath, err)
	}
	return fullPath, nil
}

// filenameFromURL converts a URL into a flat filename.
// Example: https://example.com/docs/intro → example_com_docs_intro
func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Fallback: sanitize the raw string.
		return sanitize(rawURL)
	}

	parts := []string{sanitize(parsed.Host)}
	path := strings.Trim(parsed.Path, "/")
	if path != "" {
		for _, seg := range strings.Split(path, "/") {
			parts = append(parts, sanitize(seg))
		}
	}
	return strings.Join(parts, "_")
}

// sanitize replaces non-alphanumeric characters with underscores.
func sanitize(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
