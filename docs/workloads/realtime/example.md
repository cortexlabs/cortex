# RealtimeAPI

## HTTP

Create HTTP APIs that respond to prediction requests in real-time.

### Implement

```bash
mkdir text-generator && cd text-generator
touch predictor.py requirements.txt text_generator.yaml
```

```python
# predictor.py

from transformers import pipeline

class PythonPredictor:
    def __init__(self, config):
        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]
```

```python
# requirements.txt

transformers
torch
```

```yaml
# text_generator.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
```

### Deploy

```bash
cortex deploy text_generator.yaml
```

### Monitor

```bash
cortex get text-generator --watch
```

### Stream logs

```bash
cortex logs text-generator
```

### Make a request

```bash
curl http://***.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d '{"text": "hello world"}'
```

### Delete

```bash
cortex delete text-generator
```

## gRPC

To make the above API use gRPC as its protocol, make the following changes (the rest of the steps are the same):

### Add protobuf file

Create a `predictor.proto` file in your project's directory:

```protobuf
<!-- predictor.proto -->

syntax = "proto3";
package text_generator;

service Predictor {
    rpc Predict (Message) returns (Message);
}

message Message {
    string text = 1;
}
```

Set the `predictor.protobuf_path` field in the API spec to point to the `predictor.proto` file:

```yaml
# text_generator.yaml

- name: text-generator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
    protobuf_path: predictor.proto
  compute:
    gpu: 1
```

### Make a gRPC request

```bash
grpcurl -plaintext -proto predictor.proto -d '{"text": "hello-world"}' ***.elb.us-west-2.amazonaws.com:80 text_generator.Predictor/Predict
```
