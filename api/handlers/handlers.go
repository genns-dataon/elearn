package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

	courseID := uuid.New().String()
	uploadDir := "./storage/uploads"
	os.MkdirAll(uploadDir, 0755)

	filepath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", courseID, file.Filename))
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

	course := &models.Course{
		ID:        courseID,
		PDFName:   file.Filename,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.db.Create(course).Error; err != nil {
		log.Error().Err(err).Msg("Failed to create course")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
		return
	}

	for i, chunk := range chunks {
		chunkID := uuid.New().String()
		chunkModel := &models.Chunk{
			ID:        chunkID,
			CourseID:  courseID,
			Content:   chunk,
			ChunkNum:  i,
			CreatedAt: time.Now(),
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
	CourseID          string `json:"course_id" binding:"required"`
	NumSlides         int    `json:"num_slides" binding:"required,min=3,max=50"`
	InstructorPrompt  string `json:"instructor_prompt"`
	GenerateImages    bool   `json:"generate_images"`
	GenerateVoiceover bool   `json:"generate_voiceover"`
	GenerateQuestions bool   `json:"generate_questions"`
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
	SlideNumber      int              `json:"slide_number"`
	Title            string           `json:"title"`
	Content          string           `json:"content"`
	InstructorScript string           `json:"instructor_script,omitempty"`
	ImagePrompt      string           `json:"image_prompt,omitempty"`
	Layout           string           `json:"layout,omitempty"`
	Theme            string           `json:"theme,omitempty"`
	Question         *GeneratedQuestion `json:"question,omitempty"`
}

type GeneratedQuestion struct {
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correct_answer"`
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
	for i, chunk := range chunks {
		if i >= 10 {
			break
		}
		contentBuilder.WriteString(chunk.Content)
		contentBuilder.WriteString("\n\n")
	}

	systemPrompt, _ := os.ReadFile("./api/prompts/syllabus_gen.md")
	systemPromptStr := string(systemPrompt)
	systemPromptStr = strings.ReplaceAll(systemPromptStr, "{num_slides}", fmt.Sprintf("%d", req.NumSlides))

	instructorPrompt := req.InstructorPrompt
	if instructorPrompt == "" {
		instructorPrompt = "friendly, conversational level instruction targeted at a general audience"
	}
	systemPromptStr = strings.ReplaceAll(systemPromptStr, "{instructor_style}", instructorPrompt)

	userPrompt := fmt.Sprintf("Generate a course with %d slides from this content:\n\n%s", req.NumSlides, contentBuilder.String())
	if req.GenerateQuestions {
		userPrompt += "\n\nIMPORTANT: Include a 'question' field for each slide with a multiple choice question."
	}

	response, err := h.aiProvider.GenerateText(userPrompt, systemPromptStr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate course")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate course"})
		return
	}

	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var courseStructure GeneratedCourseStructure
	if err := json.Unmarshal([]byte(response), &courseStructure); err != nil {
		log.Error().Err(err).Str("response", response).Msg("Failed to parse course JSON")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated course"})
		return
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

	for _, slide := range courseStructure.Slides {
		// Generate image if prompt exists and image generation is enabled
		var imageURL string
		if req.GenerateImages && slide.ImagePrompt != "" && h.cfg.OpenAIAPIKey != "" {
			generatedURL, err := services.GenerateImage(h.cfg.OpenAIAPIKey, slide.ImagePrompt)
			if err != nil {
				log.Warn().Err(err).Int("slide", slide.SlideNumber).Msg("Failed to generate image")
			} else {
				imageURL = generatedURL
			}
		}

		// Generate voiceover if instructor script exists and voiceover generation is enabled
		var audioURL string
		if req.GenerateVoiceover && slide.InstructorScript != "" && h.cfg.OpenAIAPIKey != "" {
			generatedAudioURL, err := services.GenerateVoiceover(h.cfg.OpenAIAPIKey, slide.InstructorScript, req.CourseID, slide.SlideNumber)
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

		// Save question if it exists
		if slide.Question != nil {
			optionsJSON, _ := json.Marshal(slide.Question.Options)
			questionModel := &models.Question{
				ID:            uuid.New().String(),
				SlideID:       slideID,
				Question:      slide.Question.Question,
				Options:       string(optionsJSON),
				CorrectAnswer: slide.Question.CorrectAnswer,
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
