package query

import (
	"math"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// BM25 parameters (standard Okapi values).
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// lexicalIndex scores a document's chunks against a question with BM25.
//
// It complements embedding similarity on exact legal terms — article numbers,
// acronyms (SARL, CNSS), defined words — that dense vectors tend to blur.
// Inverse document frequency makes it language-agnostic: words present in
// nearly every chunk ("de", "la", "the") weigh almost nothing, so no
// French or English stopword list is needed.
type lexicalIndex struct {
	termFreq map[string]map[string]int // chunk ID → term → count
	length   map[string]int            // chunk ID → number of terms
	docFreq  map[string]int            // term → number of chunks containing it
	avgLen   float64
}

func newLexicalIndex(chunks []store.Chunk) *lexicalIndex {
	ix := &lexicalIndex{
		termFreq: make(map[string]map[string]int, len(chunks)),
		length:   make(map[string]int, len(chunks)),
		docFreq:  map[string]int{},
	}
	total := 0
	for _, c := range chunks {
		tf := map[string]int{}
		terms := tokenize(c.Content)
		for _, t := range terms {
			tf[t]++
		}
		for t := range tf {
			ix.docFreq[t]++
		}
		ix.termFreq[c.ID] = tf
		ix.length[c.ID] = len(terms)
		total += len(terms)
	}
	if len(chunks) > 0 {
		ix.avgLen = float64(total) / float64(len(chunks))
	}
	return ix
}

// score returns the BM25 score of chunk id for the given query terms; 0 when
// the chunk shares no term with the query or is not in the index.
func (ix *lexicalIndex) score(id string, queryTerms []string) float64 {
	tf, ok := ix.termFreq[id]
	if !ok || ix.avgLen == 0 {
		return 0
	}
	n := float64(len(ix.termFreq))
	lenNorm := bm25K1 * (1 - bm25B + bm25B*float64(ix.length[id])/ix.avgLen)

	s := 0.0
	seen := map[string]bool{}
	for _, t := range queryTerms {
		if seen[t] {
			continue
		}
		seen[t] = true
		f := float64(tf[t])
		if f == 0 {
			continue
		}
		df := float64(ix.docFreq[t])
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		s += idf * f * (bm25K1 + 1) / (f + lenNorm)
	}
	return s
}

// tokenize lowercases, strips accents ("période" → "periode") and splits on
// anything that is not a letter or digit. Single letters are dropped (elided
// articles such as the "l" of "l'employeur"); numbers of any length are kept
// because article numbers matter.
func tokenize(s string) []string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue // combining accent
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	fields := strings.Fields(b.String())
	terms := fields[:0]
	for _, f := range fields {
		if len([]rune(f)) > 1 || unicode.IsDigit([]rune(f)[0]) {
			terms = append(terms, f)
		}
	}
	return terms
}

// lexicalCache keeps the BM25 index of recently queried documents. A
// document's chunks never change after ingestion, so entries never go stale;
// the oldest entry is evicted once the cache is full.
type lexicalCache struct {
	mu      sync.Mutex
	max     int
	order   []string
	entries map[string]*lexicalIndex
}

func newLexicalCache(max int) *lexicalCache {
	return &lexicalCache{max: max, entries: map[string]*lexicalIndex{}}
}

func (c *lexicalCache) get(documentID string) (*lexicalIndex, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ix, ok := c.entries[documentID]
	return ix, ok
}

func (c *lexicalCache) put(documentID string, ix *lexicalIndex) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[documentID]; ok {
		return
	}
	if len(c.order) >= c.max {
		delete(c.entries, c.order[0])
		c.order = c.order[1:]
	}
	c.order = append(c.order, documentID)
	c.entries[documentID] = ix
}
