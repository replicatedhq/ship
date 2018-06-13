Ship
=======

Replicated Ship on-prem components


### ship

"ship" is the container/binary that reads specs from https://console.replicated.com
and uses them to render application assets for deployment. It has three main responsibilities:

- Lifecycle -- read vendor specs, execute tasks. The `render` step will execute config and asset resolution
- Config -- load config options for the installation, from env, files, and prompts
- Assets -- Once configuration options are resolved, ship will template the specified assets and generate a state file tracking the work


### Get Started

The following will build binaries and run a simple `ship apply` on the testing file `app.yml`
in this directory, with the log level set to debug.

```bash
make build run
```

To add recommended git hooks

```bash
make githooks
```


### Architecture & Foundations

Entrypoint is a Cobra command, root commands runs the main workflow. Other commands are `e2e` for integration testing and `devtool-releaser` for interacting with g.replicated.com to manage releases.

Both commands create an instances of `ship.Ship` which, in order:

- Validates inputs
- resolves the spec 
    - default behavior is to load the spec from GQL using a customer ID
	- `ship` can be run with `--studio-file` flag to skip GQL and just load a spec from the filesystem)
- Execute each step of the lifecyle using the resolved specs


#### Output/CLI

Cobra for CLI, then use https://github.com/mitchellh/cli for its UI interface around Asking/Printing stuff.

We use pflags + viper for resolving config. Ideally Viper can also be used to resolve customer config options.

We use go-kit/log for logging, but the default log level is `off` -- For the most part, we want to suppress all output unless the Vendor has specified it as a message in `lifecycle`.

We do lots of debug logging, and allow a `log-level` param to enable this.


#### Spec

The Specs should be written in YAML. There is experimental support for HCL, but its not quite all there yet. An example spec is in `app.yml`.

The top level yaml document is an instance of `api.Spec`:

```
type Spec struct {
	Assets    Assets   `json:"assets" yaml:"assets" hcl:"asset"`
	Lifecycle Lifecyle `json:"lifecycle" yaml:"lifecycle" hcl:"lifecycle"`
	Config    Config   `json:"config" yaml:"config" hcl:"config"`
}
```

Each item has a `v1` nesting underneath the main key, which should let us mix-and-match versions
for breaking changes going forward. See `app.yml` for examples.




