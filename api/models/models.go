package models

import (
	"time"

	"gorm.io/gorm"
)

// Course represents a generated course from a PDF
type Course struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PDFName     string    `json:"pdf_name"`
	NumSlides   int       `json:"num_slides"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Slide represents a single slide in a course
type Slide struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	CourseID         string    `gorm:"index" json:"course_id"`
	SlideNumber      int       `json:"slide_number"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	InstructorScript string    `json:"instructor_script,omitempty"` // Full script for the instructor to present this slide
	ImagePrompt      string    `json:"image_prompt,omitempty"`
	ImageURL         string    `json:"image_url,omitempty"`
	AudioURL         string    `json:"audio_url,omitempty"` // URL to TTS audio file
	Layout           string    `json:"layout,omitempty"`    // "default", "title", "quote", "highlight", "comparison"
	Theme            string    `json:"theme,omitempty"`     // "blue", "green", "purple", "orange", "gradient"
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Chunk represents a text chunk from a PDF with metadata
type Chunk struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	CourseID  string    `gorm:"index" json:"course_id"`
	Content   string    `json:"content"`
	ChunkNum  int       `json:"chunk_num"`
	PageNum   int       `json:"page_num,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Embedding represents a vector embedding for a chunk
type Embedding struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	ChunkID    string    `gorm:"uniqueIndex" json:"chunk_id"`
	Vector     string    `json:"vector"` // JSON-encoded float array
	Dimension  int       `json:"dimension"`
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"created_at"`
}

// ChatMessage represents a chat interaction
type ChatMessage struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	CourseID  string    `gorm:"index" json:"course_id"`
	Role      string    `json:"role"` // user, assistant
	Content   string    `json:"content"`
	Citations []string  `gorm:"-" json:"citations,omitempty"` // Not stored, just for response
	CreatedAt time.Time `json:"created_at"`
}

// Question represents a quiz question for a slide
type Question struct {
	ID            string   `gorm:"primaryKey" json:"id"`
	SlideID       string   `gorm:"index" json:"slide_id"`
	Question      string   `json:"question"`
	Options       string   `json:"options"`        // JSON-encoded array of options
	CorrectAnswer int      `json:"correct_answer"` // Index of correct answer (0-3)
	CreatedAt     time.Time `json:"created_at"`
}

// AutoMigrate runs all migrations
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Course{},
		&Slide{},
		&Chunk{},
		&Embedding{},
		&ChatMessage{},
		&Question{},
	)
}
