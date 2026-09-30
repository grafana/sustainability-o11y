FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

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
