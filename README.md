<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Deploy machine learning in production

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->

We are building Cortex to help machine learning engineers and data scientists deploy, manage, and scale machine learning in production. Organizations worldwide use Cortex as an alternative to building in-house machine learning infrastructure. You can [get started](https://docs.cortex.dev/install) or learn more by checking out our [docs](https://docs.cortex.dev) and [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples). If you have any questions or feedback, we'd love to hear from you on our [Gitter](https://gitter.im/cortexlabs/cortex)!

<br>

```bash
$ cortex deploy --env aws

creating text-generator

$ curl http://localhost:8888 \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "deploy machine learning in"}'

"deploy machine learning in production"
```

<br>

## `cortex cluster up`

### Spin up a Cortex cluster on your AWS account

```yaml
# cluster.yaml

cluster_name: cortex
region: us-east-1
```

### Run inference on EC2's machine learning instances

```yaml
# cluster.yaml

instance_type: g4dn.xlarge
min_instances: 10
max_instances: 100
```

### Save money by configuring spot instances

```yaml
# cluster.yaml

spot: true

spot_config:
  max_price: 1.0
  on_demand_backup: true
```

<br>

## `cortex deploy`

### Import your TensorFlow, PyTorch, ONNX, and other models

```python
# predictor.py

import torch

from transformers import GPT2Tokenizer, GPT2LMHeadModel

class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload):
        input_length = len(payload["text"].split())
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + 20, do_sample=True)
        return self.tokenizer.decode(prediction[0])
```

### Deploy realtime or batch APIs

```yaml
# cortex.yaml

name: text-generator
kind: RealtimeAPI
predictor:
  path: predictor.py
```

### Scale to handle production traffic with request-based autoscaling

```yaml
# cortex.yaml

autoscaling:
  min_replicas: 1
  max_replicas: 10
  target_replica_concurrency: 10
```

### Manage compute resources depending on your inference workloads

```yaml
# cortex.yaml

compute:
  cpu: 1
  gpu: 1
  mem: 4Gi
```

### Customize networking for internal or external APIs

```yaml
# cortex.yaml

networking:
  endpoint: my-api
  api_gateway: public
```

### Manage dependencies for reproducible deployments

```txt
# requirements.txt

numpy
boto3
requests

...
```

### Configure traffic splitting for A/B tests

```yaml
# cortex.yaml

name: text-generator
kind: TrafficSplitter
apis:
  - name: tensorflow-text-generator
    weight: 80
  - name: pytorch-text-generator
    weight: 20
```

### Update APIs with no downtime

```yaml
# cortex.yaml

update_strategy:
  max_surge: 25%
  max_unavailable: 25%
```

### Test and iterate on APIs locally before deploying at scale

```bash
$ cortex deploy --env local

creating text-generator (local)

$ cortex deploy --env aws

creating text-generator (aws)
```

<br>

## `cortex get`

### See all of your APIs in one place

```bash
env     api                status     replicas   last update
local   text-generator     updating   1          5s
aws     text-generator     live       10         1h
aws     image-classifier   live       10         2h
aws     object-detector    live       10         3h
```

### Monitor predictions, requests, and latency

```bash
$ cortex get text-generator --env aws

status   replicas   last update   latency   2XX       5XX
live     10         10m           100ms     1000000   10
```

<br>

## Get started

<!-- CORTEX_VERSION_README_MINOR -->
```bash
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.20/get-cli.sh)"
```
