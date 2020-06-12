# Multi-Model Classifier API

This example deploys ResNet50, MobileNet and ShuffleNet models in one API. Query parameters are used for selecting the model.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:

```bash
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the API's endpoint for convenience. You can get the API's endpoint by running `cortex get multi-model-classifier`.

```bash
export ENDPOINT=your-api-endpoint
```

When making a prediction with [sample.json](sample.json), the following image will be used:

![cat](https://i.imgur.com/213xcvs.jpg)

### ResNet50 Classifier

Make a request to the ResNet50 model:

```bash
curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"label": "tabby"}
```

### MobileNet Classifier

Make a request to the MobileNet model:

```bash
curl "${ENDPOINT}?model=mobilenet" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"label": "tabby"}
```

### ShuffleNet Classifier

Make a request to the ShuffleNet model:

```bash
curl "${ENDPOINT}?model=shufflenet" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"label": "Egyptian_cat"}
```
