// Package render — JSON renderer.
// Builds the structured JSON output from Markdown and page metadata.
// Parses the Markdown to extract structural information (headings, links,
// code blocks, tables, lists) without inferring any business-specific fields.
package render

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gaurav-prasanna/pagepipe/core"
)

// JSONRenderer produces structured JSON output from Markdown.
type JSONRenderer struct{}

// NewJSONRenderer creates a JSONRenderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Render converts Markdown and metadata into the specified JSON structure.
func (r *JSONRenderer) Render(markdown string, meta core.PageMetadata) ([]byte, error) {
	headings := extractHeadings(markdown)
	links := extractLinks(markdown)

	// Build sections from headings.
	sections := buildSections(markdown, headings)

	// Count structural elements.
	codeBlocks := countCodeBlocks(markdown)
	tables := countTables(markdown)
	lists := countLists(markdown)

	// Strip markdown formatting to get plain text.
	plainText := stripMarkdown(markdown)

	page := core.PageJSON{
		Metadata: meta,
		Content: core.PageContent{
			Text:     plainText,
			Markdown: markdown,
			Sections: sections,
		},
		Structure: core.PageStructure{
			Headings:   headings,
			Links:      links,
			CodeBlocks: codeBlocks,
			Tables:     tables,
			Lists:      lists,
		},
	}

	data, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	return data, nil
}

// Extension returns the file extension for JSON output.
func (r *JSONRenderer) Extension() string {
	return ".json"
}

// --- Markdown parsing helpers ---

var headingRegex = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

func extractHeadings(md string) []core.Heading {
	matches := headingRegex.FindAllStringSubmatch(md, -1)
	headings := make([]core.Heading, 0, len(matches))
	for _, m := range matches {
		headings = append(headings, core.Heading{
			Level: len(m[1]),
			Text:  strings.TrimSpace(m[2]),
		})
	}
	return headings
}

// linkRegex matches Markdown links [text](url).
var linkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

func extractLinks(md string) []core.Link {
	matches := linkRegex.FindAllStringSubmatch(md, -1)
	links := make([]core.Link, 0, len(matches))
	for _, m := range matches {
		links = append(links, core.Link{
			Text: m[1],
			Href: m[2],
		})
	}
	return links
}

func buildSections(md string, headings []core.Heading) []core.Section {
	if len(headings) == 0 {
		return nil
	}

	lines := strings.Split(md, "\n")
	sections := make([]core.Section, 0, len(headings))
	headingIdx := 0

	var currentSection *core.Section
	var sectionLines []string

	for _, line := range lines {
		if headingRegex.MatchString(line) && headingIdx < len(headings) {
			// Flush previous section.
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(strings.Join(sectionLines, "\n"))
				sections = append(sections, *currentSection)
			}
			currentSection = &core.Section{
				Heading: headings[headingIdx].Text,
				Level:   headings[headingIdx].Level,
			}
			sectionLines = nil
			headingIdx++
		} else if currentSection != nil {
			sectionLines = append(sectionLines, line)
		}
	}
	// Flush last section.
	if currentSection != nil {
		currentSection.Text = strings.TrimSpace(strings.Join(sectionLines, "\n"))
		sections = append(sections, *currentSection)
	}

	return sections
}

// countCodeBlocks counts fenced code blocks (``` delimited).
func countCodeBlocks(md string) int {
	return strings.Count(md, "```") / 2
}

// countTables counts Markdown tables by looking for separator rows (|---|).
var tableRowRegex = regexp.MustCompile(`(?m)^\|[-:| ]+\|$`)

func countTables(md string) int {
	return len(tableRowRegex.FindAllString(md, -1))
}

// countLists counts top-level list items (lines starting with - or * or 1.).
var listItemRegex = regexp.MustCompile(`(?m)^[\s]*[-*]\s|^[\s]*\d+\.\s`)

func countLists(md string) int {
	return len(listItemRegex.FindAllString(md, -1))
}

// stripMarkdown removes common Markdown formatting to produce plain text.
func stripMarkdown(md string) string {
	text := md
	// Remove headings markers.
	text = headingRegex.ReplaceAllString(text, "$2")
	// Remove bold/italic.
	text = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`).ReplaceAllString(text, "$1")
	// Remove links, keep text.
	text = linkRegex.ReplaceAllString(text, "$1")
	// Remove code block fences.
	text = strings.ReplaceAll(text, "```", "")
	// Remove inline code.
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "$1")
	// Collapse whitespace.
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}
