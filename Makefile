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
DIR_ROOT = $(realpath $(CURDIR))

KUBERNETES_VERSION = 1.34.5
IMAGE_REPOSITORY = ghcr.io/cloudboss/keights
IMAGE_TAG =

CTR_IMAGE_GO = golang:1.26.1-alpine3.23
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
	$(DIR_OUT)/instance-attr/instance-attr-$(VERSION).zip \
	$(DIR_OUT)/kube-ca/kube-ca-$(VERSION).zip

$(DIR_OUT)/keights-$(OS)-$(ARCH): $(KEIGHTS_GO_DEPS) | $(DIR_OUT) $(HAS_IMAGE_LOCAL)
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
			-o /code/$(DIR_OUT)/keights-$(OS)-$(ARCH) \
			./cmd/keights

keights: $(DIR_OUT)/keights-$(OS)-$(ARCH)

keights-linux-%:
	@$(MAKE) keights OS=linux ARCH=$*

keights-darwin-%:
	@$(MAKE) keights OS=darwin ARCH=$*

keights-release: keights-linux-amd64 keights-darwin-amd64 keights-darwin-arm64

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

image: $(HAS_COMMAND_DOCKER)
	@[ -n "$(IMAGE_TAG)" ] || (echo "IMAGE_TAG is required"; exit 1)
	@[ $$(echo $(IMAGE_TAG) | cut -c 1) = v ] || (echo "IMAGE_TAG must begin with a 'v'"; exit 1)
	@docker build \
		--build-arg KUBERNETES_VERSION=$(KUBERNETES_VERSION) \
		-t $(IMAGE_REPOSITORY):$(IMAGE_TAG) \
		-f image/Containerfile \
		.

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

$(DIR_OUT)/instance-attr/bootstrap: \
		$(STACKBOT_GO_DEPS) \
		$(shell find stackbot/asgevent -type f -path '*.go' ! -path '*_test.go') \
		$(shell find stackbot/instance-attr -type f -path '*.go' ! -path '*_test.go') \
		| $(DIR_OUT)/instance-attr/ $(HAS_IMAGE_LOCAL)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code:z \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-w /code \
		$(CTR_IMAGE_LOCAL) \
		go build -o /code/$(DIR_OUT)/instance-attr/bootstrap ./stackbot/instance-attr/...

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

$(DIR_OUT)/instance-attr/instance-attr-$(VERSION).zip: $(DIR_OUT)/instance-attr/bootstrap \
		| $(HAS_COMMAND_FAKEROOT) $(HAS_COMMAND_ZIP)
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@cd $(DIR_OUT)/instance-attr && fakeroot zip instance-attr-$(VERSION).zip bootstrap

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

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)

.PHONY: keights keights-release stackbot terraform-release test clean image
