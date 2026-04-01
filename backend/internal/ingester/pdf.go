package ingester

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

var multiSpace = regexp.MustCompile(`\s{2,}`)

// ExtractText extracts plain text from PDF bytes.
// It returns ExtractionFailedError if the PDF is unreadable or contains no text.
// The returned text is unmodified — callers must not transform it (INV-05).
func ExtractText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", &ExtractionFailedError{Reason: err.Error()}
	}

	plainReader, err := r.GetPlainText()
	if err != nil {
		return "", &ExtractionFailedError{Reason: err.Error()}
	}

	content, err := io.ReadAll(plainReader)
	if err != nil {
		return "", &ExtractionFailedError{Reason: err.Error()}
	}

	text := normaliseWhitespace(string(content))
	if text == "" {
		return "", &ExtractionFailedError{Reason: "no text content found in PDF"}
	}

	return text, nil
}

func normaliseWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
