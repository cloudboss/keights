# Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

PROJECT = $(shell basename ${PWD})
VERSION =
DIR_OUT = _output
DIR_STG = $(DIR_OUT)/staging
DIR_RELEASE = $(DIR_OUT)/release
DIR_ROOT = $(realpath $(CURDIR))

IMAGE_REPOSITORY = ghcr.io/cloudboss/keights
EASYTO_VERSION = 0.10.0
SONOBUOY_VERSION = 0.57.3

# KUBERNETES_VERSION defaults to the patch from compat.json's default-test-version,
# used by the keights-tag sanity image build and by local dev.
# The AMI workflow passes KUBERNETES_VERSION explicitly per (keights, kubernetes) pair.
KUBERNETES_VERSION ?= $(shell $(CURDIR)/hack/compat-default-test-version 2>/dev/null)

CTR_IMAGE_GO = ghcr.io/cloudboss/docker.io/library/golang:1.26.1-alpine3.23
CTR_IMAGE_GOLANGCI = ghcr.io/cloudboss/golangci/golangci-lint:v2.11.4-alpine
CTR_IMAGE_TERRAFORM = ghcr.io/cloudboss/hashicorp/terraform:1.14.8

DIR_CACHE = $(DIR_OUT)/cache
DIR_AMI = $(DIR_OUT)/ami
AMI_ID_FILE = $(DIR_AMI)/ami-id
UID = $(shell id -u)
GID = $(shell id -g)
UID_SHA256 = $(shell echo -n $(UID) | sha256sum | awk '{print $$1}')
GID_SHA256 = $(shell echo -n $(GID) | sha256sum | awk '{print $$1}')
CONTAINERFILE_SHA256 = $(shell sha256sum Containerfile.build | awk '{print $$1}')
CTR_IMAGE_GO_SHA256 = $(shell echo -n $(CTR_IMAGE_GO) | sha256sum | awk '{print $$1}')
DOCKER_INPUTS_SHA256 = $(shell echo -n $(UID_SHA256)$(GID_SHA256)$(CONTAINERFILE_SHA256)$(CTR_IMAGE_GO_SHA256) | \
	sha256sum | awk '{print $$1}' | cut -c 1-40)
CTR_IMAGE_LOCAL = $(PROJECT):$(DOCKER_INPUTS_SHA256)
HAS_IMAGE_LOCAL = $(DIR_OUT)/.image-local-$(DOCKER_INPUTS_SHA256)

HAS_COMMAND_DOCKER = $(DIR_OUT)/.command-docker
HAS_COMMAND_FAKEROOT = $(DIR_OUT)/.command-fakeroot
HAS_COMMAND_ZIP = $(DIR_OUT)/.command-zip

TERRAFORM_TARBALL = $(DIR_OUT)/keights-terraform-$(VERSION).tar.gz

OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH = $(shell uname -m | sed 's/x86_64/amd64/')

DIR_STG_KEIGHTS = $(DIR_STG)/keights/$(OS)/$(ARCH)
KEIGHTS_TARBALL = $(DIR_RELEASE)/keights-$(VERSION)-$(OS)-$(ARCH).tar.gz

KEIGHTS_GO_DEPS = \
	go.mod \
	$(shell find cmd/keights -type f -path '*.go' ! -path '*_test.go') \
	$(shell find internal/deploy -type f -path '*.go' ! -path '*_test.go') \
	$(shell find internal/deps -type f -path '*.go' ! -path '*_test.go') \
	$(shell find internal/nlb -type f -path '*.go' ! -path '*_test.go') \
	$(shell find internal/quickstart -type f -path '*.go' ! -path '*_test.go') \
	$(shell find internal/whisperer -type f -path '*.go' ! -path '*_test.go')

KEIGHTS_LDFLAGS = \
	-X github.com/cloudboss/keights/cmd/keights/tree.Version=$(VERSION)

TERRAFORM_SOURCES = $(shell find terraform -type f -name '*.tf')

