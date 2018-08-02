#!/bin/sh
# Suppress logs from Docker Registry
$GOPATH/bin/registry serve config.yml > /dev/null 2>&1 &
sleep 2
./base.test
../update/update.test
