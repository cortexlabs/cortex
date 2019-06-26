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


ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
COPY pkg/workloads/onnx_serve/requirements.txt /src/onnx_serve/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/onnx_serve/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/onnx_serve /src/onnx_serve

ENTRYPOINT ["/usr/bin/python3", "/src/onnx_serve/api.py"]
