.PHONY: run build clean tidy

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
