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
import broadcast
from random import randint
from utils.bbox import BoundBox, draw_boxes
from requests_toolbelt.adapters.source import SourceAddressAdapter

logger = logging.getLogger(__name__)
stream_handler = logging.StreamHandler()
stream_handler.setLevel(logging.DEBUG)
stream_format = logging.Formatter("%(asctime)s - %(name)s - %(threadName)s - %(levelname)s - %(message)s")
stream_handler.setFormatter(stream_format)
logger.addHandler(stream_handler)
logger.setLevel(logging.DEBUG)

# api_endpoint = "http://Roberts-MacBook-2.local/predict"
api_endpoint = "http://a065a55b5459e11eaa4af06b95661afb-914798017.eu-central-1.elb.amazonaws.com/yolov3"

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

class WorkerTemplate(td.Thread):
    def __init__(self, event_stopper, name=None):
        super(WorkerTemplate, self).__init__(name=name)
        self.event_stopper = event_stopper
        self.runnable = None

    def run(self):
        if self.runnable:
            logger.debug("worker started")
            while not self.event_stopper.is_set():
                self.runnable()
                time.sleep(0.001)
            logger.debug("worker stopped")

    def stop(self):
        self.event_stopper.set()

    def compress_image(self, image, grayscale=True, desired_width=416, top_crop_percent=0.45):
        if grayscale:
            image = cv2.cvtColor(image, cv2.COLOR_RGB2GRAY)
        width = image.shape[1]
        scale_percent = desired_width / width
        width = int(image.shape[1] * scale_percent)
        height = int(image.shape[0] * scale_percent)
        image = cv2.resize(image, (width, height), interpolation = cv2.INTER_AREA)
        if top_crop_percent:
            image[:int(height * top_crop_percent)] = 128

        return image

    def image_from_bytes(self, byte_im):
        nparr = np.frombuffer(byte_im, np.uint8)
        img_np = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        return img_np

    def image_to_jpeg_bytes(self, image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
        is_success, im_buf_arr = cv2.imencode(".jpg", image, quality)
        byte_im = im_buf_arr.tobytes()
        return byte_im 

class BroadcastReassembled(WorkerTemplate):
    def __init__(self, event_stopper, in_queue, serve_address, name=None):
        super(BroadcastReassembled, self).__init__(event_stopper=event_stopper, name=name)
        self.in_queue = in_queue
        self.serve_address = serve_address
        
        self.rtt_ms = None
        self.detections = 0
        self.current_detections = 0
        self.buffer = []
        self.oldest_broadcasted_frame = 0
        self.target_buffer_size = 5
        self.max_buffer_size_variation = 5
        self.max_fps_variation = 15
        self.target_fps = 30

    def run(self):
        def lambda_func():
            server = broadcast.StreamingServer(self.serve_address, broadcast.StreamingHandler)
            server.serve_forever()

        td.Thread(
            target=lambda_func, 
            args=(),
            daemon=True).start()
        logger.debug("listening for stream on {}".format(self.serve_address))

        logger.debug("worker started")
        counter = 0
        while not self.event_stopper.is_set():
            if counter == self.target_fps:
                logger.info("buffer queue size: {}".format(len(self.buffer)))
                counter = 0
            self.reassemble()
            time.sleep(0.001)
            counter += 1
        logger.debug("worker stopped")

    def reassemble(self):
        start = time.time()
        self.pull_and_push()
        self.purge_stale_frames()
        frame, delay = self.pick_new_frame()
        # delay loop to stabilize the video fps
        end = time.time()
        elapsed_time = end - start
        elapsed_time += 0.001 # count in the millisecond in self.run
        if delay - elapsed_time > 0.0:
            time.sleep(delay - elapsed_time)
        if frame:
            # pull and push again in case 
            # write buffer (assume it takes an insignificant time to execute)
            self.pull_and_push()
            broadcast.output.write(frame)

    def pull_and_push(self):
        try:
            data = self.in_queue.get_nowait()
        except queue.Empty:
            # logger.warning("no data available for worker")
            return

        # extract data
        boxes = data["boxes"]
        frame_num = data["frame_num"]
        rtt = data["avg_rtt"]
        byte_im = data["image"]

        # run statistics
        self.statistics(rtt, len(boxes))

        # push frames to buffer and pick new frame
        self.buffer.append({
            "image": byte_im,
            "frame_num": frame_num
        })

    def purge_stale_frames(self):
        new_buffer = []
        for frame in self.buffer:
            if frame["frame_num"] > self.oldest_broadcasted_frame:
                new_buffer.append(frame)
        self.buffer = new_buffer
    
    def pick_new_frame(self):
        current_desired_fps = self.target_fps - self.max_fps_variation
        delay = 1 / current_desired_fps
        if len(self.buffer) == 0:
            return None, delay

        newlist = sorted(self.buffer, key=lambda k: k["frame_num"])
        idx_to_del = 0
        for idx, frame in enumerate(newlist):
            if frame["frame_num"] < self.oldest_broadcasted_frame:
                idx_to_del = idx + 1
        newlist = newlist[idx_to_del:]
                
        if len(newlist) == 0:
            return None, delay
        
        self.buffer = newlist[::-1]
        element = self.buffer.pop()
        frame = element["image"]
        self.oldest_broadcasted_frame = element["frame_num"]

        size = len(self.buffer)
        variation = size - self.target_buffer_size
        var_perc = variation / self.max_buffer_size_variation
        current_desired_fps = self.target_fps + var_perc * self.max_fps_variation
        if current_desired_fps < 0:
            current_desired_fps = self.target_fps - self.max_fps_variation
        try:
            delay = 1 / current_desired_fps
        except ZeroDivisionError:
            current_desired_fps = self.target_fps - self.max_fps_variation
            delay = 1 / current_desired_fps

        return frame, delay

    def statistics(self, rtt_ms, detections):
        if not self.rtt_ms:
            self.rtt_ms = rtt_ms
        else:
            self.rtt_ms = self.rtt_ms * 0.98 + rtt_ms * 0.02
        self.detections += detections
        self.current_detections = detections


class InferenceWorker(WorkerTemplate):
    def __init__(self, event_stopper, in_queue, out_queue, name=None):
        super(InferenceWorker, self).__init__(event_stopper=event_stopper, name=name)
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.rtt_ms = None
        self.runnable = self.cloud_infer

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
        # reduced = self.compress_image(image, grayscale=False, desired_width=480, top_crop_percent=None)
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
        # old_width = image.shape[1]
        upscale_width = 640
        new_width = 416
        # scale_percent = old_width / new_width
        scale_percent = 640 / new_width
        for box in boxes_raw:
            b = BoundBox(*box)
            b.xmin = int(b.xmin * scale_percent)
            b.ymin = int(b.ymin * scale_percent)
            b.xmax = int(b.xmax * scale_percent)
            b.ymax = int(b.ymax * scale_percent)
            boxes.append(b)
        logger.debug("Frame Count: {} - Avg RTT: {} - Detected: {}".format(frame_num, self.rtt_ms, len(boxes) != 0))
        
        # draw detections
        upscaled = self.compress_image(image, grayscale=False, desired_width=upscale_width, top_crop_percent=False)
        draw_image = draw_boxes(upscaled, boxes, labels=["license-plate"], obj_thresh=0.8)
        draw_byte_im = self.image_to_jpeg_bytes(draw_image, [int(cv2.IMWRITE_JPEG_QUALITY), 80])

        # base = 0.200
        # variation = randint(50, 100) / 1000
        # delay = base + variation
        # time.sleep(delay)

        # push data for further processing in the queue
        output = {
            "boxes": boxes,
            "frame_num": frame_num,
            "avg_rtt": self.rtt_ms,
            "image": draw_byte_im
        }

        self.out_queue.put(output)

class Flusher(WorkerTemplate):
    def __init__(self, event_stopper, queue, threshold, name=None):
        super(Flusher, self).__init__(event_stopper=event_stopper, name=name)
        self.queue = queue
        self.threshold = threshold
        self.runnable = self.flush_pipe

    def flush_pipe(self):
        current = self.queue.qsize()
        if current > self.threshold:
            try:
                for i in range(current):
                    self.queue.get_nowait()
                logger.warning("flushed {} elements from the producing queue".format(current))
            except queue.Empty:
                logger.warning("flushed too many elements from the queue")
        time.sleep(0.5)

class WorkerPool(mp.Process):
    def __init__(self, name, worker, pool_size, *args, **kwargs):
        super(WorkerPool, self).__init__(name=name)
        self.event_stopper = mp.Event()
        self.Worker = worker
        self.pool_size = pool_size
        self.args = args
        self.kwargs = kwargs

    def run(self):
        logger.info("spawning workers on separate process")
        pool = [self.Worker(self.event_stopper, *self.args, **self.kwargs, name="{}-Worker-{}".format(self.name, i)) for i in range(self.pool_size)]
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
        self.pool = WorkerPool("InferencePool", InferenceWorker, self.no_workers, self.in_queue, self.out_queue)
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
    
    # workers on a separate process to run inference on the data
    workers = 20
    logger.info("initializing pool w/ " + str(workers) + " workers")
    output = DistributeFramesAndInfer(workers)
    logger.info("initialized worker pool")

    # a single worker in a separate process to reassemble the data
    reassembler = WorkerPool("BroadcastReassembled", BroadcastReassembled, 1, output.out_queue, serve_address=("", 8000))
    reassembler.start()

    # a single thread to flush the producing queue
    # when there are too many frames in the pipe
    flusher = Flusher(td.Event(), output.in_queue, threshold=30, name="Flusher")
    flusher.start()

    with picamera.PiCamera() as camera:
        camera.sensor_mode = 5
        camera.resolution = (480, 270)
        camera.framerate = 30
        logger.info("picamera initialized w/ mode={} resolution={} framerate={}".format(
            camera.sensor_mode, camera.resolution, camera.framerate
        ))

        camera.start_recording(output="recording.h264", format="h264", splitter_port=0, bitrate=10000000)
        camera.start_recording(output=output, format="mjpeg", splitter_port=1, bitrate=10000000, quality=95)
        logger.info("started recording to file and to queue")

        while not killer.kill_now:
            camera.wait_recording(timeout=0.5, splitter_port=0)
            camera.wait_recording(timeout=0.5, splitter_port=1)
            logger.info('avg producing/consuming queue size: {}, {}'.format(output.in_queue.qsize(), output.out_queue.qsize()))

        logger.info("gracefully exiting")
        camera.stop_recording(splitter_port=0)
        camera.stop_recording(splitter_port=1)
        output.stop() 

    reassembler.stop()
    flusher.stop()

if __name__ == "__main__":
    main()