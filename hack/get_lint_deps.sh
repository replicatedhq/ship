#!/bin/sh
set -v

go get -u golang.org/x/tools/cmd/goimports
go get -u golang.org/x/lint/golint
go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.17.1
go get -u github.com/gordonklaus/ineffassign
