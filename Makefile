.PHONY: build-deps dep-deps docker shell githooks dep e2e run citest ci-upload-coverage goreleaser integration-test build_ship_integration_test build-ui mark-ui-gitignored fmt lint vet test build embed-ui clean


SHELL := /bin/bash -o pipefail
SRC = $(shell find . -name "*.go" ! -name "ui.bindatafs.go")
FULLSRC = $(shell find . -name "*.go")
UI = $(shell find web/dist -name "*.js")

DOCKER_REPO ?= replicated

VERSION_PACKAGE = github.com/replicatedhq/ship/pkg/version
VERSION=`git describe --tags &>/dev/null || echo "v0.0.1"`
GIT_SHA=`git rev-parse HEAD`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.state/build-deps: hack/get_build_deps.sh
	./hack/get_build_deps.sh
	@mkdir -p .state/
	@touch .state/build-deps

build-deps: .state/build-deps

dep-deps:
	go get -u github.com/golang/dep/cmd/dep

docker:
	docker build -t ship .

shell:
	docker run --rm -it \
		-p 8800:8800 \
		-v `pwd`/out:/out \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/replicatedhq/ship \
		ship

githooks:
	echo 'make test' > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	echo 'make fmt; git add `git diff --name-only --cached`' > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

_mockgen:
	rm -rf pkg/test-mocks
	mkdir -p pkg/test-mocks/ui
	mkdir -p pkg/test-mocks/config
	mkdir -p pkg/test-mocks/planner
	mkdir -p pkg/test-mocks/lifecycle
	mkdir -p pkg/test-mocks/images/saver
	mkdir -p pkg/test-mocks/docker
	mkdir -p pkg/test-mocks/helm
	mkdir -p pkg/test-mocks/dockerlayer
	mkdir -p pkg/test-mocks/github
	mkdir -p pkg/test-mocks/inline
	mkdir -p pkg/test-mocks/daemon
	mkdir -p pkg/test-mocks/tfplan
	mkdir -p pkg/test-mocks/state
	mockgen \
		-destination pkg/test-mocks/ui/ui.go \
		-package ui \
		github.com/mitchellh/cli \
		Ui
	mockgen \
		-destination pkg/test-mocks/config/resolver.go \
		-package config \
		github.com/replicatedhq/ship/pkg/lifecycle/render/config \
		Resolver
	mockgen \
		-destination pkg/test-mocks/daemon/daemon.go \
		-package daemon \
		github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes \
		Daemon
	mockgen \
		-destination pkg/test-mocks/planner/planner_mock.go \
		-package planner \
		github.com/replicatedhq/ship/pkg/lifecycle/render/planner \
		Planner
	mockgen \
		-destination pkg/test-mocks/images/saver/image_saver_mock.go \
		-package saver \
		github.com/replicatedhq/ship/pkg/images \
		ImageSaver
	mockgen \
		-destination pkg/test-mocks/images/image_manager_mock.go \
		-package images \
		github.com/replicatedhq/ship/pkg/images \
		ImageManager
	mockgen \
		-destination pkg/test-mocks/images/pull_url_resovler_mock.go \
		-package images \
		github.com/replicatedhq/ship/pkg/images \
		PullURLResolver
	mockgen \
		-destination pkg/test-mocks/helm/chart_fetcher_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		ChartFetcher
	mockgen \
		-destination pkg/test-mocks/helm/templater_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Templater
	mockgen \
		-destination pkg/test-mocks/helm/commands_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Commands
	mockgen \
		-destination pkg/test-mocks/helm/renderer_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Renderer
	mockgen \
		-destination pkg/test-mocks/docker/renderer_mock.go \
		-package docker \
		github.com/replicatedhq/ship/pkg/lifecycle/render/docker \
		Renderer
	mockgen \
		-destination pkg/test-mocks/dockerlayer/archive_mock.go \
		-package dockerlayer \
		github.com/mholt/archiver \
		Archiver
	mockgen \
		-destination pkg/test-mocks/github/github_mock.go \
		-package github \
		github.com/replicatedhq/ship/pkg/lifecycle/render/github \
		Renderer
	mockgen \
		-destination pkg/test-mocks/inline/inline_mock.go \
		-package inline \
		github.com/replicatedhq/ship/pkg/lifecycle/render/inline \
		Renderer
	mockgen \
		-destination pkg/test-mocks/tfplan/confirmer_mock.go \
		-package tfplan \
		github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan \
		PlanConfirmer
	mockgen \
		-destination pkg/test-mocks/state/manager_mock.go \
		-package state \
		github.com/replicatedhq/ship/pkg/state \
		Manager
	mockgen \
		-destination pkg/test-mocks/lifecycle/messenger_mock.go \
		-package lifecycle \
		github.com/replicatedhq/ship/pkg/lifecycle \
		Messenger
	mockgen \
		-destination pkg/test-mocks/lifecycle/renderer_mock.go \
		-package lifecycle \
		github.com/replicatedhq/ship/pkg/lifecycle \
		Renderer

