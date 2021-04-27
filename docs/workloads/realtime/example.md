# RealtimeAPI

## HTTP

Create HTTP APIs that respond to requests in real-time.

### Implement

```bash
mkdir text-generator && cd text-generator
touch handler.py requirements.txt text_generator.yaml
```

```python
# handler.py

from transformers import pipeline

class Handler:
    def __init__(self, config):
        self.model = pipeline(task="text-generation")

    def handle_post(self, payload):
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
  handler:
    type: python
    path: handler.py
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

Create a `handler.proto` file in your project's directory:

```protobuf
<!-- handler.proto -->

syntax = "proto3";
package text_generator;

service Handler {
    rpc Predict (Message) returns (Message);
}

message Message {
    string text = 1;
}
```

Set the `handler.protobuf_path` field in the API spec to point to the `handler.proto` file:

```yaml
# text_generator.yaml

- name: text-generator
  kind: RealtimeAPI
  handler:
    type: python
    path: handler.py
    protobuf_path: handler.proto
  compute:
    gpu: 1
```

### Match RPC service name

Match the name of the RPC service(s) from the protobuf definition (in this case `Predict`) with what you're defining in the handler's implementation:

```python
# handler.py

from transformers import pipeline

class Handler:
    def __init__(self, config, proto_module_pb2):
        self.model = pipeline(task="text-generation")
        self.proto_module_pb2 = proto_module_pb2

    def Predict(self, payload):
        return self.proto_module_pb2.Message(text="returned message")
```

### Make a gRPC request

```bash
grpcurl -plaintext -proto handler.proto -d '{"text": "hello-world"}' ***.elb.us-west-2.amazonaws.com:80 text_generator.Handler/Predict
```
