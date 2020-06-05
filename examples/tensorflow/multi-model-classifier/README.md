# Multi-Model Classifier API

This example deploys Iris, ResNet50 and Inception models in one API. Query parameters are used for selecting the model.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:
```
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the APIs endpoint. You can get the API's endpoint by running `cortex get multi-model-classifier`.

```bash
export ENDPOINT=your-api-endpoint
```

When making a prediction with [sample.json](sample.json), the following image will be used:

![sports car](https://i.imgur.com/zovGIKD.png)

### Iris Classifier Model

Make a request to the Iris model:

```bash
curl "${ENDPOINT}?model=iris" -X POST -H "Content-Type: application/json" -d @sample-iris.json
```

The expected response is:

```json
{"label": "setosa"}
```

### ResNet50 Classifier Model

Make a request to the ResNet50 model:

```bash
curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample-image.json
```

The expected response is:


```json
{"label": "sports_car"}
```

### Inception Classifier Model

Make a request to the Inception model:

```bash
curl "${ENDPOINT}?model=inception" -X POST -H "Content-Type: application/json" -d @sample-image.json
```

The expected response is:


```json
{"label": "sports_car"}
```
