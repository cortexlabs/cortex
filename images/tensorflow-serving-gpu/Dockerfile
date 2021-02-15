FROM tensorflow/serving:2.4.0-gpu

# https://www.tensorflow.org/install/gpu
RUN apt-get update -qq && apt-get install -y --no-install-recommends -q \
        libnvinfer7=7.1.3-1+cuda11.0 \
        libnvinfer-plugin7=7.1.3-1+cuda11.0 \
        curl \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY images/tensorflow-serving-gpu/run.sh /src/
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
