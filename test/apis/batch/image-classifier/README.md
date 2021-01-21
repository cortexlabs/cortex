# Deploy models as Batch APIs

This example shows how to deploy a batch image classification api that accepts a list of image urls as input, downloads the images, classifies them, and writes the results to S3.

**Batch APIs are only supported in AWS.** You can find cluster installation documentation [here](../../../docs/aws/install.md).

## Pre-requisites

* [Install](../../../docs/aws/install.md) Cortex and create a cluster
* Create an S3 bucket/directory to store the results of the batch job
* AWS CLI (optional)

<br>

## Implement your predictor

1. Create a Python file named `predictor.py`.
1. Define a Predictor class with a constructor that loads and initializes an image-classifier from `torchvision`.
1. Add a `predict()` function that will accept a list of images urls (http:// or s3://), downloads them, performs inference, and writes the predictions to S3.
1. Specify an `on_job_complete()` function that aggregates the results and writes them to a single file named `aggregated_results.json` in S3.

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
import re


class PythonPredictor:
    def __init__(self, config, job_spec):
        self.model = torchvision.models.alexnet(pretrained=True).eval()

        normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        self.preprocess = transforms.Compose(
            [transforms.Resize(256), transforms.CenterCrop(224), transforms.ToTensor(), normalize]
        )

        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        if len(config.get("dest_s3_dir", "")) == 0:
            raise Exception("'dest_s3_dir' field was not provided in job submission")

        self.s3 = boto3.client("s3")

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        tensor_list = []

        # download and preprocess each image
        for image_url in payload:
            if image_url.startswith("s3://"):
                bucket, image_key = re.match("s3://(.+?)/(.+)", image_url).groups()
                image_bytes = self.s3.get_object(Bucket=bucket, Key=image_key)["Body"].read()
            else:
                image_bytes = requests.get(image_url).content

            img_pil = Image.open(BytesIO(image_bytes))
            tensor_list.append(self.preprocess(img_pil))

        # classify the batch of images
        img_tensor = torch.stack(tensor_list)
        with torch.no_grad():
            prediction = self.model(img_tensor)
        _, indices = prediction.max(1)

        # extract predicted classes
        results = [
            {"url": payload[i], "class": self.labels[class_idx]}
            for i, class_idx in enumerate(indices)
        ]
        json_output = json.dumps(results)

        # save results
        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json_output)

    def on_job_complete(self):
        all_results = []

        # aggregate all classifications
        paginator = self.s3.get_paginator("list_objects_v2")
        for page in paginator.paginate(Bucket=self.bucket, Prefix=self.key):
            for obj in page["Contents"]:
                body = self.s3.get_object(Bucket=self.bucket, Key=obj["Key"])["Body"]
                all_results += json.loads(body.read().decode("utf8"))

        # save single file containing aggregated classifications
        self.s3.put_object(
            Bucket=self.bucket,
            Key=os.path.join(self.key, "aggregated_results.json"),
            Body=json.dumps(all_results),
        )
```

Here are the complete [Predictor docs](../../../docs/workloads/batch/predictors.md).

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

<br>

## Configure your API

Create a `cortex.yaml` file and add the configuration below. An `api` with `kind: BatchAPI` will expose your model as an endpoint that will orchestrate offline batch inference across multiple workers upon receiving job requests. The configuration below defines how much `compute` each worker requires and your `predictor.py` determines how each batch should be processed.

```yaml
# cortex.yaml

- name: image-classifier
  kind: BatchAPI
  predictor:
    type: python
    path: predictor.py
  compute:
    cpu: 1
```

Here are the complete [API configuration docs](../../../docs/workloads/batch/configuration.md).

<br>

## Deploy your Batch API

`cortex deploy` takes your model, your `predictor.py` implementation, and your configuration from `cortex.yaml` and creates an endpoint that can receive job submissions and manage running jobs.

```bash
$ cortex deploy --env aws

created image-classifier (BatchAPI)
```

Get the endpoint for your Batch API with `cortex get image-classifier`:

```bash
$ cortex get image-classifier --env aws

no submitted jobs

