# MPG Estimator API

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.22.*, run `git checkout -b 0.22` or switch to the `0.22` branch on GitHub)_

This project implements a multi-lingual translation API, supporting translations between over 150 languages:


```bash
curl https://***.amazonaws.com/translator -X POST -H "Content-Type: application/json" -d {"source_language": "en", "destination_language": "phi", "text": "It is a mistake to think you can solve any major problems just with potatoes." }

{"generated_text": "Sayop an paghunahuna nga masulbad mo ang bisan ano nga dagkong mga problema nga may patatas lamang."}
```

Priorities of this project include:

- __Cost effectiveness.__ Each language-to-language translation is handled by a different ~300 MB model. Traditional setups would deploy all X models across many servers to ensure availability, but this API can be run on a single server thanks to Cortex's multi-model caching.
- __Ease of use.__ Predictions are generated using Hugging Face's Transformer Library and Cortex's Predictor API, while the translation service itself runs on a Cortex cluster self-hosted on your AWS account.
- __Configurability.__ All tools used in this API are fully open source and modifiable. The deployed service and underlying infrastructure run on your AWS account. The prediction API can be run on CPU and GPU instances.

## Models used

This project uses pre-trained Opus MT neural machine translation models, trained by Jörg Tiedemann and the Language Technology Research Group at the University of Helsinki. The models are hosted for free by Hugging Face.

## How to deploy the API

To deploy the API, first spin up a Cortex cluster by running `$ cortex cluster up --config cortex.yaml`. Note that the configuration file we are providing Cortex with (accessible at `cortex.yaml`) requests a g4dn.xlarge GPU instance. If your AWS account does not have access to GPU instances, you can request an EC2 service quota increase easily [here](https://console.aws.amazon.com/servicequotas), or you can simply use CPU instances.

```bash
$ cortex cluster up --config cortex.yaml

email address [press enter to skip]:

verifying your configuration ...

aws access key id ******************** will be used to provision a cluster named "cortex" in us-east-1:

￮ using existing s3 bucket: cortex-***** ✓
￮ using existing cloudwatch log group: cortex ✓
￮ creating cloudwatch dashboard: cortex ✓
￮ creating api gateway: cortex ✓
￮ spinning up the cluster (this will take about 15 minutes) ...
￮ updating cluster configuration ✓
￮ configuring networking ✓
￮ configuring autoscaling ✓
￮ configuring logging ✓
￮ configuring metrics ✓
￮ configuring gpu support ✓
￮ starting operator ✓
￮ waiting for load balancers ...... ✓
￮ downloading docker images ✓

cortex is ready!

```

The config file referenced is in this repo. Cortex can cluster up without a config file, but for this deployment, we use it to specify that we need to disable API Gateway, as its timeout limits will interfere with our API.  

Once the cluster is spun up (roughly 20 minutes), we can deploy by running:

```bash
cortex deploy
```

I've configured my CLI to default to the AWS environment by running `cortex env default aws`)

Now, we wait for the API to become live. You can track its status with `cortex get --watch`.

Note that after the API goes live, we need to wait a few minutes for it to register all the models hosted in the S3 bucket. Because the bucket is so large, it takes Cortex a bit longer than usual. When it's done, running `cortex get translator` should return something like:

```
$ cortex get translator

using aws environment

status   up-to-date   requested   last update   avg request   2XX
live     1            1           3m            --            --

metrics dashboard: https://us-east-1.console.aws.amazon.com/cloudwatch/home#dashboards:name=***

endpoint: http://***.us-east-1.amazonaws.com/translator
example: curl: curl http://***.us-east-1.amazonaws.com/translator -X POST -H "Content-Type: application/json" -d @sample.json

model name                         model version   edit time
marian_converted_v1                1 (latest)      24 Aug 20 14:23:41 EDT
opus-mt-NORTH_EU-NORTH_EU          1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-ROMANCE-en                 1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-SCANDINAVIA-SCANDINAVIA    1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-aav-en                     1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-aed-es                     1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-de                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-en                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-eo                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-es                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-fi                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-fr                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-nl                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-ru                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-af-sv                      1 (latest)      21 Aug 20 10:42:38 EDT
opus-mt-afa-afa                    1 (latest)      21 Aug 20 10:42:38 EDT
...
```

With the list continuing on for hundreds of models. We can now query the API at the endpoint given, structuring the body of our request according to the format expected by our predictor (specified in `predictor.py`):

```
{
    "source_language": "en",
    "destination_language": "es",
    "text": "So long and thanks for all the fish."
}
```

The response should look something like this:

```
{"generated_text": "Hasta luego y gracias por todos los peces."}
```

The first time you request a specific language-to-language translation, the model will be downloaded from S3, which will take some time (roughly a minute). Every subsequent request will be much faster, as the API is defined as being able to hold 250 models on disk and 5 in memory. Models already loaded into memory will serve predictions fastest, while those on disk will take slightly longer as they need to be swapped out. Instances with more memory and disk space can naturally hold more models.

If you know which models are most likely to be needed, you can "warm up" the API by calling them immediately after deploy, or you can even deploy a separate API that only contains those models, and route relevant traffic there to ensure minimal latency.

Finally, note that both models are removed from both memory and disk according to which model was used last. You can read more about how caching works in the [Cortex docs.](https://docs.cortex.dev/)
