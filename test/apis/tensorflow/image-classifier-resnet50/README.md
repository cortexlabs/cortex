# Image Classifier with ResNet50

This example implements an image recognition system using ResNet50, which allows for the recognition of up to 1000 classes.

## Deploying

There are 4 Cortex APIs available in this example:

1. [cortex.yaml](cortex.yaml) - can be used with any instances.
1. [cortex_inf.yaml](cortex_inf.yaml) - to be used with `inf1` instances.
1. [cortex_gpu.yaml](cortex_gpu.yaml) - to be used with GPU instances.
1. [cortex_gpu_server_side_batching.yaml](cortex_gpu_server_side_batching.yaml) - to be used with GPU instances. Deployed with `max_batch_size` > 1. The exported model and the TensorFlow Predictor do not need to be modified to support server-side batching.

To deploy an API, run:

```bash
cortex deploy <cortex-deployment-yaml>
```

E.g.

```bash
cortex deploy cortex_inf.yaml
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

Before [throughput_test.py](../../utils/throughput_test.py) is run, 2 environment variables have to be exported:

```bash
export ENDPOINT=<API endpoint>  # you can find this with `cortex get image-classifier-resnet50`
export PAYLOAD=https://i.imgur.com/213xcvs.jpg # this is the cat image shown in the previous step
```

Then, deploy each API one at a time and check the results:

1. Running `python ../../utils/throughput_test.py -i 30 -p 4 -t 2` with the [cortex.yaml](cortex.yaml) API running on an `c5.xlarge` instance will get **~16.2 inferences/sec** with an average latency of **200 ms**.
1. Running `python ../../utils/throughput_test.py -i 30 -p 4 -t 48` with the [cortex_inf.yaml](cortex_inf.yaml) API running on an `inf1.2xlarge` instance will get **~510 inferences/sec** with an average latency of **80 ms**.
1. Running `python ../../utils/throughput_test.py -i 30 -p 4 -t 24` with the [cortex_gpu.yaml](cortex_gpu.yaml) API running on an `g4dn.xlarge` instance will get **~125 inferences/sec** with an average latency of **85 ms**. Optimizing the model with TensorRT to use FP16 on TF-serving only seems to achieve a 10% performance improvement - one thing to consider is that the TensorRT engines hadn't been built beforehand, so this might have affected the results negatively.
1. Running `python ../../utils/throughput_test.py -i 30 -p 4 -t 60` with the [cortex_gpu_server_side_batching.yaml](cortex_gpu_batch_sized.yaml) API running on an `g4dn.xlarge` instance will get **~186 inferences/sec** with an average latency of **500 ms**. This achieves a 49% higher throughput than the [cortex_gpu.yaml](cortex_gpu.yaml) API, at the expense of increased latency.

Alternatively to [throughput_test.py](../../utils/throughput_test.py), the `ab` GNU utility can also be used to benchmark the API. This has the advantage that it's not as taxing on your local machine, but the disadvantage that it doesn't implement a cooldown period. You can run `ab` like this:

```bash
# for making octet-stream requests, which is the default for throughput_test script
ab -n <number-of-requests> -c <concurrency-level> -p sample.bin -T 'application/octet-stream' -rks 120 $ENDPOINT

# for making json requests, will will have lower performance because the API has to download the image every time
ab -n <number-of-requests> -c <concurrency-level> -p sample.json -T 'application/json' -rks 120 $ENDPOINT
```

*Note: `inf1.xlarge` isn't used because the major bottleneck with `inf` instances for this example is with the CPU, and `inf1.2xlarge` has twice the amount of CPU cores for same number of Inferentia ASICs (which is 1), which translates to almost double the throughput.*

## Exporting SavedModels

This example deploys models that we have built and uploaded to a public S3 bucket. If you want to build the models yourself, follow these instructions.

Run the following command to install the dependencies required for the [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook:

```bash
pip install --extra-index-url=https://pip.repos.neuron.amazonaws.com \
  neuron-cc==1.0.9410.0+6008239556 \
  tensorflow-neuron==1.15.0.1.0.1333.0
```

The [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook will generate 2 SavedModels. One will be saved in the `resnet50` directory which can be run on GPU or on CPU and another in the `resnet50_neuron` directory which can only be run on `inf1` instances. For server-side batching on `inf1` instances, a different compilation of the model is required. To compile ResNet50 model for a batch size of 5, run `run_all` from [this directory](https://github.com/aws/aws-neuron-sdk/tree/master/src/examples/tensorflow/keras_resnet50).

If you'd also like to build the TensorRT version of the GPU model, run the following command in a new Python environment to install the pip dependencies required for the [generate_gpu_resnet50_model.ipynb](generate_gpu_resnet50_model.ipynb) notebook:

```bash
pip install tensorflow==2.0.0
```

TensorRT also has to be installed to export the SavedModel. Follow the instructions on [Nvidia TensorRT Documentation](https://docs.nvidia.com/deeplearning/tensorrt/install-guide/index.html#installing-debian) to download and install TensorRT on your local machine (this will require ~5GB of space, and you will have to create an Nvidia account). This notebook also requires that the SavedModel generated with the [generate_resnet50_models.ipynb](generate_resnet50_models.ipynb) notebook exists in the `resnet50` directory. The TensorRT SavedModel will be exported to the `resnet50_gpu` directory. You can then replace the existing SavedModel with the TensorRT-optimized version in [cortex_gpu.yaml](cortex_gpu.yaml) - it's a drop-in replacement that doesn't require any other dependencies on the Cortex side. By default, the API config in [cortex_gpu.yaml](cortex_gpu.yaml) uses the non-TensorRT-optimized version due to simplicity.