endpoint: http://***.elb.us-west-2.amazonaws.com/image-classifier
```

<br>

## Setup destination S3 directory

Our `predictor.py` implementation writes results to an S3 directory. Before submitting a job, we need to create an S3 directory to store the output of the batch job. The S3 directory should be accessible by the credentials used to create your Cortex cluster.

Export the S3 directory to an environment variable:

```bash
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
```

<br>

## Submit a job

Now that you've deployed a Batch API, you are ready to submit jobs. You can provide image urls directly in the request by specifying the urls in `item_list`. The curl command below showcases how to submit image urls in the request.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT \
    -X POST -H "Content-Type: application/json" \
    -d @- <<EOF
    {
        "workers": 1,
        "item_list": {
            "items": [
                "https://i.imgur.com/PzXprwl.jpg",
                "https://i.imgur.com/E4cOSLw.jpg",
                "https://user-images.githubusercontent.com/4365343/96516272-d40aa980-1234-11eb-949d-8e7e739b8345.jpg",
                "https://i.imgur.com/jDimNTZ.jpg",
                "https://i.imgur.com/WqeovVj.jpg"
            ],
            "batch_size": 2
        },
        "config": {
            "dest_s3_dir": "$CORTEX_DEST_S3_DIR"
        }
    }
EOF
```

Note: if you are prompted with `>` then type `EOF`.

After submitting the job, you should get a response like this:

```json
{"job_id":"69d6faf82e4660d3","api_name":"image-classifier", "config":{"dest_s3_dir": "YOUR_S3_BUCKET_HERE"}}
```

Take note of the job id in the response.

### List the jobs for your Batch API

```bash
$ cortex get image-classifier --env aws

job id             status    progress   start time                 duration
69d6faf82e4660d3   running   0/3        20 Jul 2020 01:07:44 UTC   3m26s

endpoint: http://***.elb.us-west-2.amazonaws.com/image-classifier
```

### Get the job status with an HTTP request

You can make a GET request to your `<BATCH_API_ENDPOINT>?JOB_ID` to get the status of your job.

```bash
$ curl http://***.elb.us-west-2.amazonaws.com/image-classifier?jobID=69d6faf82e4660d3

{
    "job_status":{
        "job_id":"69d6faf82e4660d3",
        "api_name":"image-classifier",
        ...
    },
    "endpoint":"https://***.elb.us-west-2.amazonaws.com/image-classifier"
}
```

### Get job status using Cortex CLI

You can also use the Cortex CLI to get the status of your job using `cortex get <BATCH_API_NAME> <JOB_ID>`.

```bash
$ cortex get image-classifier 69d6faf82e4660d3 --env aws

job id: 69d6faf82e4660d3
status: running

start time: 27 Jul 2020 15:02:25 UTC
end time:   -
duration:   42s

batch stats
total   succeeded   failed   avg time per batch
3       0           0        -

worker stats
requested   initializing   running   failed   succeeded
1           1              0         0        0

job endpoint: http://***.elb.us-west-2.amazonaws.com/image-classifier/69d6faf82e4660d3
```

### Stream logs

You can stream logs realtime for debugging and monitoring purposes with `cortex logs <BATCH_API_NAME> <JOB_ID>`

```bash
$ cortex logs image-classifier 69d6fdeb2d8e6647 --env aws

started enqueuing batches to queue
partitioning 5 items found in job submission into 3 batches of size 2
completed enqueuing a total of 3 batches
spinning up workers...
...
2020-08-07 14:44:05.557598:cortex:pid-25:INFO:processing batch c9136381-6dcc-45bd-bd97-cc9c66ccc6d6
2020-08-07 14:44:26.037276:cortex:pid-25:INFO:executing on_job_complete
2020-08-07 14:44:26.208972:cortex:pid-25:INFO:no batches left in queue, job has been completed
```

### Find your results

Wait for the job to complete by streaming the logs with `cortex logs <BATCH_API_NAME> <JOB_ID>` or watching for the job status to change with `cortex get <BATCH_API_NAME> <JOB_ID> --watch`.

The status of your job, which you can get from `cortex get <BATCH_API_NAME> <JOB_ID>`, should change from `running` to `succeeded` once the job has completed. If it changes to a different status, you may be able to find the stacktrace using `cortex logs <BATCH_API_NAME> <JOB_ID>`. If your job has completed successfully, you can view the results of the image classification in the S3 directory you specified in the job submission.

Using the AWS CLI:

```bash
$ aws s3 ls $CORTEX_DEST_S3_DIR/<JOB_ID>/
  161f9fda-fd08-44f3-b983-4529f950e40b.json
  40100ffb-6824-4560-8ca4-7c0d14273e05.json
  c9136381-6dcc-45bd-bd97-cc9c66ccc6d6.json
  aggregated_results.json
```

You can download the aggregated results file with `aws s3 cp $CORTEX_DEST_S3_DIR/<JOB_ID>/aggregated_results.json .` and confirm that there are 16 classifications.

<br>

## Alternative job submission: image URLs in files

In addition to providing the image URLs directly in the job submission request, it is possible to use image urls stored in newline delimited json files in S3. A newline delimited JSON file has one complete JSON object per line.

