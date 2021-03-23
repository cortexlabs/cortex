# Configuration

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```yaml
- name: <string>  # API name (required)
  kind: TaskAPI
  definition:
    path: <string>  # path to a python file with a Task class definition, relative to the Cortex root (required)
    config: <string: value>  # arbitrary dictionary passed to the callable method of the Task class (can be overridden by config passed in job submission) (optional)
    dependencies: # (optional)
      pip: <string>  # relative path to requirements.txt (default: requirements.txt)
      conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
      shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Task (default: quay.io/cortexlabs/python-predictor-cpu:0.31.1, quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.2-cudnn8, or quay.io/cortexlabs/python-predictor-inf:0.31.1 based on compute)
    env: <string: string>  # dictionary of environment variables
    log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
  compute:
    cpu: <string | int | float>  # CPU request per worker. One unit of CPU corresponds to one virtual CPU; fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (default: 200m)
    gpu: <int>  # GPU request per worker. One unit of GPU corresponds to one virtual GPU (default: 0)
    inf: <int> # Inferentia request per worker. One unit corresponds to one Inferentia ASIC with 4 NeuronCores and 8GB of cache memory. Each process will have one NeuronCore Group with (4 * inf / processes_per_replica) NeuronCores, so your model should be compiled to run on (4 * inf / processes_per_replica) NeuronCores. (default: 0) (aws only)
    mem: <string>  # memory request per worker. One unit of memory is one byte and can be expressed as an integer or by using one of these suffixes: K, M, G, T (or their power-of two counterparts: Ki, Mi, Gi, Ti) (default: Null)
```
