FROM tensorflow/serving:2.3.0-gpu

RUN apt-get update -qq && apt-get install -y --no-install-recommends -q \
        libnvinfer6=6.0.1-1+cuda10.1 \
        libnvinfer-plugin6=6.0.1-1+cuda10.1 \
        curl \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY images/tensorflow-serving-gpu/run.sh /src/
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
