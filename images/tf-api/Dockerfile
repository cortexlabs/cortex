FROM tensorflow/tensorflow:2.0.0-py3

RUN apt-get update -qq && apt-get install -y -q \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

ENV PYTHONPATH "${PYTHONPATH}:/src:/mnt/project"
RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        software-properties-common \
        zlib1g-dev \
        git \
        libsm6 \
        libxext6 \
        libxrender-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
COPY pkg/workloads/cortex/serve/tf.requirements.txt /src/cortex/serve/requirements.txt
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /src/cortex/lib/requirements.txt && \
    pip install --no-cache-dir -r /src/cortex/serve/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex/
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/serve /src/cortex/serve

ENTRYPOINT ["/src/cortex/serve/run.sh"]
