.PHONY: build-image build-binary build test fmt lint clean local-run

BINARY_NAME=azure-carbon-exporter
IMAGE_NAME=azure-carbon-exporter
VERSION?=latest

build-image:
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(VERSION) .

build-binary:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

build: build-image

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
	docker rmi $(IMAGE_NAME):$(VERSION) 2>/dev/null || true

local-run:
	go run . \
		--azure.tenant-id=<tenant-id> \
		--azure.client-id=<client-id> \
		--azure.client-secret=<client-secret> \
		--subscriptions="<subscription-id>" \
		--log-level=debug
