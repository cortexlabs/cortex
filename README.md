<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Model serving for developers

<br>

Cortex makes it easy to deploy machine learning models in production.

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [chat with us](https://gitter.im/cortexlabs/cortex)

<br>

# What you can do with Cortex

## Write APIs in Python

Define any real-time or batch inference pipeline as simple Python APIs, regardless of framework or hardware.

```python
from transformers import pipeline

class PythonPredictor:
  def __init__(self, config):
    self.analyzer = pipeline(task="text-generation")

  def predict(self, payload):
    return self.analyzer(payload["text"])[0]
```

## Configure infrastructure in YAML

Define autoscaling behavior, monitoring, compute resources, update strategies, and more—or use Cortex's defaults.

```yaml
- name: text-generator
  predictor:
    path: predictor.py
  networking:
    api_gateway: public
  compute:
    gpu: 1
  monitoring:
    model_type: classification
  autoscaling:
    min_replicas: 3
```

## Focus on machine learning engineering, not DevOps

You don't need to bring your own cluster or containerize your models. Cortex automates your cloud infrastructure.

```bash
$ cortex cluster up

confguring networking ...
configuring autoscaling ...
configuring logging ...
configuring metrics ...

cortex is ready!
```

## Scale to handle production traffic

Handle traffic with request-based autoscaling. Optimize spend with spot instances and multi-model APIs.

```bash
$ cortex get text-generator

status   last-update   replicas   requests   latency
live     10h           10         100m       100ms
```

## Integrate with your stack

Integrate Cortex with any data science platform and CI/CD tooling, without changing your workflow.

```python
import tensorflow
import torch
import transformers
import mlflow
...
```

## Run on your AWS account

Run Cortex on your AWS account (GCP support is coming soon), maintaining control over resource utilization and data access.

```bash
region: us-west-2
instance_type: p2.xlarge
min_instances: 1
max_instances: 5
```

# Get started

<!-- CORTEX_VERSION_README_MINOR -->
```bash
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.20/get-cli.sh)"
```

<!-- CORTEX_VERSION_README_MINOR -->
See our [installation guide](https://docs.cortex.dev/install), then deploy one of our [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples) or bring your own models to build [realtime APIs](https://docs.cortex.dev/deployments/realtime-api) and [batch APIs](https://docs.cortex.dev/deployments/batch-api).
