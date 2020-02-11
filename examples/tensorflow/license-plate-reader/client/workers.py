# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from utils.image import (
    resize_image,
    compress_image,
    image_from_bytes,
    image_to_jpeg_nparray,
    image_to_jpeg_bytes,
)
from utils.bbox import BoundBox, draw_boxes
from statistics import mean
import time, base64, pickle, json, cv2, logging, requests, queue, broadcast, copy, statistics

import numpy as np
import threading as td
import multiprocessing as mp

logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

session = requests.Session()


class WorkerTemplateThread(td.Thread):
    def __init__(self, event_stopper, name=None, runnable=None):
        td.Thread.__init__(self, name=name)
        self.event_stopper = event_stopper
        self.runnable = runnable

    def run(self):
        if self.runnable:
            logger.debug("worker started")
            while not self.event_stopper.is_set():
                self.runnable()
                time.sleep(0.030)
            logger.debug("worker stopped")

    def stop(self):
        self.event_stopper.set()


class WorkerTemplateProcess(mp.Process):
    def __init__(self, event_stopper, name=None, runnable=None):
        mp.Process.__init__(self, name=name)
        self.event_stopper = event_stopper
        self.runnable = runnable

    def run(self):
        if self.runnable:
            logger.debug("worker started")
            while not self.event_stopper.is_set():
                self.runnable()
                time.sleep(0.030)
            logger.debug("worker stopped")

    def stop(self):
        self.event_stopper.set()


class BroadcastReassembled(WorkerTemplateProcess):
    """
    Separate process to broadcast the stream with the overlayed predictions on top of it.
    """

    def __init__(self, in_queue, cfg, name=None):
        """
        in_queue - Queue from which to extract the frames with the overlayed predictions on top of it.
        cfg - The dictionary config for the broadcaster.
        name - Name of the process.
        """
        super(BroadcastReassembled, self).__init__(event_stopper=mp.Event(), name=name)
        self.in_queue = in_queue
        self.yolo3_rtt = None
        self.crnn_rtt = None
        self.detections = 0
        self.current_detections = 0
        self.recognitions = 0
        self.current_recognitions = 0
        self.buffer = []
        self.oldest_broadcasted_frame = 0

        for key, value in cfg.items():
            setattr(self, key, value)

    def run(self):
        # start streaming server
        def lambda_func():
            server = broadcast.StreamingServer(
                tuple(self.serve_address), broadcast.StreamingHandler
            )
            server.serve_forever()

        td.Thread(target=lambda_func, args=(), daemon=True).start()
        logger.info("listening for stream clients on {}".format(self.serve_address))

        # start polling for new processed frames from the queue and broadcast
        logger.info("worker started")
        counter = 0
        while not self.event_stopper.is_set():
            if counter == self.target_fps:
                logger.debug("buffer queue size: {}".format(len(self.buffer)))
                counter = 0
            self.reassemble()
            time.sleep(0.001)
            counter += 1
        logger.info("worker stopped")

    def reassemble(self):
        """
        Main method to run in the loop.
        """

        start = time.time()
        self.pull_and_push()
        self.purge_stale_frames()

        frame, delay = self.pick_new_frame()
        # delay loop to stabilize the video fps
        end = time.time()
        elapsed_time = end - start
        elapsed_time += 0.001  # count in the millisecond in self.run
        if delay - elapsed_time > 0.0:
            time.sleep(delay - elapsed_time)
        if frame:
            # pull and push again in case
            # write buffer (assume it takes an insignificant time to execute)
            self.pull_and_push()
            broadcast.output.write(frame)

    def pull_and_push(self):
        """
        Get new frame and push it in the broadcaster's little buffer for stabilization.
        """
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
        self.buffer.append({"image": byte_im, "frame_num": frame_num})

    def purge_stale_frames(self):
        """
        Remove any frames older than the latest broadcasted frame.
        """
        new_buffer = []
        for frame in self.buffer:
            if frame["frame_num"] > self.oldest_broadcasted_frame:
                new_buffer.append(frame)
        self.buffer = new_buffer

    def pick_new_frame(self):
        """
        Get the oldest frame from the buffer that isn't older than the last broadcasted frame.
        """
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
        """
        A bunch of RTT and detection/recognition statistics. Not used.
        """
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


