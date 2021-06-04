# Task APIs

Task APIs provide a lambda-style execution of containers. They are useful for running your containers on demand.

The TaskAPI is a good fit when you need to trigger container execution via an HTTP request. It can be used as a task runner for orchestrators such as airflow to run tasks such as training a model on a GPU.

## How it works

When you deploy a TaskAPI, an endpoint is created to receive task submissions.

Upon submitting a task, you will receive a Task ID and asynchronously trigger the execution of a Task.

Cortex will initialize a worker pod based on your API specification. After worker pod runs to completion, the Task is marked as completed and the worker pod is terminated.

You can make GET requests to the TaskAPI endpoint with the to get the status of the Task.

![]()