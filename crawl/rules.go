// Package crawl — URL filtering rules.
// Provides helpers to filter, normalize, and validate URLs during crawling.
package crawl

import (
	"net/url"
	"path"
	"strings"
)

// staticExtensions are file extensions to skip during crawling.
var staticExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".svg": true, ".webp": true, ".ico": true, ".bmp": true,
	".css": true, ".js": true, ".mjs": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp4": true, ".webm": true, ".mp3": true, ".wav": true,
	".zip": true, ".tar": true, ".gz": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
}

// IsSameDomain checks if the given URL belongs to the specified domain.
func IsSameDomain(rawURL string, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Host == domain
}

// IsStaticAsset checks if a URL points to a static asset (image, CSS, JS, etc.).
func IsStaticAsset(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	return staticExtensions[ext]
}

// NormalizeURL strips fragments and trailing slashes for deduplication.
func NormalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove fragment.
	parsed.Fragment = ""

	// Remove trailing slash (but keep root "/").
	if parsed.Path != "/" {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	return parsed.String()
}