STACKBOT_ZIPS = \
	$(DIR_OUT)/auto-namer/auto-namer-$(VERSION).zip \
	$(DIR_OUT)/kube-ca/kube-ca-$(VERSION).zip

$(DIR_STG_KEIGHTS)/keights: $(KEIGHTS_GO_DEPS) | $(DIR_STG_KEIGHTS)/ $(HAS_IMAGE_LOCAL)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOOS=$(OS) \
		-e GOARCH=$(ARCH) \
		-w /code \
		$(CTR_IMAGE_LOCAL) \
		go build -ldflags "$(KEIGHTS_LDFLAGS)" \
			-o /code/$(DIR_STG_KEIGHTS)/keights \
			./cmd/keights

$(KEIGHTS_TARBALL): $(DIR_STG_KEIGHTS)/keights | $(DIR_RELEASE)/ $(HAS_COMMAND_FAKEROOT)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@cd $(DIR_STG_KEIGHTS) && \
		fakeroot tar -czf $(DIR_ROOT)/$(KEIGHTS_TARBALL) keights

keights: $(DIR_STG_KEIGHTS)/keights

release-one: $(KEIGHTS_TARBALL)

release-linux-%:
	@$(MAKE) OS=linux ARCH=$* VERSION=$(VERSION) release-one

release-darwin-%:
	@$(MAKE) OS=darwin ARCH=$* VERSION=$(VERSION) release-one

release: release-linux-amd64 release-darwin-amd64 release-darwin-arm64

.DEFAULT_GOAL = stackbot

$(DIR_OUT):
	@mkdir -p $(DIR_OUT)

$(DIR_OUT)/%/:
	@mkdir -p $(DIR_OUT)/$*

$(DIR_OUT)/.command-%:
	@[ -f $(DIR_OUT)/.command-$* ] || { \
		which $* >/dev/null 2>&1 && \
		mkdir -p $(DIR_OUT) && touch $(DIR_OUT)/.command-$* || \
		(echo "command $* is required"; exit 1); \
	}

$(HAS_IMAGE_LOCAL): $(HAS_COMMAND_DOCKER)
	@docker build \
		--build-arg FROM=$(CTR_IMAGE_GO) \
		--build-arg GID=$(GID) \
		--build-arg UID=$(UID) \
		-f $(DIR_ROOT)/Containerfile.build \
		-t $(CTR_IMAGE_LOCAL) \
		.
	@touch $(HAS_IMAGE_LOCAL)

check-version:
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)

# Generic guard for required variables. Any target can depend on
# `check-var-FOO` to assert FOO is non-empty before running.
check-var-%:
	@[ -n "$($*)" ] || (echo "$* is required"; exit 1)

# Image tag includes the keights tag and the kubernetes patch so each (keights, kubernetes)
# pair is a distinct image.
IMAGE_TAG = $(IMAGE_REPOSITORY):$(VERSION)-k8s-$(KUBERNETES_VERSION)

# Image build-args are derived from compat.json keyed by the kubernetes minor
# version, so each (keights tag, kubernetes patch) pair pins the values its
# minor version expects.
IMAGE_BUILD_ARGS = $(shell $(CURDIR)/hack/compat-build-args $(KUBERNETES_VERSION))

image: check-version check-var-KUBERNETES_VERSION $(HAS_COMMAND_DOCKER)
	@docker build \
		--build-arg KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		$(IMAGE_BUILD_ARGS) \
		-t $(IMAGE_TAG) \
		-f image/Containerfile \
		.

image-push: check-version check-var-KUBERNETES_VERSION $(HAS_COMMAND_DOCKER)
	@docker push $(IMAGE_TAG)

image-delete: check-version check-var-KUBERNETES_VERSION
	@VERSION=$(VERSION)-k8s-$(KUBERNETES_VERSION) IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		$(DIR_ROOT)/hack/image-delete

STACKBOT_GO_DEPS = \
	go.mod \
	$(shell find internal -type f -path '*.go' ! -path '*_test.go')

