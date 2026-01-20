# ==============================================================================
# Variables
# ==============================================================================
PROJECT_NAME := game-server

# Docker Image Names
IMAGE_PREFIX := joe-shih/go-k8s
CENTRAL_IMAGE := $(IMAGE_PREFIX)-central
CONNECTOR_IMAGE := $(IMAGE_PREFIX)-connector
STATELESS_IMAGE := $(IMAGE_PREFIX)-stateless
STATEFUL_IMAGE := $(IMAGE_PREFIX)-stateful
TAG ?= latest

# Directories
DEPLOY_K8S_INFRA := deploy/k8s/local-infra
DEPLOY_K8S_APPS := deploy/k8s/apps/local
DOCKERFILE_K8S := build/package/Dockerfile.localk8s

# Shell
SHELL := /bin/bash

# ==============================================================================
# Help
# ==============================================================================
.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# ==============================================================================
# Development
# ==============================================================================
##@ Development

.PHONY: deps
deps: ## Download dependencies
	go mod download

.PHONY: tidy
tidy: ## Tidy up go.mod
	go mod tidy

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: test
test: ## Run unit tests with race detector and coverage (internal only)
	@go test -v -race -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out
	@rm coverage.out

.PHONY: ci
ci: lint test ## Run all CI steps (lint + test)

# ==============================================================================
# Code Generation
# ==============================================================================
##@ Code Generation

.PHONY: gen-proto
gen-proto: ## Generate Protobuf & gRPC code
	@echo "Generating Protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
	        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	        api/proto/*.proto api/proto/centralRPC/*.proto api/proto/gameRPC/*.proto api/proto/connectorRPC/*.proto
	@echo "Done!"

.PHONY: gen-mock
gen-mock: ## Generate Mocks using go:generate
	@echo "Generating Mocks..."
	@go generate ./...
	@echo "Done!"

# ==============================================================================
# Docker Compose
# ==============================================================================
##@ Docker Compose

.PHONY: docker-up
docker-up: ## Start local dev environment (Air hot-reload)
	docker-compose up -d --build

.PHONY: docker-down
docker-down: ## Stop local dev environment
	docker-compose down

.PHONY: docker-logs
docker-logs: ## Tail docker-compose logs
	docker-compose logs -f

# ==============================================================================
# Kubernetes
# ==============================================================================
##@ Kubernetes

# Safety: Allowed contexts for k8s-* commands
SAFE_CONTEXTS := docker-desktop minikube orbstack kind-kind rancher-desktop

.PHONY: check-context
check-context:
	@ctx=$$(kubectl config current-context); \
	found=0; \
	for safe in $(SAFE_CONTEXTS); do \
		if [ "$$ctx" = "$$safe" ]; then found=1; break; fi; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "\033[0;31m[ERROR] Current context '$$ctx' is NOT in the safe list: [$(SAFE_CONTEXTS)]\033[0m"; \
		echo "To prevent accidental deployment to production, please switch to a local cluster."; \
		exit 1; \
	else \
		echo "\033[0;32m[INFO] Context '$$ctx' verified as safe.\033[0m"; \
	fi

.PHONY: k8s-build
k8s-build: ## Build all Docker images for K8s (Locally)
	@echo "Building Central..."
	docker build --build-arg SERVICE_PATH=cmd/central -t $(CENTRAL_IMAGE):$(TAG) -f $(DOCKERFILE_K8S) .
	@echo "Building Connector..."
	docker build --build-arg SERVICE_PATH=cmd/connector -t $(CONNECTOR_IMAGE):$(TAG) -f $(DOCKERFILE_K8S) .
	@echo "Building Stateless Demo..."
	docker build --build-arg SERVICE_PATH=cmd/stateless/demo -t $(STATELESS_IMAGE):$(TAG) -f $(DOCKERFILE_K8S) .
	@echo "Building Stateful Demo..."
	docker build --build-arg SERVICE_PATH=cmd/stateful/demo -t $(STATEFUL_IMAGE):$(TAG) -f $(DOCKERFILE_K8S) .
	@echo "All images built successfully!"

.PHONY: k8s-apply
k8s-apply: check-context ## Apply K8s manifests (Infra first, then Apps)
	@echo "Deploying Infrastructure..."
	kubectl apply -f $(DEPLOY_K8S_INFRA)
	@echo "Waiting for Infra (optional pause)..."
	@sleep 2
	@echo "Deploying Apps..."
	kubectl apply -f $(DEPLOY_K8S_APPS)
	@echo "Done!"

.PHONY: k8s-delete
k8s-delete: check-context ## Delete K8s resources
	@echo "Deleting Apps..."
	-kubectl delete -f $(DEPLOY_K8S_APPS)
	@echo "Deleting Infrastructure..."
	-kubectl delete -f $(DEPLOY_K8S_INFRA)
	@echo "Done!"

# ==============================================================================
# Tools
# ==============================================================================
##@ Tools

.PHONY: install-tools
install-tools: ## Install required tools (Protoc plugins, Mockgen, Linter)
	@echo "Installing tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed!"