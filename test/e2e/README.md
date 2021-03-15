# End-to-end Tests

## Dependencies

Install the `e2e` package, from the project directory:

```shell
pip install -e test/e2e
```

This only needs to be installed once (not on every code change).

## Running the tests

Before running tests, instruct the Python client to use your development CLI binary:

```shell
export CORTEX_CLI_PATH=<cortex_repo_path>/bin/cortex
```

### AWS

From an existing cluster:

```shell
pytest test/e2e/tests -k aws --aws-env <cortex_aws_env>
```

Using a new cluster, created for testing only and deleted afterwards:

```shell
pytest test/e2e/tests -k aws --aws-config <cortex_aws_cluster_config.yaml>
```

**Note:** For the BatchAPI tests, the `--s3-path` option should be provided with an AWS S3 bucket for testing purposes.
It is more convenient however to define this bucket through an environment variable, see [configuration](#configuration)
.

### GCP

From an existing cluster:

```shell
pytest test/e2e/tests -k gcp --gcp-env <cortex_gcp_env>
```

Using a new cluster, created for testing only and deleted afterwards:

```shell
pytest test/e2e/tests -k gcp --gcp-config <cortex_gcp_cluster_config.yaml>
```

### All Tests

You can run all tests at once, however the provider specific options should be passed accordingly, or the test cases
will be skipped.

e.g.

```shell
pytest test/e2e/tests --aws-env <cortex_aws_env> --gcp-env <cortex_gcp_env>
```

### Skip GPU Tests

It is possible to skip GPU tests by passing the `--skip-gpus` flag to the pytest command.

### Skip Inferentia Tests

It is possible to skip Inferentia tests by passing the `--skip-infs` flag to the pytest command.

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
