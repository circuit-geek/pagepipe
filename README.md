# PagePipe

A Go-based CLI tool that converts website URLs into structured outputs — Markdown, PDF, JSON, and Embeddings — via a deterministic ingestion pipeline.

PagePipe is **not** a scraper with custom schemas. It is a Unix-style pipe that treats every webpage the same way: fetch it, extract the main content, normalize it to Markdown, and convert it to your desired output format.

---

## Installation

```bash
# Clone the repository
git clone https://github.com/gaurav-prasanna/pagepipe.git
cd pagepipe

# Build the binary
go build -o pagepipe .
```

---

## Quick Start

```bash
# Convert a webpage to Markdown
./pagepipe convert https://example.com --markdown

# Convert to JSON with structured metadata
./pagepipe convert https://example.com --json --output_dir ./out

# Convert to PDF
./pagepipe convert https://go.dev/doc/effective_go --pdf

# Generate embeddings (requires Ollama running locally)
./pagepipe convert https://example.com --embeddings --model nomic-embed-text

# Crawl and convert all internal pages
./pagepipe convert https://example.com --all --markdown --output_dir ./site
```

---

## CLI Reference

```
pagepipe convert <url> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--only` | Convert only the given URL | `true` |
| `--all` | Discover and convert all internal sub-pages | `false` |
| `--markdown` | Output as Markdown | — |
| `--json` | Output as structured JSON | — |
| `--pdf` | Output as PDF | — |
| `--embeddings` | Output as embeddings (from Markdown) | — |
| `--model` | Embedding model name (required with `--embeddings`) | — |
| `--chunk_size` | Token chunk size for embeddings | `512` |
| `--output_dir` | Output directory | Current directory |

### Rules

- **Exactly one** output format must be chosen per run (`--markdown`, `--json`, `--pdf`, or `--embeddings`).
- `--only` and `--all` are **mutually exclusive**. If neither is provided, defaults to `--only`.
- `--model` is **required** when using `--embeddings`.

---

## Output Formats

### Markdown (`--markdown`)

Produces a clean `.md` file with the page's main content. Navigation, footers, scripts, and images are stripped.

### JSON (`--json`)

Produces a structured `.json` file with three top-level sections:

```json
{
  "metadata": {
    "url": "https://example.com",
    "domain": "example.com",
    "path": "",
    "title": "Example Domain",
    "language": "en",
    "fetched_at": "2026-02-10T12:57:06Z"
  },
  "content": {
    "text": "plain text (markdown stripped)",
    "markdown": "# original markdown",
    "sections": [
      { "heading": "Section Title", "level": 1, "text": "section body" }
    ]
  },
  "structure": {
    "headings": [{ "level": 1, "text": "Heading" }],
    "links": [{ "text": "link text", "href": "https://..." }],
    "code_blocks": 0,
    "tables": 0,
    "lists": 0
  }
}
```

No business-specific fields (author, price, date, etc.) are inferred. The JSON represents **page structure**, not site semantics.

### PDF (`--pdf`)

Produces a styled `.pdf` with heading hierarchy, code blocks (monospace with background), lists, and paragraph text.

### Embeddings (`--embeddings`)

Produces a human-readable `.embeddings.txt` file. Markdown is split into chunks and each chunk is embedded via the Ollama API (`http://localhost:11434/api/embeddings`).

```
# source: https://example.com
# model: nomic-embed-text
# chunk_size: 512

--- chunk 1 ---
TEXT:
<markdown chunk>

VECTOR:
[0.0123, -0.334, 0.998, ...]
```

---

## Output Naming

| Mode | Example URL | Output File |
|------|-------------|-------------|
| `--only` | `https://example.com` | `example_com.md` |
| `--only` | `https://go.dev/doc/effective_go` | `go_dev_doc_effective_go.md` |
| `--all` | `https://site.com/docs/intro` | `docs/intro.md` |

In `--all` mode, the URL path structure is mirrored as subdirectories.

---

## Architecture

### Pipeline Model

Every URL flows through the same deterministic pipeline:

```
URL
 → Fetch        (HTTP GET → raw HTML)
 → Extract      (HTML → main content, strip noise)
 → Normalize    (cleaned HTML → Markdown)
 → Render       (Markdown → output format)
 → Write        (bytes → file on disk)
```

