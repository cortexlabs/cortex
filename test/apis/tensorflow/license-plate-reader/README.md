# Real-Time License Plate Identification System

This project implements a license plate identification system. On resource-constrained systems, running inferences may prove to be too computationally expensive. One solution is to run the ML in the cloud and have the local (embedded) system act as a client of these services.

![Demo GIF](https://i.imgur.com/jgkJB59.gif)

*Figure 1 - GIF taken from this real-time recording [video](https://www.youtube.com/watch?v=gsYEZtecXlA) of predictions*

![Raspberry Pi client with 4G access and onboard GPS that connects to cortex's APIs for inference](https://i.imgur.com/MvDAXWU.jpg)

*Figure 2 - Raspberry Pi-powered client with 4G access and onboard GPS that connects to cortex's APIs for inference. More on that [here](https://github.com/RobertLucian/cortex-license-plate-reader-client).*

In our example, we assume we have a dashcam mounted on a car and we want to detect and recognize all license plates in the video stream in real-time. We can use an embedded computer system to record the video, then stream and infer frame-by-frame using a web service, reassemble the stream with the licence plate annotations, and finally display the annotated stream on a screen. The web service in our case is a set of 2 web APIs deployed using cortex.

## Used Models

The identification of license plates is done in three steps:

1. Detecting the bounding boxes of each license plate using *YOLOv3* model.
1. Detecting the very specific region of each word inside each bounding box with high accuracy using a pretrained *CRAFT* text detector.
1. Recognizing the text inside the previously detected boxes using a pretrained *CRNN* model.

Out of these three models (*YOLOv3*, *CRAFT* and *CRNN*) only *YOLOv3* has been fine-tuned with a rather small dataset to better work with license plates. This dataset can be found [here](https://github.com/RobertLucian/license-plate-dataset). This *YOLOv3* model has in turn been trained using [this](https://github.com/experiencor/keras-yolo3) GitHub project. To get more details about our fine-tuned model, check the project's description page.

The other two models, *CRAFT* and *CRNN*, can be found in [keras-ocr](https://github.com/faustomorales/keras-ocr).

## Deployment - Lite Version

A lite version of the deployment is available with `cortex_lite.yaml`. The lite version accepts an image as input and returns an image with the recognized license plates overlayed on top. A single GPU is required for this deployment (i.e. `g4dn.xlarge`).

Once the cortex cluster is created, run

```bash
cortex deploy cortex_lite.yaml
```

And monitor the API with

```bash
cortex get --watch
```

To run an inference on the lite version, the only 3 tools you need are `curl`, `sed` and `base64`. This API expects an URL pointing to an image onto which the inferencing is done. This includes the detection of license plates with *YOLOv3* and the recognition part with *CRAFT* + *CRNN* models.

Export the endpoint & the image's URL by running

```bash
export ENDPOINT=your-api-endpoint
export IMAGE_URL=https://i.imgur.com/r8xdI7P.png
```

Then run the following piped commands

```bash
curl "${ENDPOINT}" -X POST -H "Content-Type: application/json" -d '{"url":"'${IMAGE_URL}'"}' |
sed 's/"//g' |
base64 -d > prediction.jpg
```

The resulting image is the same as the one in [Verifying the Deployed APIs](#verifying-the-deployed-apis).

For another prediction, let's use a generic image from the web. Export [this image's URL link](https://i.imgur.com/mYuvMOs.jpg) and re-run the prediction. This is what we get.

![annotated sample image](https://i.imgur.com/tg1PE1E.jpg)

*The above prediction has the bounding boxes colored differently to distinguish them from the cars' red bodies*

## Deployment - Full Version

The recommended number of instances to run this smoothly on a video stream is about 12 GPU instances (2 GPU instances for *YOLOv3* and 10 for *CRNN* + *CRAFT*). `cortex_full.yaml` is already set up to use these 12 instances. Note: this is the optimal number of instances when using the `g4dn.xlarge` instance type. For the client to work smoothly, the number of processes per replica can be adjusted, especially for `p3` or `g4` instances, where the GPU has a lot of compute capacity.

If you don't have access to this many GPU-equipped instances, you could just lower the number and expect dropped frames. It will still prove the point, albeit at a much lower framerate and with higher latency. More on that [here](https://github.com/RobertLucian/cortex-license-plate-reader-client).

Then after the cortex cluster is created, run

```bash
cortex deploy cortex_full.yaml
```

And monitor the APIs with

```bash
cortex get --watch
```

We can run the inference on a sample image to verify that both APIs are working as expected before we move on to running the client. Here is an example image:

![sample image](https://i.imgur.com/r8xdI7P.png)

On your local machine run:

```
pip install requests click opencv-contrib-python numpy
```

and run the following script with Python >= `3.6.x`. The application expects the argument to be a link to an image. The following link is for the above sample image.


```bash
export YOLOV3_ENDPOINT=api_endpoint_for_yolov3
export CRNN_ENDPOINT=api_endpoint_for_crnn
python sample_inference.py "https://i.imgur.com/r8xdI7P.png"
```

If all goes well, then a prediction will be saved as a JPEG image to disk. By default, it's saved to `prediction.jpg`. Here is the output for the image above:

![annotated sample image](https://i.imgur.com/JaD4A05.jpg)

You can use `python sample_inference.py --help` to find out more. Keep in mind that any detected license plates with a confidence score lower than 80% are discarded.

If this verification works, then we can move on and run the main client.

### Running the Client

Once the APIs are up and running, launch the streaming client by following the instructions at [robertlucian/cortex-license-plate-reader-client](https://github.com/RobertLucian/cortex-license-plate-reader-client).

*Note: The client is kept in a separate repository to maintain the cortex project clean and focused. Keeping some of the projects that are more complex out of this repository can reduce the confusion.*

## Customization/Optimization

### Uploading the Model to S3

The only model to upload to an S3 bucket (for Cortex to deploy) is the *YOLOv3* model. The other two models are downloaded automatically upon deploying the service.

If you would like to host the model from your own bucket, or if you want to fine tune the model for your needs, here's what you can do.

#### Lite Version

Download the *Keras* model:

```bash
wget -O license_plate.h5 "https://www.dropbox.com/s/vsvgoyricooksyv/license_plate.h5?dl=0"
```

And then upload it to your bucket (also make sure [cortex_lite.yaml](cortex_lite.yaml) points to this bucket):

```bash
BUCKET=my-bucket
YOLO3_PATH=examples/tensorflow/license-plate-reader/yolov3_keras
aws s3 cp license_plate.h5 "s3://$BUCKET/$YOLO3_PATH/model.h5"
```

#### Full Version

Download the *SavedModel*:

```bash
wget -O yolov3.zip "https://www.dropbox.com/sh/4ltffycnzfeul01/AAB7Xdmmi59w0EPOwhQ1nkvua/yolov3?dl=0"
```

Unzip it:

```bash
unzip yolov3.zip -d yolov3
```

And then upload it to your bucket (also make sure [cortex_full.yaml](cortex_full.yaml) points to this bucket):

```bash
BUCKET=my-bucket
YOLO3_PATH=examples/tensorflow/license-plate-reader/yolov3_tf
aws s3 cp yolov3/ "s3://$BUCKET/$YOLO3_PATH" --recursive
```

### Configuring YOLOv3 Predictor

The `yolov3` API predictor requires a [config.json](config.json) file to configure the input size of the image (dependent on the model's architecture), the anchor boxes, the object threshold, and the IoU threshold. All of these are already set appropriately so no other change is required.

The configuration file's content is based on [this](https://github.com/experiencor/keras-yolo3/blob/bf37c87561caeccc4f1b879e313d4a3fec1b987e/zoo/config_license_plates.json#L2-L7).

### Opportunities for performance improvements

One way to reduce the inference time is to convert the models to use FP16/BFP16 (in mixed mode or not) and then choose the accelerator that gives the best performance in half precision mode - i.e. T4/V100. A speedup of an order of magnitude can be expected.
