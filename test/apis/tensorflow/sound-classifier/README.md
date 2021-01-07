# Sound Classifier

Based on https://tfhub.dev/google/yamnet/1.

Export the endpoint:

```bash
export ENDPOINT=<API endpoint>  # you can find this with `cortex get sound-classifier`
```

And then run the inference:

```bash
curl $ENDPOINT -X POST -H "Content-type: application/octet-stream" --data-binary "@silence.wav"
```