Markdown is the **canonical intermediate format**. All renderers (PDF, JSON, Embeddings) consume Markdown, never raw HTML.

### Project Structure

```
pagepipe/
├── main.go                         # Entry point → calls cmd.Execute()
│
├── cmd/                            # CLI layer (Cobra)
│   ├── root.go                     # Root "pagepipe" command
│   └── convert.go                  # "convert" subcommand + pipeline orchestration
│
├── core/                           # Pipeline engine
│   ├── interfaces.go               # Fetcher, Extractor, Normalizer, Renderer, Embedder
│   ├── fetch/
│   │   └── fetcher.go              # HTTP client (30s timeout, User-Agent header)
│   ├── extract/
│   │   └── extractor.go            # HTML → main content (<main>/<article>/<body>)
│   ├── normalize/
│   │   └── normalizer.go           # HTML → Markdown (via html-to-markdown)
│   ├── chunk/
│   │   └── chunker.go              # Split text into token-sized chunks
│   ├── render/
│   │   ├── markdown.go             # Passthrough renderer
│   │   ├── json.go                 # Structured JSON with sections & structure
│   │   ├── pdf.go                  # Styled PDF via gofpdf
│   │   └── embeddings.go           # Ollama API embedding renderer
│   └── output/
│       └── writer.go               # File naming (--only flat / --all mirrored)
│
└── crawl/                          # URL discovery (--all mode)
    ├── discover.go                 # Sitemap.xml parsing + link-based BFS
    ├── queue.go                    # BFS queue with URL deduplication
    └── rules.go                    # Same-domain filter, static asset detection
```

### Interfaces

The pipeline is built on clean Go interfaces defined in `core/interfaces.go`:

```go
type Fetcher    interface { Fetch(ctx, url) → (*FetchResult, error) }
type Extractor  interface { Extract(html) → (string, error) }
type Normalizer interface { Normalize(html) → (string, error) }
type Renderer   interface { Render(markdown, meta) → ([]byte, error); Extension() string }
type Embedder   interface { Embed(ctx, text, model) → ([]float64, error) }
```

Each interface has a single implementation, but the design allows easy swapping — for example, replacing the HTTP fetcher with a headless browser fetcher, or adding a new output renderer.

### Crawl Mode (`--all`)

When `--all` is used, the crawl package discovers internal pages before processing:

1. **Try sitemap.xml** — fastest path if the site provides one.
2. **Fall back to BFS link crawling** — follows `<a href>` links on each page.
3. **Filter** — same domain only, no static assets (`.png`, `.css`, `.js`, etc.), no fragments (`#`).
4. **Deduplicate** — URLs are normalized (strip trailing slashes, fragments) and tracked in a visited set.
5. **Cap** — BFS is limited to 100 pages to prevent runaway crawls.

Each discovered URL is processed independently through the same pipeline.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | HTML parsing and content extraction |
| [JohannesKaufmann/html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) | HTML → Markdown conversion |
| [jung-kurt/gofpdf](https://github.com/jung-kurt/gofpdf) | PDF generation |

---

## Design Decisions

- **Markdown as canonical format**: All renderers consume Markdown, not HTML. This ensures consistent output regardless of HTML quirks and makes it trivial to add new renderers.
- **Images are ignored (v1)**: No image downloading, embedding, or rendering. This is an intentional non-goal to keep v1 focused and fast.
- **One output per run**: Keeps the CLI contract simple and predictable. Run the command twice if you need both PDF and JSON.
- **Crawl separated from ingest**: The `crawl/` package only discovers URLs. It has no knowledge of rendering or output formats. The `core/` pipeline has no knowledge of crawling. This separation makes both independently testable.
- **Explicit error messages**: Every validation failure tells the user exactly what went wrong and what their options are.

---

## v1 Limitations

- No image support (intentional)
- No JavaScript rendering (static HTML only)
- No authentication or cookie handling
- Embedding requires a locally running Ollama instance
- BFS crawl capped at 100 pages
- Token chunking uses word count as a proxy (words ≈ tokens)
- Zero chunk overlap for embeddings

---

## License

MIT
