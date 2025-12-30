
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

BINARY := netdebug

IMAGE_NAME ?= ghcr.io/ryanelliottsmith/network-debugger
IMAGE_TAG ?= $(VERSION)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/netdebug

.PHONY: install
install:
	go install $(LDFLAGS) ./cmd/netdebug
	
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...
	gofmt -s -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean: 
	rm -rf bin/
	rm -f coverage.out

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.

.PHONY: docker-build-multiarch
docker-build-multiarch: docker-buildx-setup
	docker buildx build \
		--builder multiarch \
		--platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.

.PHONY: docker-push
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker-push-multiarch
docker-push-multiarch: docker-buildx-setup
	docker buildx build \
		--builder multiarch \
		--platform linux/amd64,linux/arm64 \
		--push \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.


.PHONY: podman-build-push
podman-build-push:
	podman build \
	--platform linux/amd64,linux/arm64 \
	--manifest=$(IMAGE_NAME):$(IMAGE_TAG) \
	. && \
	podman manifest push $(IMAGE_NAME):$(IMAGE_TAG) 

.PHONY: all
all: fmt vet lint test build

.DEFAULT_GOAL := help
