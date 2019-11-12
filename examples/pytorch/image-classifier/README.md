# Deploy Image Classification as an API

This example shows how to deploy a pretrained image classifier from TorchVision.

## Predictor

We implement Cortex's Predictor interface to load the model and make predictions. Cortex will use this implementation to serve the model as an autoscaling API.

### Initialization

We can place our code to download and initialize the model in the body of the implementation:

```python
# predictor.py

# download the pretrained AlexNet model
model = torchvision.models.alexnet(pretrained=True)
model.eval()

# declare the necessary image preprocessing
normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
preprocess = transforms.Compose(
    [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
)

# download the labels
labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]
```

### Predict

The `predict()` function will be triggered once per request. The AlexNet model requires a 2-dimensional array of 3-valued tuples representing the RGB values for each pixel in the image, but the API should accept a simple input format such as a URL to an image. Also, instead of returning the model's output as an array of probabilities, the API should return the class name with the highest probability. We use the `predict()` function to download the image specified by the url in the request, process it, feed it to the model, convert the model output weights to a label, and return the label:

```python
# predictor.py

def predict(sample, metadata):
    image = requests.get(sample["url"]).content
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

A `deployment` specifies a set of resources that are deployed together. An `api` makes our implementation available as a web service that can serve real-time predictions. This configuration will deploy the implementation specified in `predictor.py`:

```yaml
# cortex.yaml

- kind: deployment
  name: image

- kind: api
  name: classifier
  predictor:
    path: predictor.py
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

creating classifier
```

Behind the scenes, Cortex containerizes our implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

We can track the statuses of the APIs using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           12s           -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ cortex get classifier

endpoint: http://***.amazonaws.com/image/classifier

$ curl http://***.amazonaws.com/image/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"url": "https://i.imgur.com/PzXprwl.jpg"}'

"hotdog"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
