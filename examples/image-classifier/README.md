# Deploy Image Classification as an API

This example shows how to deploy an Image Classifier made with Pytorch. The Pytorch Image Classifier implementation will be using a pretrained Alexnet model from Torchvision that has been exported to ONNX format.

## Define a deployment

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes a model available as a web service that can serve real-time predictions. This configuration will download the model from the `cortex-examples` S3 bucket, preprocess the request payload and postprocess the model inference with the functions defined in `alexnet_handler.py`.

```yaml
- kind: deployment
  name: image

- kind: api
  name: classifier
  model: s3://cortex-examples/image-classifier/alexnet.onnx
  request_handler: alexnet_handler.py
  tracker:
    model_type: classification
```

<!-- CORTEX_VERSION_MINOR x2 -->
You can run the code that generated the exported models used in this example folder here:
- [Pytorch Alexnet](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.9/examples/image-classifier/alexnet.ipynb)
- [TensorFlow Inception V3](https://colab.research.google.com/github/cortexlabs/cortex/blob/0.9/examples/image-classifier/inception.ipynb)


## Add request handling

The Alexnet model requires a 2 dimensional array of 3 valued tuples representing the RGB values for each pixel in the image, but the API should accept a simple input format such as a URL to an image. Instead of returning the model's output consisting of an array of probabilities, the API should return the class name with the highest probability. Define a `pre_inference` function to download the image from the specified URL and convert it to the expected model input and a `post_inference` function to return the name of the class with the highest probability:

```python
import requests
import numpy as np
import base64
from PIL import Image
from io import BytesIO
from torchvision import transforms

labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]


# https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
preprocess = transforms.Compose(
    [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
)


def pre_inference(sample, metadata):
    if "url" in sample:
        image = requests.get(sample["url"]).content
    elif "base64" in sample:
        image = base64.b64decode(sample["base64"])

    img_pil = Image.open(BytesIO(image))
    img_tensor = preprocess(img_pil)
    img_tensor.unsqueeze_(0)
    return img_tensor.numpy()


def post_inference(prediction, metadata):
    return labels[np.argmax(np.array(prediction).squeeze())]
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the models, makes them servable using ONNX Runtime, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the statuses of the APIs using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           12s           -
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get classifier

url: http://***.amazonaws.com/image/classifier

$ curl http://***.amazonaws.com/image/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"url": "https://bowwowinsurance.com.au/wp-content/uploads/2018/10/akita-700x700.jpg"}'

"Eskimo dog"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