Two newline delimited json files containing image urls for this tutorial have already been created for you and can be found at `s3://cortex-examples/image-classifier/`. If you have AWS CLI, you can list the directory and you should be able to find the files (`urls_0.json` and `urls_1.json`).

```text
$ aws s3 ls s3://cortex-examples/image-classifier/
                           PRE inception/
...
2020-07-27 14:19:30        506 urls_0.json
2020-07-27 14:19:30        473 urls_1.json
```

To use JSON files as input data for the job, we will specify `delimited_files` in the job request. The Batch API will break up the JSON files into batches of desired size and push them onto a queue that is consumed by the pool of workers.

### Dry run

Before we submit the job, let's perform a dry run to ensure that only the desired files will be read. You can perform a dry run by appending `dryRun=true` query parameter to your job request.

Get the endpoint from `cortex get image-classifier` if you haven't done so already.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT?dryRun=true \
-X POST -H "Content-Type: application/json" \
-d @- <<EOF
{
    "workers": 1,
    "delimited_files": {
        "s3_paths": ["s3://cortex-examples/image-classifier/"],
        "includes": ["**.json"],
        "batch_size": 2
    },
    "config": {
        "dest_s3_dir": "$CORTEX_DEST_S3_DIR"
    }
}
EOF
```

Note: if you are prompted with `>` then type `EOF`.

You should expect a response like this:

```text
s3://cortex-examples/image-classifier/urls_0.json
s3://cortex-examples/image-classifier/urls_1.json
validations passed
```

This shows that the correct files will be used as input for the job.

### Classify image urls stored in S3 files

When you submit a job specifying `delimited_files`, your Batch API will get all of the input S3 files based on `s3_paths` and will apply the filters specified in `includes` and `excludes`. Then your Batch API will read each file, split on the newline characters, and parse each item as a JSON object. Each item in the file is treated as a single sample and will be grouped together into batches and then placed onto a queue that is consumed by the pool of workers.

In this example `urls_0.json` and `urls_1.json` each contain 8 urls. Let's classify the images from the URLs listed in those 2 files.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT \
-X POST -H "Content-Type: application/json" \
-d @- <<EOF
{
    "workers": 1,
    "delimited_files": {
        "s3_paths": ["s3://cortex-examples/image-classifier/"],
        "includes": ["**.json"],
        "batch_size": 2
    },
    "config": {
        "dest_s3_dir": "$CORTEX_DEST_S3_DIR"
    }
}
EOF
```

Note: if you are prompted with `>` then type `EOF`.

After submitting this job, you should get a response like this:

```json
{"job_id":"69d6faf82e4660d3","api_name":"image-classifier", "config":{"dest_s3_dir": "YOUR_S3_BUCKET_HERE"}}
```

### Find results

Wait for the job to complete by streaming the logs with `cortex logs <BATCH_API_NAME> <JOB_ID>` or watching for the job status to change with `cortex get <BATCH_API_NAME> <JOB_ID> --watch`.

```bash
$ cortex logs image-classifier 69d6faf82e4660d3 --env aws

started enqueuing batches to queue
enqueuing contents from file s3://cortex-examples/image-classifier/urls_0.json
enqueuing contents from file s3://cortex-examples/image-classifier/urls_1.json
completed enqueuing a total of 8 batches
spinning up workers...
2020-08-07 15:11:21.364179:cortex:pid-25:INFO:processing batch 1de0bc65-04ea-4b9e-9e96-5a0bb52fcc37
...
2020-08-07 15:11:45.461032:cortex:pid-25:INFO:no batches left in queue, job has been completed
```

The status of your job, which you can get from `cortex get <BATCH_API_NAME> <JOB_ID>`, should change from `running` to `succeeded` once the job has completed. If it changes to a different status, you may be able to find the stacktrace using `cortex logs <BATCH_API_NAME> <JOB_ID>`. If your job has completed successfully, you can view the results of the image classification in the S3 directory you specified in the job submission.

Using the AWS CLI:

```bash
$ aws s3 ls $CORTEX_DEST_S3_DIR/<JOB_ID>/
  161f9fda-fd08-44f3-b983-4529f950e40b.json
  40100ffb-6824-4560-8ca4-7c0d14273e05.json
  6d1c933c-0ddf-4316-9956-046cd731c5ab.json
  ...
  aggregated_results.json
```

You can download the aggregated results file with `aws s3 cp $CORTEX_DEST_S3_DIR/<JOB_ID>/aggregated_results.json .` and confirm that there are 16 classifications.

<br>

## Alternative job submission: images in S3

Let's assume that rather downloading urls on the internet, you have an S3 directory containing the images. We can specify `file_path_lister` in the job request to get the list of S3 urls for the images, partition the list of S3 urls into batches, and place them on a queue that will be consumed by the workers.

