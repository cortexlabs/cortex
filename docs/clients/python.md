# Python client

* [cortex](#cortex)
  * [client](#client)
  * [new\_client](#new_client)
  * [env\_list](#env_list)
  * [env\_delete](#env_delete)
* [cortex.client.Client](#cortex-client-client)
  * [deploy](#deploy)
  * [deploy\_realtime\_api](#deploy_realtime_api)
  * [deploy\_async\_api](#deploy_async_api)
  * [deploy\_batch\_api](#deploy_batch_api)
  * [deploy\_task\_api](#deploy_task_api)
  * [deploy\_traffic\_splitter](#deploy_traffic_splitter)
  * [get\_api](#get_api)
  * [list\_apis](#list_apis)
  * [get\_job](#get_job)
  * [refresh](#refresh)
  * [patch](#patch)
  * [delete](#delete)
  * [stop\_job](#stop_job)
  * [stream\_api\_logs](#stream_api_logs)
  * [stream\_job\_logs](#stream_job_logs)

# cortex

## client

```python
client(env: Optional[str] = None) -> Client
```

Initialize a client based on the specified environment. If no environment is specified, it will attempt to use the default environment.

**Arguments**:

- `env` - Name of the environment to use.


**Returns**:

  Cortex client that can be used to deploy and manage APIs in the specified environment.

## new\_client

```python
new_client(name: str, operator_endpoint: str) -> Client
```

Create a new environment to connect to an existing cluster, and initialize a client to deploy and manage APIs on that cluster.

**Arguments**:

- `name` - Name of the environment to create.
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

```python
 | deploy(api_spec: Dict[str, Any], project_dir: str, force: bool = True, wait: bool = False)
```

Deploy an API from a project directory.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/ for schema.
- `project_dir` - Path to a python project.
- `force` - Override any in-progress api updates.
- `wait` - Streams logs until the APIs are ready.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_realtime\_api

```python
 | deploy_realtime_api(api_spec: Dict[str, Any], handler, requirements: Optional[List] = None, conda_packages: Optional[List] = None, force: bool = True, wait: bool = False) -> Dict
```

Deploy a Realtime API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/workloads/realtime-apis/configuration for schema.
- `handler` - A Cortex Handler class implementation.
- `requirements` - A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
- `conda_packages` - A list of Conda dependencies that will be installed before the handler class implementation is invoked.
- `force` - Override any in-progress api updates.
- `wait` - Streams logs until the APIs are ready.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_async\_api

```python
 | deploy_async_api(api_spec: Dict[str, Any], handler, requirements: Optional[List] = None, conda_packages: Optional[List] = None, force: bool = True) -> Dict
```

Deploy an Async API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/workloads/async-apis/configuration for schema.
- `handler` - A Cortex Handler class implementation.
- `requirements` - A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
- `conda_packages` - A list of Conda dependencies that will be installed before the handler class implementation is invoked.
- `force` - Override any in-progress api updates.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_batch\_api

```python
 | deploy_batch_api(api_spec: Dict[str, Any], handler, requirements: Optional[List] = None, conda_packages: Optional[List] = None) -> Dict
```

Deploy a Batch API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/workloads/batch-apis/configuration for schema.
- `handler` - A Cortex Handler class implementation.
- `requirements` - A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
- `conda_packages` - A list of Conda dependencies that will be installed before the handler class implementation is invoked.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_task\_api

```python
 | deploy_task_api(api_spec: Dict[str, Any], task, requirements: Optional[List] = None, conda_packages: Optional[List] = None) -> Dict
```

Deploy a Task API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/workloads/task-apis/configuration for schema.
- `task` - A callable class implementation.
- `requirements` - A list of PyPI dependencies that will be installed before the handler class implementation is invoked.
- `conda_packages` - A list of Conda dependencies that will be installed before the handler class implementation is invoked.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

## deploy\_traffic\_splitter

```python
 | deploy_traffic_splitter(api_spec: Dict[str, Any]) -> Dict
```

Deploy a Task API.

**Arguments**:

- `api_spec` - A dictionary defining a single Cortex API. See https://docs.cortex.dev/v/master/workloads/realtime-apis/traffic-splitter/configuration for schema.


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

## patch

```python
 | patch(api_spec: Dict, force: bool = False) -> Dict
```

Update the api specification for an API that has already been deployed.

**Arguments**:

- `api_spec` - The new api specification to apply
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

## stream\_api\_logs

```python
 | stream_api_logs(api_name: str)
```

Stream the logs of an API.

**Arguments**:

- `api_name` - Name of the API.

## stream\_job\_logs

```python
 | stream_job_logs(api_name: str, job_id: str)
```

Stream the logs of a Job.

**Arguments**:

- `api_name` - Name of the Batch API.
- `job_id` - Job ID.
