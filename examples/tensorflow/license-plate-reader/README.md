# Real-Time License Plate Identification System

This project is about a license plate identification system. On resource-constraint systems, running inferences may prove to be too computationally expensive or just plain impossible. That's why a solution is to run the ML in the cloud and have the local (embedded) system act as a client of these services.

![Imgur](https://i.imgur.com/jgkJB59.gif)

*Figure 1 - GIF taken from this real-time recording [video](https://www.youtube.com/watch?v=gsYEZtecXlA) of predictions*

![Imgur](https://i.imgur.com/MvDAXWU.jpg)

*Figure 2 - Raspberry Pi-powered client with 4G access and onboard GPS that connects to cortex's APIs for inference purposes. More on that [here](client/README.md#creating-your-own-device).*

In our example, we assume we have a dashcam mounted on a car and we want to detect and recognize all license plates in the stream in real-time. We can use an embedded computer system to do the video recording, then have that streamed and inferred frame-by-frame using a web service, relay back the results, reassemble the stream and finally display the stream on a screen. The web service in our case is a set of 2 web APIs deployed using cortex.

## Used Models

The identification of license plates is done in 3 steps:
1. Detecting the bounding boxes of each license plate using *YOLOv3* model.
1. Detecting the very specific region of each word inside each bounding box with high accuracy using a pretrained *CRAFT* text detector.
1. Recognizing the text inside the previously detected boxes using a pretrained *CRNN* model.

Out of all these 3 models (*YOLOv3*, *CRAFT* and *CRNN*) only *YOLOv3* has been fine-tuned with a rather small dataset to better work with license plates. This dataset can be found [here](https://github.com/RobertLucian/license-plate-dataset). This *YOLOv3* model has in turn been trained using [this](https://github.com/experiencor/keras-yolo3) GitHub project. To get more details about our fine-tuned model, check the project's description page.

The other 2 models, *CRAFT* and *CRNN*, are to be found in [keras-ocr](https://github.com/faustomorales/keras-ocr), which is a nicely packed and flexible version of these 2 models.

## Uploading the SavedModel to S3

The only model that has to be uploaded to an S3 bucket to be further used by cortex when being deployed is the *YOLOv3* model. The other 2 are downloaded automatically upon deploying the service.

*Note: The Keras model from [here](https://github.com/experiencor/keras-yolo3) has been converted to SavedModel model instead.*

Download the *SavedModel* model
```bash
wget -O yolov3.zip "https://www.dropbox.com/sh/4ltffycnzfeul01/AAB7Xdmmi59w0EPOwhQ1nkvua/yolov3?dl=0"
```
Unzip it
```bash
unzip yolov3.zip -d yolov3
```
And then upload it to your bucket (which also has to be the one used by your cortex cluster). Make sure [cortex.yaml](cortex.yaml) also has the same bucket set up.
```bash
BUCKET=cortex-bucketname
YOLO3_PATH=examples/tensorflow/license-plate-reader/yolov3
aws s3 cp yolov3/ "s3://$BUCKET/$YOLO3_PATH" --recursive
```

## Configuring YOLOv3 Predictor

The `yolov3` API predictor requires a [config.json](config.json) file to configure the input size of the image (dependent on the model's architecture), the anchor boxes, the object threshold and the IoU threshold. All of these are already set appropriately so no other change is required.

The configuration file's content is based on [this](https://github.com/experiencor/keras-yolo3/blob/bf37c87561caeccc4f1b879e313d4a3fec1b987e/zoo/config_license_plates.json#L2-L7) one.

## Deploying

Before executing `cortex deploy`, make sure you've got these 3 covered:

1. The recommended number of instances to run this smoothly is about 20 instances equipped with GPUs. It doesn't really matter how powerful the GPU is, just pick the cheapest ones (K80s or T4s) since we're already limited by how many requests the web server can answer to concurrently. `cortex.yaml` is already set up to use the 20 instances. This won't be a problem in the very near future when `waitress` will be replaced with a multiprocessed WSGI server (i.e. `gunicorn`).  A multiprocessed WSGI will allow for the GPU to be fully utilized, thus reducing the cluster's cost/hr.

1. Use the same S3 bucket for both the model and the cluster.

If you don't have access to this many GPU-equipped instances, you could just lower the number and expect dropped frames. It will still prove the point, albeit at a much lower framerate and reliability. Also, the latency will suffer considerably. More on that [here](client/README.md).

Then after the cortex cluster is created, hit
```bash
cortex deploy
```
And wait for its magic by monitoring the APIs with
```bash
cortex get --watch
```

#### Note

One other way to reduce the inference time is to convert the models to use FP16/BFP16, in mixed mode or not and then choose the accelerator that gives the best performance in half precision mode - i.e. T4/V100. A difference of an order of magnitude can be expected.

## Launching the Client

Once the APIs are up and running, go on and launch the client by following the instructions [here](client/README.md).
