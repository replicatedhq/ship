#Running the integration tests (from the parent directory)

```shell
make integration-test
```

#Running the integration tests (from a docker image)

The integration test docker image can be built with

```shell
make build_ship_integration_test
```

The resulting image can be run with

```shell
docker run -it -v /var/run/docker.sock:/var/run/docker.sock replicated/ship-e2e-test:latest
```

#Adding a new integration test

Each integration test is a folder containing a text file with the
desired customer ID, installation ID, release version, a folder 'testfiles' containing
`.ship/release.yml` and `.ship/state.yml`, and a folder 'expected'
containing the expected output of running ship with that state file, customer ID and release.yml.
