package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/local/elearn/api/config"
	"github.com/local/elearn/api/models"
	"github.com/local/elearn/api/services"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Handler struct {
	db                *gorm.DB
	cfg               *config.Config
	aiProvider        services.AIProvider
	embeddingProvider services.EmbeddingProvider
}

func New(db *gorm.DB, cfg *config.Config) *Handler {
	var aiProvider services.AIProvider
	if cfg.ModelProvider == "openai" {
		aiProvider = services.NewAIProvider("openai", cfg.OpenAIAPIKey, cfg.OpenAIModel)
	} else {
		aiProvider = services.NewAIProvider("anthropic", cfg.AnthropicAPIKey, cfg.AnthropicModel)
	}

	embeddingProvider := services.NewEmbeddingProvider(
		cfg.EmbeddingProvider,
		cfg.OpenAIAPIKey,
		cfg.EmbeddingModel,
		cfg.OllamaHost,
	)

	return &Handler{
		db:                db,
		cfg:               cfg,
		aiProvider:        aiProvider,
		embeddingProvider: embeddingProvider,
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":             "ok",
		"model_provider":     h.cfg.ModelProvider,
		"embedding_provider": h.cfg.EmbeddingProvider,
	})
}

type UploadResponse struct {
	CourseID string `json:"course_id"`
	PDFName  string `json:"pdf_name"`
	Message  string `json:"message"`
}

func (h *Handler) UploadPDF(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	if file.Size > h.cfg.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 50MB limit"})
		return
	}

	// Check if course_id is provided (for adding files to an existing course)
	courseID := c.PostForm("course_id")
	isNewCourse := false

	if courseID == "" {
		// New course - generate a new ID
		courseID = uuid.New().String()
		isNewCourse = true
	} else {
		// Verify the course exists
		var existingCourse models.Course
		if err := h.db.Where("id = ?", courseID).First(&existingCourse).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
	}

	uploadDir := "./storage/uploads"
	os.MkdirAll(uploadDir, 0755)

	sourceFileID := uuid.New().String()
	filepath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", sourceFileID, file.Filename))
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		log.Error().Err(err).Msg("Failed to save file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	text, err := services.ExtractTextFromPDF(filepath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract text")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract text from PDF"})
		return
	}

	chunks := services.ChunkText(text)

	// Create course record only if this is a new course
	if isNewCourse {
		course := &models.Course{
			ID:        courseID,
			PDFName:   file.Filename, // Legacy field - first uploaded file
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := h.db.Create(course).Error; err != nil {
			log.Error().Err(err).Msg("Failed to create course")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
			return
		}
	}

	// Create source file record
	sourceFile := &models.SourceFile{
		ID:        sourceFileID,
		CourseID:  courseID,
		Filename:  file.Filename,
		FilePath:  filepath,
		FileSize:  file.Size,
		CreatedAt: time.Now(),
	}
	if err := h.db.Create(sourceFile).Error; err != nil {
		log.Error().Err(err).Msg("Failed to create source file record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create source file record"})
		return
	}

	for i, chunk := range chunks {
		chunkID := uuid.New().String()
		chunkModel := &models.Chunk{
			ID:           chunkID,
			CourseID:     courseID,
			SourceFileID: sourceFileID,
			Content:      chunk,
			ChunkNum:     i,
			CreatedAt:    time.Now(),
		}
		if err := h.db.Create(chunkModel).Error; err != nil {
			log.Warn().Err(err).Int("chunk", i).Msg("Failed to save chunk")
			continue
		}

		embedding, err := h.embeddingProvider.Embed(chunk)
		if err != nil {
			log.Warn().Err(err).Int("chunk", i).Msg("Failed to generate embedding")
			continue
		}

		vectorJSON, _ := json.Marshal(embedding)
		embeddingModel := &models.Embedding{
			ID:        uuid.New().String(),
			ChunkID:   chunkID,
			Vector:    string(vectorJSON),
			Dimension: h.embeddingProvider.GetDimension(),
			Model:     h.embeddingProvider.GetModelName(),
			CreatedAt: time.Now(),
		}
		if err := h.db.Create(embeddingModel).Error; err != nil {
			log.Warn().Err(err).Msg("Failed to save embedding")
		}
	}

	c.JSON(http.StatusOK, UploadResponse{
		CourseID: courseID,
		PDFName:  file.Filename,
		Message:  fmt.Sprintf("PDF uploaded and processed into %d chunks", len(chunks)),
	})
}

