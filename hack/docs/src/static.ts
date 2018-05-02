export const ASSETS_INDEX_DOC = `---
categories:
- ship-assets
date: 2018-01-17T23:51:55Z
description: Reference Documentation for defining your Ship application assets 
index: docs
title: Assets
weight: "1"
gradient: "purpleToPink"
---

[Assets](/api/ship-assets/assets) | [Config](/api/ship-config/config) | [Lifecycle](/api/ship-lifecycle/lifecycle) 

## Ship Assets

This is the reference documenation for Ship assets. To get started with Ship, head on over to [Ship Guides](/guides/ship/)

Assets are the core object that enables you to describe applications managed by Ship. They allow you to define scripts, manifests, and application artifacts needed to deploy your application to your end customer's internal infrastructure. The goal of Ship assets is to give your customers controlled, transparent access to the same resources you use to deploy your SaaS application to your own private cloud. Assets can include things like:

- Scripts for installing and upgrading your application to a cloud server
- Private Docker images or ${"`tar.gz`"} archives
- Container orchestration manifests for Kubernetes or Docker Compose
- Modules for infrastructure automation tools like Chef, Ansible, Salt, or Puppet

Documented here are a number of methods Ship provides to facilitating distributing assets to your on-prem customers.

- Inline in your application spec
- Proxied from to private docker registries
- (coming soon) Proxied from to private github repos
- (coming soon) Mirrored from public github repos

In ship, a short assets section to pull and run a private docker container might look like

${"```yaml"}
assets:
  v1:
    - docker:
        dest: images/myimage.tar
        image: registry.replicated.com/myapp/myimage:1.0
        source: replicated
    - inline:
        dest: scripts/install.sh
        mode: 755
        contents: | 
          #!/bin/bash
          
          echo "starting the application..."
          docker load < images/myimage.tar
          docker run -d registry.replicated.com/myapp/myimage:1.0
          
          echo "started!"
          exit 0
        
${"```"}

We're always interested to hear more about how you're deploying your application to your customers, if there's an asset delivery method you'd like to see, drop us a line at https://vendor.replicated.com/support or https://help.replicated.com/community.

`;

export const LIFECYCLE_INDEX_DOC = `---
categories:
- ship-lifecycle
date: 2018-01-17T23:51:55Z
description: Reference Documentation for defining your Ship application lifecycle 
index: docs
title: Lifecycle
weight: "1"
gradient: "purpleToPink"
---

[Assets](/api/ship-assets/assets) | [Config](/api/ship-config/config) | [Lifecycle](/api/ship-lifecycle/lifecycle) 

## Ship Lifecycle

This is the reference documenation for Ship lifecycle. To get started with Ship, head on over to [Ship Guides](/guides/ship/)

Lifeycle is where you can define and customize the end-user experience for customers installing your application. Lifecycle has two step types at the moment:

- ${"`"}message${"`"} - print a message to the console
- ${"`"}render${"`"} - collect configuration and generate assets

In ship, a short assets section to pull and run a private docker container might look like

${"```yaml"}
lifecycle:
  v1:
    - message:
        contents: |
          This installer will prepare assets so you can run CoolTool Enterprise.
    - render: {}
    - message:
        contents: |
          Asset rendering complete! Copy the following files to your production server
          
             ./scripts/install.sh  
             ./images/myimage.tar
          
          And then, on that server, run
             
             bash ./scripts/install.sh 
          
          To start the app. Thanks for using CoolTool!
          
${"```"}

We're always interested to hear more about how you're deploying your application to your customers, if there's a lifecycle step you'd like to see, drop us a line at https://vendor.replicated.com/support or https://help.replicated.com/community.

`;

export const CONFIG_INDEX_DOC = `---
categories:
- ship-config
date: 2018-01-17T23:51:55Z
description: Reference Documentation for defining your Ship application configuration options 
index: docs
title: Config
weight: "1"
gradient: "purpleToPink"
---

[Assets](/api/ship-assets/assets) | [Config](/api/ship-config/config) | [Lifecycle](/api/ship-lifecycle/lifecycle) 

## Ship Config

This is the reference documenation for Ship config. To get started with Ship, head on over to [Ship Guides](/guides/ship/).

Config is where you can define the dynamic values that end customers need to configure before they can use your application. It might include things like: 

  - External Database connection details and credentials
  - Other internal integrations settings like SMTP auth or API keys
  - Tunable settings for your application like "number of worker processes" or "log level"
  
In ship, a minimal config section with one item might look like

${"```yaml"}
config:
  v1:
    - name: "database_info"
      title: "Database Info"
      items:
        - name: pg_connstring
          title: "connection string for a PostgreSQL server"
${"```"}
  
The configuration options API is identical to that used for applications managed by
Replicated's scheduler suite, and the documentation can be found at [Config Screen YAML
](/docs/config-screen/config-yaml/).


We're always interested to hear more about how you're deploying your application to your customers, if there's a config option type you'd like to see, drop us a line at https://vendor.replicated.com/support or https://help.replicated.com/community.

`;
