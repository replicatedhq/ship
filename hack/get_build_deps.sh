#!/bin/sh
set -v

go get golang.org/x/tools/cmd/goimports
go get -u github.com/golang/lint/golint
go get github.com/golang/mock/gomock
go install github.com/golang/mock/mockgen
go get github.com/elazarl/go-bindata-assetfs/...
go get -u github.com/jteeuwen/go-bindata/...
