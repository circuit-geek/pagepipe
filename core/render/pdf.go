// Package render — PDF renderer.
// Converts Markdown into a styled PDF using gofpdf.
// Handles headings (variable font sizes), paragraphs, code blocks, and lists.
// Images are intentionally not rendered (v1 non-goal).
package render

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/gaurav-prasanna/pagepipe/core"
	"github.com/jung-kurt/gofpdf"
)

// PDFRenderer renders Markdown content as a PDF document.
type PDFRenderer struct{}

// NewPDFRenderer creates a PDFRenderer.
func NewPDFRenderer() *PDFRenderer {
	return &PDFRenderer{}
}

// Render converts Markdown into PDF bytes.
func (r *PDFRenderer) Render(markdown string, meta core.PageMetadata) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title from metadata.
	if meta.Title != "" {
		pdf.SetFont("Helvetica", "B", 18)
		pdf.MultiCell(0, 8, meta.Title, "", "L", false)
		pdf.Ln(4)
	}

	// Source URL.
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.MultiCell(0, 5, "Source: "+meta.URL, "", "L", false)
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(6)

	// Parse and render Markdown line by line.
	lines := strings.Split(markdown, "\n")
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Toggle code block state.
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				pdf.Ln(2)
				pdf.SetFont("Courier", "", 9)
				pdf.SetFillColor(245, 245, 245)
			} else {
				pdf.Ln(2)
			}
			continue
		}

		if inCodeBlock {
			// Render code lines with monospace font and background.
			pdf.SetFont("Courier", "", 9)
			pdf.SetFillColor(245, 245, 245)
			pdf.MultiCell(0, 4.5, line, "", "L", true)
			continue
		}

		// Skip empty lines (add spacing instead).
		if strings.TrimSpace(line) == "" {
			pdf.Ln(3)
			continue
		}

		// Headings.
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, ch := range line {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			text := strings.TrimSpace(strings.TrimLeft(line, "# "))
			renderHeading(pdf, text, level)
			continue
		}

		// List items.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			pdf.SetFont("Helvetica", "", 10)
			text := "• " + strings.TrimSpace(trimmed[2:])
			text = cleanInlineMarkdown(text)
			pdf.MultiCell(0, 5, text, "", "L", false)
			continue
		}

		// Numbered list items.
		if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
			pdf.SetFont("Helvetica", "", 10)
			text := cleanInlineMarkdown(trimmed)
			pdf.MultiCell(0, 5, text, "", "L", false)
			continue
		}

		// Regular paragraph text.
		pdf.SetFont("Helvetica", "", 10)
		text := cleanInlineMarkdown(line)
		pdf.MultiCell(0, 5, text, "", "L", false)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Extension returns the file extension for PDF output.
func (r *PDFRenderer) Extension() string {
	return ".pdf"
}

// renderHeading sets the font size based on heading level and writes text.
func renderHeading(pdf *gofpdf.Fpdf, text string, level int) {
	sizes := map[int]float64{1: 18, 2: 15, 3: 13, 4: 12, 5: 11, 6: 10}
	size, ok := sizes[level]
	if !ok {
		size = 10
	}
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", size)
	pdf.MultiCell(0, size*0.6, cleanInlineMarkdown(text), "", "L", false)
	pdf.Ln(2)
}

// cleanInlineMarkdown strips inline Markdown formatting for PDF rendering.
func cleanInlineMarkdown(text string) string {
	// Remove bold markers.
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	// Remove italic markers (but not inside words like don't).
	re := regexp.MustCompile(`(?:^|\s)\*([^*]+)\*(?:\s|$)`)
	text = re.ReplaceAllString(text, " $1 ")
	// Remove inline code markers.
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "$1")
	// Remove link syntax, keep text.
	text = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(text, "$1")
	return strings.TrimSpace(text)
}
