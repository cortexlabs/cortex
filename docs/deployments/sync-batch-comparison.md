# API statuses

You can productionize your models in two ways on the Cortex platform: online inference with SyncAPI and offline inference with BatchAPI.

SyncAPI deploys your model as a realtime rest webservice that autoscales based on requests to meet the demands of traffic. It keeps track of your latency, response codes and performs rolling updates to avoid downtime.

BatchAPI

BatchAPI

. BatchAPI creates an endpoint that receives job requests.

receives large job requests, breaks the job into batches and runs your model in parallel until the Job is done. When a job is submitted

SyncAPI creates an autoscaling webservice that serves your models realtime. When the SyncAPI

Sync vs Batch

| Use case | SyncAPI or BatchAPI |
| :--- | :---: | :---: |
| Integrate your model into internal pipelines | BatchAPI |
| Serve traffic to consumers | SyncAPI |
| Each request requires 1 inference (in the order of single/double digits) | SyncAPI |
| Each request requires > 1000  inferences | BatchAPI |
| Each request takes milliseconds to minutes | SyncAPI |
| Each request takes minutes to hours | BatchAPI |
| Handle unpredictable traffic | SyncAPI |
