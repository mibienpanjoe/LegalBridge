package ingester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

// EmbeddingClient generates a vector embedding for a single piece of text.
type EmbeddingClient interface {
	Embed(ctx context.Context, text, model string) ([]float32, error)
}

// NewEmbeddingClient returns the EmbeddingClient for the provider in cfg.
func NewEmbeddingClient(cfg *config.Config) EmbeddingClient {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	switch cfg.EmbeddingProvider {
	case "huggingface":
		return &HuggingFaceClient{apiKey: cfg.HFAPIKey, http: httpClient}
	default: // "ollama" or unset
		return &OllamaClient{baseURL: cfg.OllamaBaseURL, http: httpClient}
	}
}

// OllamaClient calls the local Ollama /api/embeddings endpoint.
type OllamaClient struct {
	baseURL string
	http    *http.Client
}

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *OllamaClient) Embed(ctx context.Context, text, model string) ([]float32, error) {
	body, err := json.Marshal(ollamaEmbedRequest{Model: model, Prompt: text})
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, &EmbeddingUnavailableError{
			Reason: fmt.Sprintf("ollama returned %d: %s", resp.StatusCode, b),
		}
	}

	var out ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	return out.Embedding, nil
}
