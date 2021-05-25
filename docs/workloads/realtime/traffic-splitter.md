# Traffic Splitter

Traffic Splitters can be used to expose multiple RealtimeAPIs as a single endpoint for A/B tests, multi-armed bandits, or canary deployments.

## Configuration

```yaml
- name: <string>  # name of the traffic splitter (required)
  kind: TrafficSplitter  # must be "TrafficSplitter" for traffic splitters (required)
  networking:  # networking configuration (default: see below)
    endpoint: <string>  # the endpoint for the traffic splitter (default: <name>)
  apis:  # list of Realtime APIs to target (required)
    - name: <string>  # name of a Realtime API that is already running or is included in the same configuration file (required)
      weight: <int>   # percentage of traffic to route to the Realtime API (all non-shadow weights must sum to 100) (required)
      shadow: <bool>  # duplicate incoming traffic and send fire-and-forget to this api (only one shadow per traffic splitter) (default: false)
```

## Example

This example showcases Cortex's Python client, but these steps can also be performed by using the Cortex CLI.

### Deploy a traffic splitter

```python
traffic_splitter_spec = {
    "name": "sentiment-analyzer",
    "kind": "TrafficSplitter",
    "apis": [
        {"name": "sentiment-analyzer-a", "weight": 50},
        {"name": "sentiment-analyzer-b", "weight": 50},
    ],
}

cx.deploy(traffic_splitter_spec)
```

### Update the weights

```python
new_traffic_splitter_spec = {
    "name": "sentiment-analyzer",
    "kind": "TrafficSplitter",
    "apis": [
        {"name": "sentiment-analyzer-a", "weight": 1},
        {"name": "sentiment-analyzer-b", "weight": 99},
    ],
}

cx.deploy(new_traffic_splitter_spec)
```

### Update the target APIs

```python
new_traffic_splitter_spec = {
    "name": "sentiment-analyzer",
    "kind": "TrafficSplitter",
    "apis": [
        {"name": "sentiment-analyzer-b", "weight": 50},
        {"name": "sentiment-analyzer-c", "weight": 50},
    ],
}

cx.deploy(new_traffic_splitter_spec)
```
