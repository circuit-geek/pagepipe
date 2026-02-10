// Package extract implements the Extractor interface.
// It isolates the main content from a full HTML page by:
//  1. Finding the best content container (<main>, <article>, or <body>)
//  2. Removing noise elements (nav, footer, scripts, images, etc.)
package extract

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// noiseSelectors are HTML elements removed before extraction.
// These contribute no meaningful content to the page text.
var noiseSelectors = []string{
	"script", "style", "noscript",
	"nav", "footer", "header",
	"img", "picture", "figure", "figcaption",
	"iframe", "video", "audio",
	"svg", "canvas",
	"form", "button", "input", "select", "textarea",
	".sidebar", ".menu", ".navigation", ".ads", ".advertisement",
}

// HTMLExtractor strips noise from HTML and returns the main content fragment.
type HTMLExtractor struct{}

// New creates an HTMLExtractor.
func New() *HTMLExtractor {
	return &HTMLExtractor{}
}

// Extract takes raw HTML and returns a cleaned HTML fragment containing
// only the main content. Images are explicitly excluded per v1 spec.
func (e *HTMLExtractor) Extract(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	// Remove noise elements first (operates on the whole document).
	for _, sel := range noiseSelectors {
		doc.Find(sel).Remove()
	}

	// Find the best content container in priority order.
	// <main> is the most semantically correct, then <article>, then <body>.
	var content *goquery.Selection
	for _, tag := range []string{"main", "article", "body"} {
		sel := doc.Find(tag)
		if sel.Length() > 0 {
			content = sel.First()
			break
		}
	}

	if content == nil {
		return "", fmt.Errorf("no content container found in HTML")
	}

	result, err := goquery.OuterHtml(content)
	if err != nil {
		return "", fmt.Errorf("serializing content: %w", err)
	}

	return result, nil
}
