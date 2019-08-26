FROM ubuntu:18.04

ARG ONNXRUNTIME_VERSION="0.5.0"

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


ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
COPY pkg/workloads/cortex/onnx_serve/requirements.txt /src/cortex/onnx_serve/requirements.txt
RUN pip3 install -r /src/cortex/lib/requirements.txt && \
    pip3 install -r /src/cortex/onnx_serve/requirements.txt && \
    pip3 install onnxruntime==${ONNXRUNTIME_VERSION} && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/onnx_serve /src/cortex/onnx_serve

ENTRYPOINT ["/usr/bin/python3", "/src/cortex/onnx_serve/api.py"]
