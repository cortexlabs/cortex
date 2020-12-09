# Traffic splitter

A Traffic Splitter can be used expose multiple APIs as a single endpoint. The percentage of traffic routed to each API can be controlled. This can be useful when performing A/B tests, setting up multi-armed bandits or performing canary deployments.

**Note: Traffic Splitter is only supported on a Cortex cluster**

## Deploy APIs

```python
import cortex

class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline
        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]

requirements = ["tensorflow", "transformers"]

api_spec_cpu = {
    "name": "text-generator-cpu",
    "kind": "RealtimeAPI",
    "compute": {
        "cpu": 1,
    },
}

api_spec_gpu = {
    "name": "text-generator-gpu",
    "kind": "RealtimeAPI",
    "compute": {
        "gpu": 1,
    },
}

cx = cortex.client("aws")
cx.create_api(api_spec_cpu, predictor=PythonPredictor, requirements=requirements)
cx.create_api(api_spec_gpu, predictor=PythonPredictor, requirements=requirements)
```

## Deploy a traffic splitter

```python
traffic_splitter_spec = {
    "name": "text-generator",
    "kind": "TrafficSplitter",
    "apis": [
        {"name": "text-generator-cpu", "weight": 50},
        {"name": "text-generator-gpu", "weight": 50},
    ],
}

cx.create_api(traffic_splitter_spec)
```

## Update the weights of the traffic splitter

```python
traffic_splitter_spec = cx.get_api("text-generator")["spec"]["submitted_api_spec"]

# send 99% of the traffic to text-generator-gpu
traffic_splitter_spec["apis"][0]["weight"] = 1
traffic_splitter_spec["apis"][1]["weight"] = 99

cx.patch(traffic_splitter_spec)
```
