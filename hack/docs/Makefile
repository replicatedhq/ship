.PHONY: build

SCHEMA_JSON := ./schema.json
MUTATIONS := ./mutations.json
HACK_DOCS := ./bin/hack-docs

deps:
	yarn

prebuild:
	rm -rf assets
	rm -rf lifecycle
	rm -rf config
	mkdir -p assets
	mkdir -p lifecycle
	mkdir -p config


build:
	mkdir -p bin
	go build  -o $(HACK_DOCS) ../../cmd/hack-docs

gen-schema:
	$(HACK_DOCS) > $(SCHEMA_JSON)

merge-mutations:
	`yarn bin`/ts-node src/index.ts merge-mutations -f $(SCHEMA_JSON) -m $(MUTATIONS)  > $(SCHEMA_JSON).tmp
	mv $(SCHEMA_JSON).tmp $(SCHEMA_JSON)

validate:
	`yarn bin`/ts-node src/index.ts validate -f $(SCHEMA_JSON)


markdown-assets:
	mkdir -p build
	`yarn bin`/ts-node src/index.ts markdown-assets -f $(SCHEMA_JSON) -o ./assets


pipeline-strict: prebuild gen-schema merge-mutations validate markdown-assets
pipeline: prebuild gen-schema merge-mutations markdown-assets

copy-to-help-center:
	: To Copy to Help center, run the following command:
	:
	:    make pipeline
	:    cp assets/* [path/to/help/center]/content/api/ship-assets/
	:    cp lifecycle/* [path/to/help/center]/content/api/ship-lifecycle/
	:    cp config/* [path/to/help/center]/content/api/ship-config/

copy-replicated-lint:
	: To Copy to replicated lint, run the following command:
	:
	:    make pipeline
	:    cp schema.json  [path/to/replicated-lint]/projects/replicated-ship/

lint:
	`yarn bin`/replicated-lint validate -f ../../ship-retraced.yml --excludeDefaults --schema $(SCHEMA_JSON)

ship:
	../../bin/ship --studio-file ../../ship-retraced.yml --log-level debug
