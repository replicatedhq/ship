Contributing
=============

This is a living document and is to be expanded.
As of its initial writing, this document's main goal is to create a home for intructions pertaining to building and running the project.

Issues
------------------------

Pull Requests
------------------------

All pull requests must be made from forks enabled on CircleCI, so that unit and acceptance tests can be executed prior to code being merged.

Changelog descriptions should be in the past tense, such as 'Fixed a bug that made everything explode' or 'Added a magical new feature'.

### Getting CircleCI to run your tests

You'll need to enable your fork in CircleCI (under `Add Projects`).
You'll also need to [enable build processing](https://circleci.com/docs/2.0/build-processing/).
Finally, add an environment variable `GITHUB_TOKEN` to the CircleCI project to allow `integration_init` to complete successfully.
This token requires no permissions and is used to avoid GitHub request ratelimits.
You can generate one [here](https://github.com/settings/tokens).

Build & Run the Project
------------------------

### Prerequisites

The following tools must be installed before the project can be built.
The specified version are recommended.

- `yarn` version 1.12
- `node` version 8.11
- `go` version 1.12
- `dep` version 0.5 (https://github.com/golang/dep#installation)

### First time build

Run the following commands before building ship for the first time:

```
./hack/get_build_deps.sh
make deps
```

To build ship executable, run

```
make bin/ship
```

To rebuild everything, including tests, run

```
make build
```

### Running

To run locally-built copy of ship, use

```
./bin/ship init <chart-path>
```

for example,

```
./bin/ship init github.com/helm/charts/stable/nginx-ingress
```

### Writing tests

Tests make extensive use of mocks.
To add mocks for new types, the type needs to be added to the `mockgen` target in the make file.
The following command will generate new mocks and update existing ones:

```
make mockgen
```

### Using the UI

A webpack development server can be started for iterating on the ui with the following command:

```
make -C web serve_ship
```

The go binary serves the UI on `localhost:8800`, the webpack dev server will serve on `localhost:8880`.

### A note on node modules
On rare occasions, node modules may need to be refreshed.
If `make build` results in an error of the following flavor:
```
...
make[1]: *** [.state/build_ship] Error 2
make: *** [build-ui] Error 2
```
and/or if `make -C web serve_ship` gives results in a `Failed to compile` error, the following commands should get everything back up and running.
From the root of the project:
```
cd web
rm -rf node_modules
yarn
```
