FROM nvidia/cuda:9.0-base-ubuntu16.04

ARG TF_VERSION="1.12.0"

RUN apt-get update -qq && apt-get install -y -q \
        automake \
        build-essential \
        curl \
        libcurl3-dev \
        git \
        libtool \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        python3-dev \
        python3-numpy \
        python3-pip \
        software-properties-common \
        swig \
        zip \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN curl -o tensorflow-model-server.deb http://storage.googleapis.com/tensorflow-serving-apt/pool/tensorflow-model-server-${TF_VERSION}/t/tensorflow-model-server/tensorflow-model-server_${TF_VERSION}_all.deb
RUN dpkg -i tensorflow-model-server.deb

ENTRYPOINT ["tensorflow_model_server"]
