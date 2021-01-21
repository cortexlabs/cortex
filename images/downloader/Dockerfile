FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
        curl \
        python3.6 \
        python3.6-distutils \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3.6 get-pip.py && \
    pip install --upgrade pip && \
    rm -rf /root/.cache/pip*

COPY pkg/cortex/serve/cortex_internal.requirements.txt /src/cortex/serve/cortex_internal.requirements.txt

RUN pip install --no-cache-dir \
    -r /src/cortex/serve/cortex_internal.requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/cortex/downloader /src/cortex/downloader
COPY pkg/cortex/serve/ /src/cortex/serve
ENV CORTEX_LOG_CONFIG_FILE /src/cortex/serve/log_config.yaml

RUN pip install --no-deps /src/cortex/serve/ && \
    rm -rf /root/.cache/pip*

ENTRYPOINT ["/usr/bin/python3.6", "/src/cortex/downloader/download.py"]
