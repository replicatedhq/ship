# Console integration tests

These are tests for ship yaml, with the intention of testing functionality used with console/pg.replicated.com

## Running the integration tests (from the parent directory)

### Quick Start
```shell
make integration-test
```

### Dependencies
- Helm
- Terraform
- Test Docker Registry
    * To run a local Docker registry for tests, run the following command:
      ```sh
      docker run -d -p 5000:5000 --restart=always --name registry registry:2
      ```
- Test S3 Bucket
    * To run a local http server for tests, run the following command:
      ```sh
      npm install -g http-echo-server
      PORT=4569 http-echo-server 
      ```

## Running the integration tests (from a docker image)

The integration test docker image can be built with

```shell
make build_ship_integration_test
```

The resulting image can be run with

```shell
docker run --net="host" -it -v /var/run/docker.sock:/var/run/docker.sock replicated/ship-e2e-test:latest
```

## Adding a new integration test

Each integration test is a folder containing a yaml file with the
desired customer ID/installation ID/release semver, a folder 'input' containing
`.ship/release.yml` and `.ship/state.json`, and a folder 'expected'
containing the expected output of running ship with that state file, release yaml,
and customer ID/installation ID/release semver.

Each integration test is run twice - once in runbook (or 'local yaml') mode and once in online mode.
Both runs are headless and use the Cobra API to simulate running Ship from the CLI.
The runbook mode run will use the release yaml located at `input/.ship/release.yml` and the state file located at `input/.ship/state.json`.
The online mode run will use the state file at `input/.ship/state.json` but will get the release yaml from the graphql api using the provided customer ID, installation ID and release semver.
Files are produced in a temporary directory created within the integration test directory.
The contents of this directory is then diffed with the contents of `expected/`.
File names and contents must match.

To add a new test, create a release that should demonstrate the desired behavior in the integration test staging account.
You should also create a new directory for your test within the integration subdirectory.
After running this release in an empty directory, copy the files produced (including hidden directories) to `<your_test_dir>/expected` and the `.ship` directory to `<your_test_dir>/input/.ship`.
The contents of the `.ship` directory within expected and input can differ if desired.
Finally, add a file `<your_test_dir>/metadata.yaml` and include strings for `customer_id`, `installation_id` and `release_version`.
