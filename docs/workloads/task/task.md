# Task

Task APIs provide a lambda-style execution of containers. They are useful for running your containers on demand.

Task APIs are a good fit when you need to trigger container execution via an HTTP request. They can be used to run tasks (e.g. training models), and can be configured as task runners for orchestrators (such as airflow).

**Key Features**

* run containers on-demand
* scale to 0 (when there are no tasks)
* automatically recover from failures and spot instance termination

## How it works

When you deploy a Task API, an endpoint is created to receive task submissions. Upon submitting a Task, Cortex will respond with a Task ID and will asynchronously trigger the execution of a Task.

Cortex will initialize a worker pod based on your API specification. After the worker pod runs to completion, the Task is marked as completed and the pod is terminated.

You can make GET requests to the Task API endpoint to retrieve the status of the Task.

![](https://user-images.githubusercontent.com/4365343/121231738-c30fb680-c85e-11eb-886f-dc4d9bf3ef17.png)
