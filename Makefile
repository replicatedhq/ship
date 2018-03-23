build-deps:
	go get -u github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports

dep-deps:
	go get -u github.com/golang/dep/cmd/dep

githooks:
	echo 'make test' > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	echo 'make fmt' > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

dep:
	dep ensure

fmt:
	goimports -w pkg
	goimports -w cmd

vet: fmt
	go vet ./pkg/...
	go vet ./cmd/...

lint: vet
	golint ./pkg/... | grep -v _mock || :
	golint ./cmd/... | grep -v _mock || :

test: lint
	go test -v ./pkg/...

build: test _build

_build:
	go build \
		-o bin/ship \
		./cmd/ship

run:
	./bin/ship apply --log-level=debug --studio-file=./app.yml
