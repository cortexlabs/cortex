# Running an example

## CPU/GPU

Get the image classifier endpoint:

```bash
cortex get image-classifier-resnet50
```

```bash
python client.py http://<lb-id>.elb.<cluster-region>.amazonaws.com/image-classifier-resnet50
```

Or alternatively:

```bash
curl "http://<lb-id>.elb.<cluster-region>.amazonaws.com/image-classifier-resnet50/v1/models/resnet50:predict" -X POST -H "Content-type: application/json" -d @sample.json
```

## Inferentia

### HTTP

Get the image classifier endpoint:

```bash
cortex get image-classifier-resnet50
```

```bash
python client_inf.py http://<lb-id>.elb.<cluster-region>.amazonaws.com/image-classifier-resnet50
```

### gRPC

The Inferentia examples were inspired by [this tutorial](https://awsdocs-neuron.readthedocs-hosted.com/en/latest/neuron-deploy/tutorials/k8s_rn50_demo.html).

This guide shows how to exec into a pod, check the Neuron runtime, and make inferences using gRPC for testing purposes (exposing gRPC endpoints outside of the cluster is not currently supported by Cortex, but is on the roadmap).

#### Making an inference

Exec into the TensorFlow Serving pod:

```bash
kubectl exec api-image-classifier-resnet50-5b6df59b9b-p4s2h -c api -it -- /bin/bash
```

Create this `tensorflow-model-server-infer.py` with this contents (e.g. using `vim tensorflow-model-server-infer.py`):

```python
import numpy as np
import grpc
import tensorflow as tf
from tensorflow.keras.preprocessing import image
from tensorflow.keras.applications.resnet50 import preprocess_input
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from tensorflow.keras.applications.resnet50 import decode_predictions

if __name__ == '__main__':
    channel = grpc.insecure_channel('localhost:8500')
    stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)
    img_file = tf.keras.utils.get_file(
        "./kitten_small.jpg",
        "https://raw.githubusercontent.com/awslabs/mxnet-model-server/master/docs/images/kitten_small.jpg")
    img = image.load_img(img_file, target_size=(224, 224))
    img_array = preprocess_input(image.img_to_array(img)[None, ...])
    request = predict_pb2.PredictRequest()
    request.model_spec.name = 'resnet50_neuron'
    request.inputs['input'].CopyFrom(
        tf.make_tensor_proto(img_array, shape=img_array.shape))
    result = stub.Predict(request)
    prediction = tf.make_ndarray(result.outputs['output'])
    print(decode_predictions(prediction))
```

Run an inference:

```bash
python tensorflow-model-server-infer.py
```

#### Inspecting the neuron runtime

Exec into the TensorFlow Serving pod (the `rtd` pod will also work for the example which uses the neuron-rtd sidecar):

```bash
kubectl exec api-image-classifier-resnet50-5b6df59b9b-p4s2h -c api -it -- /bin/bash
```

Install dependencies:

```bash
apt-get update && apt-get install -y aws-neuron-dkms aws-neuron-runtime-base aws-neuron-runtime aws-neuron-tools
PATH="/opt/aws/neuron/bin:${PATH}"
```

Run CLI commands (described [here](https://awsdocs-neuron.readthedocs-hosted.com/en/latest/neuron-guide/neuron-tools/basic.html)):

```bash
neuron-ls
neuron-cli list-model
neuron-top
```
