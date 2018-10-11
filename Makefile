GOARCH := amd64
GOOS := linux
REPO_SLUG := cloudboss/keights

export GOARCH GOOS GITHUB_TOKEN

setup:
	go get github.com/goreleaser/nfpm/cmd/nfpm

keights:
	mkdir -p _output/keights
	go build -o _output/keights/keights ./keights

keights-deb: setup keights
	mkdir -p _output/keights-deb
	sed "s|__VERSION__|$(VERSION)|g" build/nfpm.yml.tmpl > _output/keights-deb/nfpm.yml
	$(HOME)/go/bin/nfpm pkg \
		-f _output/keights-deb/nfpm.yml \
		-t _output/keights-deb/keights_$(VERSION)_$(GOOS)_$(GOARCH).deb

keights-stack:
	cp -R stack/ansible/keights-stack _output
	cp stack/cloudformation/*.yml _output/keights-stack/files
	echo $(VERSION) > _output/keights-stack/version
	tar czf _output/keights-stack-$(VERSION).tar.gz -C _output keights-stack

keights-system:
	tar czf _output/keights-system-$(VERSION).tar.gz -C stack/ansible keights-system

stackbot:
	for bot in auto_namer instattr kube_ca subnet_to_az; do \
		go build -o _output/stackbot/$${bot}/$${bot} ./stackbot/$${bot}; \
		(cd _output/stackbot/$${bot} && zip $${bot}-$(VERSION).zip $${bot}); \
	done

dist: keights-deb keights-stack keights-system stackbot

github-release: dist
	VERSION=$(VERSION) REPO_SLUG=$(REPO_SLUG) ./build/github-release

test:
	go test -failfast -race -covermode=atomic ./... -run . -timeout=2m

clean:
	rm -rf _output

fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; done

.DEFAULT_GOAL := dist
.PHONY: setup keights stackbot dist github-release test clean fmt
