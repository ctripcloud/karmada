GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= '$(shell hack/version.sh)'

# Images management
REGISTRY?="docker.io/karmada"
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""
REGISTRY_SERVER_ADDRESS?=""

TARGETS := karmada-aggregated-apiserver \
			karmada-controller-manager \
			karmada-scheduler \
			karmada-descheduler \
			karmada-webhook \
			karmada-agent \
			karmada-scheduler-estimator \
			karmada-interpreter-webhook-example \
			karmada-search \
			karmada-operator

CTL_TARGETS := karmadactl kubectl-karmada

# Build code.
#
# Args:
#   GOOS:   OS to build.
#   GOARCH: Arch to build.
#
# Example:
#   make
#   make all
#   make karmada-aggregated-apiserver
#   make karmada-aggregated-apiserver GOOS=linux
CMD_TARGET=$(TARGETS) $(CTL_TARGETS)

.PHONY: all
all: $(CMD_TARGET)

.PHONY: $(CMD_TARGET)
$(CMD_TARGET):
	BUILD_PLATFORMS=$(GOOS)/$(GOARCH) hack/build.sh $@

# Build image.
#
# Args:
#   GOARCH:      Arch to build.
#   OUTPUT_TYPE: Destination to save image(docker/registry).
#
# Example:
#   make images
#   make image-karmada-aggregated-apiserver
#   make image-karmada-aggregated-apiserver GOARCH=arm64
IMAGE_TARGET=$(addprefix image-, $(TARGETS))
.PHONY: $(IMAGE_TARGET)
$(IMAGE_TARGET):
	set -e;\
	target=$$(echo $(subst image-,,$@));\
	make $$target GOOS=linux;\
	VERSION=$(VERSION) REGISTRY=$(REGISTRY) BUILD_PLATFORMS=linux/$(GOARCH) hack/docker.sh $$target

images: $(IMAGE_TARGET)

# Build and push multi-platform image to DockerHub
#
# Example
#   make multi-platform-images
#   make mp-image-karmada-aggregated-apiserver
MP_TARGET=$(addprefix mp-image-, $(TARGETS))
.PHONY: $(MP_TARGET)
$(MP_TARGET):
	set -e;\
	target=$$(echo $(subst mp-image-,,$@));\
	make $$target GOOS=linux GOARCH=amd64;\
	make $$target GOOS=linux GOARCH=arm64;\
	VERSION=$(VERSION) REGISTRY=$(REGISTRY) \
		OUTPUT_TYPE=registry \
		BUILD_PLATFORMS=linux/amd64,linux/arm64 \
		hack/docker.sh $$target

multi-platform-images: $(MP_TARGET)

.PHONY: clean
clean:
	rm -rf _tmp _output

.PHONY: update
update:
	hack/update-all.sh

.PHONY: verify
verify:
	hack/verify-all.sh

.PHONY: release-chart
release-chart:
	hack/release-helm-chart.sh $(VERSION)

.PHONY: test
test:
	mkdir -p ./_output/coverage/
	go test --race --v ./pkg/... -coverprofile=./_output/coverage/coverage_pkg.txt -covermode=atomic
	go test --race --v ./cmd/... -coverprofile=./_output/coverage/coverage_cmd.txt -covermode=atomic
	go test --race --v ./examples/... -coverprofile=./_output/coverage/coverage_examples.txt -covermode=atomic

upload-images: images
	@echo "push images to $(REGISTRY)"
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u ${REGISTRY_USER_NAME} -p ${REGISTRY_PASSWORD} ${REGISTRY_SERVER_ADDRESS}
endif
	docker push ${REGISTRY}/karmada-controller-manager:${VERSION}
	docker push ${REGISTRY}/karmada-scheduler:${VERSION}
	docker push ${REGISTRY}/karmada-descheduler:${VERSION}
	docker push ${REGISTRY}/karmada-webhook:${VERSION}
	docker push ${REGISTRY}/karmada-agent:${VERSION}
	docker push ${REGISTRY}/karmada-scheduler-estimator:${VERSION}
	docker push ${REGISTRY}/karmada-interpreter-webhook-example:${VERSION}
	docker push ${REGISTRY}/karmada-aggregated-apiserver:${VERSION}
	docker push ${REGISTRY}/karmada-search:${VERSION}

