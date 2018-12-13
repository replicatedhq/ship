#!/bin/sh
set -v

go get -u golang.org/x/tools/cmd/goimports
go get -u golang.org/x/lint/golint
go get github.com/golang/mock/gomock
go install github.com/golang/mock/mockgen
go get github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs
go get -u github.com/jteeuwen/go-bindata/go-bindata
go get -u github.com/gordonklaus/ineffassign
