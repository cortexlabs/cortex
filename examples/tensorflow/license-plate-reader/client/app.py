import signal
import time
import io
import picamera
import threading as td
import multiprocessing as mp

class GracefullKiller():
    kill_now = False
    def __init__(self):
        signal.signal(signal.SIGINT, self.exit_gracefully)
        signal.signal(signal.SIGTERM, self.exit_gracefully)
    
    def exit_gracefully(self, signum, frame):
        self.kill_now = True

class InferenceWorker(td.Thread):
    def __init__(self, event_stopper, in_queue, out_queue, name=None, verbose=None):
        super(InferenceWorker, self).__init__(name=name, verbose=verbose)
        self.event_stopper = event_stopper
        self.in_queue = in_queue
        self.out_queue = out_queue

    def run(self):
        while not self.event_stopper.is_set():
            self.cloud_infer()
            time.sleep(0.001)

    def cloud_infer(self):
        pass

class WorkerPool(mp.Process):
    def __init__(self, event_stopper, in_queue, out_queue, pool_size, name=None, daemon=None):
        super(WorkerPool, self).__init__(name=name, daemon=daemon)
        self.event_stopper = event_stopper
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.pool_size = pool_size

    def run(self):
        pool = [InferenceWorker(self.event_stopper, self.in_queue, self.out_queue) for i in range(self.pool_size)]
        [worker.start() for worker in pool]
        while not self.event_stopper.is_set():
            time.sleep(0.001)

class DistributeFramesAndInfer():
    def __init__(self, no_workers):
        self.frame_num = 0
        self.output = None
        self.in_queue = mp.Queue()
        self.out_queue = mp.Queue()
        self.no_workers = no_workers
        self.pool = WorkerPool(event_stopper, self.in_queue, self.out_queue, self.no_workers)

    def write(self, buf):
        if buf.startswith(b"\xff\xd8"):
            # start of new frame; close the old one (if any) and
            # open a new output
            if self.output:
                self.output.close()
            self.frame_num += 1
            self.output = io.open("image%02d.jpg" % self.frame_num, "wb")
        self.output.write(buf)

def main():
    killer = GracefullKiller()

    with picamera.PiCamera() as camera:
        camera.sensor_mode = 5
        camera.resolution = (1280, 720)
        camera.framerate = 30
        output = DistributeFramesAndInfer()

        camera.start_recording(output="recording.h264", format="h264", splitter_port=0, bitrate=10000000)
        camera.start_recording(output=output, format="mjpeg", splitter_port=1, bitrate=17000000, quality=100)
        while not killer.kill_now:
            camera.wait_recording(timeout=0.5, splitter_port=0)
            camera.wait_recording(timeout=0.5, splitter_port=1)
        camera.stop_recording(splitter_port=0)
        camera.stop_recording(splitter_port=1)

if __name__ == "__main__":
    main()