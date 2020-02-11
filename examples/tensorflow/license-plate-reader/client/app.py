# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import signal, os, time, json, queue, socket, click, cv2, pandas as pd
import multiprocessing as mp
import threading as td

import logging

logger = logging.getLogger()
stream_handler = logging.StreamHandler()
stream_handler.setLevel(logging.INFO)
stream_format = logging.Formatter(
    "%(asctime)s - %(name)s - %(threadName)s - %(levelname)s - %(message)s"
)
stream_handler.setFormatter(stream_format)
logger.addHandler(stream_handler)
logger.setLevel(logging.DEBUG)

disable_loggers = ["urllib3.connectionpool"]
for name, logger in logging.root.manager.loggerDict.items():
    if name in disable_loggers:
        logger.disabled = True

from gps import ReadGPSData
from workers import BroadcastReassembled, InferenceWorker, Flusher, session
from utils.image import resize_image, image_to_jpeg_bytes
from utils.queue import MPQueue
from requests_toolbelt.adapters.source import SourceAddressAdapter


class GracefullKiller:
    """
    For killing the app gracefully.
    """

    kill_now = False

    def __init__(self):
        signal.signal(signal.SIGINT, self.exit_gracefully)
        signal.signal(signal.SIGTERM, self.exit_gracefully)

    def exit_gracefully(self, signum, frame):
        self.kill_now = True


class WorkerPool(mp.Process):
    """
    Pool of threads running in a different process.
    """

    def __init__(self, name, worker, pool_size, *args, **kwargs):
        """
        name - Name of the process.
        worker - Derived class of thread to execute.
        pool_size - Number of workers to have.
        """
        super(WorkerPool, self).__init__(name=name)
        self.event_stopper = mp.Event()
        self.Worker = worker
        self.pool_size = pool_size
        self.args = args
        self.kwargs = kwargs

    def run(self):
        logger.info("spawning workers on separate process")
        pool = [
            self.Worker(
                self.event_stopper,
                *self.args,
                **self.kwargs,
                name="{}-Worker-{}".format(self.name, i),
            )
            for i in range(self.pool_size)
        ]
        [worker.start() for worker in pool]
        while not self.event_stopper.is_set():
            time.sleep(0.001)
        logger.info("stoppping workers on separate process")
        [worker.join() for worker in pool]

    def stop(self):
        self.event_stopper.set()


class DistributeFramesAndInfer:
    """
    Custom output class primarly built for the PiCamera class.
    Has 3 process-safe queues: in_queue for the incoming frames from the source,
    bc_queue for the frames with the predicted overlays heading off to the broadcaster,
    predicts_queue for the predictions to be written off to the disk.
    """

    def __init__(self, pool_cfg, worker_cfg):
        """
        pool_cfg - Configuration dictionary for the pool manager.
        worker_cfg - Configuration dictionary for the pool workers.
        """
        self.frame_num = 0
        self.in_queue = MPQueue()
        self.bc_queue = MPQueue()
        self.predicts_queue = MPQueue()
        for key, value in pool_cfg.items():
            setattr(self, key, value)
        self.pool = WorkerPool(
            "InferencePool",
            InferenceWorker,
            self.workers,
            self.in_queue,
            self.bc_queue,
            self.predicts_queue,
            worker_cfg,
        )
        self.pool.start()

    def write(self, buf):
        """
        Mandatory custom output method for the PiCamera class.
        buf - Frame as a bytes object.
        """
        if buf.startswith(b"\xff\xd8"):
            # start of new frame; close the old one (if any) and
            if self.frame_num % self.pick_every_nth_frame == 0:
                self.in_queue.put({"frame_num": self.frame_num, "jpeg": buf})
            self.frame_num += 1

    def stop(self):
        """
        Stop all workers and the process altogether.
        """
        self.pool.stop()
        self.pool.join()
        qs = [self.in_queue, self.bc_queue]
        [q.cancel_join_thread() for q in qs]

    def get_queues(self):
        """
        Retrieve all queues.
        """
        return self.in_queue, self.bc_queue, self.predicts_queue


