# Image Classifier with ResNet50

This project implements an image recognition system using ResNet50. This system allows the recognition of up to 1000 classes.

## Deploying

There are 3 Cortex APIs available in this example:

1. [cortex_asic.yaml](cortex_asic.yaml) - to be used with `inf1` instances.
1. [cortex_cpu.yaml](cortex_cpu.yaml) - to be used with any instances that have CPUs.
1. [cortex_gpu.yaml](cortex_gpu.yaml) - to be used with instances that come with GPU support.

Any of the above 3 APIs can only be used one at a time within a given Cortex cluster. To deploy an API, just run:
```bash
cortex deploy <cortex-deployment-yaml>
```

## Verifying API

To verify the API is working, check that the API is live by running `cortex get image-classifier-resnet50`. Then, export the endpoint of the API:
```
export ENDPOINT=<API endpoint>
```

The image we use for classification is the following. This image is embedded in [sample.json](sample.json):

![image](https://i.imgur.com/213xcvs.jpg)

To run the inference, run the following command:
```bash
curl "${ENDPOINT}" -X POST -H "Content-Type: application/json" -d @sample.json
```

If a 5-element list is returned containing classifications of the image ("tabby", "Egyptian_cat", "tiger_cat", "tiger", "plastic_bag", with the first classification in the list being the most likely), then it means the API is working.

## Exporting SavedModels

Run the following command to install the dependencies for [Generating Resnet50 Models](Generating%20Resnet50%20Models.ipynb) notebook:
```bash
pip install neuron-cc==1.0.9410.0+6008239556 torch-neuron==1.0.825.0
```
Also, `torchvision` has to be installed, but without any dependencies:
```bash
pip install torchvision==0.4.2 --no-deps
```

The [Generating Resnet50 Models](Generating%20Resnet50%20Models.ipynb) notebook will generate 2 torch models. One saved `resnet50.pt` which can be run on GPU or on CPU and another as `resnet50_neuron.pt` which can only be run on `inf1` instances.
