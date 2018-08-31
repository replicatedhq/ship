#!/bin/sh
set -e

# Suppress logs from Docker Registry
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
