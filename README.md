<!-- Delete on release branches -->
<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='42'>

<br>

# Model serving for developers

Cortex is a flexible, production-grade deployment platform for machine learning engineers and data scientists. It is designed to make it easy to build and deploy inference pipelines that are scalable, reproducible, and monitorable by default.

* **Simple.** Write prediction APIs using an intuitive Python interface. Configure infrastructure with a single YAML document. Trigger deployments with one CLI command.
* **Production-ready.** Cortex automates your cloud infrastructure, spinning up a cluster and provisioning it for inference. On deploy, it containerizes your APIs and serves them to the cluster, implementing load balancing, request-based autoscaling, rolling updates, monitoring, and more.
* **Easy to integrate.** Cortex fits in your stack. Train models with any data science platform, trigger deployments with any CI/CD workflow, and connect with any analytics platform. Cortex is opinionated about model serving and deployment, but nothing else.

To deploy with Cortex, write your prediction API in Python, configure your infrastructure in YAML, and deploy with the command `cortex deploy`:

![Demo](https://d1zqebknpdh033.cloudfront.net/demo/gif/v0.18.gif)

<!-- Delete on release branches -->
<!-- CORTEX_VERSION_README_MINOR -->
[install](https://docs.cortex.dev/install) • [documentation](https://docs.cortex.dev) • [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples) • [we're hiring](https://angel.co/cortex-labs-inc/jobs) • [chat with us](https://gitter.im/cortexlabs/cortex)

<br>

# Key features

### Build

* Serve models from any Python framework—including TensorFlow, PyTorch, ONNX, and others.
* Perform batch and realtime inference.
* Construct multi-model APIs.
* Import any model server or library in your prediction APIs.
* Define preprocessing and postprocessing steps.

### Deploy

* Deploy locally on your own machine/VM, or to a Cortex cluster on your AWS account.
* Run on any hardware or cloud instance type, including spot instances and Inferentia.
* Customize autoscaling, load balancing, and traffic splitting behavior.
* Configure A/B testing and other deployment strategies.

### Manage

* Update APIs with zero downtime via rolling updates.
* Stream logs from your APIs to your CLI.
* Export API performance and prediction metrics to your CLI, or as JSON.
* Automatically version deployments for reproducibility.

# Install

<!-- CORTEX_VERSION_README_MINOR -->
```bash
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.20/get-cli.sh)"
```

<!-- CORTEX_VERSION_README_MINOR -->
See our [installation guide](https://docs.cortex.dev/install), then deploy one of our [examples](https://github.com/cortexlabs/cortex/tree/0.20/examples) or bring your own models to build [realtime APIs](https://docs.cortex.dev/deployments/realtime-api) and [batch APIs](https://docs.cortex.dev/deployments/batch-api).

### Learn more

Check out our [docs](https://docs.cortex.dev) and join our [community](https://gitter.im/cortexlabs/cortex).
