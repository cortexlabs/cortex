# Image Classifier with ResNet50

This example implements an image recognition system using ResNet50, which allows for the recognition of up to 1000 classes.

## Deploying

There are 3 Cortex APIs available in this example:

1. [cortex_inf.yaml](cortex_inf.yaml) - to be used with `inf1` instances.
1. [cortex_cpu.yaml](cortex_cpu.yaml) - to be used with any instances that have CPUs.
1. [cortex_gpu.yaml](cortex_gpu.yaml) - to be used with GPU instances.

To deploy an API, run:

```bash
cortex deploy <cortex-deployment-yaml>
```

E.g.

```bash
cortex deploy cortex_cpu.yaml
```

## Verifying your API

Check that your API is live by running `cortex get image-classifier-resnet50`, and copy the example `curl` command that's shown. After the API is live, run the `curl` command, e.g.

```bash
$ curl <API endpoint> -X POST -H "Content-Type: application/json" -d @sample.json

["tabby", "Egyptian_cat", "tiger_cat", "tiger", "plastic_bag"]
```

The following image is embedded in [sample.json](sample.json):

![image](https://i.imgur.com/213xcvs.jpg)

## Throughput test

[throughput_test.py](throughput_test.py) is a Python CLI that can be used to test the throughput of your deployed API. The throughput will vary depending on your API's configuration (specified in the `cortex_*.yaml` file), your local machine's resources (mostly CPU, since it has to spawn many concurrent requests), and the internet connection on your local machine.

```bash
Usage: throughput_test.py [OPTIONS] IMG_URL ENDPOINT

  Program for testing the throughput of a Resnet50 model on instances equipped
  with CPU, GPU or Inferentia devices.

Options:
  -w, --workers INTEGER     Number of workers for prediction requests.  [default: 1]
  -t, --threads INTEGER     Number of threads per worker for prediction requests.  [default: 1]
  -s, --samples INTEGER     Number of samples to run per thread.  [default: 10]
  -i, --time-based FLOAT    How long the thread making predictions will run for in seconds.
                            If set, -s option will be ignored.
  -b, --batch-size INTEGER  Number of images sent for inference in one request.  [default: 1]
  --help                    Show this message and exit.
```

The Python CLI has been tested with Python 3.6.9. To install the CLI's dependencies, run the following:

```bash
pip install requests click opencv-contrib-python numpy
```

Before [throughput_test.py](throughput_test.py) is run, 2 environment variables have to be exported:

```bash
export ENDPOINT=<API endpoint>  # you can find this with `cortex get image-classifier-resnet50`
export IMG_URL=https://i.imgur.com/213xcvs.jpg # this is the cat image shown in the previous step
```

Then, deploy each API one at a time and check the results:

1. Running `python throughput_test.py -i 30 -w 4 -t 48` with the [cortex_inf.yaml](cortex_inf.yaml) API running on an `inf1.2xlarge` instance will get **~510 inferences/sec** with an average latency of **80 ms**.
1. Running `python throughput_test.py -i 30 -w 4 -t 2` with the [cortex_cpu.yaml](cortex_cpu.yaml) API running on an `c5.xlarge` instance will get **~16.2 inferences/sec** with an average latency of **200 ms**.
1. Running `python throughput_test.py -i 30 -w 4 -t 24` with the [cortex_gpu.yaml](cortex_gpu.yaml) API running on an `g4dn.xlarge` instance will get **~125 inferences/sec** with an average latency of **85 ms**. Optimizing the model with TensorRT to use FP16 on TF-serving only seems to achieve a 10% performance improvement - one thing to consider is that the TensorRT engines hadn't been built beforehand, so this might have affected the results negatively.

*Note: `inf1.xlarge` isn't used because the major bottleneck with `inf` instances is with the CPU, and so `inf1.2xlarge` has twice the amount of cores for same number of Inferentia ASICs (which is 1), which translates to almost double the throughput.*

## Exporting SavedModels

This example deploys models that we have built and uploaded to a public S3 bucket. If you want to build the models yourself, follow these instructions.

Run the following command to install the dependencies required for the [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook:

```bash
pip install neuron-cc==1.0.9410.0+6008239556 tensorflow-neuron==1.15.0.1.0.1333.0
```

The [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook will generate 2 SavedModels. One will be saved in the `resnet50` directory which can be run on GPU or on CPU and another in the `resnet50_neuron` directory which can only be run on `inf1` instances.

If you'd also like to build the TensorRT version of the GPU model, run the following command in a new Python environment to install the pip dependencies required for the [generate_gpu_resnet50_model.ipynb](generate_gpu_resnet50_model.ipynb) notebook:

```bash
pip install tensorflow==2.0.0
```

TensorRT also has to be installed to export the SavedModel. Follow the instructions on [Nvidia TensorRT Documentation](https://docs.nvidia.com/deeplearning/tensorrt/install-guide/index.html#installing-debian) to download and install TensorRT on your local machine (this will require ~5GB of space, and you will have to create an Nvidia account). This notebook also requires that the SavedModel generated with the [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook exists in the `resnet50` directory. The TensorRT SavedModel will be exported to the `resnet50_gpu` directory. You can then replace the existing SavedModel with the TensorRT-optimized version in [cortex_gpu.yaml](cortex_gpu.yaml) - it's a drop-in replacement that doesn't require any other dependencies on the Cortex side. By default, the API config in [cortex_gpu.yaml](cortex_gpu.yaml) uses the non-TensorRT-optimized version due to simplicity.
