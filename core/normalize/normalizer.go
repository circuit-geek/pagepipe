// Package normalize implements the Normalizer interface.
// It converts cleaned HTML into Markdown, which serves as the
// canonical intermediate format for all downstream renderers.
package normalize

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// MarkdownNormalizer converts HTML to Markdown using html-to-markdown.
type MarkdownNormalizer struct{}

// New creates a MarkdownNormalizer.
func New() *MarkdownNormalizer {
	return &MarkdownNormalizer{}
}

// Normalize converts a cleaned HTML fragment into Markdown.
func (n *MarkdownNormalizer) Normalize(html string) (string, error) {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("converting HTML to markdown: %w", err)
	}
	return markdown, nil
}
