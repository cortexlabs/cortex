# Python client

The Python client can be used to programmatically deploy models to a Cortex Cluster.

<!-- CORTEX_VERSION_BRANCH_STABLE, e.g. v0.9.0 -->
```bash
pip install git+https://github.com/cortexlabs/cortex.git@v0.9.1#egg=cortex\&subdirectory=pkg/workloads/cortex/client
```

The Python client needs to be initialized with AWS credentials and an operator URL for your Cortex cluster. You can find the operator URL by running `./cortex.sh info`.

```python
from cortex import Client

cortex = Client(
    aws_access_key_id="<string>",  # AWS access key associated with the account that the cluster is running on
    aws_secret_access_key="<string>",  # AWS secret key associated with the AWS access key
    operator_url="<string>" # operator URL of your cluster
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

`api_url` contains the URL of the deployed API. The API accepts JSON POST requests.

```python
import requests

sample = {
  "feature_1": 'a',
  "feature_2": 'b',
  "feature_3": 'c'
}

resp = requests.post(api_url, json=sample)
resp.json()
```
