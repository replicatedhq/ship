#!/bin/sh
set -e

PORT=4569 http-echo-server > /dev/null 2>&1 &

# Docker Registry
$GOPATH/bin/registry serve docker-registry.yaml > /dev/null 2>&1 &


sleep 2
cd base/
./base.test

cd ../update
./update.test

cd ../init_app
./init_app.test

cd ../init
./init.test

cd ../unfork
./unfork.test
