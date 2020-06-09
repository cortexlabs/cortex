# Multi-Model Analyzer API

This example deploys a sentiment analyzer and a text summarizer in one API. Query parameters are used for selecting the model.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:

```bash
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the APIs endpoint. You can get the API's endpoint by running `cortex get text-analyzer`.

```bash
export ENDPOINT=your-api-endpoint
```

### Sentiment Analyzer Classifier

Make a request to the sentiment analyzer model:

```bash
curl "${ENDPOINT}?model=sentiment" -X POST -H "Content-Type: application/json" -d @sample-sentiment.json
```

The expected response is:

```json
{"label": "POSITIVE", "score": 0.9998506903648376}
```

### Text Summarizer

Make a request to the text summarizer model:

```bash
curl "${ENDPOINT}?model=summarizer" -X POST -H "Content-Type: application/json" -d @sample-summarizer.json
```

The expected response is:

```text
Machine learning is the study of algorithms and statistical models that computer systems use to perform a specific task. It is seen as a subset of artificial intelligence. Machine learning algorithms are used in a wide variety of applications, such as email filtering and computer vision. In its application across business problems, machine learning is also referred to as predictive analytics.
```
