# RealtimeAPI - gRPC

Create APIs that respond to prediction requests in real-time using the gRPC protocol.

## Implement

```bash
mkdir text-generator && cd text-generator
touch predictor.proto predictor.py requirements.txt text_generator.yaml
```

```protobuf
syntax = "proto3";
package text_generator;

service Predictor {
    rpc Predict (Message) returns (Message);
}

message Message {
    string text = 1;
}

```

```python
# predictor.py

from transformers import pipeline

class PythonPredictor:
    def __init__(self, config, module_proto_pb2):
        self.model = pipeline(task="text-generation")
        self.module_proto_pb2 = module_proto_pb2

    def predict(self, payload):
        result = self.model(payload.text)[0]
        return self.module_proto_pb2.Message(text=result)
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
    protobuf_path: predictor.proto
  compute:
    gpu: 1
```

## Deploy

```bash
cortex deploy text_generator.yaml
```

## Monitor

```bash
cortex get text-generator --watch
```

## Stream logs

```bash
cortex logs text-generator
```

## Make a request

```bash
grpcurl -plaintext -proto predictor.proto -d '{"text": "hello-world"}' ***.elb.us-west-2.amazonaws.com:80 text_generator.Predictor/Predict
```

## Delete

```bash
cortex delete text-generator
```