$(DIR_OUT)/auto-namer/bootstrap: \
		$(STACKBOT_GO_DEPS) \
		$(shell find stackbot/asgevent -type f -path '*.go' ! -path '*_test.go') \
		$(shell find stackbot/auto-namer -type f -path '*.go' ! -path '*_test.go') \
		| $(DIR_OUT)/auto-namer/ $(HAS_IMAGE_LOCAL)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-w /code \
		$(CTR_IMAGE_LOCAL) \
		go build -o /code/$(DIR_OUT)/auto-namer/bootstrap ./stackbot/auto-namer/...

$(DIR_OUT)/kube-ca/bootstrap: \
		$(STACKBOT_GO_DEPS) \
		$(shell find stackbot/kube-ca -type f -path '*.go' ! -path '*_test.go') \
		| $(DIR_OUT)/kube-ca/ $(HAS_IMAGE_LOCAL)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-w /code \
		$(CTR_IMAGE_LOCAL) \
		go build -o /code/$(DIR_OUT)/kube-ca/bootstrap ./stackbot/kube-ca/...

$(DIR_OUT)/auto-namer/auto-namer-$(VERSION).zip: $(DIR_OUT)/auto-namer/bootstrap \
		| $(HAS_COMMAND_FAKEROOT) $(HAS_COMMAND_ZIP)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@cd $(DIR_OUT)/auto-namer && fakeroot zip auto-namer-$(VERSION).zip bootstrap

$(DIR_OUT)/kube-ca/kube-ca-$(VERSION).zip: $(DIR_OUT)/kube-ca/bootstrap \
		| $(HAS_COMMAND_FAKEROOT) $(HAS_COMMAND_ZIP)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@cd $(DIR_OUT)/kube-ca && fakeroot zip kube-ca-$(VERSION).zip bootstrap

$(TERRAFORM_TARBALL): $(TERRAFORM_SOURCES) | $(DIR_OUT) $(HAS_COMMAND_FAKEROOT)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@fakeroot tar -czf $(TERRAFORM_TARBALL) \
		--exclude='.terraform' \
		--exclude='*.tfvars' \
		--exclude='*.tfstate' \
		--exclude='*.tfstate.backup' \
		--exclude='.terraform.lock.hcl' \
		-C terraform .

terraform-release: $(TERRAFORM_TARBALL)

stackbot: $(STACKBOT_ZIPS)

test: | $(HAS_IMAGE_LOCAL)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-w /code \
		$(CTR_IMAGE_LOCAL) \
		sh -c "go vet -v ./... && go test -v ./..."

lint: $(HAS_COMMAND_DOCKER)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-w /code \
		$(CTR_IMAGE_GOLANGCI) \
		golangci-lint run --timeout 5m ./...

terraform-validate: $(HAS_COMMAND_DOCKER)
	@docker run --rm -t \
		-u $(UID):$(GID) \
		-v $(DIR_ROOT):/code:z \
		-w /code \
		-e HOME=/tmp \
		--entrypoint /bin/sh \
		$(CTR_IMAGE_TERRAFORM) \
		-c "terraform fmt -check -recursive terraform && \
			cd terraform && terraform init -backend=false && terraform validate"

# Extract MAJOR.MINOR from a vX.Y.Z keights tag for the keights-version-minor
# AMI tag, e.g., v2.0.5 -> v2.0.
VERSION_MINOR = $(shell echo $(VERSION) | awk -F. '{print $$1"."$$2}')

ami: check-version check-var-KUBERNETES_VERSION
	@VERSION=$(VERSION) \
		VERSION_MINOR=$(VERSION_MINOR) \
		KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		SUBNET_ID=$(SUBNET_ID) \
		IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		EASYTO_VERSION=$(EASYTO_VERSION) \
		DIR_CACHE=$(DIR_CACHE) \
		DIR_AMI=$(DIR_AMI) \
		$(DIR_ROOT)/hack/ami-build

ami-destroy:
	@AMI_ID_FILE=$(AMI_ID_FILE) \
		$(DIR_ROOT)/hack/ami-destroy

