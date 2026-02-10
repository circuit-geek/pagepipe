// Package render — Embeddings renderer.
// Generates embeddings from Markdown by chunking the text and calling
// an Ollama-compatible embedding API for each chunk.
// Output is a human-readable .embeddings.txt file.
package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gaurav-prasanna/pagepipe/core"
	"github.com/gaurav-prasanna/pagepipe/core/chunk"
)

const (
	defaultOllamaURL = "http://localhost:11434/api/embeddings"
	embeddingTimeout = 60 * time.Second
)

// EmbeddingsRenderer generates embeddings from Markdown chunks.
type EmbeddingsRenderer struct {
	Model     string
	ChunkSize int
	client    *http.Client
}

// NewEmbeddingsRenderer creates an EmbeddingsRenderer.
func NewEmbeddingsRenderer(model string, chunkSize int) *EmbeddingsRenderer {
	return &EmbeddingsRenderer{
		Model:     model,
		ChunkSize: chunkSize,
		client:    &http.Client{Timeout: embeddingTimeout},
	}
}

// ollamaRequest is the request body for the Ollama embeddings API.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaResponse is the response body from the Ollama embeddings API.
type ollamaResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Render chunks the Markdown, embeds each chunk, and produces
// the human-readable .embeddings.txt output.
func (r *EmbeddingsRenderer) Render(markdown string, meta core.PageMetadata) ([]byte, error) {
	chunker := chunk.New(r.ChunkSize)
	chunks := chunker.Chunk(markdown)

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content to embed")
	}

	var buf strings.Builder
	// Write header.
	fmt.Fprintf(&buf, "# source: %s\n", meta.URL)
	fmt.Fprintf(&buf, "# model: %s\n", r.Model)
	fmt.Fprintf(&buf, "# chunk_size: %d\n\n", r.ChunkSize)

	ctx := context.Background()
	for i, chunkText := range chunks {
		embedding, err := r.embed(ctx, chunkText)
		if err != nil {
			return nil, fmt.Errorf("embedding chunk %d: %w", i+1, err)
		}

		fmt.Fprintf(&buf, "--- chunk %d ---\n", i+1)
		fmt.Fprintf(&buf, "TEXT:\n%s\n\n", chunkText)

		// Format vector.
		vecStrs := make([]string, len(embedding))
		for j, v := range embedding {
			vecStrs[j] = fmt.Sprintf("%.4f", v)
		}
		fmt.Fprintf(&buf, "VECTOR:\n[%s]\n\n", strings.Join(vecStrs, ", "))
	}

	return []byte(buf.String()), nil
}

// Extension returns the file extension for embeddings output.
func (r *EmbeddingsRenderer) Extension() string {
	return ".embeddings.txt"
}

// embed calls the Ollama embedding API for a single text input.
func (r *EmbeddingsRenderer) embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := ollamaRequest{
		Model:  r.Model,
		Prompt: text,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultOllamaURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding Ollama response: %w", err)
	}

	return ollamaResp.Embedding, nil
}
