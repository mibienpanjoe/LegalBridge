package query

import "fmt"

// EmptyQueryError is returned when the question string is empty or whitespace.
type EmptyQueryError struct{}

func (e *EmptyQueryError) Error() string {
	return "question must not be empty"
}

// NoResultsError is returned when similarity search produces zero results.
// The LLM is never called in this case (INV-06).
type NoResultsError struct{}

func (e *NoResultsError) Error() string {
	return "no relevant passages found for this question"
}

// LLMUnavailableError is returned when the LLM API times out or returns
// a non-2xx response.
type LLMUnavailableError struct {
	Reason string
}

func (e *LLMUnavailableError) Error() string {
	return fmt.Sprintf("LLM service unavailable: %s", e.Reason)
}
