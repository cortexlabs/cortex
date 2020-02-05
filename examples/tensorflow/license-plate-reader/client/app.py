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
import copy
from random import randint
from utils.bbox import BoundBox, draw_boxes
from requests_toolbelt.adapters.source import SourceAddressAdapter

logger = logging.getLogger(__name__)
stream_handler = logging.StreamHandler()
stream_handler.setLevel(logging.INFO)
stream_format = logging.Formatter("%(asctime)s - %(name)s - %(threadName)s - %(levelname)s - %(message)s")
stream_handler.setFormatter(stream_format)
logger.addHandler(stream_handler)
logger.setLevel(logging.DEBUG)

# api_endpoint = "http://Roberts-MacBook-2.local/predict"
API_ENDPOINT_YOLO3 = "http://ab8ce587e47a011ea9f4a0ad90ccbc5a-839674254.eu-central-1.elb.amazonaws.com/yolov3"
API_ENDPOINT_RCNN = "http://ab8ce587e47a011ea9f4a0ad90ccbc5a-839674254.eu-central-1.elb.amazonaws.com/crnn"

session = requests.Session()
interface_ip = "192.168.0.46"
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
                time.sleep(0.030)
            logger.debug("worker stopped")

    def stop(self):
        self.event_stopper.set()

    def resize_image(self, image, desired_width):
        current_width = image.shape[1]
        scale_percent = desired_width / current_width
        width = int(image.shape[1] * scale_percent)
        height = int(image.shape[0] * scale_percent)
        resized = cv2.resize(image, (width, height), interpolation = cv2.INTER_AREA)
        return resized

    def compress_image(self, image, grayscale=True, desired_width=416, top_crop_percent=0.45):
        if grayscale:
            image = cv2.cvtColor(image, cv2.COLOR_RGB2GRAY)
        image = self.resize_image(image, desired_width)
        height = image.shape[0]
        if top_crop_percent:
            image[:int(height * top_crop_percent)] = 128

        return image

    def image_from_bytes(self, byte_im):
        nparr = np.frombuffer(byte_im, np.uint8)
        img_np = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        return img_np
    
    def image_to_jpeg_nparray(self, image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
        is_success, im_buf_arr = cv2.imencode(".jpg", image, quality)
        return im_buf_arr

    def image_to_jpeg_bytes(self, image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
        buf = self.image_to_jpeg_nparray(image, quality)
        byte_im = buf.tobytes()
        return byte_im 

class BroadcastReassembled(WorkerTemplate):
    def __init__(self, event_stopper, in_queue, serve_address, name=None):
        super(BroadcastReassembled, self).__init__(event_stopper=event_stopper, name=name)
        self.in_queue = in_queue
        self.serve_address = serve_address
        
        self.yolo3_rtt = None
        self.crnn_rtt = None
        self.detections = 0
        self.current_detections = 0
        self.recognitions = 0
        self.current_recognitions = 0
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
        yolo3_rtt = data["avg_yolo3_rtt"]
        crnn_rtt = data["avg_crnn_rtt"]
        byte_im = data["image"]

        # run statistics
        self.statistics(yolo3_rtt, crnn_rtt, len(boxes), 0)

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

    def statistics(self, yolo3_rtt, crnn_rtt, detections, recognitions):
        if not self.yolo3_rtt:
            self.yolo3_rtt = yolo3_rtt
        else:
            self.yolo3_rtt = self.yolo3_rtt * 0.98 + yolo3_rtt * 0.02
        if not self.crnn_rtt:
            self.crnn_rtt = crnn_rtt
        else:
            self.crnn_rtt = self.crnn_rtt * 0.98 + crnn_rtt * 0.02

        self.detections += detections
        self.current_detections = detections
        self.recognitions += recognitions
        self.current_recognitions = recognitions


class InferenceWorker(WorkerTemplate):
    def __init__(self, event_stopper, in_queue, out_queue, name=None):
        super(InferenceWorker, self).__init__(event_stopper=event_stopper, name=name)
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.rtt_yolo3_ms = None
        self.rtt_crnn_ms = 0
        self.runnable = self.cloud_infer

    def cloud_infer(self):
        try:
            data = self.in_queue.get_nowait()
        except queue.Empty:
            # logger.warning("no data available for worker")
            return

        #############################

        # extract frame
        frame_num = data["frame_num"]
        img = data["jpeg"]
        # preprocess/compress the image
        image = self.image_from_bytes(img)
        reduced = self.compress_image(image)
        byte_im = self.image_to_jpeg_bytes(reduced)
        # encode image
        img_enc = base64.b64encode(byte_im).decode("utf-8")
        img_dump = json.dumps({"img": img_enc})

        # make inference request
        resp = self.yolov3_api_request(img_dump)
        if not resp:
            return

        #############################
        
        # parse response
        r_dict = resp.json()
        boxes_raw = r_dict["boxes"]
        boxes = []
        for b in boxes_raw:
            box = BoundBox(*b)
            boxes.append(box)

        # purge bounding boxes with a low confidence score
        obj_thresh = 0.8
        aux = []
        for b in boxes:
            label = -1
            for i in range(len(b.classes)):
                if b.classes[i] > obj_thresh:
                    label = i
            if label >= 0: aux.append(b)
        boxes = aux
        del aux

        # also scale the boxes for later uses
        yolo3_image_width = 416
        upscale640_width = 640
        camera_source_width = image.shape[1]
        boxes640 = self.scale_bbox(boxes, yolo3_image_width, upscale640_width)
        boxes_source = self.scale_bbox(boxes, yolo3_image_width, camera_source_width)

        #############################
        
        # recognize the license plates in case
        # any bounding boxes have been detected
        dec_lps = []
        if len(boxes) > 0:
            # create set of images of the detected license plates
            try:
                for b in boxes_source:
                    lp = image[b.ymin:b.ymax, b.xmin:b.xmax]
                    jpeg = self.image_to_jpeg_nparray(lp)
                    lps.append(jpeg)
            except:
                logger.warning("encountered error while converting to jpeg")
                pass

            lps = pickle.dumps(lps, protocol=0)
            lps_enc = base64.b64encode(lps).decode("utf-8")
            lps_dump = json.dumps({"imgs": lps_enc})

            # make request to rcnn API
            dec_lps = self.rcnn_api_request(lps_dump)
            self.reorder_lps(dec_lps)

        if len(dec_lps) > 0:
            logger.info("Detected the following words: {}".format(dec_lps))

        #############################
        
        # draw detections
        upscaled = self.resize_image(image, upscale640_width)
        draw_image = draw_boxes(upscaled, boxes640, labels=["LP"], obj_thresh=obj_thresh)
        draw_byte_im = self.image_to_jpeg_bytes(draw_image, [int(cv2.IMWRITE_JPEG_QUALITY), 80])

        #############################

        # push data for further processing in the queue
        output = {
            "boxes": boxes,
            "frame_num": frame_num,
            "avg_yolo3_rtt": self.rtt_yolo3_ms,
            "avg_crnn_rtt": self.rtt_crnn_ms,
            "image": draw_byte_im
        }
        self.out_queue.put(output)

        logger.info("Frame Count: {} - Avg YOLO3 RTT: {} - Avg CRNN RTT: {} - Detected: {}".format(
        frame_num, self.rtt_yolo3_ms, self.rtt_crnn_ms, len(boxes)))

    def scale_bbox(self, boxes, old_width, new_width):
        boxes = copy.deepcopy(boxes)
        scale_percent = new_width / old_width
        for b in boxes:
            b.xmin = int(b.xmin * scale_percent)
            b.ymin = int(b.ymin * scale_percent)
            b.xmax = int(b.xmax * scale_percent)
            b.ymax = int(b.ymax * scale_percent)
        return boxes

    def yolov3_api_request(self, img_dump, timeout=1.200):
        # make inference request
        try:
            start = time.time()
            resp = None
            resp = session.post(API_ENDPOINT_YOLO3, 
            data=img_dump, headers={'content-type':'application/json'}, timeout=timeout)
        except requests.exceptions.Timeout as e:
            logger.warning("timeout on yolov3 inference request")
            time.sleep(0.10)
            return None
        except Exception as e:
            time.sleep(0.10)
            logger.warning("timing/connection error on yolov3", exc_info=True)
            return None
        finally:
            end = time.time()
            if not resp:
                pass
            elif resp.status_code != 200:
                logger.warning("received {} status code from yolov3 api".format(resp.status_code))
                return None

        # calculate average rtt (use complementary filter)
        current = int((end - start) * 1000)
        if not self.rtt_yolo3_ms:
            self.rtt_yolo3_ms = current
        else:
            self.rtt_yolo3_ms = self.rtt_yolo3_ms * 0.98 + current * 0.02

        return resp

    def rcnn_api_request(self, lps_dump, timeout=1.200):
        # make request to rcnn API
        try:
            start = time.time()
            resp = None
            resp = session.post(API_ENDPOINT_RCNN, 
            data=lps_dump, headers={'content-type':'application/json'}, timeout=timeout)
        except requests.exceptions.Timeout as e:
            logger.warning("timeout on rcnn inference request")
        except:
            logger.warning("timing/connection error on rnn", exc_info=True)
        finally:
            end = time.time()
            dec_lps = []
            if not resp:
                pass
            elif resp.status_code != 200:
                logger.warning("received {} status code from rcnn api".format(resp.status_code))
            else:
                r_dict = resp.json()
                dec_lps = r_dict["license-plates"]

        # calculate average rtt (use complementary filter)
        current = int((end - start) * 1000)
        self.rtt_crnn_ms = self.rtt_crnn_ms * 0.98 + current * 0.02

        return dec_lps

    def reorder_lps(self, lps):
        for lp in lps:
            lp = lp[::-1]

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
    def __init__(self, no_workers, pick_every_nth_frame=1):
        self.frame_num = 0
        self.in_queue = mp.Queue()
        self.out_queue = mp.Queue()
        self.no_workers = no_workers
        self.nth_frame = pick_every_nth_frame
        self.pool = WorkerPool("InferencePool", InferenceWorker, self.no_workers, self.in_queue, self.out_queue)
        self.pool.start()

    def write(self, buf):
        if buf.startswith(b"\xff\xd8"):
            # start of new frame; close the old one (if any) and
            if self.frame_num % self.nth_frame == 0:
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
    output = DistributeFramesAndInfer(workers, pick_every_nth_frame=1)
    logger.info("initialized worker pool")

    # a single worker in a separate process to reassemble the data
    reassembler = WorkerPool("BroadcastReassembled", BroadcastReassembled, 1, output.out_queue, serve_address=("", 8000))
    reassembler.start()

    # a single thread to flush the producing queue
    # when there are too many frames in the pipe
    flusher = Flusher(td.Event(), output.in_queue, threshold=10, name="Flusher")
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