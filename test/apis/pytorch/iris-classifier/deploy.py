import cortex
import os

dir_path = os.path.dirname(os.path.realpath(__file__))

cx = cortex.client()

api_spec = {
    "name": "iris-classifier",
    "kind": "RealtimeAPI",
    "predictor": {
        "type": "python",
        "path": "predictor.py",
        "config": {
            "model": "s3://cortex-examples/pytorch/iris-classifier/weights.pth",
        },
    },
}

print(cx.create_api(api_spec, project_dir=dir_path))

# cx.delete_api("iris-classifier")
