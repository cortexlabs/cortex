# End-to-end Tests

## Dependencies

Install the `e2e` package, from the project directory:

```shell
pip install -e test/e2e
```

This only needs to be installed once (not on every code change).

_note: you may need to run `pip3 uninstall cortex` and `pip3 install -e python/client/` before the command above_

## Running the tests

Before running tests, instruct the Python client to use your development CLI binary:

```shell
export CORTEX_CLI_PATH=<cortex_repo_path>/bin/cortex
```

From an existing cluster:

```shell
pytest test/e2e/tests --env <env_name>
```

Using a new cluster, created for testing only and deleted afterwards:

```shell
pytest test/e2e/tests --config <cluster.yaml>
```

**Note:** For the BatchAPI tests, the `--s3-path` option should be provided with an S3 bucket for testing purposes.
It is more convenient however to define this bucket through an environment variable, see [configuration](#configuration)
.

### Skip GPU Tests

It is possible to skip GPU tests by passing the `--skip-gpus` flag to the pytest command.

### Skip Inferentia Tests

It is possible to skip Inferentia tests by passing the `--skip-infs` flag to the pytest command.

### Skip Autoscaling Test

It is possible to skip the autoscaling test by passing the `--skip-autoscaling` flag to the pytest command.

### Skip Load Test

It is possible to skip the load tests by passing the `--skip-load` flag to the pytest command.

### Skip Long Running Test

It is possible to skip the long running test by passing the `--skip-long-running` flag to the pytest command.

## Configuration

It is possible to configure the behaviour of the tests by defining environment variables or a `.env` file at the project
directory.

```dotenv
# .env file
CORTEX_TEST_REALTIME_DEPLOY_TIMEOUT=120
CORTEX_TEST_BATCH_DEPLOY_TIMEOUT=60
CORTEX_TEST_BATCH_JOB_TIMEOUT=120
CORTEX_TEST_BATCH_S3_PATH=s3://<s3_bucket>/test/jobs
```
