# Stage 1: Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install ca-certificates and tzdata for timezone support
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with CGO disabled (using pure-Go glebarez/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Stage 2: Final runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create application directories
RUN mkdir -p /app/media /app/.thumbnails /app/data

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Set environment defaults
ENV PORT=8080 \
    MEDIA_DIR=/app/media \
    THUMBNAIL_DIR=/app/.thumbnails \
    DB_PATH=/app/data/gallery.db \
    THUMB_WIDTH=400 \
    THUMB_HEIGHT=400 \
    MAX_SCAN_WORKERS=4 \
    IMAGE_SCAN_WORKERS=2 \
    VIDEO_SCAN_WORKERS=1 \
    FFMPEG_THREADS=1

EXPOSE 8080

# Volumes for persistent data
VOLUME ["/app/media", "/app/.thumbnails", "/app/data"]

ENTRYPOINT ["/app/server"]
