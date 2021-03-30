import cortex

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

print(cx.create_api(api_spec, project_dir="test/apis/pytorch/iris-classifier"))

# cx.delete_api("iris-classifier")
