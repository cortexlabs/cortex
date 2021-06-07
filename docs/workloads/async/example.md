# AsyncAPI

### Define an API

```python
# main.py

from fastapi import FastAPI

app = FastAPI()

@app.post("/")
def hello_world():
    return {"msg": "hello world"}
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
docker build . --tag hello-world
```

### Run a container locally

```bash
docker run --port 8080:8080 hello-world
```

### Make a request

```bash
curl --request POST --header "Content-Type: application/json" localhost:8080
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
  kind: AsyncAPI
  pod:
    containers:
    - name: api
      image: <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/hello-world
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
cortex get hello-world
```

### Make a request

```bash
curl --request POST --header "Content-Type: application/json" http://***.amazonaws.com/hello-world
```

### Get the response

```bash
curl http://***.amazonaws.com/hello-world/<REQUEST_ID>
```
