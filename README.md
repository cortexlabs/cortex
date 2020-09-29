# Build machine learning APIs

Cortex makes deploying, scaling, and managing machine learning systems in production simple. We believe that developers in any organization should be able to add natural language processing, computer vision, and other machine learning capabilities to their applications without having to worry about infrastructure.

<br>

# Key features

### Deploy

* Run Cortex locally or as a production cluster on your AWS account.
* Deploy TensorFlow, PyTorch, Keras, ONNX, XGBoost, scikit-learn, and other models as realtime APIs or batch APIs.
* Define preprocessing and postprocessing steps in Python.

### Manage

* Update APIs with no downtime.
* Stream logs from your APIs to your CLI.
* Monitor API performance and track predictions.
* Run A/B tests.

### Scale

* Automatically scale APIs to handle production traffic.
* Reduce your cloud infrastructure spend with spot instances.
* Maximize resource utilization by deploying multiple models per API.

<br>

# How it works

Implement your predictor in `predictor.py`, configure your deployment in `cortex.yaml`, and run `cortex deploy`.

Here's how to deploy GPT-2 as a scalable text generation API:

![Demo](https://d1zqebknpdh033.cloudfront.net/demo/gif/v0.18.gif)

<br>

# Get started

### Install

<!-- CORTEX_VERSION_README_MINOR -->
```bash
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.20/get-cli.sh)"
```

<!-- CORTEX_VERSION_README_MINOR -->
See our [installation guide](https://docs.cortex.dev/install), then deploy one of our [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples) or bring your own models to build [realtime APIs](https://docs.cortex.dev/deployments/realtime-api) and [batch APIs](https://docs.cortex.dev/deployments/batch-api).

### Learn more

Check out our [docs](https://docs.cortex.dev) and join our [community](https://gitter.im/cortexlabs/cortex).
