# Configuration

```yaml
- name: <string>
  kind: AsyncAPI
  predictor: # detailed configuration below
  compute: # detailed configuration below
  autoscaling: # detailed configuration below
  update_strategy: # detailed configuration below
  networking: # detailed configuration below
```

## Predictor

### Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->

```yaml
predictor:
  type: python
  path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/python-predictor-cpu:0.31.1, quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.2-cudnn8, or quay.io/cortexlabs/python-predictor-inf:0.31.1 based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
```

## Compute

```yaml
compute:
  cpu: <string | int | float>  # CPU request per replica. One unit of CPU corresponds to one virtual CPU; fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (default: 200m)
  gpu: <int>  # GPU request per replica. One unit of GPU corresponds to one virtual GPU (default: 0)
  mem: <string>  # memory request per replica. One unit of memory is one byte and can be expressed as an integer or by using one of these suffixes: K, M, G, T (or their power-of two counterparts: Ki, Mi, Gi, Ti) (default: Null)
```

## Autoscaling

```yaml
autoscaling:
  min_replicas: <int>  # minimum number of replicas (default: 1)
  max_replicas: <int>  # maximum number of replicas (default: 100)
  init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
  max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
  target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: processes_per_replica * threads_per_process) (aws only)
  window: <duration>  # the time over which to average the API's concurrency (default: 60s) (aws only)
  downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m) (aws only)
  upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m) (aws only)
  max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75) (aws only)
  max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5) (aws only)
  downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05) (aws only)
  upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05) (aws only)
```

## Update strategy

```yaml
update_strategy:
  max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%) (set to 0 to disable rolling updates)
  max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

## Networking

```yaml
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
```
