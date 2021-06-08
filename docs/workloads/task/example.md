# TaskAPI

### Define an API

```python
# main.py

print("hello world")
```

### Create a `Dockerfile`

```Dockerfile
FROM python:3.8-slim

COPY main.py /

CMD exec python main.py
```

### Build an image

```bash
docker build . -t task
```

### Run a container locally

```bash
docker run -it --rm task
```

### Login to ECR

```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com
```

### Create a repository

```bash
aws ecr create-repository --repository-name task
```

### Tag the image

```bash
docker tag task <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/task
```

### Push the image

```bash
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/task
```

### Configure a Cortex deployment

```yaml
# cortex.yaml

- name: task
  kind: TaskAPI
  pod:
    containers:
    - name: api
      image: <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/task
      command: ["python", "main.py"]
```

### Create a Cortex deployment

```bash
cortex deploy
```

### Get the API endpoint

```bash
cortex get task
```

### Make a request

```bash
curl -X POST -H "Content-Type: application/json" -d '{}' http://***.amazonaws.com/task
```

### View the logs

```bash
cortex logs task <JOB_ID>
```
