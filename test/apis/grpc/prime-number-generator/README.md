## Prime number generator

#### Step 1

```bash
pip install grpc
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. generator.proto
```

#### Step 2

```python
import cortex
import generator_pb2
import generator_pb2_grpc

cx = cortex.client("aws")
api = cx.get_api("prime-generator")
grpc_endpoint = api["endpoint"] + ":" + str(api["grpc_ports"]["insecure"])

prime_numbers_to_generate = 5

channel = grpc.insecure_channel(grpc_endpoint)
for r in stub.Predict(generator_pb2.Input(prime_numbers_to_generate=prime_numbers_to_generate)): 
    print(r)
```