class InferenceWorker(WorkerTemplateThread):
    """
    Worker that receives frames from a queue, sends requests to 2 cortex APIs for inference reasons,
    and retrieves the results and puts them in their appropriate queues.
    """

    def __init__(self, event_stopper, in_queue, bc_queue, predicts_queue, cfg, name=None):
        """
        event_stopper - Event to stop the worker.
        in_queue - Queue that holds the unprocessed frames.
        bc_queue - Queue to push into the frames with the overlayed predictions.
        predicts_queue - Queue to push into the detected license plates that will eventually get written to the disk.
        cfg - Dictionary config for the worker.
        name - Name of the worker thread.
        """
        super(InferenceWorker, self).__init__(event_stopper=event_stopper, name=name)
        self.in_queue = in_queue
        self.bc_queue = bc_queue
        self.predicts_queue = predicts_queue
        self.rtt_yolo3_ms = None
        self.rtt_crnn_ms = 0
        self.runnable = self.cloud_infer

        for key, value in cfg.items():
            setattr(self, key, value)

    def cloud_infer(self):
        """
        Main method that runs in the loop.
        """
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
        image = image_from_bytes(img)
        reduced = compress_image(image)
        byte_im = image_to_jpeg_bytes(reduced)
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
        aux = []
        for b in boxes:
            label = -1
            for i in range(len(b.classes)):
                if b.classes[i] > self.yolov3_obj_thresh:
                    label = i
            if label >= 0:
                aux.append(b)
        boxes = aux
        del aux

        # also scale the boxes for later uses
        camera_source_width = image.shape[1]
        boxes640 = self.scale_bbox(boxes, self.yolov3_input_size_px, self.bounding_boxes_upscale_px)
        boxes_source = self.scale_bbox(boxes, self.yolov3_input_size_px, camera_source_width)

        #############################

        # recognize the license plates in case
        # any bounding boxes have been detected
        dec_words = []
        if len(boxes) > 0 and len(self.api_endpoint_crnn) > 0:
            # create set of images of the detected license plates
            lps = []
            try:
                for b in boxes_source:
                    lp = image[b.ymin : b.ymax, b.xmin : b.xmax]
                    jpeg = image_to_jpeg_nparray(
                        lp, [int(cv2.IMWRITE_JPEG_QUALITY), self.crnn_quality]
                    )
                    lps.append(jpeg)
            except:
                logger.warning("encountered error while converting to jpeg")
                pass

            lps = pickle.dumps(lps, protocol=0)
            lps_enc = base64.b64encode(lps).decode("utf-8")
            lps_dump = json.dumps({"imgs": lps_enc})

            # make request to rcnn API
            dec_lps = self.rcnn_api_request(lps_dump)
            dec_lps = self.reorder_recognized_words(dec_lps)
            for dec_lp in dec_lps:
                dec_words.append([word[0] for word in dec_lp])

        if len(dec_words) > 0:
            logger.info("Detected the following words: {}".format(dec_words))
        else:
            dec_words = [[] for i in range(len(boxes))]

        #############################

        # draw detections
        upscaled = resize_image(image, self.bounding_boxes_upscale_px)
        draw_image = draw_boxes(
            upscaled,
            boxes640,
            overlay_text=dec_words,
            labels=["LP"],
            obj_thresh=self.yolov3_obj_thresh,
        )
        draw_byte_im = image_to_jpeg_bytes(
            draw_image, [int(cv2.IMWRITE_JPEG_QUALITY), self.broadcast_quality]
        )

        #############################

        # push data for further processing in the queue
        output = {
            "boxes": boxes,
            "frame_num": frame_num,
            "avg_yolo3_rtt": self.rtt_yolo3_ms,
            "avg_crnn_rtt": self.rtt_crnn_ms,
            "image": draw_byte_im,
        }
        self.bc_queue.put(output)

        # push predictions to write to disk
        if len(dec_words) > 0:
            timestamp = time.time()
            literal_time = time.ctime(timestamp)
            predicts = {"predicts": dec_words, "date": literal_time}
            self.predicts_queue.put(predicts)

        logger.info(
            "Frame Count: {} - Avg YOLO3 RTT: {}ms - Avg CRNN RTT: {}ms - Detected: {}".format(
                frame_num, int(self.rtt_yolo3_ms), int(self.rtt_crnn_ms), len(boxes)
            )
        )

    def scale_bbox(self, boxes, old_width, new_width):
        """
        Scale a bounding box.
        """
        boxes = copy.deepcopy(boxes)
        scale_percent = new_width / old_width
        for b in boxes:
            b.xmin = int(b.xmin * scale_percent)
            b.ymin = int(b.ymin * scale_percent)
            b.xmax = int(b.xmax * scale_percent)
            b.ymax = int(b.ymax * scale_percent)
        return boxes

    def yolov3_api_request(self, img_dump):
        """
        Make a request to the YOLOv3 API.
        """
        # make inference request
        try:
            start = time.time()
            resp = None
            resp = session.post(
                self.api_endpoint_yolov3,
                data=img_dump,
                headers={"content-type": "application/json"},
                timeout=self.timeout,
            )
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
        """
        Make a request to the CRNN API.
        """
        # make request to rcnn API
        try:
            start = time.time()
            resp = None
            resp = session.post(
                self.api_endpoint_crnn,
                data=lps_dump,
                headers={"content-type": "application/json"},
                timeout=self.timeout,
            )
        except requests.exceptions.Timeout as e:
            logger.warning("timeout on crnn inference request")
        except:
            logger.warning("timing/connection error on crnn", exc_info=True)
        finally:
            end = time.time()
            dec_lps = []
            if not resp:
                pass
            elif resp.status_code != 200:
                logger.warning("received {} status code from crnn api".format(resp.status_code))
            else:
                r_dict = resp.json()
                dec_lps = r_dict["license-plates"]

        # calculate average rtt (use complementary filter)
        current = int((end - start) * 1000)
        self.rtt_crnn_ms = self.rtt_crnn_ms * 0.98 + current * 0.02

        return dec_lps

    def reorder_recognized_words(self, detected_images):
        """
        Reorder the detected words in each image based on the average horizontal position of each word.
        Sorting them in ascending order.
        """

        reordered_images = []
        for detected_image in detected_images:

            # computing the mean average position for each word
            mean_horizontal_positions = []
            for words in detected_image:
                box = words[1]
                y_positions = [point[0] for point in box]
                mean_y_position = mean(y_positions)
                mean_horizontal_positions.append(mean_y_position)
            indexes = np.argsort(mean_horizontal_positions)

            # and reordering them
            reordered = []
            for index, words in zip(indexes, detected_image):
                reordered.append(detected_image[index])
            reordered_images.append(reordered)

        return reordered_images


class Flusher(WorkerTemplateThread):
    """
    Thread which removes the elements of a queue when its size crosses a threshold.
    Used when there are too many frames are pilling up in the queue.
    """

    def __init__(self, queue, threshold, name=None):
        """
        queue - Queue to remove the elements from when the threshold is triggered.
        threshold - Number of elements.
        name - Name of the thread.
        """
        super(Flusher, self).__init__(event_stopper=td.Event(), name=name)
        self.queue = queue
        self.threshold = threshold
        self.runnable = self.flush_pipe

    def flush_pipe(self):
        """
        Main method to run in the loop.
        """
        current = self.queue.qsize()
        if current > self.threshold:
            try:
                for i in range(current):
                    self.queue.get_nowait()
                logger.warning("flushed {} elements from the frames queue".format(current))
            except queue.Empty:
                logger.debug("flushed too many elements from the queue")
        time.sleep(0.5)
