FROM tensorflow/tensorflow:1.12.0-gpu-py3 as base

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        curl \
        wget \
        python3 \
        python3-pip \
        python3-dev \
        rsync \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*


