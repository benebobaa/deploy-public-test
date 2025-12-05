# Build stage
FROM golang:1.24-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go mod file and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations for smaller size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o main .

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy templates
COPY --from=builder /app/templates ./templates

# Change ownership to non-root user
RUN chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Set default environment variables
ENV APP_TITLE="Welcome to My Landing Page"
ENV APP_DESCRIPTION="A simple, configurable landing page built with Go"
ENV APP_VERSION="1.0.0"
ENV APP_ENV="production"
ENV APP_CONTACT="contact@example.com"
ENV PORT="8080"

# Run the binary
CMD ["./main"]
