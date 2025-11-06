# Makefile for GCP Validation Demo

# Configuration
IMAGE_REGISTRY ?= quay.io/rh-ee-dawang
IMAGE_NAME ?= gcp-api-validator
IMAGE_TAG ?= latest
FULL_IMAGE = $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
PLATFORM ?= linux/amd64

.PHONY: help
help: ## Display this help
	@echo "GCP Validation Demo - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: build-image
build-image: ## Build the API validator container image
	@echo "🔨 Building API validator image..."
	podman build --platform $(PLATFORM) -f docker/Dockerfile.api-validator -t $(FULL_IMAGE) ..
	@echo "✅ Image built: $(FULL_IMAGE)"

.PHONY: build-image-local
build-image-local: ## Build the API validator image with local tag
	@echo "🔨 Building API validator image (local)..."
	podman build --platform $(PLATFORM) -f docker/Dockerfile.api-validator -t $(IMAGE_NAME):$(IMAGE_TAG) ..
	@echo "✅ Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: push-image
push-image: build-image ## Build and push the image to registry
	@echo "📤 Pushing image to registry..."
	podman push $(FULL_IMAGE)
	@echo "✅ Image pushed: $(FULL_IMAGE)"

.PHONY: test-validator
test-validator: ## Test the API validator locally
	@echo "🧪 Testing API validator..."
	@mkdir -p /tmp/tekton-results
	podman run --rm \
		-e GCP_PROJECT=demo-project \
		-e REQUIRED_APIS=compute.googleapis.com,container.googleapis.com,iam.googleapis.com \
		-v /tmp/tekton-results:/tekton/results \
		$(IMAGE_NAME):$(IMAGE_TAG) || true
	@echo ""
	@echo "📊 Results:"
	@echo "Status: $$(cat /tmp/tekton-results/status 2>/dev/null || echo 'N/A')"
	@echo "Message: $$(cat /tmp/tekton-results/message 2>/dev/null || echo 'N/A')"
	@echo "Details: $$(cat /tmp/tekton-results/details 2>/dev/null || echo 'N/A')"

.PHONY: test-validators-local
test-validators-local: ## Test all validators locally (bash scripts)
	@echo "🧪 Testing Quota Validator..."
	@mkdir -p /tmp/tekton-results
	GCP_PROJECT=demo-project GCP_REGION=us-central1 \
		bash validators/quota-validator.sh || true
	@echo ""
	@echo "🧪 Testing Network Validator..."
	GCP_PROJECT=demo-project GCP_REGION=us-central1 \
		NETWORK_NAME=default SUBNET_NAME=default-us-central1 \
		bash validators/network-validator.sh || true

.PHONY: apply-tasks
apply-tasks: ## Apply Tekton task definitions to cluster
	@echo "📋 Applying Tekton tasks..."
	kubectl apply -f tekton/tasks/
	@echo "✅ Tasks applied"

.PHONY: apply-pipeline
apply-pipeline: ## Apply Tekton pipeline definition to cluster
	@echo "📋 Applying Tekton pipeline..."
	kubectl apply -f tekton/pipelines/
	@echo "✅ Pipeline applied"

.PHONY: run-pipeline
run-pipeline: ## Create a PipelineRun to execute the validation
	@echo "🚀 Creating PipelineRun..."
	kubectl create -f tekton/pipelines/validation-pipelinerun.yaml
	@echo "✅ PipelineRun created"
	@echo ""
	@echo "📊 To watch the logs, run:"
	@echo "   tkn pipelinerun logs -f -L"

.PHONY: clean
clean: ## Clean up test results
	@echo "🧹 Cleaning up..."
	rm -rf /tmp/tekton-results
	@echo "✅ Cleaned"

.PHONY: all
all: build-image-local test-validator ## Build and test the API validator
