# =============================================================================
# Docker Build Rules
# Included via Makefile: include docker/docker.mk
# =============================================================================

HUB ?= ghcr.io/kirklin
IMG ?= boot-backend-go-clean
TAG ?= latest

# Full Docker Image Name
DOCKER_IMAGE := $(HUB)/$(IMG)



## Build production image
docker-build:
	docker build -f docker/Dockerfile \
		--build-arg GOPROXY=$(GOPROXY) \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(TAG) .

## Build development image (base)
docker-build-dev:
	docker build -f docker/Dockerfile.dev \
		-t $(DOCKER_IMAGE):dev .

## Push image to Registry
docker-push:
	docker push $(DOCKER_IMAGE):$(TAG)

## Build and push (used in CI)
docker-build-push: docker-build docker-push

## Clean local Docker images
docker-clean:
	docker rmi $(DOCKER_IMAGE):$(TAG) 2>/dev/null || true
	docker image prune -f
