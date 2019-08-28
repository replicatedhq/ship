.NOTPARALLEL:

.PHONY: build-deps docker shell githooks dep e2e run citest ci-upload-coverage goreleaser integration-test build_ship_integration_test build-ui build-ui-dev mark-ui-gitignored fmt lint vet test build embed-ui clean-ship clean clean-integration

export GO111MODULE=on

SHELL := /bin/bash -o pipefail
SRC = $(shell find pkg -name "*.go" ! -name "ui.bindatafs.go")
FULLSRC = $(shell find pkg -name "*.go")
UI = $(shell find web/app/init/build -name "*.js")

DOCKER_REPO ?= replicated

VERSION_PACKAGE = github.com/replicatedhq/ship/pkg/version
VERSION ?=`git describe --tags`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`
HELMV = v2.14.1
KUSTOMIZEV = v2.0.3
TERRAFORMV = v0.11.14

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
	-X ${VERSION_PACKAGE}.helm=${HELMV} \
	-X ${VERSION_PACKAGE}.kustomize=${KUSTOMIZEV} \
	-X ${VERSION_PACKAGE}.terraform=${TERRAFORMV} \
"
endef

.state/build-deps: hack/get_build_deps.sh
	time ./hack/get_build_deps.sh
	@mkdir -p .state/
	@touch .state/build-deps

build-deps: .state/build-deps

.state/lint-deps: hack/get_lint_deps.sh
	time ./hack/get_lint_deps.sh
	@mkdir -p .state/
	@touch .state/lint-deps

lint-deps: .state/lint-deps

docker:
	docker build -t ship .

shell:
	docker run --rm -it \
		-p 8800:8800 \
		-v `pwd`/out:/out \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/replicatedhq/ship \
		--name ship \
		ship

githooks:
	echo 'make test' > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	echo 'make fmt; git add `git diff --name-only --cached`' > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

.PHONY: pacts
pacts:
	go test -v -mod vendor ./contracts/...

.PHONY: pacts-ci
pacts-ci:
	docker build -t ship-contract-tests -f contracts/Dockerfile.testing .
	docker run --rm --name ship-contract-tests \
		ship-contract-tests \
		bash -c 'go test -v -mod vendor ./contracts/...'

.PHONY: pacts-ci-publish
pacts-ci-publish:
	docker build -t ship-contract-tests -f contracts/Dockerfile.testing .
	docker run --rm --name ship-contract-tests \
		-e PACT_BROKER_USERNAME -e PACT_BROKER_PASSWORD -e VERSION=$$CIRCLE_TAG \
		ship-contract-tests \
		bash -c 'go test -v -mod vendor ./contracts/... && ./contracts/publish.sh'

_mockgen: build-deps
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
	mkdir -p pkg/test-mocks/githubclient
	mkdir -p pkg/test-mocks/inline
	mkdir -p pkg/test-mocks/daemon
	mkdir -p pkg/test-mocks/tfplan
	mkdir -p pkg/test-mocks/state
	mkdir -p pkg/test-mocks/apptype
	mkdir -p pkg/test-mocks/replicatedapp
	mkdir -p pkg/test-mocks/util
	mockgen \
		-destination pkg/test-mocks/ui/ui.go \
		-package ui \
		github.com/mitchellh/cli \
		Ui & \
	mockgen \
		-destination pkg/test-mocks/config/resolver.go \
		-package config \
		github.com/replicatedhq/ship/pkg/lifecycle/render/config \
		Resolver & \
	mockgen \
		-destination pkg/test-mocks/daemon/daemon.go \
		-package daemon \
		github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes \
		Daemon & \
	mockgen \
		-destination pkg/test-mocks/planner/planner_mock.go \
		-package planner \
		github.com/replicatedhq/ship/pkg/lifecycle/render/planner \
		Planner & \
	mockgen \
		-destination pkg/test-mocks/images/saver/image_saver_mock.go \
		-package saver \
		github.com/replicatedhq/ship/pkg/images \
		ImageSaver & \
	mockgen \
		-destination pkg/test-mocks/images/image_manager_mock.go \
		-package images \
		github.com/replicatedhq/ship/pkg/images \
		ImageManager & \
	mockgen \
		-destination pkg/test-mocks/images/pull_url_resovler_mock.go \
		-package images \
		github.com/replicatedhq/ship/pkg/images \
		PullURLResolver & \
	mockgen \
		-destination pkg/test-mocks/helm/chart_fetcher_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		ChartFetcher & \
	mockgen \
		-destination pkg/test-mocks/helm/templater_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Templater & \
	mockgen \
		-destination pkg/test-mocks/helm/commands_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Commands & \
	mockgen \
		-destination pkg/test-mocks/helm/renderer_mock.go \
		-package helm \
		github.com/replicatedhq/ship/pkg/lifecycle/render/helm \
		Renderer & \
	mockgen \
		-destination pkg/test-mocks/docker/renderer_mock.go \
		-package docker \
		github.com/replicatedhq/ship/pkg/lifecycle/render/docker \
		Renderer & \
	mockgen \
		-destination pkg/test-mocks/dockerlayer/archive_mock.go \
		-package dockerlayer \
		github.com/mholt/archiver \
		Archiver & \
	mockgen \
		-destination pkg/test-mocks/github/github_mock.go \
		-package github \
		github.com/replicatedhq/ship/pkg/lifecycle/render/github \
		Renderer & \
	mockgen \
		-destination pkg/test-mocks/inline/inline_mock.go \
		-package inline \
		github.com/replicatedhq/ship/pkg/lifecycle/render/inline \
		Renderer & \
	mockgen \
		-destination pkg/test-mocks/tfplan/confirmer_mock.go \
		-package tfplan \
		github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan \
		PlanConfirmer & \
	mockgen \
		-destination pkg/test-mocks/state/manager_mock.go \
		-package state \
		github.com/replicatedhq/ship/pkg/state \
		Manager & \
	mockgen \
		-destination pkg/test-mocks/lifecycle/messenger_mock.go \
		-package lifecycle \
		github.com/replicatedhq/ship/pkg/lifecycle \
		Messenger & \
	mockgen \
		-destination pkg/test-mocks/lifecycle/renderer_mock.go \
		-package lifecycle \
		github.com/replicatedhq/ship/pkg/lifecycle \
		Renderer & \
	mockgen \
		-destination pkg/test-mocks/apptype/determine_type_mock.go \
		-package apptype \
		github.com/replicatedhq/ship/pkg/specs/apptype \
		Inspector,LocalAppCopy & \
	mockgen \
		-destination pkg/test-mocks/replicatedapp/resolve_replicated_app.go \
		-package replicatedapp \
		github.com/replicatedhq/ship/pkg/specs/replicatedapp \
		Resolver & \
	mockgen \
		-destination pkg/test-mocks/util/asset_uploader.go \
		-package util \
		github.com/replicatedhq/ship/pkg/util \
		AssetUploader & \
	mockgen \
		-destination pkg/test-mocks/githubclient/github_fetcher.go \
		-package githubclient \
		github.com/replicatedhq/ship/pkg/specs/githubclient \
		GitHubFetcher & \
	wait


mockgen: _mockgen fmt

deps:
	dep ensure -v


.state/fmt: $(SRC)
	goimports -w pkg
	goimports -w cmd
	goimports -w integration
	@mkdir -p .state
	@touch .state/fmt


fmt: .state/lint-deps .state/fmt

.state/vet: $(SRC)
	go vet -mod vendor ./pkg/...
	go vet -mod vendor ./cmd/...
	go vet -mod vendor ./integration/...
	@mkdir -p .state
	@touch .state/vet

vet: .state/vet

.state/ineffassign: .state/lint-deps $(SRC)
	ineffassign ./pkg
	ineffassign ./cmd
	ineffassign ./integration
	@mkdir -p .state
	@touch .state/ineffassign

ineffassign: .state/ineffassign

.state/golangci-lint: .state/lint-deps $(SRC)
	golangci-lint run ./pkg/...
	golangci-lint run ./cmd/...
	golangci-lint run ./integration/...
	@mkdir -p .state
	@touch .state/golangci-lint

golangci-lint: .state/golangci-lint

.state/lint: $(SRC)
	golint ./pkg/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" | grep -v "package comment should be of the form" | grep -v bindatafs || :
	golint ./cmd/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" | grep -v "package comment should be of the form" | grep -v bindatafs || :
	@mkdir -p .state
	@touch .state/lint

lint: vet golangci-lint .state/lint

.state/test: $(SRC)
	go test -mod vendor ./pkg/... ./integration | grep -v '?'
	@mkdir -p .state
	@touch .state/test

test: lint .state/test

.state/race: $(SRC)
	go test --race -mod vendor ./pkg/...
	@mkdir -p .state
	@touch .state/race

race: lint .state/race

.state/coverage.out: $(SRC)
	@mkdir -p .state/
	#the reduced parallelism here is to avoid hitting the memory limits - we consistently did so with two threads on a 4gb instance
	go test -parallel 1 -p 1 -coverprofile=.state/coverage.out -mod vendor ./pkg/... ./integration

citest: .state/coverage.out

.PHONY: cilint
cilint: .state/vet .state/ineffassign .state/lint

.state/cc-test-reporter:
	@mkdir -p .state/
	wget -O .state/cc-test-reporter https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64
	chmod +x .state/cc-test-reporter

ci-upload-coverage: .state/coverage.out .state/cc-test-reporter
	./.state/cc-test-reporter format-coverage -o .state/codeclimate/codeclimate.json -t gocov .state/coverage.out
	./.state/cc-test-reporter upload-coverage -i .state/codeclimate/codeclimate.json

build: fmt embed-ui-dev test bin/ship

build-ci: ci-embed-ui bin/ship

build-ci-cypress: mark-ui-gitignored pkg/lifecycle/daemon/ui.bindatafs.go bin/ship

build-minimal: build-ui pkg/lifecycle/daemon/ui.bindatafs.go bin/ship

bin/ship: $(FULLSRC)
	go build \
		-mod vendor \
		${LDFLAGS} \
		-i \
		-o bin/ship \
		./cmd/ship
	@echo built bin/ship

bin/ship.exe: $(SRC)
	GOOS=windows go build \
		-mod vendor \
		${LDFLAGS} \
		-i \
		-o bin/ship.exe \
		./cmd/ship
	@echo built bin/ship

# tests base "ship" cli
integration-test:
	ginkgo -p -stream -r integration

goreleaser: .state/goreleaser

.state/goreleaser: deploy/.goreleaser.unstable.yml deploy/Dockerfile $(SRC)
	@mkdir -p .state
	curl -sL https://git.io/goreleaser | bash -s -- --snapshot --rm-dist --config deploy/.goreleaser.unstable.yml
	@touch .state/goreleaser

run: build
	./bin/ship app --log-level=debug --runbook=./fixtures/app.yml

build_ship_integration_test:
	docker build -t $(DOCKER_REPO)/ship-e2e-test:latest -f ./integration/Dockerfile .

pkg/lifecycle/daemon/ui.bindatafs.go: .state/mark-ui-gitignored .state/build-deps web/app/.state/built-ui
	export PATH=$(GOPATH)/bin:$$PATH; go-bindata-assetfs -pkg daemon \
	  -o pkg/lifecycle/daemon/ui.bindatafs.go \
	  -prefix web/app \
	  web/app/build/...

mark-ui-gitignored: .state/mark-ui-gitignored

.state/mark-ui-gitignored:
	cd pkg/lifecycle/daemon/; $(GIT_UPDATE_INDEX_CMD) ui.bindatafs.go
	@mkdir -p .state/
	@touch .state/mark-ui-gitignored

embed-ui: mark-ui-gitignored build-ui pkg/lifecycle/daemon/ui.bindatafs.go

embed-ui-dev: mark-ui-gitignored build-ui-dev pkg/lifecycle/daemon/ui.bindatafs.go

ci-embed-ui: mark-ui-gitignored pkg/lifecycle/daemon/ui.bindatafs.go

# this file will be updated by build-ui and build-ui-dev, causing ui.bindata.fs to be regenerated
web/app/.state/built-ui:
	@mkdir -p web/app/.state/
	@touch web/app/.state/built-ui

build-ui:
	$(MAKE) -C web/app build_ship

build-ui-dev:
	$(MAKE) -C web/app build_ship_dev

test_CI:
	$(MAKE) -C web/app test_CI

cypress_base:
	CYPRESS_SPEC=cypress/integration/init/testchart.spec.js \
	CHART_URL=github.com/replicatedhq/test-charts/tree/ad1e78d13c33fae7a7ce22ed19920945ceea23e9/modify-chart \
	sh web/app/cypress/run_init_spec.sh

cypress: build cypress_base

# this shouldn't ever have to be run, but leaving here for
# posterity on how the go-bindatafs "dev" file was generated
# before we marked it as ignored. the goal here is to
# generate an empty bindata fs, so things are obviously wrong
# rather than folks just getting an old version of the UI
dev-embed-ui:
	mkdir -p .state/tmp/dist
	export PATH=$(GOPATH)/bin:$$PATH; go-bindata-assetfs -pkg daemon \
	  -o pkg/lifecycle/daemon/ui.bindatafs.go \
	  -prefix .state/tmp/ \
	  -debug \
	  .state/tmp/dist/...

clean-ship:
	rm -rf chart/
	rm -rf installer/
	rm -rf installer.bak/
	rm -rf overlays/
	rm -rf base/
	rm -rf .ship/
	rm -rf rendered.yaml

clean-integration:
	rm -rf integration/*/*/_test_*

clean:
	rm -rf .state
	$(MAKE) -C web/app clean
