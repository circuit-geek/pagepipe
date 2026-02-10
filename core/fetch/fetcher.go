// Package fetch implements the Fetcher interface.
// It performs HTTP GET requests with sensible defaults for web scraping.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gaurav-prasanna/pagepipe/core"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultUserAgent = "PagePipe/1.0 (https://github.com/gaurav-prasanna/pagepipe)"
)

// HTTPFetcher fetches web pages via HTTP.
type HTTPFetcher struct {
	client *http.Client
}

// New creates an HTTPFetcher with a sensible timeout.
func New() *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// Fetch retrieves the HTML content of the given URL.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (*core.FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &core.FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		HTML:       string(body),
	}, nil
}