We'll classify the 16 images that can be found here `s3://cortex-examples/image-classifier/samples`. You can use AWS CLI to verify that there are 16 images `aws s3 ls s3://cortex-examples/image-classifier/samples/`.

### Dry run

Let's do a dry run to make sure the correct list of images will be submitted to the job.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT?dryRun=true \
-X POST -H "Content-Type: application/json" \
-d @- <<EOF
{
    "workers": 1,
    "file_path_lister": {
        "s3_paths": ["s3://cortex-examples/image-classifier/samples"],
        "includes": ["**.jpg"],
        "batch_size": 2
    },
    "config": {
        "dest_s3_dir": "$CORTEX_DEST_S3_DIR"
    }
}
EOF
```

Note: if you are prompted with `>` then type `EOF`.

You should expect a response like this:

```text
s3://cortex-examples/image-classifier/samples/img_0.jpg
s3://cortex-examples/image-classifier/samples/img_1.jpg
...
s3://cortex-examples/image-classifier/samples/img_8.jpg
s3://cortex-examples/image-classifier/samples/img_9.jpg
validations passed
```

### Classify images in S3

Let's actually submit the job now. Your Batch API will get all of the input S3 files based on `s3_paths` and will apply the filters specified in `includes` and `excludes`.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT \
-X POST -H "Content-Type: application/json" \
-d @- <<EOF
{
    "workers": 1,
    "file_path_lister": {
        "s3_paths": ["s3://cortex-examples/image-classifier/samples"],
        "includes": ["**.jpg"],
        "batch_size": 2
    },
    "config": {
        "dest_s3_dir": "$CORTEX_DEST_S3_DIR"
    }
}
EOF
```

Note: if you are prompted with `>` then type `EOF`.

You should get a response like this:

```json
{"job_id":"69d6f8a472f0e1e5","api_name":"image-classifier", "config":{"dest_s3_dir": "YOUR_S3_BUCKET_HERE"}}
```

### Verify results

Wait for the job to complete by streaming the logs with `cortex logs <BATCH_API_NAME> <JOB_ID>` or watching for the job status to change with `cortex get <BATCH_API_NAME> <JOB_ID> --watch`.

```bash
$ cortex logs image-classifier 69d6f8a472f0e1e5 --env aws

started enqueuing batches to queue
completed enqueuing a total of 8 batches
spinning up workers...
2020-07-18 21:35:34.186348:cortex:pid-1:INFO:downloading the project code
...
2020-08-07 15:49:10.889839:cortex:pid-25:INFO:processing batch d0e695bc-a975-4115-a60f-0a55c743fc57
2020-08-07 15:49:31.188943:cortex:pid-25:INFO:executing on_job_complete
2020-08-07 15:49:31.362053:cortex:pid-25:INFO:no batches left in queue, job has been completed
```

The status of your job, which you can get from `cortex get <BATCH_API_NAME> <JOB_ID>`, should change from `running` to `succeeded` once the job has completed. If it changes to a different status, you may be able to find the stacktrace using `cortex logs <BATCH_API_NAME> <JOB_ID>`. If your job has completed successfully, you can view the results of the image classification in the S3 directory you specified in the job submission.

Using the AWS CLI:

```bash
$ aws s3 ls $CORTEX_DEST_S3_DIR/<JOB_ID>/
  6bee7412-4c16-4d9f-ab3e-e88669cf7a89.json
  3c45b4b3-953e-4226-865b-75f3961dcf95.json
  d0e695bc-a975-4115-a60f-0a55c743fc57.json
  ...
  aggregated_results.json
```

You can download the aggregated results file with `aws s3 cp $CORTEX_DEST_S3_DIR/<JOB_ID>/aggregated_results.json .` and confirm that there are 16 classifications.

<br>

## Stopping a Job

You can stop a running job by sending a DELETE request to `<BATCH_API_ENDPOINT>/<JOB_ID>`.

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT>  # e.g. export BATCH_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/image-classifier
$ curl -X DELETE $BATCH_API_ENDPOINT?jobID=69d96a01ea55da8c

stopped job 69d96a01ea55da8c
```

You can also use the Cortex CLI `cortex delete <BATCH_API_NAME> <JOB_ID>`.

```bash
$ cortex delete image-classifier 69d96a01ea55da8c --env aws

stopped job 69d96a01ea55da8c
```

<br>

## Cleanup

Run `cortex delete` to delete the API:

```bash
$ cortex delete image-classifier --env aws

deleting image-classifier
```

Running `cortex delete` will stop all in progress jobs for the API and will delete job history for that API. It will not spin down your cluster.
