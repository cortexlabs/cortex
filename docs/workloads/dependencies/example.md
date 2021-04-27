# Deploy a project

You can deploy an API by providing a project directory. Cortex will save the project directory and make it available during API initialization.

```bash
project/
  ├── model.py
  ├── util.py
  ├── handler.py
  ├── requirements.txt
  └── ...
```

You can define your Handler class in a separate python file and import code from your project.

```python
# handler.py

from model import MyModel

class Handler:
    def __init__(self, config):
        model = MyModel()

    def handle_post(payload):
        return model(payload)
```

## Deploy using the Python Client

```python
import cortex

api_spec = {
    "name": "text-generator",
    "kind": "RealtimeAPI",
    "handler": {
        "type": "python",
        "path": "handler.py"
    }
}

cx = cortex.client("aws")
cx.create_api(api_spec, project_dir=".")
```

## Deploy using the CLI

```yaml
# api.yaml

- name: text-generator
  kind: RealtimeAPI
  handler:
    type: python
    path: handler.py
```

```bash
cortex deploy api.yaml
```
