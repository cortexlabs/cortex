## Live-reloading model APIs

_WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.24.*, run `git checkout -b 0.24` or switch to the `0.24` branch on GitHub)_

The model live-reloading feature is automatically enabled for the ONNX predictors. This means that any ONNX examples found in the [examples](../..) directory will already have this running.

The live-reloading is a feature that reloads models at run-time from (a) specified S3 bucket(s) in the `cortex.yaml` config of each API. Models are added/removed from the API when the said models are added/removed from the S3 bucket(s) or reloaded when the models are edited. More on this in the [docs](insert-link).
