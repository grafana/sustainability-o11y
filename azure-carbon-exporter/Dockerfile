FROM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-carbon-exporter . && chmod +x azure-carbon-exporter

# Final stage
FROM alpine:latest@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

# Install only essential packages in a single layer
RUN apk --no-cache add 'ca-certificates>=20261020-r0' && \
    update-ca-certificates && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Use non-root user
USER appuser
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/azure-carbon-exporter ./azure-carbon-exporter

# Default command
ENTRYPOINT ["./azure-carbon-exporter"]
