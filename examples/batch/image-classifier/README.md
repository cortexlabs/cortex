# Deploy models as Batch APIs

_WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`_

This example shows how to deploy a batch image classification api that accepts a list of image urls as input, downloads the images, classifies them and writes the results to S3.

<br>

## Implement your predictor

1. Create a Python file `predictor.py`.
1. Define a Predictor class with a constructor that loads and initializes a model.
1. Add a predict function that will accept a list of images urls, downloads them, performs inference and writes the prediction to S3.

```python
# predictor.py

import os
import requests
import torch
import torchvision
from torchvision import transforms
from PIL import Image
from io import BytesIO
import boto3
import json
import os

class PythonPredictor:
    def __init__(self, config, job_spec):
        model = torchvision.models.alexnet(pretrained=True).eval()
        self.model = model

        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        self.s3 = boto3.client("s3")
        self.bucket = config["bucket"]
        self.key = os.path.join(config["key"], job_spec["job_id"])


    def predict(self, payload, batch_id):
        tensor_list = []

        for image_url in payload:
            image = requests.get(image_url).content
            img_pil = Image.open(BytesIO(image))

            tensor_list.append(self.preprocess(img_pil))
            img_tensor = torch.stack(tensor_list)
            with torch.no_grad():
                prediction = self.model(img_tensor)
            _, indices = prediction.max(1)

        results = []
        for index in indices:
            results.append(self.labels[index])

        self.s3.put_object( # write the predictions to S3
            Bucket=self.bucket,
            Key=os.path.join(self.key,  batch_id + ".json"),
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
torch
torchvision
pillow
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

Here are the complete [API configuration docs](../../../docs/deployments/batchapi/api-configuration.md).

<br>

## Deploy your Batch API

`cortex deploy` takes your model along with the configuration from `cortex.yaml` and creates an endpoint that receives job submissions:

```bash
$ cortex deploy

created image-classifier
```

Get the endpoint for your Batch API with `cortex get`:

```bash
$ cortex get

env   batch api          running jobs   latest job id   last update
aws   image-classifier   0              -               4s
```

```bash
$ cortex get image-classifier

no submitted jobs

endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier
```

## Submit a Job

```bash
$ curl http://localhost:8888/batch/image-classifier \
-X POST -H "Content-Type: application/json" \
-d @- <<BODY
{
    "workers": 1,
    "item_list": {
        "items": [
            "https://i.imgur.com/PzXprwl.jpg",
            "https://i.imgur.com/E4cOSLw.jpg"
        ],
        "batch_size": 1
    }
}
BODY
```

You should get a response like this:

```json
{
    "job_id":"a09qi234akjfs",
    "api_name":"image-classifier",
    "workers":1,
    ...
}
```

Take note of the job id returned.

## List jobs for API

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

start time: 27 Jul 2020 15:02:25 UTC
end time:   -
duration:   42s

batch stats
total   succeeded   failed   avg time per batch
2       0           0        -

worker stats
requested   initializing   running   failed   succeeded
1           1              0         0        0

job endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier/69da5bd5b2f87258

results dir (if used): s3://<YOUR_BUCKET>/job_results/image-classifier/69da5bd5b2f87258
```

Take note of `results dir` and the `job endpoint`. The `job_endpoint` is a url join of your batch api endpoint and job id: `<batch_api_endpoint>/<job_id>` You can make a GET request to endpoint to get the same information:

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
completed enqueuing a total of 2 batches
spinning up workers...
2020-07-18 21:35:34.186348:cortex:pid-1:INFO:downloading the project code
...
```

## Job Results

You can use `cortex get image-classifier <job_id> -w` to monitor the job until it has completed. Once it has completed you can find the results in the S3 directory labelled `results dir`.

```
job id: a09qi234akjfs
status: succeeded

start time: 27 Jul 2020 18:52:22 UTC
end time:   27 Jul 2020 18:53:17 UTC
duration:   54s

batch stats
total   succeeded   failed   avg time per batch
2       2           0        0.461842 s

worker stats are not available because this job is not currently running

job endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier/a09qi234akjfs
results dir (if used): s3://<YOUR_BUCKET>/job_results/image-classifier/a09qi234akjfs
```

You can navigate to your S3 bucket in AWS web console or use `aws s3 ls s3://<YOUR_BUCKET>/job_results/image-classifier/a09qi234akjfs/` to list the files in the directory and view your results.

## Submit another Job

Rather than submitting the input dataset in the request payload, let's specify a list of json s3 files containing the input dataset. There is a list of newline delimited json files stored in this directory `s3://cortex-examples/image-classifier`. If you have aws cli installed feel free to list the directory `aws s3 ls s3://cortex-examples/image-classifier/`. You should find `urls_0.json` and `urls_1.json` in that directory amongst other files, each containing 8 urls.

Let's classify the images from the URLs listed in those 2 files. This time, lets break our 16 url dataset into 8 batches with each batch containing 2 images.

```bash
$ curl http://localhost:8888/batch/image-classifier \
-X POST -H "Content-Type: application/json" \
-d @- <<BODY
{
    "workers": 1,
    "delimited_files": {
        "s3_paths": ["s3://cortex-examples/image-classifier/"],
        "includes": ["s3://cortex-examples/image-classifier/urls_*.json"],
        "batch_size": 2
    }
}
BODY
```

You should get a response like this:

```json
{
    "job_id":"oetr2p24iknva",
    "api_name":"image-classifier",
    "workers":1,
    ...
}
```

## Get the logs for the Job

```bash
$ cortex logs image-classifier oetr2p24iknva

started enqueuing batches to queue
enqueuing contents from file s3://cortex-examples/image-classifier/urls_0.json
enqueuing contents from file s3://cortex-examples/image-classifier/urls_1.json
completed enqueuing a total of 8 batches
spinning up workers...
2020-07-18 21:35:34.186348:cortex:pid-1:INFO:downloading the project code
...
```

## View Job results


## Another Batch API

The previous predictor implementation assumes that images need to be downloaded from the web. What if you have an S3 directory of images. Let's create a new predictor that takes in S3 paths as input. This time, let's also implement an `on_job_complete` function to coalesce the results to 1 file instead of having many small files. The `on_job_complete` function (if specified) will be called after all of the batches have been processed and can be used to perform post-job tasks such as triggering webhooks, submitting jobs to other APIs and operating on a completed dataset.

## Implement the new predictor

1. Update the `predict` function to receive s3 paths instead of web URLs
1. Implement a `on_job_complete` function that reads the results written in S3 and combines them to output one big file.


At the end of every job you can use implement an `on_job_complete` function that will be called after all of the batches have been processed. This can be used to implement

In the previous predictor implementation, the result of each batch was written to a file on S3. If you have a large dataset, you may end up with many small files containing

```python
# predictor.py

class PythonPredictor:
    def __init__(self, config, job_spec):
        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {device}")

        model = torchvision.models.alexnet(pretrained=True).to(device).eval()
        self.model = model
        self.device = device

        # https://github.com/pytorch/examples/blob/447974f6337543d4de6b888e244a964d3c9b71f6/imagenet/main.py#L198-L199
        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        self.s3 = boto3.client("s3")
        bucket, key = re.match("s3://(.+?)/(.+)", job_spec["results_dir"]).groups()
        self.bucket = bucket
        self.key = key

    def predict(self, payload, batch_id):
        tensor_list = []

        for image_url in payload:
            bucket, key = re.match("s3://(.+?)/(.+)", image_url).groups()
            file_byte_string = self.s3.get_object(Bucket=bucket, Key=key)["Body"].read()
            img_pil = Image.open(BytesIO(file_byte_string))
            tensor_list.append(self.preprocess(img_pil))
            img_tensor = torch.stack(tensor_list)
            img_tensor = img_tensor.to(self.device)
            with torch.no_grad():
                prediction = self.model(img_tensor)
            _, indices = prediction.max(1)

        results = []
        for index in indices:
            results.append(self.labels[index])

        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, batch_id + ".json"),
            Body=str(json.dumps(results)),
        )

    def on_job_complete(self):
        s3 = boto3.client("s3")
        all_results = []

        # download all of the results
        for obj in s3.list_objects_v2(Bucket=self.bucket, Prefix=self.key).Contents:
            if obj["Size"] > 0:
                body = s3.get_object(Bucket=self.bucket, Prefix=self.key)["Body"]
                all_results.append(json.loads(body.read().decode("utf8")))


        newline_delimited_json = "\n".join(json.dumps(result) for result in all_results)
        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, "aggregated_results" + ".csv"),
            Body=str(newline_delimited_json),
        )
```

## Submit a Job

Let us classify the 16 images can be found here `s3://cortex-examples/image-classifier/samples`. If you have aws cli installed you can use `aws s3 ls s3://cortex-examples/image-classifier/samples/` to view the list of images.

```bash
$ curl http://localhost:8888/batch/image-classifier \
-X POST -H "Content-Type: application/json" \
-d @- <<BODY
{
    "workers": 1,
    "file_path_lister": {
        "s3_files": ["s3://cortex-examples/image-classifier/samples"],
        "includes": ["**.jpg"]
        "batch_size": 1
    }
}
BODY
```



## Cleanup

Run `cortex delete` to delete each API:

```bash
$ cortex delete --env aws image-classifier

deleting image-classifier
```

Running `cortex delete` will stop in progress jobs for the API and delete job history for that API. It will not spin down your cluster.
