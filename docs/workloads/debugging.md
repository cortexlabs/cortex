# Debugging

You can test and debug your handler implementation and image by running your API container locally.

The `cortex prepare-debug` command will generate a debugging configuration file named `<api_name>.debug.json` based on your api spec, and it will print out a corresponding `docker run` command that can be used to run the container locally.

For example:

```bash
cortex prepare-debug cortex.yaml iris-classifier

> docker run -p 9000:8888 \
> -e "CORTEX_VERSION=master" \
> -e "CORTEX_API_SPEC=/mnt/project/iris-classifier.debug.json" \
> -v /home/ubuntu/iris-classifier:/mnt/project \
> quay.io/cortexlabs/python-handler-cpu:master
```

Make a request to the api container:

```bash
curl localhost:9000 -X POST -H "Content-Type: application/json" -d @sample.json
```
