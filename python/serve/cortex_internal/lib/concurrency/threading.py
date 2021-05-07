# Copyright 2021 Cortex Labs, Inc.
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

import threading as td
from typing import Optional


class ReadWriteLock:
    """
    Locking object allowing for write once, read many operations.

    The lock must not be acquired multiple times in a single thread without paired release calls.

    Can set different priority policies: "r" for read-preferring RW lock allowing for maximum concurrency
    or can be set to "w" for write-preferring RW lock to prevent from starving the writer.
    """

    def __init__(self, prefer: str = "r"):
        """
        "r" for read-preferring RW lock.

        "w" for write-preferring RW lock.
        """
        self._prefer = prefer
        self._write_preferred = td.Event()
        self._write_preferred.set()
        self._read_allowed = td.Condition(td.RLock())
        self._readers = []
        # a single writer is supported despite the fact that this is a list.
        self._writers = []

    def acquire(self, mode: str, timeout: Optional[float] = None) -> bool:
        """
        Acquire a lock.

        Args:
            mode: "r" for read lock, "w" for write lock.
            timeout: How many seconds to wait to acquire the lock.

        Returns:
            Whether the mode was valid or not.
        """
        if not timeout:
            acquire_timeout = -1
        else:
            acquire_timeout = timeout

        if mode == "r":
            # wait until "w" has been released
            if self._prefer == "w":
                if not self._write_preferred.wait(timeout):
                    self._throw_timeout_error(timeout, mode)

            # finish acquiring once all writers have released
            if not self._read_allowed.acquire(timeout=acquire_timeout):
                self._throw_timeout_error(timeout, mode)
            # while loop only relevant when prefer == "r"
            # but it's necessary when the preference policy is changed
            while len(self._writers) > 0:
                if not self._read_allowed.wait(timeout):
                    self._read_allowed.release()
                    self._throw_timeout_error(timeout, mode)

            self._readers.append(td.get_ident())
            self._read_allowed.release()

        elif mode == "w":
            # stop "r" acquirers from acquiring
            if self._prefer == "w":
                self._write_preferred.clear()

            # acquire once all readers have released
            if not self._read_allowed.acquire(timeout=acquire_timeout):
                self._write_preferred.set()
                self._throw_timeout_error(timeout, mode)
            while len(self._readers) > 0:
                if not self._read_allowed.wait(timeout):
                    self._read_allowed.release()
                    self._write_preferred.set()
                    self._throw_timeout_error(timeout, mode)
            self._writers.append(td.get_ident())
        else:
            return False

        return True

    def release(self, mode: str) -> bool:
        """
        Releases a lock.

        Args:
            mode: "r" for read lock, "w" for write lock.

        Returns:
            Whether the mode was valid or not.
        """
        if mode == "r":
            # release and let writers acquire
            self._read_allowed.acquire()
            if not len(self._readers) - 1:
                self._read_allowed.notifyAll()
            self._readers.remove(td.get_ident())
            self._read_allowed.release()

        elif mode == "w":
            # release and let readers acquire
            self._writers.remove(td.get_ident())
            # notify all only relevant when prefer == "r"
            # but it's necessary when the preference policy is changed
            self._read_allowed.notifyAll()
            self._read_allowed.release()

            # let "r" acquirers acquire again
            if self._prefer == "w":
                self._write_preferred.set()
        else:
            return False

        return True

    def set_preference_policy(self, prefer: str) -> bool:
        """
        Change preference policy dynamically.

        When readers have acquired the lock, the policy change is immediate.
        When a writer has acquired the lock, the policy change will block until the writer releases the lock.

        Args:
            prefer: "r" for read-preferring RW lock, "w" for write-preferring RW lock.

        Returns:
            True when the policy has been changed, false otherwise.
        """
        if self._prefer == prefer:
            return False

        self._read_allowed.acquire()
        self._prefer = prefer
        self._write_preferred.set()
        self._read_allowed.release()

        return True

    def _throw_timeout_error(self, timeout: float, mode: str) -> None:
        raise TimeoutError(
            "{} ms timeout on acquiring '{}' lock in {} thread".format(
                int(timeout * 1000), mode, td.get_ident()
            )
        )


class LockRead:
    """
    To be used as:

    ```python
    rw_lock = ReadWriteLock()
    with LockRead(rw_lock):
        # code
    ```
    """

    def __init__(self, lock: ReadWriteLock, timeout: Optional[float] = None):
        self._lock = lock
        self._timeout = timeout

    def __enter__(self):
        self._lock.acquire("r", self._timeout)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self._lock.release("r")
        return False


class LockWrite:
    """
    To be used as:

    ```python
    rw_lock = ReadWriteLock()
    with LockWrite(rw_lock):
        # code
    ```
    """

    def __init__(self, lock: ReadWriteLock, timeout: Optional[float] = None):
        self._lock = lock
        self._timeout = timeout

    def __enter__(self):
        self._lock.acquire("w", self._timeout)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self._lock.release("w")
        return False
