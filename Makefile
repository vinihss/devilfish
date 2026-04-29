# Makefile for DevilFish

.PHONY: build run test test-coverage docker-build docker-run docker-dev clean lint fmt tidy install-dev help

# Build binary
build:
	go build -o bin/devilfishd ./cmd/devilfishd

# Build messaging runtime binary
build-messaging:
	go build -o bin/messagingd ./cmd/messagingd

# Build all binaries
build-all: build build-messaging

# Run locally (requires config.yaml or environment variables)
run:
	go run ./cmd/devilfishd

# Run messaging runtime locally
run-messaging:
	go run ./cmd/messagingd

# Test
test:
	go test -v ./...

# Test with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker build
docker-build:
	docker build -t devilfishd:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 -p 8081:8081 \
		-e OPENAI_API_KEY=$$OPENAI_API_KEY \
		-e TELEGRAM_BOT_TOKEN=$$TELEGRAM_BOT_TOKEN \
		-e DISCORD_BOT_TOKEN=$$DISCORD_BOT_TOKEN \
		-e SLACK_BOT_TOKEN=$$SLACK_BOT_TOKEN \
		devilfishd:latest

# Docker compose dev (hot-reload)
docker-dev:
	docker-compose -f configs/docker-compose.yaml up --build

# Clean build artifacts
clean:
	rm -rf bin/ tmp/ coverage.out coverage.html

# Lint
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# tidy go.mod
tidy:
	go mod tidy

# Install dev tools
install-dev:
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "DevilFish Makefile Commands:"
	@echo "  make build           - Build core runtime binary (devilfishd)"
	@echo "  make build-messaging - Build messaging runtime binary (messagingd)"
	@echo "  make build-all       - Build both binaries"
	@echo "  make run             - Run core runtime locally"
	@echo "  make run-messaging   - Run messaging runtime locally"
	@echo "  make test            - Run tests"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-run      - Run Docker container"
	@echo "  make docker-dev      - Run with hot-reload (Air)"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make lint            - Run linter"
	@echo "  make fmt             - Format code"
	@echo "  make tidy            - Tidy go modules"
	@echo "  make install-dev     - Install development tools"