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

If a 5-element list is returned containing classifications of the image ("tabby", "tiger_cat", "Egyptian_cat", "lynx", "tiger", with the first classification in the list being the most likely), then it means the API is working.

## Throughput test

[throughput_test.py](throughput_test.py) is a Python CLI that can be used to test the throughput of the API. The throughput will vary depending on the used Cortex configuration file, your local machine's resources (mostly CPU) and the internet connection speed/latency on your machine.
```bash
Usage: throughput_test.py [OPTIONS] IMG_URL ENDPOINT

  Program for testing the throughput of Resnet50 model on instances equipped
  with CPU, GPU or ASIC devices.

Options:
  -w, --workers INTEGER     Number of workers for prediction requests.
                            [default: 1]
  -t, --threads INTEGER     Number of threads per worker.  [default: 1]
  -s, --samples INTEGER     Number of samples to run per thread.  [default:
                            10]
  -i, --time-based FLOAT    How long the thread making predictions will run for in seconds.
                            If set, -s option won't be considered anymore.
  -b, --batch-size INTEGER  Number of images sent for inference in one
                            request.  [default: 1]
  --help                    Show this message and exit.
```

Python 3.6.9 has been used for the Python CLI. To install the CLI's dependencies, run the following:
```bash
pip install requests click opencv-contrib-python numpy
```

Before [throughput_test.py](throughput_test.py) is run, 2 environment variables have to be exported:
```bash
export ENDPOINT=<API endpoint> # which has already been exported in the previous step
export IMG_URL=https://i.imgur.com/213xcvs.jpg # this is the cat image shown in the previous step
```

Then, deploy each API one at a time and check the results:

1. Running `python throughput_test.py -i 30 -w 4 -t 48` on an `inf1.2xlarge` instance using the [cortex_asic.yaml](cortex_asic.yaml) config will get **~510 inferences/sec** with an average latency of **80 ms**.
1. Running `python throughput_test.py -i 30 -w 4 -t 2` on a `c5.xlarge` instance using the [cortex_cpu.yaml](cortex_cpu.yaml) config will get **~16.2 inferences/sec** with an average latency of **200 ms**.
1. Running `python throughput_test.py -i 30 -w 4 -t 24` on a `g4dn.xlarge` instance using the [cortex_gpu.yaml](cortex_gpu.yaml) config will get **~125 inferences/sec** with an average latency of **85 ms**. Optimizing the model with TensorRT to use FP16 on TF-serving only seems to achieve a 10% performance improvement - one thing to consider is that the TensorRT engines hadn't been built beforehand, so this might have affected the results negatively.

## Exporting SavedModels

Run the following command to install the dependencies for [Generating Resnet50 Models](Generating%20Resnet50%20Models.ipynb) notebook:
```bash
pip install neuron-cc==1.0.9410.0+6008239556 tensorflow-neuron==1.15.0.1.0.1333.0 
```

The [Generating Resnet50 Models](Generating%20Resnet50%20Models.ipynb) notebook will generate 2 SavedModels. One will be saved in the `resnet50` directory which can be run on GPU or on CPU and another in the `resnet50_neuron` directory which can only be run on `inf1` instances.

Next, run the following command in a new Python environment to install the pip dependencies for [Generating GPU Resnet50 Model](Generating%20GPU%20Resnet50%20Model.ipynb) notebook:
```bash
pip install tensorflow==2.0.0
```
Alongside `tensorflow` package, TensorRT also has to be installed. Follow the instructions on [Nvidia TensorRT Documentation](https://docs.nvidia.com/deeplearning/tensorrt/install-guide/index.html#installing-debian) to download and install the TensorRT (will require about 5GB of space) on your local machine (you will have to create an Nvidia account). TensorRT is required for the exporting process of the SavedModel. The notebook also requires the SavedModel from `resnet50` directory generated with [Generating Resnet50 Models](Generating%20Resnet50%20Models.ipynb) notebook. Finally, the SavedModel will be exported to `resnet50_gpu` directory. You can then replace the existing SavedModel with the TensorRT-optimized version in [cortex_gpu.yaml](cortex_gpu.yaml) - it's a drop-in replacement that doesn't require any other dependencies on the Cortex side. By default, [cortex_gpu.yaml](cortex_gpu.yaml) config uses the non-TensorRT-optimized version due to simplicity.