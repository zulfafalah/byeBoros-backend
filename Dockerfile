# Build stage
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Install required packages for build
RUN apk add --no-cache make git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN make build

# Final stage
# =============================
FROM alpine:latest

# Install CA certificates and timezone data (required for making HTTPS requests and handling time correctly)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/bin/byeboros-backend .

EXPOSE 8080
CMD ["./byeboros-backend"]
