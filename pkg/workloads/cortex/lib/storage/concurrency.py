#!/bin/bash

# Copyright 2020 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os, fcntl, time
from cortex.lib.exceptions import CortexException


class FileLock:
    def __init__(self, lock_file: str, timeout: float = None, reader_lock: bool = False):
        """
        Lock for files. Not thread-safe. Instantiate one lock per thread.

        lock_file - File to use as lock.
        timeout - If used, a timeout exception will be raised if the lock can't be acquired. Measured in seconds.
        reader_lock - When set to true, a shared lock (LOCK_SH) will be used. Otherwise, an exclusive lock (LOCK_EX) is used.
        """
        self._lock_file = lock_file
        self._file_handle = None

        self.timeout = timeout
        self.reader_lock = reader_lock
        self._time_loop = 0.001

        # create lock if it doesn't exist
        with open(self._lock_file, "w+") as f:
            pass

    def acquire(self):
        """
        To acquire the lock to resource.
        """
        if self._file_handle:
            return

        if not self.timeout:
            self._file_handle = open(self._lock_file, "w")
            if self.reader_lock:
                fcntl.flock(self._file_handle, fcntl.LOCK_SH)
            else:
                fcntl.flock(self._file_handle, fcntl.LOCK_EX)
        else:
            start = time.time()
            acquired = False
            while start + self.timeout >= time.time():
                try:
                    self._file_handle = open(self._lock_file, "w")
                    if self.reader_lock:
                        fcntl.flock(self._file_handle, fcntl.LOCK_SH | fcntl.LOCK_NB)
                    else:
                        fcntl.flock(self._file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    acquired = True
                    break
                except OSError:
                    time.sleep(self._time_loop)

            if not acquired:
                self._file_handle = None
                raise TimeoutError(
                    "{} ms timeout on acquiring {} lock".format(
                        int(self.timeout * 1000), self._lock_file
                    )
                )

    def release(self):
        """
        To release the lock to resource.
        """
        if not self._file_handle:
            return

        fd = self._file_handle
        self._file_handle = None
        fcntl.flock(fd, fcntl.LOCK_UN)
        fd.close()

    def __enter__(self):
        self.acquire()
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.release()
        return None

    def __del__(self):
        self.release()
        return None


class LockedFile:
    """
    Create a lock-based file.
    """

    def __init__(self, filename: str, mode: str, timeout: float = None, reader_lock: bool = False):
        self.dir_path, self.basename = os.path.split(filename)
        if self.basename == "":
            raise CortexException(f"{filename} does not represent a path to file")
        if not self.basename.startswith("."):
            self.lockname = "."
        self.lockname += self.basename

        self.filename = filename
        self.mode = mode
        self.timeout = timeout
        self.reader_lock = reader_lock

    def __enter__(self):
        lockfilepath = os.path.join(self.dir_path, self.lockname + ".lock")
        self._lock = FileLock(lockfilepath, self.timeout, self.reader_lock)
        self._lock.acquire()
        try:
            self._fd = open(self.filename, self.mode)
        except Exception as e:
            self._lock.release()
            raise e
        return self._fd

    def __exit__(self, exc_type, exc_value, traceback):
        close(self._fd)
        self._lock.release()

    def __del__(self):
        close(self._fd)
        self._lock.release()
