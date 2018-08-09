.PHONY: clean deps serve_config serve_ship build_ship embed_ship test

SHELL := /bin/bash
$SRC = $(shell find . -path "./node_modules" -prune -o -path "./dist" -prune -o \( -name "*.js" -o -name "*.jsx" -o -name "*.scss" -o -name "*.json" -o -name "*.svg" -o -name "*.png" \) -print)

.state/package: package.json package-lock.json yarn.lock
	yarn install
	@mkdir -p .state
	@touch .state/package

clean:
	rm -rf node_modules
	rm -rf dist
	rm -rf .state

deps: .state/package

serve_config:
	`yarn bin`/webpack-dev-server --config webpack.config.js --progress -w --debug --env configOnly --mode development

serve_ship:
	`yarn bin`/webpack-dev-server --config webpack.config.js --progress -w --debug --env shipDev --mode development

.state/build_ship: .state/package webpack* postcss* $(shell find src -type f)
	rm -rf dist
	`yarn bin`/webpack --config webpack.config.js --env production --mode production
	@mkdir -p .state
	@touch .state/build_ship

build_ship: .state/build_ship

embed_ship:
	rm -r ${GOPATH}/src/github.com/replicatedhq/ship/ui/*
	cp -r dist/* ${GOPATH}/src/github.com/replicatedhq/ship/ui/
	:
	: Assets embedded. You will need to rebuild ship to see changes reflected


test:
	yarn test

test_CI:
	CI=1 yarn test
