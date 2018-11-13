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
This is possible because of how the three operating modes of Ship invoke, store and apply Kustomizations made by the cluster operator.

# Three operating modes

## ship init
Prepares a new application for deployment. Use for:
- Generating initial config (state.json) for an application
- Creating and managing [Kustomize](https://kustomize.io/) overlays to be applied before deployment

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
- Deploy [Helm](https://helm.sh/) charts to a Kubernetes cluster without Tiller
- Enables [GitOps](https://www.weave.works/blog/the-gitops-pipeline) workflows to update third party applications
- Configuration workflow `ship.yaml` files can be included in [Helm](https://helm.sh/) chart repos, to customize the initial `ship init` experience

# Installation
There are two ways you can get started with Ship:

## Installing locally
Ship is packaged as a single binary, and Linux and MacOS versions are distributed:
- To download the latest Linux build, run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.23.0/ship_0.23.0_linux_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

- To download the latest MacOS build, run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.23.0/ship_0.23.0_darwin_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

Ship is also available through [Homebrew](https://brew.sh/):
```shell
brew tap replicatedhq/ship
brew install ship
```

After ship is installed, run it with:

```shell
ship init <path-to-chart> # github.com/helm/charts/stable/mysql
```

## Running in Docker
To run ship in Docker:
```shell
docker run -p 8800:8800 replicated/ship init <path-to-chart> # github.com/helm/charts/stable/mysql
```

Note, you will need to mount and configure a shared volume, in order to persist any changes made within the Ship admin console when launched via Docker.


## Ship Modes
![Replicated Ship Modes](https://github.com/replicatedhq/ship/blob/master/logo/ship-flow.png)

# CI/CD Integration
Once you've prepared an application using `ship init`, the deployable application assets can be generated, using any version of the application, by running:

```shell
ship update
```

The `watch` command is designed to be a trigger for a CI/CD process by watching the upstream application for changes. Running `ship watch` will load the state file and periodically poll the upstream application and exit when it finds a change.
A simple, starting workflow could be to run `ship watch && ship update` after completing `ship init`.
This will apply an update to the base directory.

# Community

For questions about using Ship, there's a [Replicated Community](https://help.replicated.com/community) forum.

For bug reports, please [open an issue](https://github.com/replicatedhq/ship/issues/new) in this repo.

For instructions on building the project and making contributions, see [Contributing](./CONTRIBUTING.md)

