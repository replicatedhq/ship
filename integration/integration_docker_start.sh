#!/bin/sh
set -e

PORT=4569 http-echo-server > /dev/null 2>&1 &

# Docker Registry
$GOPATH/bin/registry serve docker-registry.yaml > /dev/null 2>&1 &


sleep 2
$GOPATH/bin/ginkgo -p base/base.test
$GOPATH/bin/ginkgo -p update/update.test
$GOPATH/bin/ginkgo -p init_app/init_app.test
$GOPATH/bin/ginkgo -p init/init.test
$GOPATH/bin/ginkgo -p unfork/unfork.test

