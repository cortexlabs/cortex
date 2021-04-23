# Models

In addition to the [standard Python Handler](handler.md), Cortex also supports another handler called the TensorFlow handler, which can be used to run TensorFlow models exported as `SavedModel` models.

## Interface

**Uses TensorFlow version 2.3.0 by default**

```python
class Handler:
    def __init__(self, tensorflow_client, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in handle_batch().
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
            job_spec (optional): Dictionary containing the following fields:
                "job_id": A unique ID for this job
                "api_name": The name of this batch API
                "config": The config that was provided in the job submission
                "workers": The number of workers for this job
                "total_batch_count": The total number of batches in this job
                "start_time": The time that this job started
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def handle_batch(self, payload, batch_id):
        """(Required) Called once per batch. Preprocesses the batch payload (if
        necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), postprocesses the inference output
        (if necessary), and writes the predictions to storage (i.e. S3 or a
        database, if desired).

        Args:
            payload (required): a batch (i.e. a list of one or more samples).
            batch_id (optional): uuid assigned to this batch.
        Returns:
            Nothing
        """
        pass

    def handle_on_job_complete(self):
        """(Optional) Called once after all batches in the job have been
        processed. Performs post job completion tasks such as aggregating
        results, executing web hooks, or triggering other jobs.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` to your Handler class' constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/cortex/serve/cortex_internal/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Handler class, and your `handle_batch()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `handle_batch()` function as well.

When multiple models are defined using the Handler's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`).

If you need to share files between your handler implementation and the TensorFlow Serving container, you can create a new directory within `/mnt` (e.g. `/mnt/user`) and write files to it. The entire `/mnt` directory is shared between containers, but do not write to any of the directories in `/mnt` that already exist (they are used internally by Cortex).