stackbot-upload: check-version stackbot
	@VERSION=$(VERSION) DIR_OUT=$(DIR_OUT) \
		BUCKET=$(BUCKET) PREFIX=$(PREFIX) \
		$(DIR_ROOT)/hack/stackbot-upload

stackbot-bucket-create:
	@AWS_REGION=$(AWS_REGION) BUCKET_PREFIX=$(BUCKET_PREFIX) \
		$(DIR_ROOT)/hack/stackbot-bucket-create

stackbot-bucket-delete:
	@BUCKET=$(BUCKET) \
		$(DIR_ROOT)/hack/stackbot-bucket-delete

stackbot-delete: check-version
	@VERSION=$(VERSION) BUCKET=$(BUCKET) PREFIX=$(PREFIX) \
		$(DIR_ROOT)/hack/stackbot-delete

release-delete:
	@RELEASE_TAG=$(RELEASE_TAG) \
		$(DIR_ROOT)/hack/release-delete

KEIGHTS_BIN = $(DIR_STG_KEIGHTS)/keights
KEIGHTS_PATH = $(DIR_ROOT)/$(shell dirname $(KEIGHTS_BIN))

cluster-provision: check-version check-var-KUBERNETES_VERSION check-var-AMI_NAME $(KEIGHTS_BIN)
	@SCENARIO=$(SCENARIO) \
		CLUSTER_NAME=$(CLUSTER_NAME) \
		AWS_REGION=$(AWS_REGION) \
		VPC_ID=$(VPC_ID) \
		SUBNET_ID_API=$(SUBNET_ID_PUBLIC) \
		SUBNET_IDS_PRIVATE=$(SUBNET_IDS_PRIVATE) \
		CLUSTER_DIR=$(CLUSTER_DIR) \
		AMI_NAME=$(AMI_NAME) \
		VERSION=$(VERSION) \
		KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		MODULE_SOURCE=$(MODULE_SOURCE) \
		STACKBOT_BUCKET=$(STACKBOT_BUCKET) \
		KMS_KEY_ID=$(KMS_KEY_ID) \
		STATE_BUCKET=$(STATE_BUCKET) \
		KEIGHTS=$(KEIGHTS_BIN) \
		$(DIR_ROOT)/hack/cluster-provision

terraform-state-delete:
	@STATE_BUCKET=$(STATE_BUCKET) CLUSTER_NAME=$(CLUSTER_NAME) \
		$(DIR_ROOT)/hack/terraform-state-delete

e2e-run:
	@PATH=$(KEIGHTS_PATH):$(PATH) DIR_CACHE=$(DIR_CACHE) SONOBUOY_VERSION=$(SONOBUOY_VERSION) \
		$(DIR_ROOT)/hack/e2e-run

cluster-wait: $(KEIGHTS_BIN)
	@PATH=$(KEIGHTS_PATH):$(PATH) CLUSTER_NAME=$(CLUSTER_NAME) KEIGHTS=$(KEIGHTS_BIN) \
		$(DIR_ROOT)/hack/cluster-wait

cluster-kubeconfig: $(KEIGHTS_BIN)
	@[ -n "$(CLUSTER_NAME)" ] || (echo "CLUSTER_NAME is required"; exit 1)
	@[ -n "$(OUTPUT)" ] || (echo "OUTPUT is required"; exit 1)
	@$(KEIGHTS_BIN) kubeconfig --cluster-name $(CLUSTER_NAME) -o $(OUTPUT)

cluster-destroy: $(KEIGHTS_BIN)
	@[ -n "$(CLUSTER_DIR)" ] || (echo "CLUSTER_DIR is required"; exit 1)
	@$(KEIGHTS_BIN) destroy $(CLUSTER_DIR) --auto-approve

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)

.PHONY: check-version check-var-% keights release-one release stackbot \
	terraform-release test lint terraform-validate ami ami-destroy \
	release-delete image image-push image-delete stackbot-upload \
	stackbot-delete stackbot-bucket-create stackbot-bucket-delete \
	cluster-provision terraform-state-delete cluster-wait \
	cluster-kubeconfig cluster-destroy e2e-run clean
