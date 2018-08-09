Contributing
=============

This is a living document and is to be expanded. As of its initial writing, this document's main goal is to create a home for
intructions pertaining to building and running the project.

Issues
------------------------

Pull Requests
------------------------

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
make build-ui embed-ui build
```

If you're planning to only work on "headless" modes, you can omit the `build-ui` and `embed-ui` steps.

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

For iterating on the go (not the UI), you can use

```
make build
```

to re-build the project.

For iterating on the ui, you can start a webpack development server with

```
make -C web serve_ship
```

