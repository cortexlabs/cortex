import signal
import time
import base64
import pickle
import json
import picamera
import cv2
import logging
import requests
import queue
import threading as td
import multiprocessing as mp
import numpy as np
from utils.bbox import BoundBox
from requests_toolbelt.adapters.source import SourceAddressAdapter

logger = logging.getLogger(__name__)
stream_handler = logging.StreamHandler()
stream_handler.setLevel(logging.DEBUG)
stream_format = logging.Formatter("%(asctime)s - %(name)s - %(threadName)s - %(levelname)s - %(message)s")
stream_handler.setFormatter(stream_format)
logger.addHandler(stream_handler)
logger.setLevel(logging.DEBUG)

api_endpoint = "http://Roberts-MacBook-2.local/predict"
# api_endpoint = "http://a8ebac3e744d111ea981b06a64f9c668-1180365019.eu-central-1.elb.amazonaws.com/yolov3"

session = requests.Session()
interface_ip = "192.168.0.59"
session.mount('http://', SourceAddressAdapter(interface_ip))
session.mount('https://', SourceAddressAdapter(interface_ip))

class GracefullKiller():
    kill_now = False
    def __init__(self):
        signal.signal(signal.SIGINT, self.exit_gracefully)
        signal.signal(signal.SIGTERM, self.exit_gracefully)
    
    def exit_gracefully(self, signum, frame):
        self.kill_now = True

class InferenceWorker(td.Thread):
    def __init__(self, event_stopper, in_queue, out_queue, name=None):
        super(InferenceWorker, self).__init__(name=name)
        self.event_stopper = event_stopper
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.rtt_ms = None

    def run(self):
        logger.debug("worker started")
        while not self.event_stopper.is_set():
            self.cloud_infer()
            time.sleep(0.001)
        logger.debug("worker stopped")

    def cloud_infer(self):
        try:
            data = self.in_queue.get_nowait()
        except queue.Empty:
            # logger.warning("no data available for worker")
            return

        # extract frame
        frame_num = data["frame_num"]
        img = data["jpeg"]

        # preprocess/compress the image
        image = self.image_from_bytes(img)
        reduced = self.compress_image(image)
        byte_im = self.image_to_jpeg_bytes(reduced)
        # logger.debug("reduced image size from {}KB down to {}KB".format(len(img) // 1024, len(byte_im) // 1024))

        # encode image
        img_enc = base64.b64encode(byte_im).decode("utf-8")
        dump = json.dumps({"img": img_enc})

        # make inference request
        try:
            start = time.time()
            resp = session.post(api_endpoint, 
            data=dump, headers={'content-type':'application/json'}, timeout=1.000)
            end = time.time()
        except requests.exceptions.Timeout as e:
            logger.warning("timeout on inference request")
            return
        except:
            logger.warning("timing/connection error")
            return

        # calculate average rtt (use complementary filter)
        current = int((end - start) * 1000)
        if not self.rtt_ms:
            self.rtt_ms = current
        else:
            self.rtt_ms = self.rtt_ms * 0.98 + current * 0.02
        
        # parse response
        r_dict = resp.json()
        boxes_raw = r_dict["boxes"]
        boxes = []
        for box in boxes_raw:
            boxes.append(BoundBox(*box))
        logger.debug("Frame Count: {} - Avg RTT: {} - Detected: {}".format(frame_num, self.rtt_ms, len(boxes) != 0))
        
        # push data for further processing in the queue
        output = {
            "boxes": boxes,
            "frame_num": frame_num,
            "avg_rtt": self.rtt_ms,
            "image": img
        }

        self.out_queue.put(output)

    def compress_image(self, image, desired_square_height=416, top_crop_percent=0.45):
        image = cv2.cvtColor(image, cv2.COLOR_RGB2GRAY)
        width = image.shape[1]
        scale_percent = desired_square_height / width
        width = int(image.shape[1] * scale_percent)
        height = int(image.shape[0] * scale_percent)
        image = cv2.resize(image, (width, height), interpolation = cv2.INTER_AREA)
        image[:int(height * top_crop_percent)] = 128

        return image

    def image_from_bytes(self, byte_im):
        nparr = np.frombuffer(byte_im, np.uint8)
        img_np = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        return img_np

    def image_to_jpeg_bytes(self, image):
        is_success, im_buf_arr = cv2.imencode(".jpg", image)
        byte_im = im_buf_arr.tobytes()
        return byte_im

class WorkerPool(mp.Process):
    def __init__(self, in_queue, out_queue, pool_size, name=None, daemon=None):
        super(WorkerPool, self).__init__(name=name, daemon=daemon)
        self.event_stopper = mp.Event()
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.pool_size = pool_size

    def run(self):
        logger.info("spawning workers on separate process")
        pool = [InferenceWorker(self.event_stopper, self.in_queue, self.out_queue, "Worker-{}".format(i)) for i in range(self.pool_size)]
        [worker.start() for worker in pool]
        while not self.event_stopper.is_set():
            time.sleep(0.001)
        [worker.join() for worker in pool]

    def stop(self):
        self.event_stopper.set()

class DistributeFramesAndInfer():
    def __init__(self, no_workers):
        self.frame_num = 0
        self.in_queue = mp.Queue()
        self.out_queue = mp.Queue()
        self.no_workers = no_workers
        self.pool = WorkerPool(self.in_queue, self.out_queue, self.no_workers, name="WorkerPool")
        self.pool.start()

    def write(self, buf):
        if buf.startswith(b"\xff\xd8"):
            # start of new frame; close the old one (if any) and
            self.in_queue.put({
                "frame_num": self.frame_num,
                "jpeg": buf
            })
            self.frame_num += 1

    def stop(self):
        self.pool.stop()
        self.pool.join()
        qs = [self.in_queue, self.out_queue]
        [q.cancel_join_thread() for q in qs]

def main():
    killer = GracefullKiller()
    workers = 12
    logger.info("initializing pool w/ " + str(workers) + " workers")
    output = DistributeFramesAndInfer(workers)
    logger.info("initialized worker pool")

    with picamera.PiCamera() as camera:
        camera.sensor_mode = 5
        camera.resolution = (1280, 720)
        camera.framerate = 30
        logger.info("picamera initialized w/ mode={} resolution={} framerate={}".format(
            camera.sensor_mode, camera.resolution, camera.framerate
        ))

        camera.start_recording(output="recording.h264", format="h264", splitter_port=0, bitrate=10000000)
        camera.start_recording(output=output, format="mjpeg", splitter_port=1, bitrate=17000000, quality=100)
        logger.info("started recording to file and to queue")

        while not killer.kill_now:
            camera.wait_recording(timeout=0.5, splitter_port=0)
            camera.wait_recording(timeout=0.5, splitter_port=1)
            logger.info('avg producing/consuming queue size: {}, {}'.format(output.in_queue.qsize(), output.out_queue.qsize()))

        logger.info("gracefully exiting")
        camera.stop_recording(splitter_port=0)
        camera.stop_recording(splitter_port=1)
        output.stop() 

if __name__ == "__main__":
    main()