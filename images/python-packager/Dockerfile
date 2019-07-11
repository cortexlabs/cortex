FROM ubuntu:16.04

RUN apt-get update -qq && apt-get install -y -q \
        python3 \
        python3-dev \
        python3-pip \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    pip3 install --upgrade \
        pip \
        setuptools \
    && rm -rf /root/.cache/pip*

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
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY pkg/workloads/cortex/consts.py /src/cortex/
COPY pkg/workloads/cortex/lib /src/cortex/lib

COPY pkg/workloads/cortex/tf_api/requirements.txt /src/cortex/tf_api/requirements.txt

RUN pip3 install -r /src/cortex/lib/requirements.txt && \
    pip3 install -r /src/cortex/tf_api/requirements.txt && \
    rm -rf /root/.cache/pip*

ENV PYTHONPATH="/src:${PYTHONPATH}"

ENTRYPOINT ["/usr/bin/python3", "/src/cortex/lib/package.py"]
