package ingester

import (
	"fmt"
	"strings"
	"testing"
)

// words returns n distinct words "prefix0 prefix1 …".
func words(prefix string, n int) string {
	w := make([]string, n)
	for i := range w {
		w[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return strings.Join(w, " ")
}

func TestChunkPages_Empty(t *testing.T) {
	if got := ChunkPages(nil); len(got) != 0 {
		t.Fatalf("expected 0 segments for no pages, got %d", len(got))
	}
}

func TestChunkPages_PacksShortArticles(t *testing.T) {
	pages := []Page{
		{Number: 3, Lines: []string{
			"TITRE I - DISPOSITIONS GENERALES",
			"Article 1 : La présente loi est applicable aux travailleurs.",
			"Article 2 : Est considérée comme travailleur toute personne.",
		}},
		{Number: 4, Lines: []string{
			"Article 3 : Les agents de la fonction publique ne sont pas soumis.",
		}},
	}
	got := ChunkPages(pages)
	if len(got) != 1 {
		t.Fatalf("expected short articles packed into 1 segment, got %d", len(got))
	}
	seg := got[0]
	if seg.Section != "Article 1–3" {
		t.Errorf("expected section %q, got %q", "Article 1–3", seg.Section)
	}
	if seg.PageStart != 3 || seg.PageEnd != 4 {
		t.Errorf("expected pages 3–4, got %d–%d", seg.PageStart, seg.PageEnd)
	}
	// INV-05: lines are kept verbatim and in order.
	want := strings.Join(append(pages[0].Lines, pages[1].Lines...), "\n")
	if seg.Text != want {
		t.Errorf("segment text altered:\n got %q\nwant %q", seg.Text, want)
	}
}

func TestChunkPages_ChunksStartAtArticleBoundaries(t *testing.T) {
	// Two articles that together exceed chunkSize must not share a chunk.
	pages := []Page{{Number: 1, Lines: []string{
		"Article 5 : " + words("a", 200),
		"Article 6 : " + words("b", 200),
	}}}
	got := ChunkPages(pages)
	if len(got) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(got))
	}
	if got[0].Section != "Article 5" || got[1].Section != "Article 6" {
		t.Errorf("unexpected sections %q, %q", got[0].Section, got[1].Section)
	}
	if !strings.HasPrefix(got[1].Text, "Article 6 :") {
		t.Errorf("second segment does not start at its article heading: %.30q", got[1].Text)
	}
}

func TestChunkPages_SplitsLongArticleWithOverlap(t *testing.T) {
	var lines []string
	lines = append(lines, "Article 9 :")
	for i := 0; i < 40; i++ {
		lines = append(lines, words(fmt.Sprintf("l%d_", i), 20)) // 800 words in 20-word lines
	}
	got := ChunkPages([]Page{{Number: 7, Lines: lines}})
	if len(got) < 3 {
		t.Fatalf("expected ≥3 windows for an 800-word article, got %d", len(got))
	}
	for i, seg := range got {
		if n := len(strings.Fields(seg.Text)); n > chunkSize {
			t.Errorf("segment %d has %d words, exceeds chunkSize %d", i, n, chunkSize)
		}
		if seg.Section != "Article 9" {
			t.Errorf("segment %d: expected section %q, got %q", i, "Article 9", seg.Section)
		}
	}
	// Consecutive windows share their boundary line(s).
	firstLines := strings.Split(got[0].Text, "\n")
	if !strings.HasPrefix(got[1].Text, firstLines[len(firstLines)-1]) {
		t.Error("expected the second window to start with the first window's last line (overlap)")
	}
}

func TestChunkPages_NoHeadingsFallsBackToWindows(t *testing.T) {
	// A contract-like document: one 1000-word paragraph, no article headings.
	got := ChunkPages([]Page{{Number: 1, Lines: []string{words("w", 1000)}}})
	if len(got) < 3 {
		t.Fatalf("expected ≥3 segments for 1000 words, got %d", len(got))
	}
	first := strings.Fields(got[0].Text)
	second := strings.Fields(got[1].Text)
	if len(first) != chunkSize {
		t.Errorf("expected first window of %d words, got %d", chunkSize, len(first))
	}
	if second[0] != first[chunkSize-chunkOverlap] {
		t.Errorf("expected %d-word overlap: second window starts with %q, want %q",
			chunkOverlap, second[0], first[chunkSize-chunkOverlap])
	}
	if got[0].Section != "" {
		t.Errorf("expected no section label, got %q", got[0].Section)
	}
}

func TestChunkPages_SectionsGroupArticlesInCodes(t *testing.T) {
	pages := []Page{{Number: 1, Lines: []string{
		"Section 1 : Formation et orientation professionnelles",
		"Article 7 : Il est créé un conseil national.",
		"Article 8 : La formation professionnelle est l’ensemble des activités.",
	}}}
	got := ChunkPages(pages)
	if len(got) != 1 || got[0].Section != "Article 7–8" {
		t.Fatalf("expected one segment labelled %q, got %+v", "Article 7–8", got)
	}

	// Without articles (common-law statute), sections are the units.
	pages = []Page{{Number: 1, Lines: []string{
		"Section 12 — Registration of foreign companies",
		words("a", 200),
		"Section 13 — Documents to be delivered",
		words("b", 200),
	}}}
	got = ChunkPages(pages)
	if len(got) != 2 || got[0].Section != "Section 12" || got[1].Section != "Section 13" {
		t.Fatalf("expected sections 12 and 13 as units, got %d segments", len(got))
	}
}

func TestHeadingRe(t *testing.T) {
	cases := map[string]string{
		"Article 1 : La présente loi":     "Article 1",
		"Article premier.":                "Article premier",
		"ARTICLE 4 BIS - Dispositions":    "Article 4 bis",
		"Art. 3 : Le contrat":             "Article 3",
		"Section 12 — Registration":       "Section 12",
		"Article 25":                      "Article 25",
		"Article 12 de la présente loi":   "",
		"article 7 : lowercase reference": "",
		"Les articles 3 et 4 :":           "",
	}
	for in, want := range cases {
		m := articleRe.FindStringSubmatch(in)
		if m == nil {
			m = sectionRe.FindStringSubmatch(in)
		}
		got := ""
		if m != nil {
			got = normaliseLabel(m[1])
		}
		if got != want {
			t.Errorf("%q: got label %q, want %q", in, got, want)
		}
	}
}

func TestRemoveRunningHeaders(t *testing.T) {
	bodies := []string{"Le travail forcé est interdit.", "Le salaire est payé en monnaie.",
		"La durée du travail est fixée.", "Le repos hebdomadaire est obligatoire.",
		"Les congés payés sont dus.", "Le préavis est réciproque."}
	var pages []Page
	for i := 1; i <= 6; i++ {
		pages = append(pages, Page{Number: i, Lines: []string{
			fmt.Sprintf("Page %d sur 116MINISTERE DU TRAVAIL ET%d DE LA SECURITE", i, i),
			fmt.Sprintf("%d", i), // bare page number
			"SOCIALE /BURKINA FASO",
			fmt.Sprintf("Article %d :", i), // heading alone on its line: same key on every page
			bodies[i-1],
		}})
	}
	got := removeRunningHeaders(pages)
	for _, p := range got {
		want := []string{fmt.Sprintf("Article %d :", p.Number), bodies[p.Number-1]}
		if strings.Join(p.Lines, "|") != strings.Join(want, "|") {
			t.Errorf("page %d: expected %q, got %q", p.Number, want, p.Lines)
		}
	}
}
