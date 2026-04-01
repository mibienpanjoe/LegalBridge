package ingester

import "strings"

const (
	chunkSize    = 500 // words per chunk
	chunkOverlap = 50  // overlapping words between consecutive chunks
)

// Chunk splits text into overlapping word-boundary chunks.
// Each chunk is approximately chunkSize words; consecutive chunks share
// chunkOverlap words at their boundaries.
func Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	step := chunkSize - chunkOverlap
	var chunks []string

	for start := 0; start < len(words); start += step {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[start:end], " "))
		if end == len(words) {
			break
		}
	}

	return chunks
}
