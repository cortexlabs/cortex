# Deploy Image Classification as an API

This example shows how to deploy a Pretrained Image Classifier from TorchVision.

## Predictor

We implement Cortex's Python Predictor interface that describes how to load the model and make predictions using the model. Cortex will use this implementation to serve your model as an API of autoscaling replicas. We specify a `requirements.txt` to install dependencies necessary to implement the Cortex Predictor interface.

### Initialization

Cortex executes the Python implementation once per replica startup. We can place our initializations in the body of the implementation. Let us download the pretrained Alexnet model and set it to evaluation:

```python
model = torchvision.models.alexnet(pretrained=True)
model.eval()
```

We declare the necessary image preprocessing:

```python
normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
preprocess = transforms.Compose(
    [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
)
```

We download the labels:

```python
labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]
```

### Predict

The `predict` function will be triggered once per request to run the Alexnet model on the request payload and respond with a prediction. The Alexnet model requires a 2 dimensional array of 3 valued tuples representing the RGB values for each pixel in the image, but the API should accept a simple input format such as a URL to an image. Instead of returning the model's output consisting of an array of probabilities, the API should return the class name with the highest probability. We define the `predict` function to download the image specified by the url in the request, process it, feed it to the model, convert the model output weights to a label and return the label.

```python
def predict(sample, metadata):
    if "url" in sample:
        image = requests.get(sample["url"]).content
    elif "base64" in sample:
        image = base64.b64decode(sample["base64"])

    img_pil = Image.open(BytesIO(image))
    img_tensor = preprocess(img_pil)
    img_tensor.unsqueeze_(0)
    with torch.no_grad():
        prediction = model(img_tensor)
    _, index = prediction[0].max(0)
    return labels[index]
```

See [predictor.py](./predictor.py) for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes the Cortex python implementation available as a web service that can serve real-time predictions. This configuration will deploy the implementation specified in `predictor.py` and trigger the `predict` function once per request.

```yaml
- kind: deployment
  name: image

- kind: api
  name: classifier
  python:
    predictor: predictor.py
  tracker:
    model_type: classification
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the Predictor implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

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
