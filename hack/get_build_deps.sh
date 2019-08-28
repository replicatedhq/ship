#!/bin/sh
set -v

go get github.com/golang/mock/gomock
go install github.com/golang/mock/mockgen
go get github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs
GO111MODULE=off go get -u github.com/jteeuwen/go-bindata/go-bindata
