# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import serial, pynmea2, time, threading as td
import logging

logger = logging.getLogger(__name__)


class ReadGPSData(td.Thread):
    """
    Class to read the data off of the EC25-E's GPS module.

    Can be easily adapted to work with any other GPS module.
    """

    def __init__(self, write_port, read_port, baudrate, name="GPS"):
        """
        write_port - The serial port to use for activating the GPS.
        read_port - The serial port from which to read the GPS data.
        baudrate - Transport rate over the serial port.
        name - Name of the thread.
        """
        super(ReadGPSData, self).__init__(name=name)
        self.write_port = write_port
        self.read_port = read_port
        self.baudrate = baudrate
        self.event = td.Event()
        self.lock = td.Lock()

    def run(self):
        logger.info("configuring GPS on port {}".format(self.write_port))
        self.serw = serial.Serial(
            self.write_port, baudrate=self.baudrate, timeout=1, rtscts=True, dsrdtr=True
        )
        self.serw.write("AT+QGPS=1\r".encode("utf-8"))
        self.serw.close()
        time.sleep(0.5)

        self.serr = serial.Serial(
            self.read_port, baudrate=self.baudrate, timeout=1, rtscts=True, dsrdtr=True
        )
        logger.info("configured GPS to read from port {}".format(self.read_port))

        while not self.event.is_set():
            data = self.serr.readline()
            self.lock.acquire()
            try:
                self.__msg = pynmea2.parse(data.decode("utf-8"))
            except:
                pass
            finally:
                self.lock.release()
            logger.info(self.__msg)
            time.sleep(1)

        logger.info("stopped GPS thread")

    @property
    def parsed(self):
        """
        Get the whole parsed data.
        """
        self.lock.acquire()
        try:
            data = self.__msg
        except:
            data = None
        finally:
            self.lock.release()
        return data

    @property
    def latitude(self):
        """
        Returns latitude expressed as a float.
        """
        self.lock.acquire()
        try:
            latitude = self.__msg.latitude
        except:
            latitude = 0.0
        finally:
            self.lock.release()
        return latitude

    @property
    def longitude(self):
        """
        Returns longitude expressed as a float.
        """
        self.lock.acquire()
        try:
            longitude = self.__msg.longitude
        except:
            longitude = 0.0
        finally:
            self.lock.release()
        return longitude

    def stop(self):
        """
        Stop the thread.
        """
        self.event.set()