mockgen: _mockgen fmt

deps:
	dep ensure -v


.state/fmt: $(SRC)
	goimports -w pkg
	goimports -w cmd
	@mkdir -p .state
	@touch .state/fmt


fmt: .state/build-deps .state/fmt

.state/vet: $(SRC)
	go vet ./pkg/...
	go vet ./cmd/...
	@mkdir -p .state
	@touch .state/vet

vet: .state/vet

.state/lint: $(SRC)
	golint ./pkg/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindatafs || :
	golint ./cmd/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindatafs || :
	@mkdir -p .state
	@touch .state/lint

lint: vet .state/lint

.state/test: $(SRC)
	go test ./pkg/... | grep -v '?'
	@mkdir -p .state
	@touch .state/test

test: lint .state/test

.state/coverage.out: $(SRC)
	@mkdir -p .state/
	#the reduced parallelism here is to avoid hitting the memory limits - we consistently did so with two threads on a 4gb instance
	go test -parallel 1 -p 1 -coverprofile=.state/coverage.out ./pkg/...

citest: .state/vet .state/lint .state/coverage.out

.state/cc-test-reporter:
	@mkdir -p .state/
	wget -O .state/cc-test-reporter https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64
	chmod +x .state/cc-test-reporter

ci-upload-coverage: .state/coverage.out .state/cc-test-reporter
	./.state/cc-test-reporter format-coverage -o .state/codeclimate/codeclimate.json -t gocov .state/coverage.out
	./.state/cc-test-reporter upload-coverage -i .state/codeclimate/codeclimate.json

build: fmt embed-ui test bin/ship

bin/ship: $(FULLSRC)
	go build \
		${LDFLAGS} \
		-i \
		-o bin/ship \
		./cmd/ship
	@echo built bin/ship

# tests base "ship" cli
integration-test:
	ginkgo -p -stream integration/base

# tests "ship kustomize"
integration-test-update:
	ginkgo -p -stream integration/update

goreleaser: .state/goreleaser

.state/goreleaser: deploy/.goreleaser.unstable.yml deploy/Dockerfile $(SRC)
	@mkdir -p .state
	curl -sL https://git.io/goreleaser | bash -s -- --snapshot --rm-dist --config deploy/.goreleaser.unstable.yml
	@touch .state/goreleaser

run: build
	./bin/ship app --log-level=debug --runbook=./fixtures/app.yml

build_ship_integration_test:
	docker build -t $(DOCKER_REPO)/ship-e2e-test:latest -f ./integration/Dockerfile .

pkg/lifeycle/daemon/ui.bindatafs.go: .state/build-deps
	go-bindata-assetfs -pkg daemon \
	  -o pkg/lifecycle/daemon/ui.bindatafs.go \
	  -prefix web/ \
	  web/dist/...

mark-ui-gitignored:
	cd pkg/lifecycle/daemon/; git update-index --assume-unchanged ui.bindatafs.go


embed-ui: mark-ui-gitignored build-ui-dev pkg/lifeycle/daemon/ui.bindatafs.go


ci-embed-ui: mark-ui-gitignored pkg/lifeycle/daemon/ui.bindatafs.go
build-ui:
	$(MAKE) -C web build_ship

build-ui-dev:
	$(MAKE) -C web build_ship_dev

ci-build-ui-dev:
	$(MAKE) -C web build_ship_dev PROGRESS=

test_CI:
	$(MAKE) -C web test_CI

# this shouldn't ever have to be run, but leaving here for
# posterity on how the go-bindatafs "dev" file was generated
# before we marked it as ignored. the goal here is to
# generate an empty bindata fs, so things are obviously wrong
# rather than folks just getting an old version of the UI
dev-embed-ui:
	mkdir -p .state/tmp/dist
	go-bindata-assetfs -pkg daemon \
	  -o pkg/lifecycle/daemon/ui.bindatafs.go \
	  -prefix .state/tmp/ \
	  -debug \
	  .state/tmp/dist/...

clean:
	rm -rf .state
	$(MAKE) -C web clean
