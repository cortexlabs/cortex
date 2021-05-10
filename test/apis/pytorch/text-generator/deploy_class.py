import cortex
import os
import requests

dir_path = os.path.dirname(os.path.realpath(__file__))

cx = cortex.client()

api_spec = {
    "name": "text-generator",
    "kind": "RealtimeAPI",
}


class Handler:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def handle_post(self, payload):
        return self.model(payload["text"])[0]


api = cx.deploy_realtime_api(
    api_spec,
    handler=Handler,
    requirements=["torch", "transformers"],
    wait=True,
)

response = requests.post(
    api["endpoint"],
    json={"text": "machine learning is great because"},
)

print(response.status_code)
print(response.text)

cx.delete(api_spec["name"])
