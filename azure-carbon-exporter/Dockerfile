FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary with executable permissions
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-carbon-exporter . && chmod +x azure-carbon-exporter

# Final stage
FROM alpine:latest@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

# Install only essential packages in a single layer
RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Use non-root user
USER appuser
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/azure-carbon-exporter ./azure-carbon-exporter

# Default command
ENTRYPOINT ["./azure-carbon-exporter"]
