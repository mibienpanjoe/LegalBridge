package query

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

// LLMClient generates a completion given a system prompt and user message.
type LLMClient interface {
	Complete(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// NewLLMClient returns the LLMClient for the provider in cfg.
func NewLLMClient(cfg *config.Config) LLMClient {
	// 10-second timeout enforced at the HTTP client level (FR-051).
	httpClient := &http.Client{Timeout: 10 * time.Second}
	switch cfg.LLMProvider {
	case "groq":
		return &GroqClient{apiKey: cfg.GroqAPIKey, http: httpClient}
	default: // "ollama" or unset
		return &OllamaLLMClient{baseURL: cfg.OllamaBaseURL, model: cfg.OllamaModel, http: httpClient}
	}
}

// ─── shared OpenAI-compatible chat types ────────────────────────────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ─── GroqClient ─────────────────────────────────────────────────────────────

const (
	groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"
	groqModel    = "llama-3.3-70b-versatile"
)

// GroqClient calls the Groq chat completions API.
type GroqClient struct {
	apiKey string
	http   *http.Client
}

func (c *GroqClient) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return doChat(ctx, c.http, groqEndpoint, groqModel, systemPrompt, userMessage,
		func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		},
	)
}

// ─── OllamaLLMClient ────────────────────────────────────────────────────────

// OllamaLLMClient uses Ollama's OpenAI-compatible /v1/chat/completions endpoint
// so both Groq and Ollama share the same request/response format.
type OllamaLLMClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func (c *OllamaLLMClient) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return doChat(ctx, c.http, c.baseURL+"/v1/chat/completions", c.model, systemPrompt, userMessage, nil)
}

// ─── shared helper ──────────────────────────────────────────────────────────

func doChat(
	ctx context.Context,
	client *http.Client,
	endpoint, model, systemPrompt, userMessage string,
	setHeaders func(*http.Request),
) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
	})
	if err != nil {
		return "", &LLMUnavailableError{Reason: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", &LLMUnavailableError{Reason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if setHeaders != nil {
		setHeaders(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", &LLMUnavailableError{Reason: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", &LLMUnavailableError{
			Reason: fmt.Sprintf("API returned %d: %s", resp.StatusCode, b),
		}
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", &LLMUnavailableError{Reason: err.Error()}
	}
	if len(out.Choices) == 0 {
		return "", &LLMUnavailableError{Reason: "API returned no choices"}
	}
	return out.Choices[0].Message.Content, nil
}
