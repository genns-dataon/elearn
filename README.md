# eLearning App - RAG-Powered Course Generator

A production-ready eLearning application that transforms PDFs into interactive courses with AI-generated slides and an intelligent chatbot.

## Features

- **PDF Upload & Processing**: Upload PDFs and automatically extract, chunk, and embed content
- **AI-Powered Course Generation**: Generate structured courses with customizable slide counts
- **RAG Chatbot**: Ask questions about the course material with grounded, citation-backed answers
- **Multi-Provider Support**:
  - AI Models: Anthropic Claude or OpenAI GPT
  - Embeddings: OpenAI or Ollama (local)
- **Beautiful UI**: Dark mode by default with Tailwind CSS and shadcn/ui components
- **Vector Search**: Cosine similarity-based retrieval for relevant content chunks

## Tech Stack

**Backend:**
- Go 1.22+ with Gin framework
- SQLite with GORM
- Provider-agnostic AI architecture

**Frontend:**
- React + TypeScript
- Vite for blazing-fast dev experience
- Tailwind CSS + shadcn/ui
- TanStack Query for data fetching

## Prerequisites

- Go 1.22 or higher
- Node.js 16+ and npm
- (Optional) Ollama for local embeddings

## Quick Start

### 1. Clone and Setup

```bash
cd elearn
cp .env.example .env
```

### 2. Configure Environment

Edit `.env` with your API keys:

```bash
# Choose your AI provider
MODEL_PROVIDER=anthropic  # or openai

# Add your API keys
ANTHROPIC_API_KEY=your_anthropic_key_here
OPENAI_API_KEY=your_openai_key_here

# Choose embedding provider
EMBEDDING_PROVIDER=openai  # or ollama for local
EMBEDDING_MODEL=text-embedding-3-small
```

### 3. Install Dependencies

```bash
make install-deps
```

### 4. Run the App

```bash
make dev
```

This starts:
- API server on `http://localhost:8080`
- Web app on `http://localhost:5173`

## Usage

1. **Upload a PDF**: Click "Upload PDF" and select a document
2. **Generate Course**: Choose number of slides (3-50) and generate
3. **View Slides**: Navigate through AI-generated course slides
4. **Ask Questions**: Use the chatbot to ask questions about the material

## API Endpoints

```
GET  /api/health              - Health check
POST /api/upload              - Upload and process PDF
POST /api/course/generate     - Generate course structure
GET  /api/course/:courseId    - Get course details
GET  /api/slides/:courseId    - Get all slides
POST /api/chat/ask            - Ask chatbot a question
```

## Switching AI Providers

### Anthropic Claude

```bash
MODEL_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...
```

### OpenAI

```bash
MODEL_PROVIDER=openai
OPENAI_API_KEY=sk-...
```

## Using Ollama for Local Embeddings

### 1. Install Ollama

```bash
# macOS
brew install ollama

# Start Ollama service
ollama serve
```

### 2. Pull the embedding model

```bash
ollama pull nomic-embed-text
```

### 3. Update .env

```bash
EMBEDDING_PROVIDER=ollama
EMBEDDING_MODEL=nomic-embed-text
OLLAMA_HOST=http://localhost:11434
```

## Project Structure

```
elearn/
├── api/
│   ├── main.go              # Server entry point
│   ├── config/              # Configuration
│   ├── db/                  # Database initialization
│   ├── handlers/            # HTTP handlers
│   ├── models/              # Data models
│   ├── services/            # Business logic (AI, embeddings, PDF)
│   └── prompts/             # System prompts for AI
├── web/
│   ├── src/
│   │   ├── components/      # React components
│   │   ├── lib/             # Utilities
│   │   └── App.tsx          # Main app
│   └── package.json
├── storage/                 # SQLite DB and uploads
├── .env                     # Environment variables (create from .env.example)
├── Makefile                 # Build and run commands
└── README.md
```

## Development

### Available Make Commands

```bash
make dev          # Run both API and web servers
make run-api      # Run API server only
make dev-web      # Run web dev server only
make build        # Build everything
make test         # Run tests
make clean        # Clean build artifacts
make help         # Show all commands
```

### Building for Production

```bash
make build
```

Binaries will be created in:
- `api/elearn` (backend)
- `web/dist/` (frontend)

## Security & Privacy

- Never log or store full document text
- All API keys in `.env`, never hardcoded
- File upload limited to 50MB
- PDF-only uploads enforced
- CORS configured for local development

## Troubleshooting

### "Failed to connect to database"
Ensure the `storage/` directory exists and is writable.

### "API error (401)"
Check that your API keys in `.env` are correct and valid.

### Ollama connection error
Ensure Ollama is running: `ollama serve`

### Frontend can't reach API
Verify the API is running on port 8080 and CORS is configured.

## Future Enhancements

- Quiz/testing system
- Voice-over and narration
- Image generation for slides
- Course export (PDF/PowerPoint)
- Multi-language support

## License

MIT

## Contributing

Contributions welcome! Please open an issue or PR.
