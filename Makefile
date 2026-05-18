.PHONY: build test run clean docker docker-run migrate lint fmt tidy help

APP_NAME := gateway
BUILD_DIR := ./bin
DOCKER_IMAGE := aigateway

build:
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/gateway

test:
	go test -v -race -coverprofile=coverage.out ./...

run: build
	$(BUILD_DIR)/$(APP_NAME) --config config/config.yaml

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

docker:
	docker build -t $(DOCKER_IMAGE) -f docker/Dockerfile .

docker-run:
	docker-compose -f docker/docker-compose.yaml up -d

migrate:
	go run ./cmd/migrate up

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  test        - Run tests"
	@echo "  run         - Build and run the application"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker      - Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  migrate     - Run database migrations"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  tidy        - Tidy dependencies"
