package query

import (
	"fmt"
	"strings"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// systemInstructions is the invariant-bearing part of every RAG prompt.
//
// INV-01: "Answer using only the provided passages. Do not use external knowledge."
// INV-03: "You do NOT provide legal advice, legal opinions, or legal counsel."
// INV-07: "Respond in the same language as the user's question."
//
// These strings must remain present in any future edit of this constant.
const systemInstructions = `You are a legal document retrieval assistant for West African businesses.
You provide information based exclusively on the legal documents provided to you.
You do NOT provide legal advice, legal opinions, or legal counsel.
You MUST cite the specific passage(s) from the provided context that support each claim in your answer.
Respond in the same language as the user's question.
Answer using only the provided passages. Do not use external knowledge.
For each claim, reference the passage number in square brackets (e.g., "[1]", "[2]").`

// BuildRAGPrompt constructs the system prompt sent to the LLM.
// It embeds the context passages and all invariant instructions.
// The caller passes the question separately as the user message.
func BuildRAGPrompt(chunks []store.Chunk) string {
	var sb strings.Builder
	sb.WriteString(systemInstructions)
	sb.WriteString("\n\nContext passages:\n")
	for i, c := range chunks {
		fmt.Fprintf(&sb, "[%d]%s %s\n\n", i+1, passageLocation(c), c.Content)
	}
	sb.WriteString("Instructions: Answer the question using only the context passages above. " +
		"Do not use any knowledge beyond what is provided in the passages.")
	return sb.String()
}

// passageLocation renders " (Article 12, p. 4)" so the model can name the
// article it cites. It returns "" when nothing is known about the location.
func passageLocation(c store.Chunk) string {
	var parts []string
	if c.Section != "" {
		parts = append(parts, c.Section)
	}
	switch {
	case c.PageStart > 0 && c.PageEnd > c.PageStart:
		parts = append(parts, fmt.Sprintf("p. %d–%d", c.PageStart, c.PageEnd))
	case c.PageStart > 0:
		parts = append(parts, fmt.Sprintf("p. %d", c.PageStart))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
