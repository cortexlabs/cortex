# Python client

* [cortex](#cortex)
  * [client](#cortex.client)
  * [local\_client](#cortex.local_client)
  * [cluster\_client](#cortex.cluster_client)
  * [env\_list](#cortex.env_list)
  * [env\_delete](#cortex.env_delete)
* [cortex.client.Client](#cortex.client.Client)
  * [deploy](#cortex.client.Client.deploy)
  * [get\_api](#cortex.client.Client.get_api)
  * [list\_apis](#cortex.client.Client.list_apis)
  * [get\_job](#cortex.client.Client.get_job)
  * [refresh](#cortex.client.Client.refresh)
  * [delete\_api](#cortex.client.Client.delete_api)
  * [stop\_job](#cortex.client.Client.stop_job)
  * [stream\_api\_logs](#cortex.client.Client.stream_api_logs)
  * [stream\_job\_logs](#cortex.client.Client.stream_job_logs)

<a name="cortex"></a>
# cortex

<a name="cortex.client"></a>
## client

```python
client(env: str)
```

Initialize a client based on the specified environment.

To deploy and manage APIs locally:
import cortex
c = cortex.client("local")
c.deploy("./cortex.yaml")

To deploy and manage APIs on a new cluster:
1. Spin up a cluster using the CLI command `cortex cluster up`.
An environment named "aws" will be created once the cluster is ready.
2. Initialize your client:
import cortex
c = cortex.client("aws")
c.deploy("./cortex.yaml")

To deploy and manage APIs on an existing cluster:
1. Use the command `cortex cluster info` to get the Operator Endpoint.
2. Configure a client to your cluster:
import cortex
c = cortex.cluster_client("aws", operator_endpoint, aws_access_key_id, aws_secret_access_key)
c.deploy("./cortex.yaml")

**Arguments**:

- `env` - Name of the environment to use.


**Returns**:

  Cortex client that can be used to deploy and manage APIs in the specified environment.

<a name="cortex.local_client"></a>
## local\_client

```python
local_client(aws_access_key_id: str, aws_secret_access_key: str, aws_region: str) -> Client
```

Initialize a client to deploy and manage APIs locally.

The specified AWS credentials will be used by the CLI to download models
from S3 and authenticate to ECR, and will be set in your Predictor.

**Arguments**:

- `aws_access_key_id` - AWS access key ID.
- `aws_secret_access_key` - AWS secret access key.
- `aws_region` - AWS region.


**Returns**:

  Cortex client that can be used to deploy and manage APIs locally.

<a name="cortex.cluster_client"></a>
## cluster\_client

```python
cluster_client(name: str, operator_endpoint: str, aws_access_key_id: str, aws_secret_access_key: str) -> Client
```

Create a new environment to connect to an existing Cortex Cluster, and initialize a client to deploy and manage APIs on that cluster.

**Arguments**:

- `name` - Name of the environment to create.
- `operator_endpoint` - The endpoint for the operator of your Cortex Cluster. You can get this endpoint by running the CLI command `cortex cluster info`.
- `aws_access_key_id` - AWS access key ID.
- `aws_secret_access_key` - AWS secret access key.


**Returns**:

  Cortex client that can be used to deploy and manage APIs on a Cortex Cluster.

<a name="cortex.env_list"></a>
## env\_list

```python
env_list() -> list
```

List all environments configured on this machine.

<a name="cortex.env_delete"></a>
## env\_delete

```python
env_delete(name: str)
```

Delete an environment configured on this machine.

**Arguments**:

- `name` - Name of the environment to delete.

<a name="cortex.client.Client"></a>
# cortex.client.Client

<a name="cortex.client.Client.deploy"></a>
## deploy

```python
 | deploy(config_file: str, force: bool = False, wait: bool = False) -> list
```

Deploy or update APIs specified in the config_file.

**Arguments**:

- `config_file` - Local path to a yaml file defining Cortex APIs.
- `force` - Override any in-progress api updates.
- `wait` - Streams logs until the APIs are ready.


**Returns**:

  Deployment status, API specification, and endpoint for each API.

<a name="cortex.client.Client.get_api"></a>
## get\_api

```python
 | get_api(api_name: str) -> dict
```

Get information about an API.

**Arguments**:

- `api_name` - Name of the API.


**Returns**:

  Information about the API, including the API specification, endpoint, status, and metrics (if applicable).

<a name="cortex.client.Client.list_apis"></a>
## list\_apis

```python
 | list_apis() -> list
```

List all APIs in the environment.

**Returns**:

  List of APIs, including information such as the API specification, endpoint, status, and metrics (if applicable).

<a name="cortex.client.Client.get_job"></a>
## get\_job

```python
 | get_job(api_name: str, job_id: str) -> dict
```

Get information about a submitted job.

**Arguments**:

- `api_name` - Name of the Batch API.
- `job_id` - Job ID.


**Returns**:

  Information about the job, including the job status, worker status, and job progress.

<a name="cortex.client.Client.refresh"></a>
## refresh

```python
 | refresh(api_name: str, force: bool = False)
```

Restart all of the replicas for a Realtime API without downtime.

**Arguments**:

- `api_name` - Name of the API to refresh.
- `force` - Override an already in-progress API update.

<a name="cortex.client.Client.delete_api"></a>
## delete\_api

```python
 | delete_api(api_name: str, keep_cache: bool = False)
```

Delete an API.

**Arguments**:

- `api_name` - Name of the API to delete.
- `keep_cache` - Whether to retain the cached data for this API.

<a name="cortex.client.Client.stop_job"></a>
## stop\_job

```python
 | stop_job(api_name: str, job_id: str, keep_cache: bool = False)
```

Stop a running job.

**Arguments**:

- `api_name` - Name of the Batch API.
- `job_id` - ID of the Job to stop.

<a name="cortex.client.Client.stream_api_logs"></a>
## stream\_api\_logs

```python
 | stream_api_logs(api_name: str)
```

Stream the logs of an API.

**Arguments**:

- `api_name` - Name of the API.

<a name="cortex.client.Client.stream_job_logs"></a>
## stream\_job\_logs

```python
 | stream_job_logs(api_name: str, job_id: str)
```

Stream the logs of a Job.

**Arguments**:

- `api_name` - Name of the Batch API.
- `job_id` - Job ID.
