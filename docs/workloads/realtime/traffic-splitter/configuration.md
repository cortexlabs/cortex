# Configuration

```yaml
- name: <string>  # Traffic Splitter name (required)
  kind: TrafficSplitter
  networking:
    endpoint: <string>  # the endpoint for the Traffic Splitter (default: <name>)
  apis:  # list of Realtime APIs to target
    - name: <string>  # name of a Realtime API that is already running or is included in the same configuration file (required)
      weight: <int>   # percentage of traffic to route to the Realtime API (all weights must sum to 100) (required)
```
