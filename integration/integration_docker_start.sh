#!/bin/sh
# Suppress logs from Docker Registry
$GOPATH/bin/registry serve config.yml > /dev/null 2>&1 &
sleep 2
cd base/
echo "START BASE TESTS"
./base.test
echo "END BASE TESTS"

cd ../update
echo "START UPDATE TESTS"
./update.test
echo "END UPDATE TESTS"