@click.command(
    help=(
        "Identify license plates from a given video source"
        " while outsourcing the predictions using REST API endpoints."
    )
)
@click.option("--config", "-c", required=True, type=str)
def main(config):
    killer = GracefullKiller()

    # open config file
    try:
        file = open(config)
        cfg = json.load(file)
        file.close()
    except Exception as error:
        logger.critical(str(error), exc_info=1)
        return

    # give meaningful names to each sub config
    source_cfg = cfg["video_source"]
    broadcast_cfg = cfg["broadcaster"]
    pool_cfg = cfg["inferencing_pool"]
    worker_cfg = cfg["inferencing_worker"]
    flusher_cfg = cfg["flusher"]
    gps_cfg = cfg["gps"]
    gen_cfg = cfg["general"]

    # bind requests module to use a given network interface
    try:
        socket.inet_aton(gen_cfg["bind_ip"])
        session.mount("http://", SourceAddressAdapter(gen_cfg["bind_ip"]))
        logger.info("binding requests module to {} IP".format(gen_cfg["bind_ip"]))
    except OSError as e:
        logger.error("bind IP is invalid, resorting to default interface", exc_info=True)

    # start polling the GPS
    if gps_cfg["use_gps"]:
        wport = gps_cfg["write_port"]
        rport = gps_cfg["read_port"]
        br = gps_cfg["baudrate"]
        gps = ReadGPSData(wport, rport, br)
        gps.start()
    else:
        gps = None

    # workers on a separate process to run inference on the data
    logger.info("initializing pool w/ " + str(pool_cfg["workers"]) + " workers")
    output = DistributeFramesAndInfer(pool_cfg, worker_cfg)
    frames_queue, bc_queue, predicts_queue = output.get_queues()
    logger.info("initialized worker pool")

    # a single worker in a separate process to reassemble the data
    reassembler = BroadcastReassembled(bc_queue, broadcast_cfg, name="BroadcastReassembled")
    reassembler.start()

    # a single thread to flush the producing queue
    # when there are too many frames in the pipe
    flusher = Flusher(frames_queue, threshold=flusher_cfg["frame_count_threshold"], name="Flusher")
    flusher.start()

    # data aggregator to write things to disk
    def results_writer():
        if len(gen_cfg["saved_data"]) > 0:
            df = pd.DataFrame(columns=["Date", "License Plate", "Coordinates"])
            while not killer.kill_now:
                time.sleep(0.01)
                try:
                    data = predicts_queue.get_nowait()
                except queue.Empty:
                    continue
                predicts = data["predicts"]
                date = data["date"]
                for lp in predicts:
                    if len(lp) > 0:
                        lp = " ".join(lp)
                        entry = {"Date": date, "License Plate": lp, "Coordinates": ""}
                        if gps:
                            entry["Coordinates"] = "{}, {}".format(
                                gps.latitude, gps.longitude
                            ).upper()
                        df = df.append(entry, ignore_index=True)

            logger.info("dumping results to csv file {}".format(gen_cfg["saved_data"]))
            if os.path.isfile(gen_cfg["saved_data"]):
                header = False
            else:
                header = True
            with open(gen_cfg["saved_data"], "a") as f:
                df.to_csv(f, header=header)

    # data aggregator thread
    results_thread = td.Thread(target=results_writer)
    results_thread.start()

    if source_cfg["type"] == "camera":
        # import module
        import picamera

        # start the pi camera
        with picamera.PiCamera() as camera:
            # configure the camera
            camera.sensor_mode = source_cfg["sensor_mode"]
            camera.resolution = source_cfg["resolution"]
            camera.framerate = source_cfg["framerate"]
            logger.info(
                "picamera initialized w/ mode={} resolution={} framerate={}".format(
                    camera.sensor_mode, camera.resolution, camera.framerate
                )
            )

            # start recording both to disk and to the queue
            camera.start_recording(
                output=source_cfg["output_file"], format="h264", splitter_port=0, bitrate=10000000,
            )
            camera.start_recording(
                output=output, format="mjpeg", splitter_port=1, bitrate=10000000, quality=95,
            )
            logger.info("started recording to file and to queue")

            # wait until SIGINT is detected
            while not killer.kill_now:
                camera.wait_recording(timeout=0.5, splitter_port=0)
                camera.wait_recording(timeout=0.5, splitter_port=1)
                logger.info(
                    "frames qsize: {}, broadcast qsize: {}, predicts qsize: {}".format(
                        frames_queue.qsize(), bc_queue.qsize(), predicts_queue.qsize()
                    )
                )

            # stop recording
            logger.info("gracefully exiting")
            camera.stop_recording(splitter_port=0)
            camera.stop_recording(splitter_port=1)
            output.stop()

    elif source_cfg["type"] == "file":
        # open video file
        video_reader = cv2.VideoCapture(source_cfg["input"])
        video_reader.set(cv2.CAP_PROP_POS_FRAMES, source_cfg["frames_to_skip"])

        # get # of frames and determine target width
        nb_frames = int(video_reader.get(cv2.CAP_PROP_FRAME_COUNT))
        frame_h = int(video_reader.get(cv2.CAP_PROP_FRAME_HEIGHT))
        frame_w = int(video_reader.get(cv2.CAP_PROP_FRAME_WIDTH))
        target_h = int(frame_h * source_cfg["scale_video"])
        target_w = int(frame_w * source_cfg["scale_video"])
        period = 1.0 / source_cfg["framerate"]

        logger.info(
            "file-based video stream initialized w/ resolution={} framerate={} and {} skipped frames".format(
                (target_w, target_h), source_cfg["framerate"], source_cfg["frames_to_skip"],
            )
        )

        # serve each frame to the workers iteratively
        last_log = time.time()
        for i in range(nb_frames):
            try:
                # write frame to queue
                _, frame = video_reader.read()
                if target_w != frame_w:
                    frame = resize_image(frame, target_w)
                jpeg = image_to_jpeg_bytes(frame)
                output.write(jpeg)
            except Exception as error:
                logger.error("unexpected error occurred", exc_info=True)
                break

            # check if SIGINT has been sent
            if killer.kill_now:
                break

            # do logs every second
            current = time.time()
            if current - last_log >= 1.0:
                logger.info(
                    "frames qsize: {}, broadcast qsize: {}, predicts qsize: {}".format(
                        frames_queue.qsize(), bc_queue.qsize(), predicts_queue.qsize()
                    )
                )
                last_log = current

        logger.info("gracefully exiting")
        video_reader.release()
        output.stop()

    if gps_cfg["use_gps"]:
        gps.stop()

    reassembler.stop()
    flusher.stop()


if __name__ == "__main__":
    main()
