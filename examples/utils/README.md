## Throughput tester

[throughput_test.py](throughput_test.py) is a Python CLI that can be used to test the throughput of your deployed API. The throughput will vary depending on your API's configuration (specified in your `cortex.yaml` file), your local machine's resources (mostly CPU, since it has to spawn many concurrent requests), and the internet connection on your local machine.

```bash
Usage: throughput_test.py [OPTIONS] ENDPOINT PAYLOAD

  Program for testing the throughput of Cortex-deployed APIs.

Options:
  -w, --processes INTEGER   Number of processes for prediction requests.  [default: 1]
  -t, --threads INTEGER     Number of threads per process for prediction requests.  [default: 1]
  -s, --samples INTEGER     Number of samples to run per thread.  [default: 10]
  -i, --time-based FLOAT    How long the thread making predictions will run for in seconds.
                            If set, -s option will be ignored.
  --help                    Show this message and exit.
```

`ENDPOINT` is the API's endpoint, which you can get by running `cortex get <API-name>`. This argument can also be exported as an environment variable instead of being passed to the CLI.

`PAYLOAD` can either be a local file or an URL resource that points to a file. The allowed extension types for the file are `json` and `jpg`. This argument can also be exported as an environment variable instead of being passed to the CLI.

* `json` files are generally `sample.json`s as they are found in most Cortex examples. Each of these is attached to the request as payload. The content type of the request is `"application/json"`.
* `jpg` images are read as numpy arrays and then are converted to a bytes object using `cv2.imencode` function. The content type of the request is `"application/octet-stream"`.

The same payload `PAYLOAD` is attached to all requests the script makes.

### Dependencies

The [throughput_test.py](throughput_test.py) CLI has been tested with Python 3.6.9. To install the CLI's dependencies, run the following:

```bash
pip install requests click opencv-contrib-python numpy validator-collection imageio
```
