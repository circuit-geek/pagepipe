// Package chunk splits Markdown text into token-sized chunks for embedding.
// Uses a simple whitespace tokenizer (words ≈ tokens) for v1.
// Chunk overlap is 0 per spec.
package chunk

import "strings"

// Chunker splits text into fixed-size token chunks.
type Chunker struct {
	ChunkSize int // number of tokens (words) per chunk
}

// New creates a Chunker with the given chunk size.
// Defaults to 512 if chunkSize <= 0.
func New(chunkSize int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	return &Chunker{ChunkSize: chunkSize}
}

// Chunk splits the input text into slices of at most ChunkSize words.
// Each chunk is a contiguous block of words joined by spaces.
func (c *Chunker) Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	for i := 0; i < len(words); i += c.ChunkSize {
		end := i + c.ChunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
	}
	return chunks
}
