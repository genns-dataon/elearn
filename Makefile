.PHONY: run-api dev-web dev install-deps

# Run API server
run-api:
	go run ./api/main.go

# Run frontend dev server
dev-web:
	cd web && npm run dev

# Run both API and web in parallel
dev:
	@echo "Starting eLearning App..."
	@$(MAKE) -j2 run-api dev-web

# Install all dependencies
install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing npm dependencies..."
	cd web && npm install

# Build API
build-api:
	@echo "Building API..."
	go build -o api/elearn ./api

# Build web
build-web:
	@echo "Building web..."
	cd web && npm run build

# Build everything
build: build-api build-web

# Clean build artifacts
clean:
	rm -f api/elearn
	rm -rf web/dist
	rm -f storage/*.db storage/*.db-shm storage/*.db-wal

# Run tests
test:
	go test ./...

help:
	@echo "Available commands:"
	@echo "  make dev          - Run both API and web servers"
	@echo "  make run-api      - Run API server only"
	@echo "  make dev-web      - Run web dev server only"
	@echo "  make install-deps - Install all dependencies"
	@echo "  make build        - Build API and web"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
