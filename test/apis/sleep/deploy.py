import cortex
import os

dir_path = os.path.dirname(os.path.realpath(__file__))

cx = cortex.client()

api_spec = {
    "name": "sleep",
    "kind": "RealtimeAPI",
    "handler": {
        "type": "python",
        "path": "handler.py",
    },
}

print(cx.create_api(api_spec, project_dir=dir_path))

# cx.delete_api("sleep")
