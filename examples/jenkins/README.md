# Ship and Jenkins

This example shows one approach to creating a Jenkins job that updates an application using ship. This example assumes the application (Sourcegraph) is delivered as a public Helm chart in GitHub and the organization installing Sourcegraph is using GitHub Enterprise (git.somebigbank.com) to store the state.

A screenshot showing the GitHub Enterprise repo is:

![GitHub Enterprise](https://github.com/replicatedhq/ship/blob/master/examples/jenkins/github-enterprise.png)

In this example, we store the version of the application that's running on the cluster in `master` and the latest version that hasn't been deployed in a branch named `sourcegraph`. The Jenkins job defined here will update `sourcegraph` with the lastest version and create a pull request to master with the changes that should be deployed. Then, a deployment tool such as Weave can manage the actual rollout and deployment of the Kubernetes application.

## Jenkins Configuration
Create a Jenkins job with 1 "shell" step:

```shell
#!/bin/bash

set -x

sudo rm -rf third-party-apps
git clone git@git.somebigbank.com:ops/third-party-apps.git
cd third-party-apps
git checkout --track origin/sourcegraph
cd sourcegraph

# Decrypt state file
openssl enc -d -aes-256-cbc -a -pass pass:$ENCRYPTION_KEY -in ./.ship/state.json.enc -out ./.ship/state.json

ship update github.com/sourcegraph/deploy-sourcegraph

# Encrypt state file and remove the clear text one
openssl enc -aes-256-cbc -a -salt -pass pass:$ENCRYPTION_KEY -in ./.ship/state.json -out ./.ship/state.json.enc
rm ./.ship/state.json

# Commit, if there were any differences
git status
git diff
git add .
git commit -m "Update to sourcegraph"

if [ $? -eq 0 ]; then
  git push origin sourcegraph
  GITHUB_HOST=git.somebigbank.com USER=jenkins hub pull-request -m "$(git log -1 --pretty=%B)"
fi
```
