import os, fcntl, time


class FileLock:
    def __init__(self, lock_file, timeout=None):
        """
        lock_file - File to use as lock.
        timeout - If used, a timeout exception will be raised if the lock can't be acquired. Measured in seconds.
        """
        self._lock_file = lock_file
        self._file_handle = None

        self.timeout = timeout
        self._time_loop = 0.001

        # create lock if it doesn't exist
        try:
            with open(self._lock_file_handlefile, "w+") as f:
                pass
        except:
            pass

    def acquire(self):
        """
        To acquire rw access to resource.
        """
        if self._file_handle:
            return

        if not self.timeout:
            self._file_handle = open(self._lock_file, "w")
            fcntl.lockf(self._file_handle, fcntl.LOCK_EX)
        else:
            start = time.time()
            acquired = False
            while start + self.timeout >= time.time():
                try:
                    self._file_handle = open(self._lock_file, "w")
                    fcntl.lockf(self._file_handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
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
        To release rw access to resource.
        """
        if not self._file_handle:
            return

        fd = self._file_handle
        self._file_handle = None
        fcntl.lockf(fd, fcntl.LOCK_UN)
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
