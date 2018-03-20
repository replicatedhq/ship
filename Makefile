build-deps:
	go get -u github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports

dep-deps:
	go get -u github.com/golang/dep/cmd/dep

githooks:
	echo 'make test' > .git/pre-push
	chmod +x .git/pre-push
	echo 'make fmt' > .git/pre-commit
	chmod +x .git/pre-commit

dep:
	dep ensure

fmt:
	goimports -w pkg
	goimports -w cmd

vet: fmt
	go vet ./pkg/...
	go vet ./cmd/...

lint: vet
	golint ./pkg/...
	golint ./pkg/...

test: lint
	go test -v ./pkg/...

build: test _build

_build:
	go build \
		-o bin/ship \
		./cmd/ship

run:
	./bin/ship apply --log_level=debug --studio_file=./app.yml

integration-test-cloud:
	./bin/ship integration-test --target http://localhost:8036/ship/v1 --verbose

integration-test-licensed:
	./bin/ship integration-test --target http://localhost:8035/ship/v1 --verbose
