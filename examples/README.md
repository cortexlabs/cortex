# Examples

| Example | Packaging |
|:--- | :--- |
| [BERT Sentiment Analysis](tensorflow/sentiment-analysis) </br> Demonstrates how to deploy a BERT for sentiment analysis with text preprocessing on requests. | TensorFlow |
| [DistilGPT2 Text Generation](pytorch/text-generator) </br> Demonstrates how to deploy HuggingFace's DistilGPT2 model. HuggingFace's tokenizers are used to encode incoming text and decode generate text. Model is sampled consecutively times to generate a response of desired length. | Cortex Predictor |
| [GPT-2 Text Generation](tensorflow/text-generator) </br> Demonstrates how to deploy a OpenAI's GPT-2 to generate text with text encoding and decoding. | TensorFlow |
| [Alexnet Image Classifier](pytorch/image-classifier) </br> Demonstrates how to deploy a pretrained Alexnet model from TorchVision with image preprocessing. | Cortex Predictor |
| [MPG Linear Regression](sklearn/mpg-regression) </br> Demonstrates how to deploy an Sklearn Linear Regression model. Pickled model is downloaded from S3 and served. | Cortex Predictor |
| [Inception V3 Image Classifier](tensorflow/image-classifier) </br> Demonstrates how to deploy an Inception V3 Image Classifier with image preprocessing on requests. | TensorFlow |
| [Alexnet Image Classifier](pytorch/image-classifier-onnx) </br> Demonstrates how to export a pretrained Alexnet from TorchVision to ONNX and deploy it. | ONNX |
| [Iris Classifier](pytorch/iris-classifier) </br> Demonstrates how to deploy a PyTorch model using Cortex Predictor. Model state is downloaded from S3 and and loaded into an IrisNet PyTorch Class for serving. | Cortex Predictor |
| [Iris Classifier](tensorflow/iris-classifier) </br> Iris classifier written in TensorFlow | TensorFlow |
| [Iris Classifier](xgboost/iris-classifier-onnx) </br> Iris classifier written in XGBoost and exported to ONNX | ONNX |
