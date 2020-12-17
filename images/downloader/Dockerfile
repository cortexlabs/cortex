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

COPY pkg/workloads/cortex/internal/requirements.txt /src/cortex/internal/requirements.txt
COPY pkg/workloads/cortex/downloader /src/cortex/downloader

RUN pip install --no-cache-dir -r \
    /src/cortex/internal/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/internal/ /src/cortex/internal
RUN pip install --no-deps /src/cortex/internal/ && \
    rm -rf /root/.cache/pip*

ENTRYPOINT ["/usr/bin/python3.6", "/src/cortex/downloader/download.py"]
