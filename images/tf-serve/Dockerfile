FROM tensorflow/serving:2.1.0

RUN apt-get update -qq && apt-get install -y --no-install-recommends -q \
        netcat \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*