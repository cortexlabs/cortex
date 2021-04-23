# Models

In addition to the [standard Python Handler](handler.md), Cortex also supports another handler called the TensorFlow handler, which can be used to deploy TensorFlow models exported as `SavedModel` models.

## Interface

**Uses TensorFlow version 2.3.0 by default**

```python
class Handler:
    def __init__(self, config, tensorflow_client, metrics_client):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing a vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified).
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in handle_async().
            metrics_client (optional): The cortex metrics client, which allows
                you to push custom metrics in order to build custom dashboards
                in grafana.
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def handle_async(self, payload, request_id):
        """(Required) Called once per request. Preprocesses the request payload
        (if necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), and postprocesses the inference
        output (if necessary).

        Args:
            payload (optional): The request payload (see below for the possible
                payload types).
            request_id (optional): The request id string that identifies a workload

        Returns:
            Prediction or a batch of predictions.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->

Cortex provides a `tensorflow_client` to your handler's constructor. `tensorflow_client` is an instance
of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/cortex/serve/cortex_internal/lib/client/tensorflow.py)
that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as
an instance variable in your handler class, and your `handle_async()` function should call `tensorflow_client.predict()` to make
an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions
can be implemented in your `handle_async()` function as well.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as
from where to download the model and initialization files, or any configurable model parameters. You define `config` in
your API configuration, and it is passed through to your handler class' constructor.

Your API can accept requests with different types of payloads. Navigate to the [API requests](#api-requests) section to
learn about how headers can be used to change the type of `payload` that is passed into your `handler_async` method.

At this moment, the AsyncAPI `handler_async` method can only return `JSON`-parseable objects. Navigate to
the [API responses](#api-responses) section to learn about how to configure it.
