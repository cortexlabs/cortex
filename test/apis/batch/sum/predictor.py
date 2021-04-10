import os
import boto3
import json
import re


class PythonPredictor:
    def __init__(self, config, job_spec):
        if len(config.get("dest_s3_dir", "")) == 0:
            raise Exception("'dest_s3_dir' field was not provided in job submission")

        self.s3 = boto3.client("s3")

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])
        self.list = []

    def predict(self, payload, batch_id):
        for numbers_list in payload:
            self.list.append(sum(numbers_list))

    def on_job_complete(self):
        json_output = json.dumps(self.list)
        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}.json", Body=json_output)
