FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS builder

# Distrust the compromised CA before downloading build dependencies. The tag
# also sets the CI Go version: .github/actions/setup-goversion parses it out of
# this FROM line, so it has to keep the `golang:<version>-alpine ... AS builder`
# shape.
RUN apk --no-cache add 'ca-certificates>=20261020-r0' && \
    update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary with executable permissions
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o aws-water-exporter . && chmod +x aws-water-exporter

# Final stage
FROM alpine:latest@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

# Install only essential packages in a single layer
RUN apk --no-cache add 'ca-certificates>=20261020-r0' && \
    update-ca-certificates && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Use non-root user
USER appuser
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/aws-water-exporter ./aws-water-exporter

# Default command
ENTRYPOINT ["./aws-water-exporter"]
