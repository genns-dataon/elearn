package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ModelProvider     string
	AnthropicAPIKey   string
	AnthropicModel    string
	OpenAIAPIKey      string
	OpenAIModel       string
	EmbeddingProvider string
	EmbeddingModel    string
	OllamaHost        string
	Port              string
	DBPath            string
	MaxUploadSize     int64
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		ModelProvider:     getEnv("MODEL_PROVIDER", "anthropic"),
		AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:    getEnv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:       getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		EmbeddingProvider: getEnv("EMBEDDING_PROVIDER", "openai"),
		EmbeddingModel:    getEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		OllamaHost:        getEnv("OLLAMA_HOST", "http://localhost:11434"),
		Port:              getEnv("PORT", "8080"),
		DBPath:            getEnv("DB_PATH", "./storage/elearn.db"),
		MaxUploadSize:     52428800, // 50MB default
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
