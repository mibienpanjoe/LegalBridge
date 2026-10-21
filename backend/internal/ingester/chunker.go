package ingester

import (
	"regexp"
	"strings"
)

const (
	// chunkSize is the word budget per chunk. 350 words of French or English
	// legal text is roughly 500 bge-m3 tokens (FR-013).
	chunkSize = 350
	// chunkOverlap is the word overlap between consecutive windows when a
	// single article is too long to fit one chunk (~50 tokens).
	chunkOverlap = 35
)

// articleRe and sectionRe match a line that opens an article or section, e.g.
// "Article 1 : …", "Article 12.", "ARTICLE 4 bis -", "Article premier",
// "Art. 3 :", "Section 12 —". The separator (or end of line) after the number
// avoids matching wrapped cross-references such as "Article 12 de la loi".
var (
	articleRe = headingPattern(`Article|ARTICLE|Art\.`)
	sectionRe = headingPattern(`Section|SECTION`)
)

func headingPattern(keywords string) *regexp.Regexp {
	return regexp.MustCompile(`^((?:` + keywords + `)\s+(?:\d+(?:\.\d+)*|premier|1er|PREMIER)` +
		`(?:\s*(?:bis|ter|quater|BIS|TER|QUATER))?)\s*(?:[:.\-–—]|$)`)
}

// Segment is a chunk of document text with its location in the source PDF.
type Segment struct {
	Text      string
	PageStart int
	PageEnd   int
	Section   string // e.g. "Article 12" or "Article 12–14"; empty when unknown
}

type line struct {
	text  string
	page  int
	words int
}

// unit is an article (heading line plus body) or, before the first heading or
// in documents without headings, a run of plain lines.
type unit struct {
	label string
	lines []line
	words int
}

// ChunkPages splits extracted pages into retrieval segments.
//
// Articles are the natural unit of legal text, so chunks start at article
// headings: consecutive short articles are packed together up to chunkSize
// words, and an article longer than chunkSize is split into overlapping
// windows. Documents without recognisable headings (contracts, letters) fall
// back to overlapping windows over their lines. Line text is never altered
// (INV-05); lines are joined with newlines to keep paragraph structure.
func ChunkPages(pages []Page) []Segment {
	units := splitUnits(pages)

	var segments []Segment
	var pack []unit
	packWords := 0

	flush := func() {
		if len(pack) > 0 {
			segments = append(segments, packSegment(pack))
			pack, packWords = nil, 0
		}
	}

	for _, u := range units {
		switch {
		case u.words > chunkSize:
			flush()
			segments = append(segments, splitUnit(u)...)
		case packWords+u.words > chunkSize:
			flush()
			pack, packWords = []unit{u}, u.words
		default:
			pack = append(pack, u)
			packWords += u.words
		}
	}
	flush()
	return segments
}

// splitUnits cuts the document at its article headings. In civil-law codes
// "Section" is a division that groups articles, so sections only delimit units
// in documents that have no articles at all (e.g. common-law statutes).
func splitUnits(pages []Page) []unit {
	headingRe := sectionRe
	for _, p := range pages {
		for _, text := range p.Lines {
			if articleRe.MatchString(text) {
				headingRe = articleRe
			}
		}
	}

	var units []unit
	cur := unit{}
	for _, p := range pages {
		for _, text := range p.Lines {
			l := line{text: text, page: p.Number, words: len(strings.Fields(text))}
			if m := headingRe.FindStringSubmatch(text); m != nil {
				if len(cur.lines) > 0 {
					units = append(units, cur)
				}
				cur = unit{label: normaliseLabel(m[1])}
			}
			cur.lines = append(cur.lines, l)
			cur.words += l.words
		}
	}
	if len(cur.lines) > 0 {
		units = append(units, cur)
	}
	return units
}

// packSegment joins whole units into one segment.
func packSegment(units []unit) Segment {
	var lines []line
	var labels []string
	for _, u := range units {
		lines = append(lines, u.lines...)
		if u.label != "" {
			labels = append(labels, u.label)
		}
	}
	return newSegment(lines, rangeLabel(labels))
}

// splitUnit cuts an oversized unit into overlapping windows of whole lines.
// A single line longer than chunkSize is first cut into word pieces.
func splitUnit(u unit) []Segment {
	var lines []line
	for _, l := range u.lines {
		lines = append(lines, splitLongLine(l)...)
	}

	var segments []Segment
	start := 0
	for start < len(lines) {
		end, words := start, 0
		for end < len(lines) && (end == start || words+lines[end].words <= chunkSize) {
			words += lines[end].words
			end++
		}
		segments = append(segments, newSegment(lines[start:end], u.label))
		if end == len(lines) {
			break
		}

		// Step back over trailing lines worth at most chunkOverlap words, but
		// always advance by at least one line.
		next, overlap := end, 0
		for next-1 > start && overlap+lines[next-1].words <= chunkOverlap {
			next--
			overlap += lines[next].words
		}
		start = next
	}
	return segments
}

func splitLongLine(l line) []line {
	if l.words <= chunkSize {
		return []line{l}
	}
	words := strings.Fields(l.text)
	var out []line
	for start := 0; start < len(words); start += chunkSize - chunkOverlap {
		end := min(start+chunkSize, len(words))
		out = append(out, line{text: strings.Join(words[start:end], " "), page: l.page, words: end - start})
		if end == len(words) {
			break
		}
	}
	return out
}

func newSegment(lines []line, label string) Segment {
	texts := make([]string, len(lines))
	for i, l := range lines {
		texts[i] = l.text
	}
	return Segment{
		Text:      strings.Join(texts, "\n"),
		PageStart: lines[0].page,
		PageEnd:   lines[len(lines)-1].page,
		Section:   label,
	}
}

// normaliseLabel turns a matched heading into a display label:
// "ARTICLE 4 BIS" → "Article 4 bis", "Art. 3" → "Article 3".
func normaliseLabel(h string) string {
	fields := strings.Fields(h)
	switch strings.ToLower(strings.TrimSuffix(fields[0], ".")) {
	case "article", "art":
		fields[0] = "Article"
	case "section":
		fields[0] = "Section"
	}
	for i := 1; i < len(fields); i++ {
		fields[i] = strings.ToLower(fields[i])
	}
	return strings.Join(fields, " ")
}

// rangeLabel summarises the labels of packed units: "Article 12" for one,
// "Article 12–14" when first and last share a keyword, otherwise
// "Article 12 – Section 3".
func rangeLabel(labels []string) string {
	switch len(labels) {
	case 0:
		return ""
	case 1:
		return labels[0]
	}
	first, last := labels[0], labels[len(labels)-1]
	fk, fr, _ := strings.Cut(first, " ")
	lk, lr, _ := strings.Cut(last, " ")
	if fk == lk {
		return fk + " " + fr + "–" + lr
	}
	return first + " – " + last
}