type GenerateCourseRequest struct {
	CourseID           string `json:"course_id" binding:"required"`
	NumSlides          int    `json:"num_slides" binding:"required,min=3,max=50"`
	PresentationStyle  string `json:"presentation_style"`
	InstructorPrompt   string `json:"instructor_prompt"`
	GenerateImages     bool   `json:"generate_images"`
	UseWebImages       bool   `json:"use_web_images"`
	UseDalle           bool   `json:"use_dalle"`
	GenerateVoiceover  bool   `json:"generate_voiceover"`
	GenerateQuestions  bool   `json:"generate_questions"`
	Language           string `json:"language"`
}

type GenerateCourseResponse struct {
	CourseID string                   `json:"course_id"`
	Course   GeneratedCourseStructure `json:"course"`
}

type GeneratedCourseStructure struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Slides      []GeneratedSlide `json:"slides"`
}

type GeneratedSlide struct {
	SlideNumber      int                     `json:"slide_number"`
	Title            string                  `json:"title"`
	Content          string                  `json:"content"`
	InstructorScript string                  `json:"instructor_script,omitempty"`
	ImagePrompt      string                  `json:"image_prompt,omitempty"`
	Layout           string                  `json:"layout,omitempty"`
	Theme            string                  `json:"theme,omitempty"`
	Question         json.RawMessage         `json:"question,omitempty"` // Use RawMessage to handle inconsistent format
	ParsedQuestion   *GeneratedQuestion      `json:"-"`                  // Parsed question data
}

type GeneratedQuestion struct {
	Question            string      `json:"question"`
	Options             []string    `json:"options"`
	CorrectAnswerRaw    interface{} `json:"correct_answer"` // Can be int or string
	CorrectAnswerParsed int         `json:"-"`              // Parsed index
}

