# Deploy models as Batch APIs

_WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`_

This example shows how to deploy a batch image classification api that accepts a list of image urls as input, downloads the images, classifies them and writes the results to S3.

<br>

## Implement your predictor

1. Create a Python file `predictor.py`.
2. Define a Predictor class with a constructor that loads and initializes your model.
3. Add a predict function that will accept a list of images urls, performs inference and writes the prediction to S3.

```python
# predictor.py

import os
import boto3
from botocore import UNSIGNED
from botocore.client import Config
import pickle

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config, job_spec):
        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {device}")

        model = torchvision.models.alexnet(pretrained=True).to(device)
        model.eval()
        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])

        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )
        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]
        self.model = model
        self.device = device

        self.s3 = boto3.client("s3")
        self.job_id = job_spec["job_id"]
        self.bucket = config["bucket"]
        self.prefix = os.path.join(config["prefix"], self.job_id)


    def predict(self, payload, batch_id):
        tensor_list = []

        for image_url in payload:
            image = requests.get(image_url).content
            img_pil = Image.open(BytesIO(image))
            tensor_list.append(self.preprocess(img_pil))
            img_tensor = torch.stack(tensor_list)
            img_tensor = img_tensor.to(self.device)
            with torch.no_grad():
                prediction = self.model(img_tensor)
            _, indices = prediction.max(1)

        results = []
        for index in indices:
            results.append(self.labels[index])

        self.s3.put_object( # write the predictions to S3
            Bucket=self.bucket,
            Key=os.path.join(self.prefix, batch_id + ".json"),
            Body=str(json.dumps(results)),
        )
```

Here are the complete [Predictor docs](../../../docs/deployments/predictors). // TODO

<br>

## Specify your Python dependencies

Create a `requirements.txt` file to specify the dependencies needed by `predictor.py`. Cortex will automatically install them into your runtime once you deploy:

```python
# requirements.txt

boto3
```

You can skip dependencies that are [pre-installed](../../../docs/deployments/predictors) to speed up the deployment process. Note that `pickle` is part of the Python standard library so it doesn't need to be included.

<br>

## Configure your API

Create a `cortex.yaml` file and add the configuration below and replace `cortex-examples` with your S3 bucket. An `api` provides a runtime for inference and makes your `predictor.py` implementation available as a web service that can serve real-time predictions:

```yaml
# cortex.yaml

- name: image-classifier
  kind: BatchAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    cpu: 1
    mem: 1G
```

Here are the complete [API configuration docs](../../../docs/deployments/api-configuration.md). // TODO

<br>

## Deploy your Batch API

`cortex deploy` takes your model along with the configuration from `cortex.yaml` and creates a web API:

```bash
$ cortex deploy

created image-classifier
```

Get the endpoint for your Batch API with `cortex get`:

```bash
$ cortex get image-classifier

no submitted jobs

endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier
```

## Submit a Job

Before we submit a job, we need to get an S3 bucket for the workers to upload their inferences. The bucket should be accessible by the Cortex cluster.

If you don't already have an S3 bucket you can create a bucket in the (AWS Web console](https://docs.aws.amazon.com/AmazonS3/latest/user-guide/create-bucket.html) or use the following command if you have aws CLI installed and the credentials you've used : `aws s3api create-bucket --bucket my-bucket --region us-east-1`.

```bash
$ curl https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier \
-X POST -H "Content-Type: application/json" \
-d @- <<BODY
{
    "workers": 1,
    "items": [
        "https://i.imgur.com/PzXprwl.jpg",
        "https://i.imgur.com/E4cOSLw.jpg"
    ]
}
BODY
```

curl http://localhost:8888/batch/image-classifier \
-X POST -H "Content-Type: application/json" \
-d @- <<BODY
{
    "workers": 1,
    "items": [
        "https://i.imgur.com/PzXprwl.jpg",
        "https://i.imgur.com/E4cOSLw.jpg"
    ]
}
BODY

You should get a response like this:

```json
{
    "job_id":"a09qi234akjfs",
    "api_name":"image-classifier",
    "workers":1,
    "sqs_url":"https://sqs.us-west-2.amazonaws.com/123456789/123456789-image-classifier-a09qi234akjfs.fifo"
}
```

Take note of the job id returned.

## List submitted jobs
```bash
$ cortex get image-classifier

job id          status    progress   start time                 duration
a09qi234akjfs   running   2/2        20 Jul 2020 01:07:44 UTC   3m26s

endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier
```

## Get Job Status

```bash
$ cortex get image-classifier a09qi234akjfs

job id: a09qi234akjfs
status: running

start time: 17 Jul 2020 04:13:16 UTC
end time:   -
duration:   24s

batch stats
total   succeeded   failed   avg time per batch
2       0           0        -


job endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier/a09qi234akjfs
```

You can use `curl` to get batch status by making a GET request to `job endpoint`

```bash
$ curl https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier/a09qi234akjfs

{
    "job_status":{
        "job_id":"a09qi234akjfs",
        "api_name":"image-classifier",
        "workers":1,
        "total_batch_count":2,
        "status":"status_running",
        "queue_metrics":null,
        "batch_metrics":{
            "succeeded":0,
            "failed":0,
            "average_time_per_batch":null,
        },
        "created_time":"2020-07-17T04:13:16.296128812Z",
        "start_time":"2020-07-17T04:13:16.296128812Z",
    },
    "base_url":"https://abcdefg.execute-api.us-west-2.amazonaws.com"
}
```

## Get the logs for the Job

```bash
$ cortex logs image-classifier a09qi234akjfs

started enqueuing batches to queue
partitioning 2 items found in job submission into 2 batches of size 1
completed enqueuing a total 2 batches
spinning up workers...
2020-07-18 21:35:34.186348:cortex:pid-1:INFO:downloading the project code
...
```

## Cleanup

Run `cortex delete` to delete each API:

```bash
$ cortex delete --env aws image-classifier

deleting image-classifier
```

Running `cortex delete` will stop in progress jobs for the API and delete job history for that API. It will not spin down your cluster.
