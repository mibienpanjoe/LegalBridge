package query

import (
	"sort"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// rrfK is the reciprocal rank fusion constant; 60 is the value from the
// original RRF paper and works well without tuning.
const rrfK = 60

// lexicalSignificance is the fraction of the best BM25 score a candidate needs
// to earn a lexical rank. Without it, a word shared by every candidate (such
// as "article") would hand each of them full rank credit for a near-zero score.
const lexicalSignificance = 0.5

// fuseRankings reorders vector candidates (nearest-first) by reciprocal rank
// fusion of their vector rank and their BM25 rank, and returns the best topK.
//
// Only the given candidates are ranked — every one of them already passed the
// similarity threshold — so lexical matching can promote a passage but never
// admit one the embedding judged irrelevant (INV-06).
func fuseRankings(candidates []store.Chunk, ix *lexicalIndex, question string, topK int) []store.Chunk {
	terms := tokenize(question)

	type scored struct {
		chunk   store.Chunk
		vecRank int
		lexical float64
		fused   float64
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		items[i] = scored{chunk: c, vecRank: i + 1, lexical: ix.score(c.ID, terms)}
	}

	// Lexical rank among candidates with a significant BM25 score.
	best := 0.0
	for _, it := range items {
		best = max(best, it.lexical)
	}
	byLexical := make([]int, len(items))
	for i := range byLexical {
		byLexical[i] = i
	}
	sort.SliceStable(byLexical, func(a, b int) bool {
		return items[byLexical[a]].lexical > items[byLexical[b]].lexical
	})
	for rank, i := range byLexical {
		items[i].fused = 1.0 / float64(rrfK+items[i].vecRank)
		if best > 0 && items[i].lexical >= lexicalSignificance*best {
			items[i].fused += 1.0 / float64(rrfK+rank+1)
		}
	}

	sort.SliceStable(items, func(a, b int) bool { return items[a].fused > items[b].fused })

	n := min(topK, len(items))
	out := make([]store.Chunk, n)
	for i := 0; i < n; i++ {
		out[i] = items[i].chunk
	}
	return out
}
