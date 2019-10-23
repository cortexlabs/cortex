FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        curl \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        rsync \
        software-properties-common \
        unzip \
        zlib1g-dev \
        python3.6-dev \
        python3.6-distutils \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3.6 get-pip.py && \
    rm -rf /root/.cache/pip*

ENV PYTHONPATH "${PYTHONPATH}:/src:/mnt/project"

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
COPY pkg/workloads/cortex/pytorch/requirements.txt /src/cortex/pytorch/requirements.txt
RUN pip install -r /src/cortex/lib/requirements.txt && \
    pip install -r /src/cortex/pytorch/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/pytorch /src/cortex/pytorch

ENTRYPOINT ["/src/cortex/pytorch/run.sh"]
