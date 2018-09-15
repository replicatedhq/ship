.PHONY: clean_build_artifacts clean deps serve_ship build_ship build_ship_dev test test_CI

SHELL := /bin/bash
$SRC = $(shell find . -path "./node_modules" -prune -o -path "./build" -prune -o \( -name "*.js" -o -name "*.jsx" -o -name "*.scss" -o -name "*.json" -o -name "*.svg" -o -name "*.png" \) -print)

.state/package: package.json yarn.lock
	yarn install
	@mkdir -p .state
	@touch .state/package

clean_build_artifacts:
	rm -rf build
	rm -rf dist

clean: clean_build_artifacts
	rm -rf node_modules
	rm -rf .state

deps: .state/package
	cd ../init && \
	yarn --pure-lockfile && \
	yarn build && \
	yarn link
	yarn link @replicatedhq/ship-init

# TODO: Dev builds should use --mode development
deps-dev: .state/package
	cd ../init && \
	yarn --pure-lockfile && \
	yarn build && \
	yarn link
	yarn link @replicatedhq/ship-init

serve_ship:
	yarn start

.state/build_ship: deps $(shell find src -type f) clean_build_artifacts
	yarn build
	@mkdir -p .state
	@touch .state/build_ship

.state/build_ship_dev: deps-dev clean_build_artifacts
	yarn build
	@mkdir -p .state
	@touch .state/build_ship_dev

build_ship: .state/build_ship

build_ship_dev: .state/build_ship_dev

test:
	yarn test

test_CI:
	cd ../init && \
	yarn --pure-lockfile && \
	CI=1 yarn test
