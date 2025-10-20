package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// ImageSearchResult represents a search result from Unsplash
type ImageSearchResult struct {
	URL         string
	Description string
	Author      string
}

// SearchWebImages provides placeholder images using Picsum Photos (no API key needed)
// For production, replace with a proper image search API like Pexels or Unsplash with your own API key
func SearchWebImages(query string) (*ImageSearchResult, error) {
	// Picsum Photos - Free placeholder images, no auth required
	// Generates a random image with seed based on query for consistency
	// Size: 1280x720 (16:9 aspect ratio, perfect for slides)

	// Create a deterministic seed from the query so same query = same image
	seed := hashString(query)
	imageURL := fmt.Sprintf("https://picsum.photos/seed/%d/1280/720", seed)

	// Picsum works by redirecting to actual images, just return the URL
	// No need to verify - it always works
	log.Info().Str("url", imageURL).Int("seed", seed).Msg("Generated Picsum image URL")

	return &ImageSearchResult{
		URL:         imageURL,
		Description: fmt.Sprintf("Professional image for: %s", query),
		Author:      "Picsum Photos",
	}, nil
}

// hashString creates a simple hash from a string for deterministic image selection
func hashString(s string) int {
	hash := 0
	for _, char := range s {
		hash = (hash << 5) - hash + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % 10000 // Limit to reasonable range
}

// GenerateImage generates an image using DALL-E
func GenerateImage(apiKey, prompt string) (string, error) {
	url := "https://api.openai.com/v1/images/generations"

	reqBody := map[string]interface{}{
		"model":   "dall-e-3",
		"prompt":  prompt,
		"n":       1,
		"size":    "1024x1024",
		"quality": "standard",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("no image data in response")
	}

	return result.Data[0].URL, nil
}

// GetImageForSlide intelligently fetches an image based on preferences
// useWebImages: try web search first
// useDalle: use DALL-E as fallback or primary
func GetImageForSlide(imagePrompt string, useWebImages bool, useDalle bool, apiKey string) (string, error) {
	if imagePrompt == "" {
		return "", nil
	}

	var imageURL string
	var err error

	// Try web images first if enabled
	if useWebImages {
		// Extract key search terms from the image prompt
		searchQuery := extractSearchTerms(imagePrompt)
		log.Info().Str("search_query", searchQuery).Msg("Searching for web image")

		result, webErr := SearchWebImages(searchQuery)
		if webErr == nil && result != nil {
			imageURL = result.URL
			log.Info().Str("url", imageURL).Str("author", result.Author).Msg("Found web image")
		} else {
			log.Warn().Err(webErr).Msg("Web image search failed")
			err = webErr
		}
	}

	// If web search failed or wasn't enabled, try DALL-E
	if imageURL == "" && useDalle && apiKey != "" {
		log.Info().Msg("Using DALL-E for image generation")
		imageURL, err = GenerateImage(apiKey, imagePrompt)
		if err != nil {
			log.Warn().Err(err).Msg("DALL-E generation failed")
		}
	}

	if imageURL == "" {
		return "", fmt.Errorf("failed to get image: %w", err)
	}

	return imageURL, nil
}

// extractSearchTerms simplifies the image prompt to better search terms
func extractSearchTerms(prompt string) string {
	// Remove common AI image generation phrases
	prompt = strings.ToLower(prompt)
	prompt = strings.ReplaceAll(prompt, "a professional illustration of", "")
	prompt = strings.ReplaceAll(prompt, "an image showing", "")
	prompt = strings.ReplaceAll(prompt, "a photo of", "")
	prompt = strings.ReplaceAll(prompt, "depicting", "")
	prompt = strings.ReplaceAll(prompt, "illustration", "")
	prompt = strings.ReplaceAll(prompt, "professional", "business")

	// Take first meaningful words (usually the core concept)
	words := strings.Fields(strings.TrimSpace(prompt))
	if len(words) > 4 {
		words = words[:4]
	}

	return strings.Join(words, " ")
}
