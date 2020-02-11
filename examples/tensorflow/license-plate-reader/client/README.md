# Client for License Plate Identification System

This app which uses the deployed APIs as a PaaS captures the frames from a video camera, sends them for inferencing to the cortex APIs, recombines them after the responses are received and then the detections/recognitions are overlayed on the output stream.

The app must be configured to use the API endpoints as shown when calling `cortex get yolov3` and `cortex get crnn`.

The app also saves a `csv` file containing the dates and GPS coordinates of each identified license plate.

The observable latency between capturing the frame and broadcasting the predictions in the browser (with all the inference stuff going on) takes about 0.5-0.7 seconds.

To learn more about how the actual device was constructed, check out [this](https://www.robertlucian.com/2020/02/12/real-time-license-plate-identification/) article.

## Target Machine

Target machine **1**: Raspberry Pi.

Target machine **2**: Any x86 machine.

The app's primary target machine is the *Raspberry Pi* (3/3B+/4) - the Raspberry Pi is a small embedded computer system that got adopted by many hobbyists and institutions all around the world as the de-facto choice for hardware/software experiments.

Unfortunately for the Raspberry Pi, the $35 pocket-sized computer, doesn't have enough oomph (far from it) to do any inference: not only it doesn't have enough memory to load a model, but should it have enough RAM, it would still take dozens of minutes just to get a single inference. Let alone run inferences at 30 FPS, with a number of inferences for each frame.

The app is built to be used with a *Pi Camera* alongside the *Raspberry Pi*. The details on how to build such a system are found [here](#creating-your-own-device).

Since many developers don't have a Raspberry Pi laying around or are just purely interested in seeing results right away, the app can also be configured to take in a video file and treat it exactly as if it was a camera.

## Dependencies

For either target machine, the minimum version of Python has to be `3.6.x`.

#### For the Raspberry Pi

```bash
pip3 install --user -r deps/requirements_rpi.txt
sudo apt-get update
# dependencies for opencv and pandas package
sudo apt-get install --no-install-recommends $(cat deps/requirements_dpkg_rpi.txt)
```

#### For x86 Machine

```bash
pip3 install -r deps/requirements_base.txt
```

## Configuring

The configuration file can be in this form
```jsonc
{
    "video_source": {
        // "file" for reading from file or "camera" for pi camera
        "type": "file",
        // video file to read from; applicable just for "file" type
        "input": "airport_ride_480p.mp4",
        // scaling for the video file; applicable just for "file" type
        "scale_video": 1.0,
        // how many frames to skip on the video file; applicable just for "file" type
        "frames_to_skip": 0,
        // framerate
        "framerate": 30
        // camera sensor mode; applicable just for "camera" type
        // "sensor_mode": 5
        // where to save camera's output; applicable just for "camera" type
        // "output_file": "recording.h264"
    },
    "broadcaster": {
        // when broadcasting, a buffer is required to provide framerate fluidity; measured in frames
        "target_buffer_size": 5,
        // how much the size of the buffer can variate +-
        "max_buffer_size_variation": 5,
        // the maximum variation of the fps when extracting frames from the queue
        "max_fps_variation": 15,
        // target fps - must match the camera/file's framerate
        "target_fps": 30,
        // address to bind the web server to
        "serve_address": ["0.0.0.0", 8000]
    },
    "inferencing_worker": {
        // YOLOv3's input size of the image in pixels (must consider the existing model)
        "yolov3_input_size_px": 416,
        // when drawing the bounding boxes, use a higher res image to draw boxes more precisely
        // (this way text is more readable)
        "bounding_boxes_upscale_px": 640,
        // object detection accuracy threshold in percentages with range (0, 1)
        "yolov3_obj_thresh": 0.8,
        // the jpeg quality of the image for the CRAFT/CRNN models in percentanges;
        // these models receive the cropped images of each detected license plate
        "crnn_quality": 98,
        // broadcast quality - aim for a lower value since this stream doesn't influence the predictions; measured in percentages
        "broadcast_quality": 90,
        // connection timeout for both API endpoints measured in seconds
        "timeout": 1.20,
        // YOLOv3 API endpoint
        "api_endpoint_yolov3": "http://a23893c574c0511ea9f430a8bed50c69-1100298247.eu-central-1.elb.amazonaws.com/yolov3",
        // CRNN API endpoint
        // Can be set to "" value to turn off the recognition inference
        // By turning it off, the latency is reduced and the output video appears smoother
        "api_endpoint_crnn": "http://a23893c574c0511ea9f430a8bed50c69-1100298247.eu-central-1.elb.amazonaws.com/crnn"
    },
    "inferencing_pool": {
        // number of workers to do inferencing (YOLOv3 + CRAFT + CRNN)
        // depending on the source's framerate, a balance must be achieved
        "workers": 20,
        // pick the nth frame from the input stream
        // if the input stream runs at 30 fps, then setting this to 2 would act
        // as if the input stream runs at 30/2=15 fps
        // ideally, you have a high fps camera (90-180) and you only pick every 3rd-6th frame
        "pick_every_nth_frame": 1
    },
    "flusher": {
        // if there are more than this given number of frames in the input stream's buffer, flush them
        // it's useful if the inference workers (due to a number of reasons) can't keep up with the input flow
        // also keeps the current broadcasted stream up-to-date with the reality
        "frame_count_threshold": 5
    },
    "gps": {
        // set to false when using a video file to read the stream from
        // set to true when you have a GPS connected to your system
        "use_gps": false,
        // port to write to to activate the GPS (built for EC25-E modules)
        "write_port": "/dev/ttyUSB2",
        // port to read from in NMEA standard
        "read_port": "/dev/ttyUSB1",
        // baudrate as measured in bits/s
        "baudrate": 115200
    },
    "general": {
        // to which IP the requests module is bound to
        // useful if you only want to route the traffic through a specific interface
        "bind_ip": "0.0.0.0",
        // where to save the csv data containing the date, the predicted license plate number and GPS data
        // can be an empty string, in which case, nothing is stored
        "saved_data": "saved_data.csv"
    }
}
```

Be aware that for having a functional application, the minimum amount of things that have to be adjusted in the config file are:

1. The input file in case you are using a video to feed the application with. You can download the following `mp4` video file to use as input. Download it by running `wget -O airport_ride_480p.mp4 "https://www.dropbox.com/s/q9j57y5k95wg2zt/airport_ride_480p.mp4?dl=0"`
1. Both API endpoints from your cortex APIs. Use `cortex get your-api-name-here` command to get those.

## Running It

Make sure both APIs are already running in the cluster. Launching it the first time might raise some timeout exceptions, but let it run for a few moments. If there's enough compute capacity, you'll start getting `200` response codes.

Run it like
```bash
python app.py -c config.json
```

Once it's running, you can head off to its browser page to see the live broadcast with its predictions overlayed on top.

To save the broascasted MJPEG stream, you can run the following command

```bash
PORT=8000
FRAMERATE=30
ffmpeg -i http://localhost:PORT/stream.mjpg -an -vcodec libx264 -r FRAMERATE saved_video.h264
```

To terminate the app, press `CTRL-C` and wait a bit.

## Creating Your Own Device

![Imgur](https://i.imgur.com/MvDAXWU.jpg)

To create your own Raspberry Pi-powered device to record and display the predictions in real time in your car, you're gonna need the following things:
1. A Raspberry Pi - preferably a 4, because that one has more oomph.
1. A Pi Camera - doesn't matter which version of it.
1. A good buck converter to step-down from 12V down to 5V - aim for 4-5 amps. You can use a SBEC/UBEC/BEC regulators - they are easy to find and cheap.
1. A power outlet for the car's cigarette port to get the 12V DC.
1. 4G/GPS shield to host a GSM module - in this project, an EC25-E module has been used. You will also need antennas.
1. A 3D-printed support to hold the electronics and be able to hold it against the rear mirror or dashboard. Must be built to accomodate to your own car.

Without convoluting this README too much:

* Here are the [STLs/SLDPRTs/Renders](https://www.dropbox.com/sh/fw16vy1okrp606y/AAAwkoWXODmoaOP4yR-z4T8Va?dl=0) to the car's 3D printed support.
* Here's an [article](https://www.robertlucian.com/2020/02/12/real-time-license-plate-identification/) that talks about this in full detail.
