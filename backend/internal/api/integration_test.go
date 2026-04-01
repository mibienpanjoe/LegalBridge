//go:build integration

// Integration tests for the full LegalBridge query pipeline.
//
// These tests require a running backend with a seeded demo document.
// Run with:
//
//	INTEGRATION_SERVER_URL=http://localhost:8080 go test -tags integration ./internal/api/...
//
// The server must be started separately (go run ./cmd/server) with Ollama or
// Groq configured, and the demo document must already be seeded
// (go run ./cmd/seed).
package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// demoQuestions are the five representative questions for the hackathon demo.
// Each must return at least one citation against the seeded demo document.
var demoQuestions = []string{
	"What are the requirements to register a company?",
	"How long does the registration process take?",
	"What documents are needed for registration?",
	"What are the penalties for late registration?",
	"What is the minimum share capital required?",
}

func serverURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("INTEGRATION_SERVER_URL")
	if u == "" {
		t.Skip("INTEGRATION_SERVER_URL not set — skipping integration tests")
	}
	return u
}

// TestDemoQuestions_E2E posts each demo question to the running backend and
// asserts that each response contains at least one citation.
func TestDemoQuestions_E2E(t *testing.T) {
	base := serverURL(t)

	for _, q := range demoQuestions {
		q := q
		t.Run(q, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"question": q})
			resp, err := http.Post(
				base+"/api/query",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("POST /api/query: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}

			var result struct {
				Answer    string `json:"answer"`
				Citations []struct {
					Index        int    `json:"index"`
					DocumentName string `json:"document_name"`
					Passage      string `json:"passage"`
				} `json:"citations"`
				NoResults bool `json:"no_results"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if result.NoResults {
				t.Errorf("no_results=true for question %q — demo document may not be seeded", q)
				return
			}
			if len(result.Citations) == 0 {
				t.Errorf("expected ≥1 citation, got 0 for question %q", q)
			}
			if result.Answer == "" {
				t.Errorf("expected non-empty answer for question %q", q)
			}

			// Record answer for manual review (visible with -v flag).
			t.Logf("answer: %s", result.Answer)
			for _, c := range result.Citations {
				t.Logf("  [%d] %s: %q", c.Index, c.DocumentName, c.Passage)
			}
		})
	}
}

// TestHealth_Integration checks that the health endpoint reports a healthy DB.
func TestHealth_Integration(t *testing.T) {
	base := serverURL(t)

	resp, err := http.Get(base + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected status=ok, got %q", result.Status)
	}
	if result.Database != "ok" {
		t.Errorf("expected database=ok, got %q", result.Database)
	}
	fmt.Printf("health: status=%s database=%s\n", result.Status, result.Database)
}
