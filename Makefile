GOARCH := amd64
GOOS := linux
REPO_SLUG := cloudboss/keights

setup:
	go get github.com/goreleaser/nfpm/cmd/nfpm

keights:
	mkdir -p _output/keights
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -o _output/keights/keights ./keights

keights-deb: setup keights
	mkdir -p _output/keights-deb
	sed "s|__VERSION__|$(VERSION)|g" build/nfpm.yml.tmpl > _output/keights-deb/nfpm.yml
	$(GOPATH)/bin/nfpm pkg \
		-f _output/keights-deb/nfpm.yml \
		-t _output/keights-deb/keights_$(VERSION)_$(GOOS)_$(GOARCH).deb

export GITHUB_TOKEN
github-release: keights-deb
	VERSION=$(VERSION) REPO_SLUG=$(REPO_SLUG) ./build/github-release

test:
	go test -failfast -race -covermode=atomic ./... -run . -timeout=2m

clean:
	rm -rf _output

fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; done

.DEFAULT_GOAL := keights
.PHONY: setup keights test clean fmt
