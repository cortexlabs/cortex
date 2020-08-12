Please refer to the [tutorial](https://docs.cortex.dev/batchapi/image-classifier) to see how to deploy a Batch API example with Cortex.

# Batch Image Classifier in TensorFlow

You can deploy this example by cloning this repo with `git clone -b <TAG> https://github.com/cortexlabs/cortex`. The tag should be substituted with the minor version of your Cortex CLI `cortex version`. For example, if the output of `cortex version` is 0.18.1, the clone command would be `git clone -b 0.18 https://github.com/cortexlabs/cortex`.

Navigate to this example directory and deploy this example with `cortex deploy`.

```bash
$ cortex deploy --env aws

created image-classifier
```

## Find your Batch API Endpoint

You can get the endpoint for your Batch API using the command `cortex get <API_NAME>`

```bash
$ cortex get image-classifier --env aws

no submitted jobs

endpoint: https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier
```

Note the Batch API endpoint.

## Setup destination S3 directory

The `predictor.py` implementation writes results to an S3 directory. Before submitting a job, we need to create or provide an S3 directory to store the output of the batch job. The S3 directory should be accessible by the credentials used to create your Cortex cluster.

Export the S3 directory to an environment variable:

```bash
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY>
```

## Submit a job

```bash
$ export BATCH_API_ENDPOINT=<BATCH_API_ENDPOINT> # e.g. export BATCH_API_ENDPOINT=https://abcdefg.execute-api.us-west-2.amazonaws.com/image-classifier
$ export CORTEX_DEST_S3_DIR=<YOUR_S3_DIRECTORY> # e.g. export CORTEX_DEST_S3_DIR=s3://my-bucket/dir
$ curl $BATCH_API_ENDPOINT \
    -X POST -H "Content-Type: application/json" \
    -d @- <<EOF
    {
        "workers": 1,
        "item_list": {
            "items": [
                "https://i.imgur.com/PzXprwl.jpg",
                "https://i.imgur.com/E4cOSLw.jpg",
                "http://farm1.static.flickr.com/13/17868690_fe11bdc16e.jpg",
                "https://i.imgur.com/jDimNTZ.jpg",
                "http://farm2.static.flickr.com/1140/950904728_0d84ac956b.jpg"
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

After submitting this job, you should get a response like this:

```json
{"job_id":"69d6faf82e4660d3","api_name":"image-classifier", "config":{"dest_s3_dir": "YOUR_S3_BUCKET_HERE"}}
```

### Find results

Wait for the job to complete by streaming the logs with `cortex logs <BATCH_API_NAME> <JOB_ID>` or watching for the job status to change with `cortex get <BATCH_API_NAME> <JOB_ID> --watch`.

```bash
$ cortex logs image-classifier 69d6faf82e4660d3 --env aws

started enqueuing batches to queue
partitioning 5 items found in job submission into 3 batches of size 2
completed enqueuing a total of 3 batches
spinning up workers...
...
2020-08-07 14:44:05.557598:cortex:pid-25:INFO:processing batch c9136381-6dcc-45bd-bd97-cc9c66ccc6d6
2020-08-07 14:44:26.037276:cortex:pid-25:INFO:executing on_job_complete
2020-08-07 14:44:26.208972:cortex:pid-25:INFO:no batches left in queue, job has been completed
```

The status of your job in the output of `cortex get <BATCH_API_NAME> <JOB_ID>` should change from `running` to `succeeded`. If it changes to a different status, you may be able to find the stacktrace using `cortex logs <BATCH_API_NAME> <JOB_ID>`. If your job has completed successfully, you can find the results of the image classification in the S3 directory on AWS console or using AWS CLI you specified in the job submission.

Using AWS CLI:

```bash
$ aws s3 ls $CORTEX_DEST_S3_DIR/<JOB_ID>/
  1de0bc65-04ea-4b9e-9e96-5a0bb52fcc37.json
  40100ffb-6824-4560-8ca4-7c0d14273e05.json
  6d1c933c-0ddf-4316-9956-046cd731c5ab.json
```
