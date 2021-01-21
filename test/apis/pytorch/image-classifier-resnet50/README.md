# Image Classifier with ResNet50

This example implements an image recognition system using ResNet50, which allows for the recognition of up to 1000 classes.

## Deploying

There are 3 Cortex APIs available in this example:

1. [cortex.yaml](cortex.yaml) - can be used with any instances.
1. [cortex_inf.yaml](cortex_inf.yaml) - to be used with `inf1` instances.
1. [cortex_gpu.yaml](cortex_gpu.yaml) - to be used with GPU instances.

To deploy an API, run:

```bash
cortex deploy <cortex-deployment-yaml>
```

E.g.

```bash
cortex deploy cortex_gpu.yaml
```

## Verifying your API

Check that your API is live by running `cortex get image-classifier-resnet50`, and copy the example `curl` command that's shown. After the API is live, run the `curl` command, e.g.

```bash
$ curl <API endpoint> -X POST -H "Content-Type: application/json" -d @sample.json

["tabby", "Egyptian_cat", "tiger_cat", "tiger", "plastic_bag"]
```

The following image is embedded in [sample.json](sample.json):

![image](https://i.imgur.com/213xcvs.jpg)

## Exporting SavedModels

This example deploys models that we have built and uploaded to a public S3 bucket. If you want to build the models yourself, follow these instructions.

Run the following command to install the dependencies required for the [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook:

```bash
pip install --extra-index-url=https://pip.repos.neuron.amazonaws.com \
 neuron-cc==1.0.9410.0+6008239556 \
 torch-neuron==1.0.825.0
```

Also, `torchvision` has to be installed, but without any dependencies:

```bash
pip install torchvision==0.4.2 --no-deps
```

The [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook will generate 2 torch models. One is saved as `resnet50.pt` which can be run on GPU or CPU, and another is saved as `resnet50_neuron.pt`, which can only be run on `inf1` instances.
