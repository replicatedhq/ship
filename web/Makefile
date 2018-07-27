clean:
	rm -rf node_modules
	rm -rf dist

deps:
	yarn install

serve_config:
	`yarn bin`/webpack-dev-server --config webpack.config.js --progress -w --debug --env configOnly --mode development

serve_ship:
	`yarn bin`/webpack-dev-server --config webpack.config.js --progress -w --debug --env shipDev --mode development

build_ship:
	`yarn bin`/webpack --config webpack.config.js --env ship --mode production

embed_ship:
	rm -r ${GOPATH}/src/github.com/replicatedhq/ship/ui/*
	cp -r dist/* ${GOPATH}/src/github.com/replicatedhq/ship/ui/
	:
	: Assets embedded. You will need to rebuild ship to see changes reflected

test:
	yarn test