func (h *Handler) GenerateCourse(c *gin.Context) {
	var req GenerateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var chunks []models.Chunk
	if err := h.db.Where("course_id = ?", req.CourseID).Order("chunk_num ASC").Find(&chunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chunks"})
		return
	}

	if len(chunks) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No content found for this course"})
		return
	}

	contentBuilder := strings.Builder{}
	const maxContentLength = 12000 // Limit total content to ~12k characters to avoid Cloudflare blocking
	const maxChunks = 5             // Limit to 5 chunks maximum
	for i, chunk := range chunks {
		if i >= maxChunks {
			break
		}
		// Check if adding this chunk would exceed the limit
		if contentBuilder.Len()+len(chunk.Content) > maxContentLength {
			log.Info().Int("chunks_used", i).Int("total_length", contentBuilder.Len()).Msg("Reached content length limit")
			break
		}
		contentBuilder.WriteString(chunk.Content)
		contentBuilder.WriteString("\n\n")
	}

	log.Info().
		Int("total_content_length", contentBuilder.Len()).
		Bool("generate_images", req.GenerateImages).
		Bool("use_web_images", req.UseWebImages).
		Bool("use_dalle", req.UseDalle).
		Bool("generate_voiceover", req.GenerateVoiceover).
		Bool("generate_questions", req.GenerateQuestions).
		Msg("Content prepared for course generation")

	systemPrompt, _ := os.ReadFile("./api/prompts/syllabus_gen.md")
	systemPromptStr := string(systemPrompt)
	systemPromptStr = strings.ReplaceAll(systemPromptStr, "{num_slides}", fmt.Sprintf("%d", req.NumSlides))

	// Apply presentation style guidelines
	presentationStyle := req.PresentationStyle
	if presentationStyle == "" {
		presentationStyle = "balanced"
	}

	styleInstructions := map[string]string{
		"minimal": "MINIMAL & VISUAL STYLE:\n- Use VERY SHORT, punchy text (2-3 sentences max per slide)\n- Focus on bold statements and key takeaways\n- Emphasize visual impact with striking image prompts\n- Modern, clean design aesthetic\n- Generate vivid, eye-catching image prompts for modern stock photos",
		"balanced": "BALANCED STYLE:\n- Use moderate amount of text (4-6 sentences per slide)\n- Mix of explanations and key points\n- Balance between text and visual elements\n- Professional yet accessible\n- Generate clear, relevant image prompts for professional stock photos",
		"detailed": "DETAILED & PROFESSIONAL STYLE:\n- Use comprehensive, information-rich content with 5-8 bullet points per slide\n- Format content as clear bullet points (use '-' or '•' prefix)\n- Each bullet should be a complete, detailed point with specific information\n- Include specific examples, data points, and thorough coverage\n- Professional corporate presentation aesthetic with substantial on-screen text\n- Information-dense slides suitable for detailed handouts\n- Generate businesslike, professional image prompts for corporate stock photos",
		"fun": "FUN & ENTERTAINING STYLE:\n- Use MINIMAL text with playful, engaging language (2-4 sentences)\n- Emphasize entertainment value and engagement\n- Light, fun tone throughout\n- Use creative, unexpected angles\n- Generate playful, colorful, dynamic image prompts for fun stock photos",
	}

	if styleGuide, ok := styleInstructions[presentationStyle]; ok {
		systemPromptStr += "\n\n" + styleGuide
	}

	instructorPrompt := req.InstructorPrompt
	if instructorPrompt == "" {
		instructorPrompt = "friendly, conversational level instruction targeted at a general audience"
	}
	systemPromptStr = strings.ReplaceAll(systemPromptStr, "{instructor_style}", instructorPrompt)

	// Determine language for content generation
	language := req.Language
	if language == "" {
		language = "english"
	}

	// Add language instruction to system prompt (stronger enforcement)
	if language != "english" {
		languageMap := map[string]string{
			"indonesian": "Indonesian (Bahasa Indonesia)",
			"thai":       "Thai",
			"german":     "German",
		}
		if langName, ok := languageMap[language]; ok {
			languageInstruction := fmt.Sprintf("\n\n**CRITICAL LANGUAGE REQUIREMENT:**\nYou MUST generate ALL course content in %s language ONLY.\nThis includes:\n- Course title and description\n- All slide titles\n- All slide content\n- All instructor scripts\n- All quiz questions and options\n\nDo NOT use English anywhere in the course content. The ENTIRE course must be in %s.", langName, langName)
			systemPromptStr += languageInstruction
		}
	}

	userPrompt := fmt.Sprintf("Generate a JSON course with %d slides from this content:\n\n%s", req.NumSlides, contentBuilder.String())
	userPrompt += "\n\nCRITICAL: You MUST include the 'instructor_script' field for EVERY slide with 3-5 paragraphs of presentation content."
	if req.GenerateQuestions {
		userPrompt += "\n\nIMPORTANT: Include a 'question' field for EVERY slide. The question MUST be a JSON object (not a string) with this exact structure:\n{\n  \"question\": \"Question text here?\",\n  \"options\": [\"Option 1\", \"Option 2\", \"Option 3\", \"Option 4\"],\n  \"correct_answer\": 0\n}\nDo NOT use a string for the question field. It must be a JSON object."
	}

	response, err := h.aiProvider.GenerateJSON(userPrompt, systemPromptStr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate course")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate course"})
		return
	}

	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Log first 500 chars of response to debug
	log.Info().Str("response_preview", response[:min(500, len(response))]).Msg("AI response received")

	var courseStructure GeneratedCourseStructure

	// Try to parse the response - handle multiple formats
	var wrappedResponse struct {
		Course GeneratedCourseStructure `json:"course"`
	}

	// Also handle array format {"course":[...]}
	var arrayResponse struct {
		Course []GeneratedSlide `json:"course"`
	}

	// First try wrapped format (OpenAI) - {"course":{"title":"","slides":[...]}}
	if err := json.Unmarshal([]byte(response), &wrappedResponse); err == nil {
		log.Info().
			Int("wrapped_slides_count", len(wrappedResponse.Course.Slides)).
			Str("wrapped_title", wrappedResponse.Course.Title).
			Msg("Successfully unmarshaled to wrapped format")

		if len(wrappedResponse.Course.Slides) > 0 {
			courseStructure = wrappedResponse.Course
			log.Info().Msg("Parsed wrapped course structure (OpenAI format)")
		} else {
			log.Warn().Msg("Wrapped structure parsed but no slides found, trying other formats")
			// Try array format {"course":[...]}
			if err := json.Unmarshal([]byte(response), &arrayResponse); err == nil && len(arrayResponse.Course) > 0 {
				courseStructure.Slides = arrayResponse.Course
				log.Info().
					Int("array_slides_count", len(arrayResponse.Course)).
					Msg("Parsed array format")
			} else {
				// Try direct format
				if err := json.Unmarshal([]byte(response), &courseStructure); err != nil {
					log.Error().Err(err).Str("response", response).Msg("Failed to parse course JSON in all formats")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated course"})
					return
				}
				log.Info().
					Int("direct_slides_count", len(courseStructure.Slides)).
					Str("direct_title", courseStructure.Title).
					Msg("Parsed direct course structure")
			}
		}
	} else {
		log.Warn().Err(err).Msg("Failed to parse wrapped format, trying other formats")
		// Try array format {"course":[...]}
		if err := json.Unmarshal([]byte(response), &arrayResponse); err == nil && len(arrayResponse.Course) > 0 {
			courseStructure.Slides = arrayResponse.Course
			log.Info().
				Int("array_slides_count", len(arrayResponse.Course)).
				Msg("Parsed array format")
		} else {
			// Fall back to direct parsing (for Anthropic)
			if err := json.Unmarshal([]byte(response), &courseStructure); err != nil {
				log.Error().Err(err).Str("response", response).Msg("Failed to parse course JSON")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated course"})
				return
			}
			log.Info().
				Int("direct_slides_count", len(courseStructure.Slides)).
				Str("direct_title", courseStructure.Title).
				Msg("Parsed direct course structure (Anthropic format)")
		}
	}

	// Debug log to see what was parsed
	if len(courseStructure.Slides) > 0 {
		log.Info().
			Int("total_slides", len(courseStructure.Slides)).
			Str("first_slide_title", courseStructure.Slides[0].Title).
			Int("first_slide_number", courseStructure.Slides[0].SlideNumber).
			Int("content_length", len(courseStructure.Slides[0].Content)).
			Msg("Parsed course structure")
	} else {
		log.Warn().Msg("No slides in parsed structure")
	}

	if err := h.db.Model(&models.Course{}).Where("id = ?", req.CourseID).Updates(map[string]interface{}{
		"title":       courseStructure.Title,
		"description": courseStructure.Description,
		"num_slides":  len(courseStructure.Slides),
		"updated_at":  time.Now(),
	}).Error; err != nil {
		log.Error().Err(err).Msg("Failed to update course")
	}

	h.db.Where("course_id = ?", req.CourseID).Delete(&models.Slide{})

	for i, slide := range courseStructure.Slides {
		// Parse question if it exists
		if len(slide.Question) > 0 {
			var parsedQ GeneratedQuestion
			if err := json.Unmarshal(slide.Question, &parsedQ); err != nil {
				log.Warn().Err(err).Int("slide", i+1).Str("question_raw", string(slide.Question)).Msg("Failed to parse question, skipping")
			} else {
				// Convert correct_answer to index
				switch v := parsedQ.CorrectAnswerRaw.(type) {
				case float64:
					// Already a number
					parsedQ.CorrectAnswerParsed = int(v)
				case string:
					// Try to parse as number first
					if idx, err := strconv.Atoi(v); err == nil {
						parsedQ.CorrectAnswerParsed = idx
					} else {
						// It's a text answer - find matching option
						foundIndex := -1
						for optIdx, opt := range parsedQ.Options {
							if strings.TrimSpace(strings.ToLower(opt)) == strings.TrimSpace(strings.ToLower(v)) {
								foundIndex = optIdx
								break
							}
						}
						if foundIndex >= 0 {
							parsedQ.CorrectAnswerParsed = foundIndex
						} else {
							log.Warn().
								Str("correct_answer_text", v).
								Int("slide", i+1).
								Msg("Could not match correct_answer text to any option, defaulting to 0")
							parsedQ.CorrectAnswerParsed = 0
						}
					}
				default:
					log.Warn().Int("slide", i+1).Msg("Unknown correct_answer type, defaulting to 0")
					parsedQ.CorrectAnswerParsed = 0
				}
				slide.ParsedQuestion = &parsedQ
			}
		}

		// Fix slide numbering - ensure it starts from 1
		if slide.SlideNumber == 0 {
			slide.SlideNumber = i + 1
		}

		// Determine slide template and theme based on content
		isTitle := slide.SlideNumber == 1 || strings.ToLower(slide.Layout) == "title"
		hasImage := req.GenerateImages && slide.ImagePrompt != ""
		template := services.GetSlideTemplate(slide.SlideNumber, slide.Title, slide.Content, isTitle, hasImage)

		// Apply template to slide if not already set
		if slide.Layout == "" {
			slide.Layout = template.Layout
		}
		if slide.Theme == "" {
			slide.Theme = template.Theme
		}

		// Generate image if prompt exists and image generation is enabled
		var imageURL string
		log.Info().
			Int("slide", slide.SlideNumber).
			Bool("req_generate_images", req.GenerateImages).
			Str("image_prompt", slide.ImagePrompt).
			Bool("will_generate_image", req.GenerateImages && slide.ImagePrompt != "").
			Msg("Image generation check")

		if req.GenerateImages && slide.ImagePrompt != "" {
			// Enhance the image prompt based on template
			enhancedPrompt := services.GetImagePromptEnhanced(slide.ImagePrompt, template)

			// Use the new intelligent image fetching
			var err error
			imageURL, err = services.GetImageForSlide(enhancedPrompt, req.UseWebImages, req.UseDalle, h.cfg.OpenAIAPIKey)
			if err != nil {
				log.Warn().Err(err).Int("slide", slide.SlideNumber).Msg("Failed to get image")
			} else {
				log.Info().Str("url", imageURL).Int("slide", slide.SlideNumber).Msg("Image obtained for slide")
			}
		}

		// Generate voiceover if instructor script exists and voiceover generation is enabled
		var audioURL string
		if req.GenerateVoiceover && slide.InstructorScript != "" && h.cfg.OpenAIAPIKey != "" {
			generatedAudioURL, err := services.GenerateVoiceover(h.cfg.OpenAIAPIKey, slide.InstructorScript, req.CourseID, language, slide.SlideNumber)
			if err != nil {
				log.Warn().Err(err).Int("slide", slide.SlideNumber).Msg("Failed to generate voiceover")
			} else {
				audioURL = generatedAudioURL
			}
		}

		slideID := uuid.New().String()
		slideModel := &models.Slide{
			ID:               slideID,
			CourseID:         req.CourseID,
			SlideNumber:      slide.SlideNumber,
			Title:            slide.Title,
			Content:          slide.Content,
			InstructorScript: slide.InstructorScript,
			ImagePrompt:      slide.ImagePrompt,
			ImageURL:         imageURL,
			AudioURL:         audioURL,
			Layout:           slide.Layout,
			Theme:            slide.Theme,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := h.db.Create(slideModel).Error; err != nil {
			log.Warn().Err(err).Msg("Failed to save slide")
		}

		// Save question if it exists and was successfully parsed
		if slide.ParsedQuestion != nil {
			optionsJSON, _ := json.Marshal(slide.ParsedQuestion.Options)
			questionModel := &models.Question{
				ID:            uuid.New().String(),
				SlideID:       slideID,
				Question:      slide.ParsedQuestion.Question,
				Options:       string(optionsJSON),
				CorrectAnswer: slide.ParsedQuestion.CorrectAnswerParsed,
				CreatedAt:     time.Now(),
			}
			if err := h.db.Create(questionModel).Error; err != nil {
				log.Warn().Err(err).Msg("Failed to save question")
			}
		}
	}

	c.JSON(http.StatusOK, GenerateCourseResponse{
		CourseID: req.CourseID,
		Course:   courseStructure,
	})
}

