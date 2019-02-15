FROM ubuntu:18.04

ARG TF_VERSION="1.12.0"

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        curl \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        python3 \
        python3-dev \
        python3-pip \
        rsync \
        software-properties-common \
        unzip \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN pip3 install https://storage.googleapis.com/tensorflow/linux/cpu/tensorflow-${TF_VERSION}-cp36-cp36m-linux_x86_64.whl && rm -rf /root/.cache/pip*
