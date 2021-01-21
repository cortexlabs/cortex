# YOLOv5 Detection model

This example deploys a detection model trained using [ultralytics' yolo repo](https://github.com/ultralytics/yolov5) using ONNX.
We'll use the `yolov5s` model as an example here.
In can be used to run inference on youtube videos and returns the annotated video with bounding boxes.

The example can be run on both CPU and on GPU hardware.

## Sample Prediction

Deploy the model by running:

```bash
cortex deploy
```

And wait for it to become live by tracking its status with `cortex get --watch`.

Once the API has been successfully deployed, export the API's endpoint for convenience. You can get the API's endpoint by running `cortex get yolov5-youtube`.

```bash
export ENDPOINT=your-api-endpoint
```

When making a prediction with [sample.json](sample.json), [this](https://www.youtube.com/watch?v=aUdKzb4LGJI) youtube video will be used.

To make a request to the model:

```bash
curl "${ENDPOINT}" -X POST -H "Content-Type: application/json" -d @sample.json --output video.mp4
```

After a few seconds, `curl` will save the resulting video `video.mp4` in the current working directory. The following is a sample of what should be exported:

![yolov5](https://user-images.githubusercontent.com/26958764/86545098-e0dce900-bf34-11ea-83a7-8fd544afa11c.gif)


## Exporting ONNX

To export a custom model from the repo, use the [`model/export.py`](https://github.com/ultralytics/yolov5/blob/master/models/export.py) script.
The only change we need to make is to change the line

```bash
model.model[-1].export = True  # set Detect() layer export=True
```

to

```bash
model.model[-1].export = False
```

Originally, the ultralytics repo does not export postprocessing steps of the model, e.g. the conversion from the raw CNN outputs to bounding boxes.
With newer ONNX versions, these can be exported as part of the model making the deployment much easier.

With this modified script, the ONNX graph used for this example has been exported using
```bash
python models/export.py --weights weights/yolov5s.pt --img 416 --batch 1
```
