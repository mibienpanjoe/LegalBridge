package query

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

func TestTokenize(t *testing.T) {
	got := tokenize("Article 37 : La période d’essai de l'employé — CNSS, 2 mois.")
	want := []string{"article", "37", "la", "periode", "essai", "de", "employe", "cnss", "2", "mois"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokenize:\n got %q\nwant %q", got, want)
	}
}

func TestLexicalIndex_RareTermsOutweighCommonOnes(t *testing.T) {
	chunks := []store.Chunk{
		{ID: "a", Content: "Le salaire de la travailleuse est payé à la fin du mois de la période."},
		{ID: "b", Content: "La période d'essai ne peut excéder la durée prévue de la convention."},
		{ID: "c", Content: "Le travailleur a droit au repos de la semaine et de la journée."},
	}
	ix := newLexicalIndex(chunks)
	q := tokenize("Quelle est la durée de la période d'essai ?")

	if ix.score("b", q) <= ix.score("a", q) || ix.score("b", q) <= ix.score("c", q) {
		t.Errorf("expected chunk b (essai, durée) to score highest: a=%.3f b=%.3f c=%.3f",
			ix.score("a", q), ix.score("b", q), ix.score("c", q))
	}
	if s := ix.score("missing", q); s != 0 {
		t.Errorf("expected 0 for unknown chunk, got %f", s)
	}
}

func TestFuseRankings_PromotesExactTermMatch(t *testing.T) {
	// Vector order puts the chunk that actually contains "Article 37" fifth.
	var candidates []store.Chunk
	for i := 1; i <= 6; i++ {
		candidates = append(candidates, store.Chunk{
			ID:      fmt.Sprintf("c%d", i),
			Content: fmt.Sprintf("Article %d : dispositions relatives au contrat de travail.", 30+i),
		})
	}
	candidates[4].Content = "Article 37 : la période d'essai est renouvelable une fois."

	ix := newLexicalIndex(candidates)
	got := fuseRankings(candidates, ix, "Que dit l'article 37 sur l'essai ?", 3)

	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got))
	}
	if got[0].ID != "c5" {
		t.Errorf("expected the Article 37 chunk first, got order %s %s %s", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestFuseRankings_KeepsVectorOrderWithoutLexicalSignal(t *testing.T) {
	candidates := []store.Chunk{{ID: "x", Content: "alpha"}, {ID: "y", Content: "beta"}, {ID: "z", Content: "gamma"}}
	got := fuseRankings(candidates, newLexicalIndex(candidates), "omega", 2)
	if got[0].ID != "x" || got[1].ID != "y" {
		t.Errorf("expected vector order x, y; got %s, %s", got[0].ID, got[1].ID)
	}
}

// TestQuery_HybridRerankBuildsIndexOnce verifies that the BM25 index is built
// from the store on the first query only and reused afterwards.
func TestQuery_HybridRerankBuildsIndexOnce(t *testing.T) {
	var chunks []store.Chunk
	for i := 0; i < 5; i++ {
		chunks = append(chunks, store.Chunk{ID: fmt.Sprint(i), Content: strings.Repeat("mot ", i+1), DocumentFilename: "doc.pdf"})
	}
	fs := &fakeStore{chunks: chunks}
	p := NewQueryProcessor(fs, &fakeEmbedder{}, &fakeLLM{response: "Réponse [1]."})

	for i := 0; i < 3; i++ {
		if _, err := p.Query(context.Background(), testDocID, "mot ?"); err != nil {
			t.Fatalf("query %d: %v", i, err)
		}
	}
	if fs.contentsCalls != 1 {
		t.Errorf("expected the lexical index to be loaded once, got %d loads", fs.contentsCalls)
	}
}
