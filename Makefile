.PHONY: run build clean tidy dev stt-model stt-up

# Run the application
run:
	go run cmd/web/main.go

# Build the application
build:
	go build -o bin/byeboros-backend cmd/web/main.go

# Clean build artifacts
clean:
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air -c .air.toml || go run cmd/web/main.go

# Download the Whisper model used by the whisper-stt service (~142MB)
stt-model:
	mkdir -p models
	curl -fL -o models/ggml-base.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin

# Start only the self-hosted speech-to-text service
stt-up:
	docker compose up -d whisper-stt
