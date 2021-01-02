# MPG Estimator API

This example deploys an MPG estimator model of multiple versions in one API. Query parameters are used for selecting the model and the version.

Since model caching is enabled, there can only be 2 models loaded into memory (counting the versioned models as well) - loading a 3rd one will lead to the removal of the least recently used one. To witness the adding/removal process of models, check the logs of the API by running `cortex logs mpg-estimator` once the API is up.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:

```bash
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the API's endpoint for convenience. You can get the API's endpoint by running `cortex get mpg-estimator`.

```bash
export ENDPOINT=your-api-endpoint
```

### Version 1

Make a request version `1` of the `mpg-estimator` model:

```bash
curl "${ENDPOINT}?version=1" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"prediction": 26.929889872154185, "model": {"name": "mpg-estimator", "version": "1"}}
```

### Version 2

At this point, there is one model loaded into memory (as specified by `cache_size`). Loading another versioned model as well will lead to the removal of the least recently used model - in this case, it will be version 1 that will get evicted. Since the `disk_cache_size` is set to 2, no model will be removed from disk.

Make a request version `2` of the `mpg-estimator` model:

```bash
curl "${ENDPOINT}?version=2" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"prediction": 26.929889872154185, "model": {"name": "mpg-estimator", "version": "2"}}
```

### Version 3

With the following request, version 2 of the model will have to be evicted from the memory. Since `disk_cache_size` is set to 2, this time, version 1 of the model will get removed from the disk.

Make a request version `3` of the `mpg-estimator` model:

```bash
curl "${ENDPOINT}?version=3" -X POST -H "Content-Type: application/json" -d @sample.json
```

The expected response is:

```json
{"prediction": 26.929889872154185, "model": {"name": "mpg-estimator", "version": "3"}}
```

---

Now, inspect `cortex get mpg-estimator` to see when and which models were removed in this process of making requests to different versions of the same model. The same algorithm is applied to different models as well, not just for the versions of a specific model.
