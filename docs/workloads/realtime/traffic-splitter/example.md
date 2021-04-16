# TrafficSplitter

Expose multiple RealtimeAPIs as a single endpoint for A/B tests, multi-armed bandits, or canary deployments.

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
