// Package render provides output renderers for the PagePipe pipeline.
// This file implements the Markdown renderer, which is a simple passthrough.
package render

import (
	"github.com/gaurav-prasanna/pagepipe/core"
)

// MarkdownRenderer writes Markdown as-is. It's the simplest renderer
// since Markdown is already the canonical pipeline format.
type MarkdownRenderer struct{}

// NewMarkdownRenderer creates a MarkdownRenderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render returns the Markdown as bytes (passthrough).
func (r *MarkdownRenderer) Render(markdown string, meta core.PageMetadata) ([]byte, error) {
	return []byte(markdown), nil
}

// Extension returns the file extension for Markdown output.
func (r *MarkdownRenderer) Extension() string {
	return ".md"
}
