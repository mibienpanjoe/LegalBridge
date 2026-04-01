package config

import (
	"errors"
	"os"
)

// EmbeddingModel is the single source of truth for the embedding model name.
// Both ingester and query packages must import and use this constant — never
// hardcode the model name at a call site (INV-04).
const EmbeddingModel = "BAAI/bge-m3"

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL string

	// EmbeddingProvider is "ollama" (local dev) or "huggingface" (production).
	EmbeddingProvider string

	// LLMProvider is "ollama" (local dev) or "groq" (production).
	LLMProvider string

	// Ollama — used when EmbeddingProvider or LLMProvider is "ollama".
	OllamaBaseURL string
	OllamaModel   string

	// HuggingFace — used when EmbeddingProvider is "huggingface".
	HFAPIKey string

	// Groq — used when LLMProvider is "groq".
	GroqAPIKey string

	// CORSAllowedOrigin is an additional origin to allow in production
	// (set to the Vercel frontend URL). localhost:3000 is always allowed.
	CORSAllowedOrigin string
}

// Load reads all configuration from environment variables. It returns a Config
// with defaults applied for optional fields but does not validate required
// fields — call Validate() for that.
func Load() *Config {
	return &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		EmbeddingProvider: getEnvOrDefault("EMBEDDING_PROVIDER", "ollama"),
		LLMProvider:       getEnvOrDefault("LLM_PROVIDER", "ollama"),
		OllamaBaseURL:     getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:       getEnvOrDefault("OLLAMA_MODEL", "llama3.2"),
		HFAPIKey:          os.Getenv("HF_API_KEY"),
		GroqAPIKey:        os.Getenv("GROQ_API_KEY"),
		CORSAllowedOrigin: os.Getenv("CORS_ALLOWED_ORIGIN"),
	}
}

// Validate checks that all required fields for the configured providers are
// present. It returns an error describing the first missing field.
//
// INV-08: if a production API key is empty, the server must not start.
func Validate(c *Config) error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	switch c.EmbeddingProvider {
	case "huggingface":
		if c.HFAPIKey == "" {
			return errors.New("HF_API_KEY is required when EMBEDDING_PROVIDER=huggingface")
		}
	case "ollama":
		// no key required
	default:
		return errors.New("EMBEDDING_PROVIDER must be 'ollama' or 'huggingface'")
	}

	switch c.LLMProvider {
	case "groq":
		if c.GroqAPIKey == "" {
			return errors.New("GROQ_API_KEY is required when LLM_PROVIDER=groq")
		}
	case "ollama":
		// no key required
	default:
		return errors.New("LLM_PROVIDER must be 'ollama' or 'groq'")
	}

	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
