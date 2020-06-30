# Image Classifier with Inception

This example implements an image recognition system using Inception, which allows for the recognition of up to 1000 classes.

## Deploying

There are 2 Cortex APIs available in this example:

1. [cortex.yaml](cortex.yaml) - standard deployment for Inception model.
1. [cortex_batch_sized.yaml](cortex_batch_sized.yaml) - Inception model deployed with a batch size > 1. The input/output doesn't suffer modifications.

To deploy an API, run:

```bash
cortex deploy <cortex-deployment-yaml>
```

E.g.

```bash
cortex deploy cortex_batch_sized.yaml

# by default it uses cortex.yaml
cortex deploy
```

## Making predictions

Check that your API is live by running `cortex get image-classifier-inception`, and copy the example `curl` command that's shown. After the API is live, run the `curl` command, e.g.

```bash
$ curl <API endpoint> -X POST -H "Content-Type: application/json" -d @sample.json

["tabby", "Egyptian_cat", "tiger_cat", "tiger", "plastic_bag"]
```

## Throughput test

Before [throughput_test.py](../../utils/throughput_test.py) is run, 2 environment variables have to be exported:

```bash
export ENDPOINT=<API endpoint>  # you can find this with `cortex get image-classifier-resnet50`
export PAYLOAD=sample.json # this is the sample.json file that contains the text to make the prediction for
```

Then, deploy each of the 2 APIs one at a time and check the results:

1. Running `python ../../utils/throughput_test.py -i X -p 4 -t X` with the [cortex.yaml](cortex.yaml) API running on a `g4dn.xlarge` instance will get **~X inferences/sec** with an average latency of **X ms**.
1. Running `python ../../utils/throughput_test.py -i X -p 4 -t X` with the [cortex_batch_sized.yaml](cortex_batch_sized.yaml) API running on a `g4dn.xlarge` instance will get **~X inferences/sec** with an average latency of **X ms**.
