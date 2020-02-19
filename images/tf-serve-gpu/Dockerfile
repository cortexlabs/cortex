FROM tensorflow/serving:2.1.0-gpu

RUN apt-get update -qq && apt-get install -y -q \
        libnvinfer7 \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*
