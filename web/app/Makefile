.PHONY: clean deps serve_ship build_ship test test_CI

SHELL := /bin/bash
$SRC = $(shell find . -path "./node_modules" -prune -o -path "./dist" -prune -o \( -name "*.js" -o -name "*.jsx" -o -name "*.scss" -o -name "*.json" -o -name "*.svg" -o -name "*.png" \) -print)

.state/package: package.json yarn.lock
	yarn install
	@mkdir -p .state
	@touch .state/package

clean:
	rm -rf node_modules
	rm -rf dist
	rm -rf .state

deps: .state/package
	cd ../ship-init && \
	yarn --pure-lockfile && \
	yarn build && \
	yarn link
	yarn link @replicatedhq/ship-init

serve_ship:
	yarn start

.state/build_ship: .state/package $(shell find src -type f)
	rm -rf dist
	rm -rf build
	yarn build
	@mkdir -p .state
	@touch .state/build_ship

build_ship: .state/build_ship

test:
	yarn test

test_CI:
	CI=1 yarn test
