package ingester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HuggingFaceClient calls the HuggingFace Inference API feature-extraction pipeline.
type HuggingFaceClient struct {
	apiKey string
	http   *http.Client
}

type hfEmbedRequest struct {
	Inputs string `json:"inputs"`
}

func (c *HuggingFaceClient) Embed(ctx context.Context, text, model string) ([]float32, error) {
	url := fmt.Sprintf("https://router.huggingface.co/hf-inference/models/%s/pipeline/feature-extraction", model)

	body, err := json.Marshal(hfEmbedRequest{Inputs: text})
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, &EmbeddingUnavailableError{
			Reason: fmt.Sprintf("huggingface returned %d: %s", resp.StatusCode, b),
		}
	}

	// HF router returns a flat []float32 (shape: dim) for a single string input.
	var out []float32
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, &EmbeddingUnavailableError{Reason: err.Error()}
	}
	if len(out) == 0 {
		return nil, &EmbeddingUnavailableError{Reason: "empty embedding response"}
	}
	return out, nil
}
