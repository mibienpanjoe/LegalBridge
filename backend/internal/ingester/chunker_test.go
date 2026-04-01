package ingester

import (
	"fmt"
	"strings"
	"testing"
)

func TestChunk_Empty(t *testing.T) {
	if got := Chunk(""); len(got) != 0 {
		t.Fatalf("expected 0 chunks for empty input, got %d", len(got))
	}
}

func TestChunk_SingleChunk(t *testing.T) {
	// 100 words fits inside one chunk (chunkSize = 500)
	text := strings.TrimSpace(strings.Repeat("word ", 100))
	chunks := Chunk(text)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for 100 words, got %d", len(chunks))
	}
}

func TestChunk_ExactChunkSize(t *testing.T) {
	// Exactly chunkSize words → still one chunk, no second chunk needed
	words := make([]string, chunkSize)
	for i := range words {
		words[i] = "word"
	}
	chunks := Chunk(strings.Join(words, " "))
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for exactly %d words, got %d", chunkSize, len(chunks))
	}
}

func TestChunk_MultipleChunks(t *testing.T) {
	// 1000 words → step = 450, so we need more than one chunk
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	chunks := Chunk(strings.Join(words, " "))
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for 1000 words, got %d", len(chunks))
	}
}

func TestChunk_Overlap(t *testing.T) {
	// Build 600 uniquely-named words so we can verify overlap by content.
	words := make([]string, 600)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	chunks := Chunk(strings.Join(words, " "))
	if len(chunks) < 2 {
		t.Fatal("expected at least 2 chunks for overlap test")
	}

	chunk0 := strings.Fields(chunks[0])
	chunk1 := strings.Fields(chunks[1])

	// The last chunkOverlap words of chunk[0] must equal the first chunkOverlap of chunk[1].
	tail0 := chunk0[len(chunk0)-chunkOverlap:]
	head1 := chunk1[:chunkOverlap]

	for i, w := range tail0 {
		if w != head1[i] {
			t.Fatalf("overlap mismatch at index %d: chunk0 tail=%q, chunk1 head=%q", i, w, head1[i])
		}
	}
}

func TestChunk_ChunkWordCount(t *testing.T) {
	// All chunks except possibly the last must have exactly chunkSize words.
	words := make([]string, 1200)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	chunks := Chunk(strings.Join(words, " "))

	for i, c := range chunks[:len(chunks)-1] {
		n := len(strings.Fields(c))
		if n != chunkSize {
			t.Errorf("chunk[%d] has %d words, want %d", i, n, chunkSize)
		}
	}
}
