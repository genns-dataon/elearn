package services

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

const (
	ChunkSize    = 1000 // characters per chunk
	ChunkOverlap = 200  // overlap between chunks
)

// ExtractTextFromPDF extracts all text from a PDF file
func ExtractTextFromPDF(filepath string) (string, error) {
	f, r, err := pdf.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			// Continue even if one page fails
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	return textBuilder.String(), nil
}

// ChunkText splits text into overlapping chunks
func ChunkText(text string) []string {
	if len(text) == 0 {
		return []string{}
	}

	var chunks []string
	runes := []rune(text)
	start := 0

	for start < len(runes) {
		end := start + ChunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[start:end])
		chunk = strings.TrimSpace(chunk)

		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		// Move forward, but overlap
		start += ChunkSize - ChunkOverlap
	}

	return chunks
}
