package ingester

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

var (
	multiSpace = regexp.MustCompile(`\s+`)
	digits     = regexp.MustCompile(`\d+`)
	nonLetters = regexp.MustCompile(`[^\p{L}]+`)
)

// Page is the text of one PDF page as an ordered list of non-empty lines.
type Page struct {
	Number int // 1-based page number in the source PDF
	Lines  []string
}

// ExtractPages extracts text from PDF bytes, page by page and line by line.
// It returns ExtractionFailedError if the PDF is unreadable or contains no text.
//
// Lines are kept verbatim apart from whitespace normalisation (INV-05). Running
// headers and footers — lines repeated on most pages, such as
// "Page 3 sur 116 MINISTERE DU TRAVAIL" — are layout artifacts, not legal text,
// and are removed so they do not pollute passages and embeddings.
func ExtractPages(data []byte) (pages []Page, err error) {
	// ledongthuc/pdf panics on some malformed content streams.
	defer func() {
		if r := recover(); r != nil {
			pages, err = nil, &ExtractionFailedError{Reason: fmt.Sprintf("malformed PDF: %v", r)}
		}
	}()

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, &ExtractionFailedError{Reason: err.Error()}
	}

	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		rows, err := p.GetTextByRow()
		if err != nil {
			return nil, &ExtractionFailedError{Reason: fmt.Sprintf("page %d: %v", i, err)}
		}

		page := Page{Number: i}
		for _, row := range rows {
			var sb strings.Builder
			for _, t := range row.Content {
				sb.WriteString(t.S)
			}
			if line := normaliseWhitespace(sb.String()); line != "" {
				page.Lines = append(page.Lines, line)
			}
		}
		pages = append(pages, page)
	}

	pages = removeRunningHeaders(pages)

	for _, p := range pages {
		if len(p.Lines) > 0 {
			return pages, nil
		}
	}
	return nil, &ExtractionFailedError{Reason: "no text content found in PDF"}
}

// removeRunningHeaders drops header/footer lines: lines among the first or
// last edgeLines of a page that recur on at least half of the pages (and on at
// least 3) once only their letters are compared, so "Page 3 sur 116 … ET3 DE"
// and "Page 4 sur 116 … DU4TRAVAIL" count as the same line. Article headings are never dropped, even when one
// sits at the top of many pages. Short documents are left untouched.
func removeRunningHeaders(pages []Page) []Page {
	const edgeLines = 3
	if len(pages) < 4 {
		return pages
	}

	key := func(line string) string {
		if k := nonLetters.ReplaceAllString(line, ""); k != "" {
			return strings.ToLower(k)
		}
		return digits.ReplaceAllString(line, "#") // e.g. a bare page number
	}
	isEdge := func(i, n int) bool { return i < edgeLines || i >= n-edgeLines }

	seenOn := map[string]int{}
	for _, p := range pages {
		onPage := map[string]bool{}
		for i, l := range p.Lines {
			if isEdge(i, len(p.Lines)) {
				onPage[key(l)] = true
			}
		}
		for k := range onPage {
			seenOn[k]++
		}
	}

	threshold := max(3, len(pages)/2)
	for pi := range pages {
		lines := pages[pi].Lines
		kept := make([]string, 0, len(lines))
		for i, l := range lines {
			if isEdge(i, len(lines)) && seenOn[key(l)] >= threshold && !articleRe.MatchString(l) && !sectionRe.MatchString(l) {
				continue
			}
			kept = append(kept, l)
		}
		pages[pi].Lines = kept
	}
	return pages
}

func normaliseWhitespace(s string) string {
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}
