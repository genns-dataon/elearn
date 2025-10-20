package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/local/elearn/api/config"
	"github.com/local/elearn/api/db"
	"github.com/local/elearn/api/handlers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	log.Info().Str("db_path", cfg.DBPath).Msg("Database initialized")

	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize handlers
	h := handlers.New(database, cfg)

	// Health check
	router.GET("/api/health", h.Health)

	// Serve static audio files
	router.Static("/audio", "./storage/audio")

	// API routes
	api := router.Group("/api")
	{
		api.POST("/upload", h.UploadPDF)
		api.POST("/course/generate", h.GenerateCourse)
		api.GET("/course/:courseId", h.GetCourse)
		api.GET("/slides/:courseId", h.GetSlides)
		api.GET("/files/:courseId", h.GetSourceFiles)
		api.DELETE("/files/:courseId/:fileId", h.DeleteSourceFile)
		api.GET("/questions/:courseId", h.GetQuestions)
		api.POST("/chat/ask", h.ChatAsk)
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Info().
		Str("port", cfg.Port).
		Str("model_provider", cfg.ModelProvider).
		Str("embedding_provider", cfg.EmbeddingProvider).
		Msg("Starting eLearning API server")

	if err := router.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
