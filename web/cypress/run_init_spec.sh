set -e

# REQUIRED: CYPRESS_SPEC, CHART_URL
HOST=${CYPRESS_HOST:-"localhost:8080"}

rm -rf web/cypress/test
mkdir -p web/cypress/test

cd web/cypress/test
../../../bin/ship init $CHART_URL --no-open &
SHIP_PID=$!
cd ../../..

# Always exit 0 in trap on EXIT, $SHIP_PID may not be found
trap "kill -2 $SHIP_PID 2> /dev/null || exit 0" EXIT
trap "kill -2 $SHIP_PID 2> /dev/null" HUP

cd web
CYPRESS_HOST="localhost:8800" npx cypress run --spec $CYPRESS_SPEC
