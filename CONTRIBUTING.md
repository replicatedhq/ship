Contributing
=============

This is a living document and is to be expanded. As of its initial writing, this document's main goal is to create a home for
intructions pertaining to building and running the project.

Issues
------------------------

Pull Requests
------------------------

Before submitting a pull request, please ensure you have enabled your fork on CircleCI, so that unit and acceptance tests can be run against the request.

Build & Run the Project
------------------------

### Prerequisites

Ensure you have (min versions to be added):

- `yarn`
- `go`
- `node`

### First time build

The first time you build ship, you'll need to run the following

```
make build
```

If you're planning to only work on "headless" mode, and don't need the UI built, you can just run

```
make bin/ship
```

### Running

To run your locally-built copy of ship, use

```
./bin/ship init <chart-path>
```

for example,

```
./bin/ship init github.com/helm/charts/stable/nginx-ingress
```

### Iterating

You can run

```
make build
```

to re-build the project.

For iterating on the ui, you can start a webpack development server with

```
make -C web serve_ship
```

The go binary serves the UI on `localhost:8800`, the webpack dev server will serve on `localhost:8880`.

### A note on node modules
On rare occasions, you may need to refresh your node modules. If `make build` gives you an error of the following flavor:
```
...
make[1]: *** [.state/build_ship] Error 2
make: *** [build-ui] Error 2
``` 
and/or if `make -C web serve_ship` gives you a `Failed to compile` error, the following commands should get you back up and running. From the root of the project:
```
cd web
rm -rf node_modules
yarn
```

