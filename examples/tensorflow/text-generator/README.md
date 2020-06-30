# Text Generator with GPT-2

This example implements a text generator system based on GPT-2 model.

## Deploying

There are 2 Cortex APIs available in this example:

1. [cortex.yaml](cortex.yaml) - standard deployment for GPT-2 model.
1. [cortex_batch_sized.yaml](cortex_batch_sized.yaml) - GPT-2 model deployed with a batch size > 1. The input/output doesn't suffer modifications.

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

Check that your API is live by running `cortex get text-generator`, and copy the example `curl` command that's shown. After the API is live, run the `curl` command, e.g.

```bash
$ curl <API endpoint> -X POST -H "Content-Type: application/json" -d @sample.json

machine learning is just the tip of the iceberg. One thing that hasn't been covered is the fact that learning is just about as simple as it gets. It really does take a good understanding of the brain and all the various stages of evolution to gain the skills needed for human experience ...
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
