# BatchAPI

### Define an API

```python
# main.py

from fastapi import FastAPI
from typing import List

app = FastAPI()

@app.post("/")
def handle_batch(batch: List[int]):
    print(batch)

@app.post("/on-job-complete")
def on_job_complete():
    print("done")
```

### Create a `Dockerfile`

```Dockerfile
FROM python:3.8-slim

RUN pip install --no-cache-dir fastapi uvicorn

COPY main.py /

CMD uvicorn --host 0.0.0.0 --port 8080 main:app
```

### Build an image

```bash
docker build . -t hello-world
```

### Run a container locally

```bash
docker run -p 8080:8080 hello-world
```

### Make a request

```bash
curl -X POST -H "Content-Type: application/json" -d '[1,2,3,4]' localhost:8080
```

### Login to ECR

```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com
```

### Create a repository

```bash
aws ecr create-repository --repository-name hello-world
```

### Tag the image

```bash
docker tag hello-world <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/hello-world
```

### Push the image

```bash
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/hello-world
```

### Configure a Cortex deployment

```yaml
# cortex.yaml

- name: hello-world
  kind: BatchAPI
  pod:
    containers:
    - name: api
      image: <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/hello-world
      command: ["uvicorn", "--host", "0.0.0.0", "--port", "8080", "main:app"]
```

### Create a Cortex deployment

```bash
cortex deploy
```

### Get the API endpoint

```bash
cortex get hello-world
```

### Make a request

```bash
curl -X POST -H "Content-Type: application/json" -d '{"workers": 2, "item_list": {"items": [1,2,3,4], "batch_size": 2}}' http://***.amazonaws.com/hello-world
```

### View the logs

```bash
cortex logs hello-world <JOB_ID>
```
