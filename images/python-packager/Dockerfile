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

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib

COPY pkg/workloads/spark_job/requirements* /src/spark_job/
COPY pkg/workloads/tf_api/requirements* /src/tf_api/
COPY pkg/workloads/tf_train/requirements* /src/tf_train/

RUN find /src -name "requirements*" | xargs printf -- '-r %s ' | xargs pip3 install && \
    rm -rf /root/.cache/pip*

ENV PYTHONPATH="/src:${PYTHONPATH}"

ENTRYPOINT ["/usr/bin/python3", "/src/lib/package.py"]
