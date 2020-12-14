# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.23.*, run `git checkout -b 0.23` or switch to the `0.23` branch on GitHub)

import boto3
import json
import time


class PythonPredictor:
    def __init__(self, config, job_spec):
        sqs = boto3.client('sqs', region_name='us-east-1')
        queue_url = job_spec['sqs_url']
        dead_letter_queue_arn = 'arn:aws:sqs:us-east-1:186623276624:dl-test.fifo'
        redrive_policy = {
            'deadLetterTargetArn': dead_letter_queue_arn,
            'maxReceiveCount': '1'
        }
        # Configure queue to send messages to dead letter queue
        sqs.set_queue_attributes(
            QueueUrl=queue_url,
            Attributes={
                'RedrivePolicy': json.dumps(redrive_policy)
            }
        )

    def predict(self, payload, batch_id):
        time.sleep(300)

    def on_job_complete(self):
        pass
