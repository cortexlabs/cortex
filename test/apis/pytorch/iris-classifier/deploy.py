import cortex
import os

dir_path = os.path.dirname(os.path.realpath(__file__))

cx = cortex.client()

api_spec = {
    "name": "iris-classifier",
    "kind": "RealtimeAPI",
    "handler": {
        "type": "python",
        "path": "handler.py",
        "config": {
            "model": "s3://cortex-examples/pytorch/iris-classifier/weights.pth",
        },
    },
}

print(cx.deploy(api_spec, project_dir=dir_path))

# cx.delete("iris-classifier")
