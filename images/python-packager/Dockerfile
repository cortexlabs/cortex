FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
        python3 \
        python3-dev \
        python3-pip \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    pip3 install --upgrade \
        pip \
        setuptools \
    && rm -rf /root/.cache/pip*

ENV PYSPARK_PYTHON=/usr/bin/python3

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

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib

COPY pkg/workloads/spark_job/requirements.txt /src/spark_job/requirements.txt
COPY pkg/workloads/tf_api/requirements.txt /src/tf_api/requirements.txt
COPY pkg/workloads/tf_train/requirements.txt /src/tf_train/requirements.txt

RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/tf_train/requirements.txt && \
    pip3 install -r /src/spark_job/requirements.txt && \
    pip3 install -r /src/tf_api/requirements.txt && \
    rm -rf /root/.cache/pip*

ENV PYTHONPATH="/src:${PYTHONPATH}"

ENTRYPOINT ["/usr/bin/python3", "/src/lib/package.py"]
