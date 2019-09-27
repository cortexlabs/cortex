# Python Client

Python Client can be used to programmatically deploy to a Cortex Cluster.

<!-- CORTEX_VERSION_MINOR -->
```
pip install git+https://github.com/cortexlabs/cortex.git@master#egg=cortex\&subdirectory=pkg/workloads/cortex/client
```

Python client needs to be initialized with AWS credentials and a url to the operator your Cortex cluster.

```python
from cortex import Client

cortex = Client(
    aws_access_key_id="<string>",  # AWS access key associated with the account that created the Cortex cluster
    aws_secret_access_key="<string>",  # AWS secrey key associated with the provided AWS access key  
    operator_url="<string>" # Operator URL of your cortex cluster
)

api_url = cortex.deploy(
    deployment_name="<string>",  # deployment name (required)
    api_name="<string>",  # API name (required)
    model_path="<string>",  # S3 path to an exported model (required)
    pre_inference=callable,  # function used to prepare requests for model input
    post_inference=callable,  # function used to prepare model output for response
    model_format="<string>",  # model format, must be "tensorflow" or "onnx" (default: "onnx" if model path ends with .onnx, "tensorflow" if model path ends with .zip or is a directory)
    tf_serving_key="<string>"  # name of the signature def to use for prediction (required if your model has more than one signature def)
)
```

`api_url` is the url to the deployed API, it accepts JSON POST requests.

```python
import requests

sample = {
    "feature_1" [0,1,2]
}

resp = requests.post(api_url, json=sample)

resp.json() # your model prediction
```