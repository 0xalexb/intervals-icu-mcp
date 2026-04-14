IMAGE ?= ghcr.io/0xalexb/intervals-icu-mcp
VERSION ?= dev
DI_VERSION ?= 0.5.0
COMPILED_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

PLATFORMS ?= linux/amd64,linux/arm64
BUILDER_NAME ?= intervals-icu-mcp-builder
NATIVE_PLATFORM := linux/$(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

.PHONY: build push test lint tidy setup-buildx

setup-buildx:
	@docker buildx inspect $(BUILDER_NAME) >/dev/null 2>&1 || \
		docker buildx create --name $(BUILDER_NAME) --driver docker-container --use
	@docker buildx use $(BUILDER_NAME)

build: setup-buildx
	docker buildx build \
		--platform $(NATIVE_PLATFORM) \
		--load \
		--build-arg VERSION=$(VERSION) \
		--build-arg DI_VERSION=$(DI_VERSION) \
		--build-arg COMPILED_AT=$(COMPILED_AT) \
		-t $(IMAGE):$(VERSION) \
		.

push: setup-buildx
	docker buildx build \
		--platform $(PLATFORMS) \
		--push \
		--build-arg VERSION=$(VERSION) \
		--build-arg DI_VERSION=$(DI_VERSION) \
		--build-arg COMPILED_AT=$(COMPILED_AT) \
		-t $(IMAGE):$(VERSION) \
		.

test:
	go test -race -count=1 ./src/...

lint:
	golangci-lint run ./src/...

tidy:
	go mod tidy
