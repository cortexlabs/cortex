# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from multiprocessing.queues import Queue as mp_queue
import multiprocessing as mp


class SharedCounter(object):
    """ A synchronized shared counter.

    The locking done by multiprocessing.Value ensures that only a single
    process or thread may read or write the in-memory ctypes object. However,
    in order to do n += 1, Python performs a read followed by a write, so a
    second process may read the old value before the new one is written by the
    first process. The solution is to use a multiprocessing.Lock to guarantee
    the atomicity of the modifications to Value.

    This class comes almost entirely from Eli Bendersky's blog:
    http://eli.thegreenplace.net/2012/01/04/shared-counter-with-pythons-multiprocessing/

    """

    def __init__(self, n=0):
        self.count = mp.Value("i", n)

    def increment(self, n=1):
        """ Increment the counter by n (default = 1) """
        with self.count.get_lock():
            self.count.value += n

    def reset(self):
        """ Reset the counter to 0 """
        with self.count.get_lock():
            self.count.value = 0

    @property
    def value(self):
        """ Return the value of the counter """
        return self.count.value


class MPQueue(mp_queue):
    """ A portable implementation of multiprocessing.Queue.

    Because of multithreading / multiprocessing semantics, Queue.qsize() may
    raise the NotImplementedError exception on Unix platforms like Mac OS X
    where sem_getvalue() is not implemented. This subclass addresses this
    problem by using a synchronized shared counter (initialized to zero) and
    increasing / decreasing its value every time the put() and get() methods
    are called, respectively. This not only prevents NotImplementedError from
    being raised, but also allows us to implement a reliable version of both
    qsize() and empty().

    """

    def __init__(self, *args, **kwargs):
        self.ctx = mp.get_context()
        super(MPQueue, self).__init__(*args, **kwargs, ctx=self.ctx)
        self.size = SharedCounter(0)

    def put(self, *args, **kwargs):
        self.size.increment(1)
        super(MPQueue, self).put(*args, **kwargs)

    def get(self, *args, **kwargs):
        self.size.increment(-1)
        if self.size.value < 0:
            self.size.reset()
        return super(MPQueue, self).get(*args, **kwargs)

    def qsize(self):
        """ Reliable implementation of multiprocessing.Queue.qsize() """
        return self.size.value

    def empty(self):
        """ Reliable implementation of multiprocessing.Queue.empty() """
        return not self.qsize()
