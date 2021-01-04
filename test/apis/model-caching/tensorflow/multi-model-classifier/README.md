# Multi-Model Classifier API

This example deploys Iris, ResNet50 and Inception models in one API. Query parameters are used for selecting the model.

Since model caching is enabled, there can only be 2 models loaded into memory - loading a 3rd one will lead to the removal of the least recently used one. To witness the adding/removal process of models, check the logs of the API by running `cortex logs multi-model-classifier` once the API is up.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:

```bash
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the APIs endpoint. You can get the API's endpoint by running `cortex get multi-model-classifier`.

```bash
export ENDPOINT=your-api-endpoint
```

When making a prediction with [sample-image.json](sample-image.json), the following image will be used:

![sports car](https://i.imgur.com/zovGIKD.png)

### ResNet50 Classifier

Make a request to the ResNet50 model:

```bash
curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample-image.json
```

The expected response is:

```json
{"label": "sports_car"}
```

### Inception Classifier

Make a request to the Inception model:

```bash
curl "${ENDPOINT}?model=inception" -X POST -H "Content-Type: application/json" -d @sample-image.json
```

The expected response is:

```json
{"label": "sports_car"}
```

### Iris Classifier

At this point, there are 2 models loaded into memory (as specified by `cache_size`). Loading the `iris` classifier will lead to the removal of the least recently used model - in this case, it will be the ResNet50 model that will get evicted. Since the `disk_cache_size` is set to 3, no model will be removed from disk.

Make a request to the Iris model:

```bash
curl "${ENDPOINT}?model=iris" -X POST -H "Content-Type: application/json" -d @sample-iris.json
```

The expected response is:

```json
{"label": "setosa"}
```

---

Now, inspect `cortex get multi-model-classifier` to see when and which models were removed in this process of making requests to different versions of the same model.