func (h *Handler) GetCourse(c *gin.Context) {
	courseID := c.Param("courseId")

	var course models.Course
	if err := h.db.Where("id = ?", courseID).First(&course).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	c.JSON(http.StatusOK, course)
}

func (h *Handler) GetSlides(c *gin.Context) {
	courseID := c.Param("courseId")

	var slides []models.Slide
	if err := h.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve slides"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID,
		"slides":    slides,
	})
}

func (h *Handler) GetSourceFiles(c *gin.Context) {
	courseID := c.Param("courseId")

	var files []models.SourceFile
	if err := h.db.Where("course_id = ?", courseID).Order("created_at ASC").Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID,
		"files":     files,
	})
}

func (h *Handler) DeleteSourceFile(c *gin.Context) {
	courseID := c.Param("courseId")
	fileID := c.Param("fileId")

	// Get the source file
	var sourceFile models.SourceFile
	if err := h.db.Where("id = ? AND course_id = ?", fileID, courseID).First(&sourceFile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete associated chunks and their embeddings
	var chunks []models.Chunk
	h.db.Where("source_file_id = ?", fileID).Find(&chunks)

	for _, chunk := range chunks {
		// Delete embeddings for this chunk
		h.db.Where("chunk_id = ?", chunk.ID).Delete(&models.Embedding{})
	}

	// Delete chunks
	h.db.Where("source_file_id = ?", fileID).Delete(&models.Chunk{})

	// Delete the physical file
	if err := os.Remove(sourceFile.FilePath); err != nil {
		log.Warn().Err(err).Str("file_path", sourceFile.FilePath).Msg("Failed to delete physical file")
	}

	// Delete the source file record
	if err := h.db.Delete(&sourceFile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record"})
		return
	}

	// Check if this was the last file for the course
	var remainingFiles []models.SourceFile
	h.db.Where("course_id = ?", courseID).Find(&remainingFiles)

	if len(remainingFiles) == 0 {
		// Delete the entire course and all related data
		h.db.Where("course_id = ?", courseID).Delete(&models.Slide{})
		h.db.Where("course_id = ?", courseID).Delete(&models.ChatMessage{})
		h.db.Delete(&models.Course{}, "id = ?", courseID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "File deleted successfully",
		"files_remaining": len(remainingFiles),
	})
}

type ChatRequest struct {
	CourseID string `json:"course_id" binding:"required"`
	Question string `json:"question" binding:"required"`
}

type ChatResponse struct {
	Answer    string   `json:"answer"`
	Citations []string `json:"citations"`
}

func (h *Handler) GetQuestions(c *gin.Context) {
	courseID := c.Param("courseId")

	var slides []models.Slide
	if err := h.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve slides"})
		return
	}

	type QuestionResponse struct {
		SlideID       string   `json:"slide_id"`
		Question      string   `json:"question"`
		Options       []string `json:"options"`
		CorrectAnswer int      `json:"correct_answer"`
	}

	var responses []QuestionResponse
	for _, slide := range slides {
		var question models.Question
		if err := h.db.Where("slide_id = ?", slide.ID).First(&question).Error; err == nil {
			var options []string
			json.Unmarshal([]byte(question.Options), &options)
			responses = append(responses, QuestionResponse{
				SlideID:       slide.ID,
				Question:      question.Question,
				Options:       options,
				CorrectAnswer: question.CorrectAnswer,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID,
		"questions": responses,
	})
}

func (h *Handler) ChatAsk(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queryEmbedding, err := h.embeddingProvider.Embed(req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to embed question"})
		return
	}

	var chunks []models.Chunk
	var embeddings []models.Embedding

	h.db.Where("course_id = ?", req.CourseID).Find(&chunks)
	h.db.Joins("JOIN chunks ON embeddings.chunk_id = chunks.id").
		Where("chunks.course_id = ?", req.CourseID).
		Find(&embeddings)

	type ChunkWithScore struct {
		Chunk models.Chunk
		Score float64
	}
	var scoredChunks []ChunkWithScore

	for _, emb := range embeddings {
		var vector []float64
		json.Unmarshal([]byte(emb.Vector), &vector)

		score := services.CosineSimilarity(queryEmbedding, vector)

		for _, chunk := range chunks {
			if chunk.ID == emb.ChunkID {
				scoredChunks = append(scoredChunks, ChunkWithScore{
					Chunk: chunk,
					Score: score,
				})
				break
			}
		}
	}

	topK := 6
	if len(scoredChunks) < topK {
		topK = len(scoredChunks)
	}

	for i := 0; i < len(scoredChunks); i++ {
		for j := i + 1; j < len(scoredChunks); j++ {
			if scoredChunks[j].Score > scoredChunks[i].Score {
				scoredChunks[i], scoredChunks[j] = scoredChunks[j], scoredChunks[i]
			}
		}
	}

	topChunks := scoredChunks[:topK]

	contextBuilder := strings.Builder{}
	citations := []string{}
	for i, sc := range topChunks {
		contextBuilder.WriteString(fmt.Sprintf("[Chunk %d] %s\n\n", i+1, sc.Chunk.Content))
		citations = append(citations, fmt.Sprintf("Chunk %d (similarity: %.2f)", i+1, sc.Score))
	}

	systemPromptBytes, _ := os.ReadFile("./api/prompts/answer_grounded.md")
	systemPrompt := string(systemPromptBytes)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{context}", contextBuilder.String())
	systemPrompt = strings.ReplaceAll(systemPrompt, "{question}", req.Question)

	answer, err := h.aiProvider.GenerateText(req.Question, systemPrompt)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate answer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate answer"})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		Answer:    answer,
		Citations: citations,
	})
}