# Build and package binary
#
# Example
#   make release-karmadactl
#   make release-kubectl-karmada
#   make release-kubectl-karmada GOOS=darwin GOARCH=amd64
RELEASE_TARGET=$(addprefix release-, $(CTL_TARGETS))
.PHONY: $(RELEASE_TARGET)
$(RELEASE_TARGET):
	@set -e;\
	target=$$(echo $(subst release-,,$@));\
	make $$target;\
	hack/release.sh $$target $(GOOS) $(GOARCH)

# Build and package binary for all platforms
#
# Example
#   make release
release:
	@make release-karmadactl GOOS=linux GOARCH=amd64
	@make release-karmadactl GOOS=linux GOARCH=arm64
	@make release-karmadactl GOOS=darwin GOARCH=amd64
	@make release-karmadactl GOOS=darwin GOARCH=arm64
	@make release-kubectl-karmada GOOS=linux GOARCH=amd64
	@make release-kubectl-karmada GOOS=linux GOARCH=arm64
	@make release-kubectl-karmada GOOS=darwin GOARCH=amd64
	@make release-kubectl-karmada GOOS=darwin GOARCH=arm64


# For Ctrip using
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_prj := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))
GOPROXY = http://goproxy.release.ctripcorp.com,direct
GOREPO ?= hub.cloud.ctripcorp.com/golang/golang
GOVERSION ?= 1.19.5
GODOCKER ?= docker run --rm \
		-e GOPROXY=${GOPROXY} \
		-e GO111MODULE=on \
		--network=host \
		-v /etc/resolv.conf:/etc/resolv.conf:ro \
		-v ${CURDIR}:/tmp/${current_prj}:rw \
		-w /tmp/${current_prj} \
		${GOREPO}:${GOVERSION} 

CTRIP_TARGETS := karmada-aggregated-apiserver \
					karmada-controller-manager \
					karmada-scheduler \
					karmada-webhook \
					karmada-search 

CTRIP_KARMADA_REGISTRY ?= "hub.cloud.ctripcorp.com/karmada"

# Build binaries for Ctrip
build_ctrip_cross_all: 
	@for target in ${CTRIP_TARGETS}; do \
		BUILD_PLATFORMS="linux/amd64,linux/arm64" hack/build.sh $$target; \
	done

build_ctrip_in_docker:
	$(GODOCKER) sh -c "git config --global --add safe.directory /tmp/karmada; \
	make build_ctrip_cross_all"

# Make images for Ctrip
image_ctrip_cross_all: 
	@docker buildx use docker-container || \
		docker buildx create \
			--name docker-container \
			--driver docker-container \
			--driver-opt image=hub.cloud.ctripcorp.com/moby/buildkit:buildx-stable-1-ctrip \
			--use
	@for target in ${CTRIP_TARGETS}; do \
		BUILD_PLATFORMS=linux/amd64,linux/arm64 \
		VERSION=${VERSION} \
		REGISTRY=${CTRIP_KARMADA_REGISTRY} \
		OUTPUT_TYPE=registry \
		hack/docker.sh $$target; \
	done
	@docker buildx prune -f || echo "no buildx cache needs cleaning"

.PHONY: info
info:
	@echo "targets: [${CTRIP_TARGETS}]"
	@echo "registry: ${CTRIP_KARMADA_REGISTRY}"
	@echo "version: ${VERSION}"

# Test and verify for Ctrip
test_ctrip_in_docker:
	@$(GODOCKER) sh -c "make test"

verify_ctrip_in_docker:
	@$(GODOCKER) sh -c "make verify"