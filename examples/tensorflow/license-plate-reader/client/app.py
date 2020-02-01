import signal
import time
import base64
import pickle
import json
import picamera
import logging
import requests
import queue
import threading as td
import multiprocessing as mp
from utils.bbox import BoundBox

logger = logging.getLogger(__name__)
stream_handler = logging.StreamHandler()
stream_handler.setLevel(logging.DEBUG)
stream_format = logging.Formatter("%(asctime)s - %(name)s - %(threadName)s - %(levelname)s - %(message)s")
stream_handler.setFormatter(stream_format)
logger.addHandler(stream_handler)
logger.setLevel(logging.DEBUG)

api_endpoint = "http://Roberts-MacBook-2.local/predict"

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

        frame_num = data["frame_num"]
        img = data["jpeg"]
        img_enc = base64.b64encode(img).decode("utf-8")
        dump = json.dumps({"img": img_enc})

        try:
            start = time.time()
            resp = requests.post("http://Roberts-MacBook-2.local/predict", 
            data=dump, headers={'content-type':'application/json'}, timeout=1.000)
            end = time.time()
        except requests.exceptions.Timeout as e:
            logger.warning("timeout on inference request")
            return
        except:
            logger.warning("timing/connection error")
            return
        current = int((end - start) * 1000)
        if not self.rtt_ms:
            self.rtt_ms = current
        else:
            self.rtt_ms = self.rtt_ms * 0.98 + current * 0.02
        logger.debug("Frame Count: {} - Avg RTT: {}".format(frame_num, self.rtt_ms))
        
        r_dict = resp.json()
        data = pickle.loads(r_dict.encode('utf-8'))
        

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
    workers = 4
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
            logger.info('avg producing queue size: {}'.format(output.in_queue.qsize()))

        logger.info("gracefully exiting")
        camera.stop_recording(splitter_port=0)
        camera.stop_recording(splitter_port=1)
        output.stop() 

if __name__ == "__main__":
    main()