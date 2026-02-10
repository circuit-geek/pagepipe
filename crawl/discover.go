// Package crawl provides URL discovery and crawling for --all mode.
// It discovers internal pages via sitemap.xml and link extraction,
// keeping crawling logic separate from the ingest pipeline.
package crawl

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gaurav-prasanna/pagepipe/core"
)

// sitemapURL holds a URL from a sitemap.xml.
type sitemapURL struct {
	Loc string `xml:"loc"`
}

// sitemapIndex is the root element of a sitemap.xml.
type sitemapIndex struct {
	URLs []sitemapURL `xml:"url"`
}

// DiscoverAll finds all internal URLs to process starting from baseURL.
// It first tries sitemap.xml, then falls back to link crawling.
// The baseURL itself is always included.
func DiscoverAll(ctx context.Context, baseURL string, fetcher core.Fetcher) ([]string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	domain := parsed.Host

	// Try sitemap first.
	sitemapURLStr := fmt.Sprintf("%s://%s/sitemap.xml", parsed.Scheme, domain)
	urls, err := discoverFromSitemap(ctx, sitemapURLStr, domain)
	if err == nil && len(urls) > 0 {
		return urls, nil
	}

	// Fall back to BFS link crawling.
	return discoverFromLinks(ctx, baseURL, domain, fetcher)
}

// discoverFromSitemap fetches and parses sitemap.xml for internal URLs.
func discoverFromSitemap(ctx context.Context, sitemapURL string, domain string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sitemap sitemapIndex
	if err := xml.Unmarshal(body, &sitemap); err != nil {
		return nil, err
	}

	var urls []string
	for _, u := range sitemap.URLs {
		if IsSameDomain(u.Loc, domain) && !IsStaticAsset(u.Loc) {
			urls = append(urls, NormalizeURL(u.Loc))
		}
	}
	return urls, nil
}

// discoverFromLinks performs BFS crawling to find internal links.
func discoverFromLinks(ctx context.Context, startURL string, domain string, fetcher core.Fetcher) ([]string, error) {
	queue := NewQueue()
	queue.Add(NormalizeURL(startURL))

	// BFS with a reasonable limit to avoid runaway crawls.
	const maxPages = 100

	for queue.HasNext() && queue.Visited() < maxPages {
		currentURL := queue.Next()

		result, err := fetcher.Fetch(ctx, currentURL)
		if err != nil {
			continue // Skip failed pages, don't block the crawl.
		}

		links, err := extractLinks(result.HTML, currentURL)
		if err != nil {
			continue
		}

		for _, link := range links {
			if IsSameDomain(link, domain) && !IsStaticAsset(link) {
				queue.Add(NormalizeURL(link))
			}
		}
	}

	return queue.All(), nil
}

// extractLinks extracts all href values from <a> tags, resolving relative URLs.
func extractLinks(html string, baseURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	base, _ := url.Parse(baseURL)
	var links []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		resolved := resolveURL(href, base)
		if resolved != "" {
			links = append(links, resolved)
		}
	})

	return links, nil
}

// resolveURL resolves a potentially relative URL against a base.
func resolveURL(href string, base *url.URL) string {
	// Skip mailto, javascript, etc.
	if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "#") {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)
	// Strip fragments.
	resolved.Fragment = ""
	return resolved.String()
}
