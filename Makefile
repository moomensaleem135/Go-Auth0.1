PROJ=dex
ORG_PATH=github.com/coreos
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

VERSION=$(shell ./scripts/git-version)

DOCKER_REPO=quay.io/ericchiang/dex
DOCKER_IMAGE=$(DOCKER_REPO):$(VERSION)

export GOBIN=$(PWD)/bin
export GO15VENDOREXPERIMENT=1

LD_FLAGS="-w -X $(REPO_PATH)/version.Version=$(VERSION)"

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

build: bin/dex bin/example-app

bin/dex: FORCE server/templates_default.go
	@go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/dex

bin/example-app: FORCE
	@go install -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/example-app

test:
	@go test -v -i $(shell go list ./... | grep -v '/vendor/')
	@go test -v $(shell go list ./... | grep -v '/vendor/')

testrace:
	@go test -v -i --race $(shell go list ./... | grep -v '/vendor/')
	@go test -v --race $(shell go list ./... | grep -v '/vendor/')

vet:
	@go vet $(shell go list ./... | grep -v '/vendor/')

fmt:
	@go fmt $(shell go list ./... | grep -v '/vendor/')

lint:
	@for package in $(shell go list ./... | grep -v '/vendor/' | grep -v 'api/apipb'); do \
      golint -set_exit_status $$package; \
	done

server/templates_default.go: $(wildcard web/templates/**)
	@go run server/templates_default_gen.go

.PHONY: docker-build
docker-build: bin/dex
	@docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-push
docker-push: docker-build
	@docker tag $(DOCKER_IMAGE) $(DOCKER_REPO):latest
	@docker push $(DOCKER_IMAGE)
	@docker push $(DOCKER_REPO):latest

clean:
	@rm bin/*

testall: testrace vet fmt lint

FORCE:

.PHONY: test testrace vet fmt lint testall
