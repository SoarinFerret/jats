# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled and static linking
#RUN CGO_ENABLED=1 go build -ldflags '-s -w -extldflags "-static"' -o jatsd ./cmd/jatsd
RUN CGO_ENABLED=1 go build -o jatsd ./cmd/jatsd

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/jatsd .

# Copy frontend templates
COPY --from=builder /app/frontend ./frontend

# Create attachments directory and set ownership
RUN mkdir -p /app/attachments && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the binary
CMD ["./jatsd"]
