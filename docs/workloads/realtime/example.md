# RealtimeAPI

### Define an API

```python
# main.py

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class Data(BaseModel):
    msg: str

@app.post("/")
def realtime(data: Data):
    return data
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
docker build . -t realtime
```

### Run a container locally

```bash
docker run -p 8080:8080 realtime
```

### Make a request

```bash
curl -X POST -H "Content-Type: application/json" -d '{"msg": "hello world"}' localhost:8080
```

### Login to ECR

```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com
```

### Create a repository

```bash
aws ecr create-repository --repository-name realtime
```

### Tag the image

```bash
docker tag realtime <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/realtime
```

### Push the image

```bash
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/realtime
```

### Configure a Cortex deployment

```yaml
# cortex.yaml

- name: realtime
  kind: RealtimeAPI
  pod:
    containers:
    - name: api
      image: <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/realtime
```

### Create a Cortex deployment

```bash
cortex deploy
```

### Wait for the API to be ready

```bash
cortex get --watch
```

### Get the API endpoint

```bash
cortex get realtime
```

### Make a request

```bash
curl -X POST -H "Content-Type: application/json" -d '{"msg": "hello world"}' http://***.amazonaws.com/realtime
```
