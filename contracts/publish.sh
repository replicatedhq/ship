#!/bin/sh

curl \
		--silent --output /dev/null --show-error --fail \
		--user ${PACT_BROKER_USERNAME}:${PACT_BROKER_PASSWORD} \
		-X PUT \
		-H "Content-Type: application/json" \
		-d@pacts/ship-prem-graphql-api.json \
		https://replicated-pact-broker.herokuapp.com/pacts/provider/prem-graphql-api/consumer/ship/version/${VERSION}
