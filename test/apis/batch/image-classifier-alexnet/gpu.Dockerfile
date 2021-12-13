FROM nvidia/cuda:10.2-cudnn8-runtime-ubuntu18.04

RUN apt-get update \
    && apt-get install -y \
        python3 \
        python3-pip \
        pkg-config \
        build-essential \
        git \
        cmake \
        libjpeg8-dev \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

ENV LC_ALL=C.UTF-8 LANG=C.UTF-8
ENV PYTHONUNBUFFERED TRUE

COPY app/requirements-gpu.txt /app/requirements.txt
RUN pip3 install --no-cache-dir -r /app/requirements.txt

COPY app /app
WORKDIR /app/
ENV PYTHONPATH=/app

ENV CORTEX_PORT=8080
CMD uvicorn --workers 1 --host 0.0.0.0 --port $CORTEX_PORT main:app
