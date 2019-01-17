Ship Cypress E2E Tests
======================
## Local (Iterative) Development
There are a few steps to get local development set up with Cypress and Ship.
1. Make sure you are `cd`'d into the `web` directory.
1. `CYPRESS_HOST=localhost:8800 npx cypress open`
    - This command opens the Cypress UI, displaying all available tests,
1. Before execution of a test, run the following command in a test folder:
   ```sh
   $GOPATH/src/github.com/replicatedhq/ship/bin/ship init <CHART_URL> --no-open
   ```
    - This command must be executed before a test run since the Cypress tests rely on a newly started server.
      There is no way to orchestrate this command from within Cypress.
    - It is recommended to not run this command in the `ship` folder, but some test folder or scratch workspace.
1. Click on a test in the Cypress UI to start execution. When the test finishes, the `ship init` command will exit.
1. Repeat steps 3 & 4 between executions.

## Run Full Test Suite
From the base directory, run `make cypress` to run all tests.

## Run in Docker
From the root directory of the project:
```sh
docker build -t replicatedhq/ship-cypress:latest -f ./web/cypress/Dockerfile .
docker run -it replicatedhq/ship-cypress:latest
```
