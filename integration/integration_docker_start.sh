#!/bin/sh
set -e

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

cd ../init_app
echo "START INIT_APP TESTS"
./init_app.test
echo "END INIT_APP TESTS"

cd ../init_chart
echo "START INIT_CHART TESTS"
./init_chart.test
echo "END INIT_CHART TESTS"
