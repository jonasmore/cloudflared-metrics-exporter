.PHONY: build clean install test run help

# Build variables
BINARY_NAME=cloudflared-metrics-exporter
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d-%H:%M UTC')
LDFLAGS=-ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

build-all: ## Build for all platforms
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe

clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

install: build ## Install the binary to /usr/local/bin
	sudo cp $(BINARY_NAME) /usr/local/bin/

test: ## Run tests
	$(GOTEST) -v ./...

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

run: build ## Build and run with example configuration
	./$(BINARY_NAME) \
		--metrics localhost:2000 \
		--metricsfile /tmp/metrics.jsonl \
		--metricsinterval 10s \
		--log-level info

run-compressed: build ## Build and run with compression enabled
	./$(BINARY_NAME) \
		--metrics localhost:2000 \
		--metricsfile /tmp/metrics-compressed.jsonl \
		--metricsinterval 10s \
		--metricscompress \
		--log-level info

run-filtered: build ## Build and run with filtering
	./$(BINARY_NAME) \
		--metrics localhost:2000 \
		--metricsfile /tmp/metrics-filtered.jsonl \
		--metricsinterval 10s \
		--metricsfilter "cloudflared_tunnel_*,quic_client_*" \
		--metricscompress \
		--log-level info

docker-build: ## Build Docker image
	docker build -t cloudflared-metrics-exporter:$(VERSION) .

docker-run: ## Run in Docker
	docker run --rm \
		-v /tmp:/tmp \
		cloudflared-metrics-exporter:$(VERSION) \
		--metrics host.docker.internal:2000 \
		--metricsfile /tmp/metrics.jsonl \
		--metricsinterval 10s

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...
