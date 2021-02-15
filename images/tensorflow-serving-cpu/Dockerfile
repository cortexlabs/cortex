FROM tensorflow/serving:2.4.1

RUN apt-get update -qq && apt-get install -y -q \
    curl \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY images/tensorflow-serving-cpu/run.sh /src/
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
