# PyTorch Models

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Python Predictor

PyTorch models can be deployed using the Python Predictor. This is the most common approach for deploying PyTorch models. Please refer to the [Python Predictor documentation](../predictors/python.md).

<!-- CORTEX_VERSION_MINOR -->
PyTorch examples can be found in [examples/pytorch](https://github.com/cortexlabs/cortex/blob/master/examples/pytorch).

### Exporting PyTorch models

You can export your trained model with [torch.save()](https://pytorch.org/docs/stable/torch.html?highlight=save#torch.save) (here is PyTorch's documentation on [saving and loading models](https://pytorch.org/tutorials/beginner/saving_loading_models.html)).

For example, the iris classifier example exports the trained model like this:

```python
torch.save(model.state_dict(), "weights.pth")
```

`weights.pth` is then uploaded to S3, referenced in `cortex.yaml`, and downloaded and loaded in `predictor.py`:

```python
model = IrisNet()
model.load_state_dict(torch.load("weights.pth"))
model.eval()
```

<!-- CORTEX_VERSION_MINOR -->
The complete example can be found in [examples/pytorch/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/pytorch/iris-classifier).

## ONNX Predictor

It is also possible to export a model into the ONNX format and deploy the exported model using the ONNX Predictor. Please refer to the [ONNX Predictor documentation](../predictors/onnx.md).

<!-- CORTEX_VERSION_MINOR -->
One example which uses the ONNX Predictor is [examples/xgboost/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/xgboost/iris-classifier); this model was trained using XGBoost instead of PyTorch, but the approach is similar.

### Exporting PyTorch models to ONNX

You can export your trained model to the ONNX format with [torch.onnx.export()](https://pytorch.org/docs/stable/onnx.html#torch.onnx.export).

For example, the iris-classifier example above would export the trained model like this:

```python
placeholder = torch.randn(1, 4)
torch.onnx.export(
    model, placeholder, "model.onnx", input_names=["input"], output_names=["species"]
)
```

You would then upload `model.onnx` to S3 and reference it in `cortex.yaml`:

```yaml
- name: my-api
  predictor:
    type: onnx
    model: s3://my-bucket/model.onnx
    path: predictor.py
```
