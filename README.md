Ship
=======

[![Test Coverage](https://api.codeclimate.com/v1/badges/7e19355b20109fd50ada/test_coverage)](https://codeclimate.com/repos/5b217b8b536ddc029d005c48/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/7e19355b20109fd50ada/maintainability)](https://codeclimate.com/repos/5b217b8b536ddc029d005c48/maintainability)
[![CircleCI](https://circleci.com/gh/replicatedhq/ship.svg?style=svg&circle-token=471765bf5ec85ede48fcf02ea6a886dc6c5a73f1)](https://circleci.com/gh/replicatedhq/ship)
[![Docker Image](https://images.microbadger.com/badges/image/replicated/ship.svg)](https://microbadger.com/images/replicated/ship)


![Replicated Ship](https://github.com/replicatedhq/ship/blob/master/logo/logo.png)

Ship enables the operation of third-party applications through modern software deployment pipelines (i.e. [GitOps](https://www.weave.works/blog/the-gitops-pipeline)). Ship is a command line and UI that prepares workflows to enable deploying and updating of [Helm](https://helm.sh/) charts, Kubernetes applications and other third-party software. Ship handles the process of merging custom settings (`state.json`) with custom overlays (using [Kustomize](https://kustomize.io/)), and preparing a deployable set of assets (an application). Ship is designed to provide first-time configuration UI and/or be used headless in a CI/CD pipeline to automate deployment of third party applications.

Ship includes first-class support for [Helm](https://helm.sh/) charts, and automates the "last-mile" of custom configuration via [Kustomize](https://kustomize.io/).

# Features
- Web based "admin console" provides initial configuration of [Helm](https://helm.sh/) values and creates [Kustomize](https://kustomize.io/) overlays
- Headless mode supports automated pipelines
- Merge [Helm](https://helm.sh/) charts with override values and apply custom overlays with [Kustomize](https://kustomize.io/) to avoid merge conflicts when upstream or local values are changed
- Deploy [Helm](https://helm.sh/) charts to a Kubernetes cluster without Tiller
- Enables [GitOps](https://www.weave.works/blog/the-gitops-pipeline) workflows to update third party applications
- Configuration workflow `ship.yml` files can be included in [Helm](https://helm.sh/) chart repos, to customize the initial `ship init` experience

# Operating modes

## ship init
Prepares a new application for deployment. Use for:
- Generating initial config (state.json) for an application
- Creating and managing [Kustomize](https://kustomize.io/) overlays to be applied before deployment

## ship update
Updates an existing application by merging the latest release with the local state and overlays. Use for:
- Preparing an update to be deployed to a third party application
- Automating the update process to start from a continuous integration (CI) service

## ship watch
Polls an upstream source, blocking until any change has been published.  Use for:
- Triggering creation of pull requests in a CI pipeline, so that third party updates can be manually reviewed, and then automatically deployed once merged

# Installation
There are two ways you can get started with Ship:

## Installing locally
Ship is packaged as a single binary, and Linux and MacOS versions are distributed:
- To download the latest Linux build, run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.14.0/ship_0.14.0_linux_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

- To download the latest MacOS build, run:
```shell
curl -sSL https://github.com/replicatedhq/ship/releases/download/v0.14.0/ship_0.14.0_darwin_amd64.tar.gz | tar xv && sudo mv ship /usr/local/bin
```

After ship is installed, run it with:

```shell
ship init <path-to-chart> # github.com/kubernetes/charts/mysql
```

## Running in Docker
To run ship in Docker:
```shell
docker run replicated/ship init <path-to-chart> # github.com/kubernetes/charts/mysql
```

Note, you will need to mount and configure a shared volume, in order to persist any changes made within the Ship admin console when launched via Docker.


## Ship Modes
![Replicated Ship Modes](https://github.com/replicatedhq/ship/blob/master/logo/ship-flow.png)

# CI/CD Integration
Once you've prepared an application using `ship init`, the deployable application assets can be generated, using any version of the application, by running:

```shell
ship update
```

The `watch` command is designed to be a trigger for a CI/CD process by watching the upstream application for changes. Running `ship watch` will load the state file and periodically poll the upstream application and exit when it finds a change. A simple, starting workflow could be to run `ship watch && ship update` after completing `ship init`. This will apply an update to the base directory.

# Community

For questions about using Ship, there's a [Replicated Community](https://help.replicated.com/community) forum.

For bug reports, please [open an issue](https://github.com/replicatedhq/ship/issues/new) in this repo.

For instructions on building the project and making contributions, see [Contributing](./CONTRIBUTING.md)

