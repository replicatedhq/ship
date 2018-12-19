Ship
=======

[![Test Coverage](https://api.codeclimate.com/v1/badges/a00869c41469d016a3c8/test_coverage)](https://codeclimate.com/github/replicatedhq/ship/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/a00869c41469d016a3c8/maintainability)](https://codeclimate.com/github/replicatedhq/ship/maintainability)
[![CircleCI](https://circleci.com/gh/replicatedhq/ship.svg?style=svg&circle-token=471765bf5ec85ede48fcf02ea6a886dc6c5a73f1)](https://circleci.com/gh/replicatedhq/ship)
[![Docker Image](https://images.microbadger.com/badges/image/replicated/ship.svg)](https://microbadger.com/images/replicated/ship)
[![Go Report Card](https://goreportcard.com/badge/github.com/replicatedhq/ship)](https://goreportcard.com/report/github.com/replicatedhq/ship)
[![GitHub stars](https://img.shields.io/github/stars/replicatedhq/ship.svg)](https://github.com/replicatedhq/ship/stargazers)

![Replicated Ship](https://github.com/replicatedhq/ship/blob/master/logo/logo.png)

Replicated Ship is an open source project by [Replicated](https://www.replicated.com) designed to extend Google’s [Kustomize](https://www.kustomize.io) project in a way that can reduce the operational overhead of maintaining 3rd party applications (open source or proprietary) deployed to a [Kubernetes](https://kubernetes.io) cluster.

The initial release of Replicated Ship exposes the power of Kustomize as an advanced custom configuration management tool for [Helm charts](https://www.github.com/helm/charts), Kubernetes manifests and [Knative](https://github.com/knative/) applications.
With Ship, cluster operators can automatically stay in sync with upstream changes (ie. via automated pull requests or another form of automation) while preserving their local, custom configurations and extensions (add, deletes and edits) without git merge conflicts.
This is possible because of how the three operating modes of Ship invoke, store, and apply Kustomizations made by the cluster operator.

Read on for more details on Ship features and operating modes, or skip ahead to [getting started](#getting-started).

# Three operating modes

![Replicated Ship Modes](https://github.com/replicatedhq/ship/blob/master/logo/ship-flow.png)

## ship init
Prepares a new application for deployment. Use for:
- Specifying the upstream source for an application to be managed -- typically a repo with raw Kubernetes yaml or a Helm chart
- Creating and managing [Kustomize](https://kustomize.io/) overlays to be applied before deployment
- Generating initial config (state.json) for the application, and persisting that config to disk for use with the other modes

## ship watch
Polls an upstream source, blocking until any change has been published.  Use for:
- Triggering creation of pull requests in a CI pipeline, so that third party updates can be manually reviewed, and then automatically deployed once merged

## ship update
Updates an existing application by merging the latest release with the local state and overlays. Use for:
- Preparing an update to be deployed to a third party application
- Automating the update process to start from a continuous integration (CI) service

# Features
Ship is designed to provide first-time configuration UI and/or be used headless in a CI/CD pipeline to automate deployment of third party applications.

- Web based "admin console" provides initial configuration of [Helm](https://helm.sh/) values and creates [Kustomize](https://kustomize.io/) overlays
- Headless mode supports automated pipelines
- Merge [Helm](https://helm.sh/) charts with override values and apply custom overlays with [Kustomize](https://kustomize.io/) to avoid merge conflicts when upstream or local values are changed
- Watch upstream repos for updates & sync changes to your local version.
- Render [Helm](https://helm.sh/) charts to pure Kubernetes yaml that can be deployed to clusters without Tiller
- Enables [GitOps](https://www.weave.works/blog/the-gitops-pipeline) workflows to update third party applications
- Configuration workflow `ship.yaml` files can be included in [Helm](https://helm.sh/) chart repos, to customize the initial `ship init` experience

# Getting Started

## Installation

Ship is packaged as a single binary, and Linux and MacOS versions are distributed:
- To download the latest Linux build, run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.29.0/ship_0.29.0_linux_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

- To download the latest MacOS build, you can either run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.29.0/ship_0.29.0_darwin_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

- ... or you can install with [Homebrew](https://brew.sh/):
```shell
brew tap replicatedhq/ship
brew install ship
```

Alternately, you can run Ship in Docker, in which case you can pull the latest ship image with:
```shell
docker pull replicated/ship
```

## Initializing 
After Ship is installed, create a directory for the application you'll be managing with Ship, and launch Ship from there, specifying an upstream Helm chart or Kubernetes yaml:

```shell
mkdir -p ~/my-ship/example
cd ~/my-ship/example
ship init <path-to-chart> # github.com/helm/charts/tree/master/stable/grafana
```

Alternately, the same command run through Docker:
```shell
mkdir -p ~/my-ship/example
cd ~/my-ship/example
docker run -p 8800:8800 -v "$PWD":/wd -w /wd \
    replicated/ship init <path-to-chart> # github.com/helm/charts/tree/master/stable/grafana
```
_Note: you may need to point your browser to http://127.0.0.1:8800 if ship's suggested localhost URL doesn't resolve._

You'll be prompted to open a browser and walk through the steps to configure site-specific values for your installation, updating Helm values (if it's a chart), and making direct edits to the Kubernetes yaml (or Helm-generated yaml), which will be converted to patches to apply via Kustomize.

After completing the guided 'ship init' workflow, you'll see that Ship has generated several directories and files within the current working directory.

```
├── .ship
│   └── state.json
├── base
│   ├── clusterrole.yaml
│   ├── ...
│   └── serviceaccount.yaml
├── overlays
│   └── ship
│       └── kustomization.yaml
└── rendered.yaml
```

`.ship/state.json` - maintains all the configuration decisions made within the `ship init` flow, including the path to the upstream, the upstream's original `values.yaml`, any modifications made to `values.yaml`, and any patch directives configured in the Kustomize phase.

The `base/` and `overlays/` folders contain the various files that drive the Kustomization process.

The `rendered.yaml` file is the final output, suitable to deploy to your Kubernetes cluster via 

```shell
kubectl apply -f rendered.yaml
```

If you need to revise any of the configuration details, you can re-invoke `ship init <path-to-chart>` to start fresh, or `ship update --headed` to walk through the configuration steps again, starting with your previously entered values & patches as a baseline.

## Unforking
Another initialization option is to start with a Helm chart or Kubernetes manifest that has been forked from an upstream source, and to "unfork" it.

```shell
ship unfork <path-to-forked> --upstream <path-to-upstream>
```
or

```shell
docker run -v "$PWD":/wd -w /wd \
    replicated/ship unfork <path-to-forked> \
    --upstream <path-to-upstream>
```

With this workflow, Ship will attempt to move the changes that prompted the fork into 'overlays' that can be applied as patches onto the unmodified upstream base.  You can inspect the `rendered.yaml` to verify the final output, or run through `ship update --headed` to review the generated overlays in the Ship admin console.


# CI/CD Integration
Once you've prepared an application using `ship init`, a simple starting CI/CD workflow could be:

```shell
ship watch && ship update 
```

or
```shell
docker run -v "$PWD":/wd -w /wd replicated/ship watch && \
    docker run -v "$PWD":/wd -w /wd replicated/ship update
```

The `watch` command is a trigger for CI/CD processes, watching the upstream application for changes. Running `ship watch` will load the local state file (which includes a content hash of the most recently used upstream) and periodically poll the upstream application and exit when it finds a change. `ship update` will regenerate the deployable application assets, using the most recent upstream version of the application, and any local configuration from `state.json`.  The new `rendered.yaml` output can be deployed directly to the cluster, or submitted as a pull request into a [GitOps](https://www.weave.works/blog/what-is-gitops-really) repo.

With chart repo you have commit privileges on, you, you can see this flow in action by running `ship init <path-to-chart>` and going through the workflow, then `ship watch --interval 10s && ship update` to start polling, then commit a change to the upstream chart and see the `ship watch` process exit, with `rendered.yaml` updated to reflect the change.  

# Customizing the Configuration Experience

Maintainers of OTS (Off the Shelf) software can customize the `ship init` experience by including a `ship.yaml` manifest alongside a Helm Chart or Kubernetes manifest.  The [Replicated Ship YAML](https://help.replicated.com/docs/ship/getting-started/yaml-overview/) format allows further customization of the installation process, including infrastructure automation steps to spin up and configure clusters to deploy to.  (If you're wondering about some of the more obscure Ship CLI option flags, these mostly apply to ship.yaml features)

# Ship Cloud

For those not interested in operating and maintaining a fleet of Ship instances, [Ship Cloud](https://www.replicated.com/ship) is available as a hosted solution.   With Ship Cloud, teams can collaborate and manage multiple OTS Kubernetes application settings in one place, with Ship watching and updating on any upstream or local configuration changes, and creating Pull Requests and other integrations into CI/CD systems.

# Community

For questions about using Ship, there's a [Replicated Community](https://help.replicated.com/community) forum.

For bug reports, please [open an issue](https://github.com/replicatedhq/ship/issues/new) in this repo.

For instructions on building the project and making contributions, see [Contributing](./CONTRIBUTING.md)

