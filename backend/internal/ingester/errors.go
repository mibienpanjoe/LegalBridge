package ingester

import "fmt"

// InvalidFileTypeError is returned when the uploaded file is not a valid PDF.
type InvalidFileTypeError struct {
	MIMEType string
}

func (e *InvalidFileTypeError) Error() string {
	return fmt.Sprintf("invalid file type: expected application/pdf, got %s", e.MIMEType)
}

// ExtractionFailedError is returned when text extraction from the PDF fails
// (e.g. the PDF is encrypted, image-only, or corrupted).
type ExtractionFailedError struct {
	Reason string
}

func (e *ExtractionFailedError) Error() string {
	return fmt.Sprintf("text extraction failed: %s", e.Reason)
}

// EmbeddingUnavailableError is returned when the embedding API is unreachable
// or returns a non-2xx response.
type EmbeddingUnavailableError struct {
	Reason string
}

func (e *EmbeddingUnavailableError) Error() string {
	return fmt.Sprintf("embedding service unavailable: %s", e.Reason)
}
