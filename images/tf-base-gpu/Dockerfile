FROM tensorflow/tensorflow:1.12.0-gpu-py3

RUN apt-get update -qq && apt-get install -y -q \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*
