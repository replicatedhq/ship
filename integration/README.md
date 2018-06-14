#Running the integration tests (from the parent directory)

```shell
make integration-test
```

#Adding a new integration test

Each integration test is a folder containing a text file with the 
desired customer ID, installation ID, release version, a folder 'testfiles' containing
`.ship/release.yml` and `.ship/state.yml`, and a folder 'expected' 
containing the expected output of running ship with that state file, customer ID and release.yml.