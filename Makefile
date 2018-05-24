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
	$(GOPATH)/bin/nfpm pkg \
		-f _output/keights-deb/nfpm.yml \
		-t _output/keights-deb/keights_$(VERSION)_$(GOOS)_$(GOARCH).deb

stackbot:
	go build -o _output/stackbot/kube_ca/kube_ca ./stackbot/kube_ca
	(cd _output/stackbot/kube_ca && zip kube_ca-$(VERSION).zip kube_ca)

	go build -o _output/stackbot/subnet_to_az/subnet_to_az ./stackbot/subnet_to_az
	(cd _output/stackbot/subnet_to_az && zip subnet_to_az-$(VERSION).zip subnet_to_az)

dist: keights-deb stackbot

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
