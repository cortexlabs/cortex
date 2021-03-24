## gRPC client

#### Step 1

```bash
pip install grpc
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. iris_classifier.proto
```

#### Step 2

```python
import cortex
import grpcio
import iris_classifier_pb2

sample = iris_classifier_pb2.Sample()
with open("sample.bin", "rb") as f:
    serialized_sample = f.read()
sample.ParseFromString(serialized_sample)

cx = cortex.client("aws")
grpc_endpoint = cx.get_api("iris-classifier")["endpoint"]
channel = grpc.insecure_channel(grpc_endpoint)
stub = iris_classifier_pb2.PredictorStub(channel)

response = stub.Predict(sample)
print("prediction:", response.classification)
```

