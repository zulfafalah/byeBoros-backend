.PHONY: run build clean tidy dev stt-model stt-model-small stt-model-base stt-up

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

# Whisper model used by the whisper-stt service (override STT_MODEL_FILE).
# base < small < medium < large-v3-turbo-q5_0 (bigger = more accurate, slower).
STT_MODEL_FILE ?= ggml-large-v3-turbo-q5_0.bin

# Download the Whisper model (large-v3-turbo q5 ~547MB by default)
stt-model:
	mkdir -p models
	curl -fL -o models/$(STT_MODEL_FILE) https://huggingface.co/ggerganov/whisper.cpp/resolve/main/$(STT_MODEL_FILE)

# Download lighter, faster models instead
stt-model-small:
	$(MAKE) stt-model STT_MODEL_FILE=ggml-small.bin
stt-model-base:
	$(MAKE) stt-model STT_MODEL_FILE=ggml-base.bin

# Start only the self-hosted speech-to-text service
stt-up:
	docker compose up -d whisper-stt
