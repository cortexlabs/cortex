# Python client

* [cortex](#cortex)
  * [client](#client)
  * [new\_client](#new_client)
  * [env\_list](#env_list)
  * [env\_delete](#env_delete)
* [cortex.client.Client](#cortex-client-client)
  * [deploy](#deploy)
  * [deploy\_from\_file](#deploy_from_file)
  * [get\_api](#get_api)
  * [list\_apis](#list_apis)
  * [get\_job](#get_job)
  * [refresh](#refresh)
  * [delete](#delete)
  * [stop\_job](#stop_job)

# cortex

## client

```python
client(env_name: Optional[str] = None) -> Client
```

Initialize a client based on the specified environment. If no environment is specified, it will attempt to use the default environment.

**Arguments**:

- `env_name` - Name of the environment to use.


**Returns**:

  Cortex client that can be used to deploy and manage APIs in the specified environment.

## new\_client

```python
new_client(env_name: str, operator_endpoint: str) -> Client
```

Create a new environment to connect to an existing cluster, and initialize a client to deploy and manage APIs on that cluster.

**Arguments**:

- `env_name` - Name of the environment to create.
- `operator_endpoint` - The endpoint for the operator of your Cortex cluster. You can get this endpoint by running the CLI command `cortex cluster info`.


**Returns**:

  Cortex client that can be used to deploy and manage APIs on a cluster.

## env\_list

```python
env_list() -> List
```

List all environments configured on this machine.

## env\_delete

```python
env_delete(name: str)
```

Delete an environment configured on this machine.

**Arguments**:

- `name` - Name of the environment to delete.

# cortex.client.Client

## deploy

<!-- CORTEX_VERSION_MINOR -->

```python
 | deploy(api_spec: Dict[str, Any], force: bool = True, wait: bool = False)
```

Deploy or update an API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortexlabs.com/v/master/ for schema.
- `force` - Override any in-progress api updates.
- `wait` - Block until the API is ready.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_from\_file

<!-- CORTEX_VERSION_MINOR -->

```python
 | deploy_from_file(config_file: str, force: bool = False, wait: bool = False) -> Dict
```

Deploy or update APIs specified in a configuration file.

**Arguments**:

- `config_file` - Local path to a yaml file defining Cortex API(s). See https://docs.cortexlabs.com/v/master/ for schema.
- `force` - Override any in-progress api updates.
- `wait` - Block until the API is ready.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## get\_api

```python
 | get_api(api_name: str) -> Dict
```

Get information about an API.

**Arguments**:

- `api_name` - Name of the API.


**Returns**:

  Information about the API, including the API specification, endpoint, status, and metrics (if applicable).

## list\_apis

```python
 | list_apis() -> List
```

List all APIs in the environment.

**Returns**:

  List of APIs, including information such as the API specification, endpoint, status, and metrics (if applicable).

## get\_job

```python
 | get_job(api_name: str, job_id: str) -> Dict
```

Get information about a submitted job.

**Arguments**:

- `api_name` - Name of the Batch/Task API.
- `job_id` - Job ID.


**Returns**:

  Information about the job, including the job status, worker status, and job progress.

## refresh

```python
 | refresh(api_name: str, force: bool = False)
```

Restart all of the replicas for a Realtime API without downtime.

**Arguments**:

- `api_name` - Name of the API to refresh.
- `force` - Override an already in-progress API update.

## delete

```python
 | delete(api_name: str, keep_cache: bool = False)
```

Delete an API.

**Arguments**:

- `api_name` - Name of the API to delete.
- `keep_cache` - Whether to retain the cached data for this API.

## stop\_job

```python
 | stop_job(api_name: str, job_id: str, keep_cache: bool = False)
```

Stop a running job.

**Arguments**:

- `api_name` - Name of the Batch/Task API.
- `job_id` - ID of the Job to stop.
