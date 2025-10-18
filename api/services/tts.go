package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// GenerateVoiceover generates an audio file using OpenAI TTS
func GenerateVoiceover(apiKey, text, courseID, language string, slideNumber int) (string, error) {
	url := "https://api.openai.com/v1/audio/speech"

	// Select voice based on language for better pronunciation
	voice := "alloy" // default voice
	// OpenAI TTS supports multiple languages automatically based on input text
	// We just use the best general voice for all languages

	reqBody := map[string]interface{}{
		"model": "tts-1",
		"voice": voice,
		"input": text,
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Create audio directory
	audioDir := filepath.Join("./storage/audio", courseID)
	os.MkdirAll(audioDir, 0755)

	// Save audio file
	audioFilename := fmt.Sprintf("slide_%d_%s.mp3", slideNumber, uuid.New().String()[:8])
	audioPath := filepath.Join(audioDir, audioFilename)

	audioFile, err := os.Create(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to create audio file: %w", err)
	}
	defer audioFile.Close()

	_, err = io.Copy(audioFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save audio: %w", err)
	}

	// Return relative URL path
	return fmt.Sprintf("/audio/%s/%s", courseID, audioFilename), nil
}
