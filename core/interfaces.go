// Package core defines the pipeline interfaces for PagePipe.
// Each stage of the pipeline is a clean, testable interface.
package core

import "context"

// FetchResult holds the raw HTML and response metadata from a fetch.
type FetchResult struct {
	URL        string
	StatusCode int
	HTML       string
}

// PageMetadata holds metadata extracted from the page and URL.
type PageMetadata struct {
	URL       string `json:"url"`
	Domain    string `json:"domain"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Language  string `json:"language"`
	FetchedAt string `json:"fetched_at"` // ISO8601
}

// Section represents a heading-delimited section of content.
type Section struct {
	Heading string `json:"heading"`
	Level   int    `json:"level"`
	Text    string `json:"text"`
}

// Heading represents a single heading found in the content.
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// Link represents a hyperlink found in the content.
type Link struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

// PageContent holds the text and structured content of a page.
type PageContent struct {
	Text     string    `json:"text"`
	Markdown string    `json:"markdown"`
	Sections []Section `json:"sections"`
}

// PageStructure holds structural metadata parsed from the content.
type PageStructure struct {
	Headings   []Heading `json:"headings"`
	Links      []Link    `json:"links"`
	CodeBlocks int       `json:"code_blocks"`
	Tables     int       `json:"tables"`
	Lists      int       `json:"lists"`
}

// PageJSON is the complete JSON output for a single page.
type PageJSON struct {
	Metadata  PageMetadata  `json:"metadata"`
	Content   PageContent   `json:"content"`
	Structure PageStructure `json:"structure"`
}

// Fetcher retrieves raw HTML from a URL.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*FetchResult, error)
}

// Extractor pulls the main content from raw HTML, stripping noise.
type Extractor interface {
	Extract(html string) (string, error)
}

// Normalizer converts cleaned HTML into Markdown (the canonical format).
type Normalizer interface {
	Normalize(html string) (string, error)
}

// Renderer converts Markdown (and metadata) into a final output format.
type Renderer interface {
	Render(markdown string, meta PageMetadata) ([]byte, error)
	// Extension returns the file extension for this renderer (e.g. ".md", ".pdf").
	Extension() string
}

// Embedder generates a vector embedding for a text input.
type Embedder interface {
	Embed(ctx context.Context, text string, model string) ([]float64, error)
}
