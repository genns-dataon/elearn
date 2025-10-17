package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
)

// EmbeddingProvider is the interface for embedding providers
type EmbeddingProvider interface {
	Embed(text string) ([]float64, error)
	GetDimension() int
	GetModelName() string
}

// OpenAIEmbedding implements OpenAI embeddings
type OpenAIEmbedding struct {
	APIKey    string
	Model     string
	Dimension int
}

// OllamaEmbedding implements Ollama local embeddings
type OllamaEmbedding struct {
	Host      string
	Model     string
	Dimension int
}

func NewEmbeddingProvider(provider, apiKey, model, ollamaHost string) EmbeddingProvider {
	switch strings.ToLower(provider) {
	case "openai":
		dim := 1536
		if model == "text-embedding-3-large" {
			dim = 3072
		}
		return &OpenAIEmbedding{
			APIKey:    apiKey,
			Model:     model,
			Dimension: dim,
		}
	case "ollama":
		return &OllamaEmbedding{
			Host:      ollamaHost,
			Model:     model,
			Dimension: 768, // nomic-embed-text dimension
		}
	default:
		return &OpenAIEmbedding{
			APIKey:    apiKey,
			Model:     "text-embedding-3-small",
			Dimension: 1536,
		}
	}
}

func (o *OpenAIEmbedding) GetDimension() int {
	return o.Dimension
}

func (o *OpenAIEmbedding) GetModelName() string {
	return o.Model
}

func (o *OpenAIEmbedding) Embed(text string) ([]float64, error) {
	url := "https://api.openai.com/v1/embeddings"

	reqBody := map[string]interface{}{
		"input": text,
		"model": o.Model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return result.Data[0].Embedding, nil
}

func (ol *OllamaEmbedding) GetDimension() int {
	return ol.Dimension
}

func (ol *OllamaEmbedding) GetModelName() string {
	return ol.Model
}

func (ol *OllamaEmbedding) Embed(text string) ([]float64, error) {
	url := fmt.Sprintf("%s/api/embeddings", ol.Host)

	reqBody := map[string]interface{}{
		"model":  ol.Model,
		"prompt": text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Embedding, nil
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
