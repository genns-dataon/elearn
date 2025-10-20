package services

import (
	"strings"
)

// SlideTemplate defines different layout templates
type SlideTemplate struct {
	Layout           string
	Theme            string
	ImagePlacement   string // "background", "inline", "split", "none"
	ImageSize        string // "large", "medium", "small"
	UseImage         bool
	ContentAlignment string // "left", "center"
}

// GetSlideTemplate intelligently selects a template based on slide characteristics
func GetSlideTemplate(slideNumber int, title string, content string, isTitle bool, hasImage bool) SlideTemplate {
	// Title slide (first slide)
	if slideNumber == 1 || isTitle {
		return SlideTemplate{
			Layout:           "title",
			Theme:            "gradient",
			ImagePlacement:   "background",
			ImageSize:        "large",
			UseImage:         hasImage,
			ContentAlignment: "center",
		}
	}

	// Summary/conclusion slide (usually last or contains summary keywords)
	contentLower := strings.ToLower(content)
	titleLower := strings.ToLower(title)
	if strings.Contains(titleLower, "summary") ||
		strings.Contains(titleLower, "conclusion") ||
		strings.Contains(titleLower, "recap") ||
		strings.Contains(contentLower, "in summary") {
		return SlideTemplate{
			Layout:           "summary",
			Theme:            "purple",
			ImagePlacement:   "inline",
			ImageSize:        "medium",
			UseImage:         hasImage,
			ContentAlignment: "left",
		}
	}

	// List/bullet point slides (content with bullets or numbered items)
	if strings.Contains(content, "\n- ") ||
		strings.Contains(content, "\n• ") ||
		strings.Contains(content, "\n1.") ||
		strings.Contains(content, "\n2.") {
		return SlideTemplate{
			Layout:           "list",
			Theme:            "blue",
			ImagePlacement:   "split",
			ImageSize:        "medium",
			UseImage:         hasImage,
			ContentAlignment: "left",
		}
	}

	// Comparison slides (contains "vs", "versus", "compared to")
	if strings.Contains(contentLower, " vs ") ||
		strings.Contains(contentLower, "versus") ||
		strings.Contains(contentLower, "compared to") ||
		strings.Contains(contentLower, "difference between") {
		return SlideTemplate{
			Layout:           "comparison",
			Theme:            "orange",
			ImagePlacement:   "split",
			ImageSize:        "medium",
			UseImage:         hasImage,
			ContentAlignment: "left",
		}
	}

	// Data/statistics slides (contains numbers, percentages, data)
	if strings.Contains(contentLower, "%") ||
		strings.Contains(contentLower, "data") ||
		strings.Contains(contentLower, "statistics") ||
		strings.Contains(contentLower, "research shows") {
		return SlideTemplate{
			Layout:           "data",
			Theme:            "green",
			ImagePlacement:   "inline",
			ImageSize:        "small",
			UseImage:         hasImage,
			ContentAlignment: "left",
		}
	}

	// Concept explanation slides (longer content, explanatory)
	contentLength := len(content)
	if contentLength > 300 {
		return SlideTemplate{
			Layout:           "concept",
			Theme:            "blue",
			ImagePlacement:   "background",
			ImageSize:        "large",
			UseImage:         hasImage,
			ContentAlignment: "left",
		}
	}

	// Default template with rotation of themes
	themes := []string{"blue", "green", "purple", "orange"}
	themeIndex := slideNumber % len(themes)

	return SlideTemplate{
		Layout:           "standard",
		Theme:            themes[themeIndex],
		ImagePlacement:   "inline",
		ImageSize:        "medium",
		UseImage:         hasImage,
		ContentAlignment: "left",
	}
}

// GetImagePromptEnhanced creates better image prompts based on template
func GetImagePromptEnhanced(originalPrompt string, template SlideTemplate) string {
	if originalPrompt == "" {
		return ""
	}

	// Add style hints based on image placement
	switch template.ImagePlacement {
	case "background":
		return originalPrompt + ", wide angle, professional background image, subtle, atmospheric"
	case "split":
		return originalPrompt + ", professional photograph, high quality, clear focal point"
	case "inline":
		return originalPrompt + ", icon style, clean, professional illustration"
	default:
		return originalPrompt
	}
}
