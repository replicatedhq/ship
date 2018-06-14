.PHONY: build-deps -dep-deps docker shell githooks dep fmt _vet vet _lint lint _test test build e2e run build_yoonit_docker_image _build


SHELL := /bin/bash
SRC = $(shell find . -name "*.go")

build-deps:
	go get -u github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports


dep-deps:
	go get -u github.com/golang/dep/cmd/dep

docker:
	docker build -t ship .

shell:
	docker run --rm -it \
		-p 8880:8880 \
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
	mkdir -p pkg/test-mocks/images
	mkdir -p pkg/test-mocks/docker
	mkdir -p pkg/test-mocks/helm
	mkdir -p pkg/test-mocks/dockerlayer
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
		-destination pkg/test-mocks/config/daemon.go \
		-package config \
		github.com/replicatedhq/ship/pkg/lifecycle/render/config \
		Daemon
	mockgen \
		-destination pkg/test-mocks/planner/planner_mock.go \
		-package planner \
		github.com/replicatedhq/ship/pkg/lifecycle/render/planner \
		Planner
	mockgen \
		-destination pkg/test-mocks/images/image_saver_mock.go \
		-package images \
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

mockgen: _mockgen fmt

deps:
	dep ensure -v; dep prune -v


fmt:
	goimports -w pkg
	goimports -w cmd

_vet:
	go vet ./pkg/...
	go vet ./cmd/...

vet: fmt _vet

_lint:
	golint ./pkg/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" || :
	golint ./cmd/... | grep -vE '_mock|e2e' | grep -v "should have comment" | grep -v "comment on exported" || :

lint: vet _lint

_test:
	go test -v ./pkg/...

test: lint _test

_citest:
	go test -coverprofile=coverage.out -v ./pkg/...

citest: lint _citest

ci-build-deps:
	wget -O cc-test-reporter https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64
	chmod +x cc-test-reporter

ci-upload-coverage:
	./cc-test-reporter format-coverage -t gocov coverage.out
	./cc-test-reporter upload-coverage


build: test bin/ship

_build: bin/ship

bin/ship: $(SRC)
	go build \
		-i \
		-o bin/ship \
		./cmd/ship
	@echo built bin/ship

e2e: bin/ship
	./bin/ship e2e



run: bin/ship
	./bin/ship --log-level=debug --studio-file=./app.yml

# this should really be in a different repo
build_yoonit_docker_image:
	docker build -t replicated/yoonit:latest -f deploy/Dockerfile-yoonit .
